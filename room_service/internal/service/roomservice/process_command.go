package roomservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	"github.com/chempik1234/room-service/internal/projectutils"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"go.uber.org/zap"
)

const (
	roomIDZapKey       = "room_id"
	joinedUserIDZapKey = "joined_user_id"
	kickedUserIDZapKey = "kicked_user_id"
	ownerUserIDZapKey  = "owner_user_id"
)

// processCommand - idempotent processes commands, returns error on internal failure
func (s *RoomService) processCommand(ctx context.Context, in *r.Command) (returnEvent *r.Event, ok bool, internalErr error) {
	//check command id no-repeat
	// no return: error's put into returnedEvent later
	commandID, err := s.noRepeatCommandID(ctx, in)

	returnEvent = &r.Event{
		Timestamp: projectutils.NowTimestamp(),
		RoomId:    in.GetRoomId(),
		UserId:    in.GetUserId(),
		Payload:   nil,
	}

	// early return on DDoS/duplicated error
	if err != nil {
		returnEvent = quickErrorEvent(returnEvent.RoomId, returnEvent.UserId, err.Error())
		return returnEvent, false, nil
	}

	logger.GetLoggerFromCtx(ctx).Info(ctx, "command received, processing", zap.String(commandIDZapKey, commandID))

	//region validate userID, roomID

	// user id is always required, so we validate it before switch
	var userIDValid types.NotEmptyText

	userIDValid, err = types.NewNotEmptyText(in.GetUserId())
	if err != nil {
		err = errors.New("userID is empty")
		// no return: error's put into returnedEvent later
	}

	// room id isn't really required in all commands, so we check it only when really need
	var roomIDValidated *models.RoomID
	if err == nil {
		roomIDValidated, err = s.getValidRoomID(in) // inside: check only if needed
		if err != nil {
			err = fmt.Errorf("failed to get valid room id: %w", err)
		}
		// no return: error's put into returnedEvent later
	}
	//endregion

	// We need 2 errors: err for client-side problems and internalErr for internal problems.
	// The latter is returned and logged, first one is to slap the client

	// proceed if both userID and roomID are correct
	if err == nil {

		switch payload := in.Payload.(type) {
		case *r.Command_CreateRoom:
			var roomID models.RoomID
			roomID, returnEvent.Payload, internalErr = s.createRoom(ctx, userIDValid, payload)
			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to create room", zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to create room: %w", internalErr) // it's put into returnedEvent later
				break
			}

			returnEvent.RoomId = roomID.String()
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "created room", zap.Stringer(roomIDZapKey, &roomID))

			//endregion
		case *r.Command_DeleteRoom:
			returnEvent.Payload, internalErr = s.deleteRoom(ctx, userIDValid, roomIDValidated)
			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to delete room", zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to delete room: %w", internalErr) // it's put into returnedEvent later
				break
			}
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "deleted room", zap.Stringer(roomIDZapKey, roomIDValidated))
			//endregion
		case *r.Command_JoinRoom:
			var joinedUserID, joinedUserName types.NotEmptyText
			var joinedUserMetadata map[string]string

			joinedUserID, joinedUserName, joinedUserMetadata, err = s.getJoinedUserFull(payload.JoinRoom.UserFull)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid join room params from client", zap.Error(err))
				err = fmt.Errorf("invalid join room params from client: %w", err) // it's put into returnedEvent later
				break
			}
			returnEvent.Payload, internalErr = s.joinRoom(ctx, &roomServiceJoinRoomParams{
				roomID:             roomIDValidated,
				joinedUserID:       joinedUserID,
				joinedUserName:     joinedUserName,
				joinedUserMetadata: joinedUserMetadata,
			})
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "joined room",
				zap.Stringer(roomIDZapKey, roomIDValidated),
				zap.Stringer(joinedUserIDZapKey, joinedUserID))
			//endregion
		case *r.Command_LeaveRoom:
			var kickedUserIDValid types.NotEmptyText
			kickedUserIDValid, err = s.getKickedUserID(payload.LeaveRoom)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid kicked_user_id", zap.Error(err))
				err = fmt.Errorf("invalid kicked_user_id: %w", err) // it's put into returnedEvent later
				break
			}

			returnEvent.Payload, internalErr = s.leaveRoom(ctx, &leaveRoomParams{
				roomID:       roomIDValidated,
				userID:       userIDValid,
				kickedUserID: kickedUserIDValid,
			})
			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to leave room", zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to leave room: %w", internalErr) // it's put into returnedEvent later
				break
			}

			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "user was kicked from room",
				zap.Stringer(roomIDZapKey, roomIDValidated),
				zap.Stringer(kickedUserIDZapKey, kickedUserIDValid))
		case *r.Command_AffectData:
			itemIndex := ""
			if payload.AffectData.ItemIndex != nil {
				itemIndex = *payload.AffectData.ItemIndex
			}

			returnEvent.Payload, internalErr = s.affectDataInRoom(ctx,
				payload.AffectData.DataValue,
				payload.AffectData.CommandMode,
				&affectDataParams{
					RoomID:    roomIDValidated,
					DataID:    types.NewAnyText(payload.AffectData.DataId),
					ItemIndex: types.NewAnyText(itemIndex),
					Action:    ports.Action(payload.AffectData.CommandMode),
				})
			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to affect data in room", zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to affect data in room: %w", internalErr) // it's put into returnedEvent later
				break
			}

			logger.GetLoggerFromCtx(ctx).Info(ctx, "changed data in room",
				zap.Stringer(roomIDZapKey, roomIDValidated))
		case *r.Command_SetOwner:
			var newOwnerID types.NotEmptyText
			newOwnerID, err = types.NewNotEmptyText(payload.SetOwner.GetNewOwnerId())
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid new_owner_id from client", zap.Error(err))
				err = fmt.Errorf("invalid new_owner_id from client: %w", err)
				break
			}
			returnEvent.Payload, internalErr = s.setOwner(ctx, &roomServiceSetOwnerParams{
				roomID:     roomIDValidated,
				newOwnerID: newOwnerID,
			})

			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set owner of room",
					zap.Stringer(roomIDZapKey, roomIDValidated),
					zap.Stringer(ownerUserIDZapKey, newOwnerID),
					zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to set owner of room: %w", internalErr)
			}

		case *r.Command_RefreshRoom:
			// TODO: rate limiter per room
			returnEvent.Payload, internalErr = s.refreshRoom(ctx, roomIDValidated)
			if internalErr != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to refresh room", zap.Error(internalErr))
				internalErr = fmt.Errorf("failed to refresh room: %w", internalErr) // it's put into returnedEvent later
				break
			}

			logger.GetLoggerFromCtx(ctx).Info(ctx, "asked for full room",
				zap.Stringer(roomIDZapKey, roomIDValidated))

		default:
			panic("unknown type of command payload")
		}
	}

	ok = true

	if err != nil {
		returnEvent = quickErrorEvent(returnEvent.RoomId, returnEvent.UserId, err.Error())
		ok = false
	} else if internalErr != nil {
		returnEvent = quickErrorEvent(returnEvent.RoomId, returnEvent.UserId, internalErr.Error())
		ok = false
	} else if returnEvent.Payload == nil {
		returnEvent = quickErrorEvent(returnEvent.RoomId, returnEvent.UserId, "nil payload returned")
		ok = false
	}

	return returnEvent, ok, internalErr
}
