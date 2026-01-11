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

func (s *RoomService) processCommand(ctx context.Context, in *r.Command) (*r.Event, error) {
	var err error

	//check command id no-repeat
	commandID, err := s.noRepeatCommandID(ctx, in)
	logger.GetLoggerFromCtx(ctx).Info(ctx, "command received, processing", zap.String(commandIDZapKey, commandID))

	returnEvent := &r.Event{
		Timestamp: projectutils.NowTimestamp(),
		RoomId:    in.GetRoomId(),
		UserId:    in.GetUserId(),
		Payload:   nil,
	}

	//region validate userID

	// user id is always required, so we validate it before switch
	// room id isn't really required in all commands, so we check it only when really need
	var userIDValid types.NotEmptyText
	userIDValid, err = types.NewNotEmptyText(in.GetUserId())
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "someone entered empty userID")
		err = errors.New("userID is empty") // it's put into returnedEvent later
	}
	//endregion

	//region validate roomID
	roomIDValidated, err := s.getValidRoomID(in)
	if err != nil {
		err = fmt.Errorf("failed to get valid room id: %w", err) // it's put into returnedEvent later
	}
	//endregion

	if err == nil {

		switch payload := in.Payload.(type) {
		case *r.Command_CreateRoom:
			var roomID models.RoomID
			roomID, returnEvent.Payload, err = s.createRoom(ctx, userIDValid, payload)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to create room", zap.Error(err))
				err = fmt.Errorf("failed to create room: %w", err) // it's put into returnedEvent later
				break
			}

			returnEvent.RoomId = roomID.String()
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "created room", zap.Stringer(roomIDZapKey, &roomID))

			break
			//endregion
		case *r.Command_DeleteRoom:
			returnEvent.Payload, err = s.deleteRoom(ctx, userIDValid, roomIDValidated)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to delete room", zap.Error(err))
				err = fmt.Errorf("failed to delete room: %w", err) // it's put into returnedEvent later
				break
			}
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "deleted room", zap.Stringer(roomIDZapKey, roomIDValidated))
			break
			//endregion
		case *r.Command_JoinRoom:
			joinedUserID, joinedUserName, joinedUserMetadata, err := s.getJoinedUserFull(payload.JoinRoom.UserFull)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid join room params from client", zap.Error(err))
				err = fmt.Errorf("invalid join room params from client: %w", err) // it's put into returnedEvent later
				break
			}
			returnEvent.Payload, err = s.joinRoom(ctx, &roomServiceJoinRoomParams{
				roomID:             roomIDValidated,
				joinedUserID:       joinedUserID,
				joinedUserName:     joinedUserName,
				joinedUserMetadata: joinedUserMetadata,
			})
			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "joined room",
				zap.Stringer(roomIDZapKey, roomIDValidated),
				zap.Stringer(joinedUserIDZapKey, joinedUserID))
			break
			//endregion
		case *r.Command_LeaveRoom:
			kickedUserIDValid, err := s.getKickedUserID(payload.LeaveRoom)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid kicked_user_id", zap.Error(err))
				err = fmt.Errorf("invalid kicked_user_id: %w", err) // it's put into returnedEvent later
				break
			}

			returnEvent.Payload, err = s.leaveRoom(ctx, &leaveRoomParams{
				roomID:       roomIDValidated,
				userID:       userIDValid,
				kickedUserID: kickedUserIDValid,
			})
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to leave room", zap.Error(err))
				err = fmt.Errorf("failed to leave room: %w", err) // it's put into returnedEvent later
				break
			}

			logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "user was kicked from room",
				zap.Stringer(roomIDZapKey, roomIDValidated),
				zap.Stringer(kickedUserIDZapKey, kickedUserIDValid))

			break
		case *r.Command_AffectData:
			itemIndex := ""
			if payload.AffectData.ItemIndex != nil {
				itemIndex = *payload.AffectData.ItemIndex
			}

			returnEvent.Payload, err = s.affectDataInRoom(ctx,
				payload.AffectData.DataValue,
				payload.AffectData.CommandMode,
				&affectDataParams{
					RoomID:    roomIDValidated,
					DataID:    types.NewAnyText(payload.AffectData.DataId),
					ItemIndex: types.NewAnyText(itemIndex),
					Action:    ports.Action(payload.AffectData.CommandMode),
				})
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to affect data in room", zap.Error(err))
				err = fmt.Errorf("failed to affect data in room: %w", err) // it's put into returnedEvent later
				break
			}

			logger.GetLoggerFromCtx(ctx).Info(ctx, "changed data in room",
				zap.Stringer(roomIDZapKey, roomIDValidated))

			break
		case *r.Command_SetOwner:
			var newOwnerID types.NotEmptyText
			newOwnerID, err = types.NewNotEmptyText(payload.SetOwner.GetNewOwnerId())
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid new_owner_id from client", zap.Error(err))
				err = fmt.Errorf("invalid new_owner_id from client: %w", err)
				break
			}
			returnEvent.Payload, err = s.setOwner(ctx, &roomServiceSetOwnerParams{
				roomID:     roomIDValidated,
				newOwnerID: newOwnerID,
			})

			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set owner of room",
					zap.Stringer(roomIDZapKey, roomIDValidated),
					zap.Stringer(ownerUserIDZapKey, newOwnerID),
					zap.Error(err))
				err = fmt.Errorf("failed to set owner of room: %w", err)
			}

			break
		case *r.Command_RefreshRoom:
			// TODO: rate limiter per room
			returnEvent.Payload, err = s.refreshRoom(ctx, roomIDValidated)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to refresh room", zap.Error(err))
				err = fmt.Errorf("failed to refresh room: %w", err) // it's put into returnedEvent later
				break
			}

			logger.GetLoggerFromCtx(ctx).Info(ctx, "asked for full room",
				zap.Stringer(roomIDZapKey, roomIDValidated))

			break
		default:
			panic("unknown type of command payload")
		}
	}

	if returnEvent.Payload == nil || err != nil {
		returnEvent = quickErrorEvent(returnEvent.RoomId, returnEvent.UserId, err.Error())
	}

	return returnEvent, err
}
