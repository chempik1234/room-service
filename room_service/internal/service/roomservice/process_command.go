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
		return returnEvent, errors.New("userID is empty")
	}
	//endregion

	//region validate roomID
	roomIDValidated, err := s.getValidRoomID(in)
	if err != nil {
		return returnEvent, fmt.Errorf("failed to get valid room id: %w", err)
	}
	//endregion

	switch payload := in.Payload.(type) {
	case *r.Command_CreateRoom:
		var roomID models.RoomID
		roomID, returnEvent.Payload, err = s.createRoom(ctx, userIDValid, payload)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to create room", zap.Error(err))
		}
		returnEvent.RoomId = roomID.String()
		break
		//endregion
	case *r.Command_DeleteRoom:
		returnEvent.Payload, err = s.deleteRoom(ctx, userIDValid, roomIDValidated)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to delete room", zap.Error(err))
		}
		break
		//endregion
	case *r.Command_JoinRoom:
		joinedUserID, joinedUserName, joinedUserMetadata, err := s.getJoinedUserFull(payload.JoinRoom.UserFull)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to get joined user full", zap.Error(err))
			return returnEvent, fmt.Errorf("failed to get joined user full: %w", err)
		}
		returnEvent.Payload, err = s.joinRoom(ctx, &roomServiceJoinRoomParams{
			roomID:             roomIDValidated,
			joinedUserID:       joinedUserID,
			joinedUserName:     joinedUserName,
			joinedUserMetadata: joinedUserMetadata,
		})
		break
		//endregion
	case *r.Command_LeaveRoom:
		kickedUserIDValid, err := s.getKickedUserID(payload.LeaveRoom)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to get kicked user id", zap.Error(err))
			return returnEvent, fmt.Errorf("failed to get kicked user id: %w", err)
		}

		returnEvent.Payload, err = s.leaveRoom(ctx, &leaveRoomParams{
			roomID:       roomIDValidated,
			userID:       userIDValid,
			kickedUserID: kickedUserIDValid,
		})
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to leave room", zap.Error(err))
			return returnEvent, fmt.Errorf("failed to leave room: %w", err)
		}
		break
	case *r.Command_AffectData:
		returnEvent.Payload, err = s.affectDataInRoom(ctx,
			payload.AffectData.DataValue,
			payload.AffectData.CommandMode,
			&affectDataParams{
				RoomID: roomIDValidated,
				DataID: types.NewAnyText(payload.AffectData.DataId),
				Action: ports.Action(payload.AffectData.CommandMode),
			})
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to affect data in room", zap.Error(err))
			return returnEvent, fmt.Errorf("failed to affect data in room: %w", err)
		}
		break
	case *r.Command_RefreshRoom:
		// TODO: rate limiter per room
		returnEvent.Payload, err = s.refreshRoom(ctx, roomIDValidated)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to refresh room", zap.Error(err))
			return returnEvent, fmt.Errorf("failed to refresh room: %w", err)
		}
		break
	default:
		panic("unknown type of command payload")
	}

	return returnEvent, err
}
