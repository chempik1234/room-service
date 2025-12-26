package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func (s *RoomService) sendError(ctx context.Context, stream grpc.BidiStreamingServer[r.Command, r.Event], baseEvent *r.Event, err error) {
	err2 := stream.Send(&r.Event{
		Timestamp: baseEvent.Timestamp,
		RoomId:    baseEvent.RoomId,
		UserId:    baseEvent.UserId,
		Payload:   &r.Event_ErrorMessage{ErrorMessage: &r.ErrorMessage{Error: err.Error()}},
	})
	if err2 != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to send error", zap.Error(err), zap.String("original_error", err.Error()))
	}
}

// getValidRoomID - func that's separated from processCommand
//
// if it's CreateRoom, we return empty roomID
//
// if it's other command, we ensure roomID is valid.
func (s *RoomService) getValidRoomID(in *r.Command) (roomIDValidated *models.RoomID, err error) {
	// it's only omitted in create room
	switch _ := in.Payload.(type) {
	case *r.Command_CreateRoom:
		// skip, generate locally
		break
	default:
		_roomIdParsed, err := types.NewUUID(in.GetRoomId())
		if err != nil {
			return nil, fmt.Errorf("room id '%s' - invalid uuid", in.GetRoomId())
		}
		_v := models.RoomID(_roomIdParsed)
		roomIDValidated = &_v
		break
	}

	return roomIDValidated, nil
}

func (s *RoomService) getJoinedUserFull(user *r.User) (joinedUserIDValid types.NotEmptyText, joinedUserNameValid types.NotEmptyText, joinedUserMetadata map[string]string, err error) {
	joinedUserIDValid, err = types.NewNotEmptyText(user.GetId())
	if err != nil {
		return joinedUserIDValid, joinedUserNameValid, joinedUserMetadata, fmt.Errorf("user_full userID is empty")
	}
	joinedUserNameValid, err = types.NewNotEmptyText(user.GetName())
	if err != nil {
		return joinedUserIDValid, joinedUserNameValid, joinedUserMetadata, fmt.Errorf("user_full name is empty")
	}

	joinedUserMetadata = user.GetMetadata()

	return joinedUserIDValid, joinedUserNameValid, joinedUserMetadata, nil
}

func (s *RoomService) getKickedUserID(leaveRoom *r.LeaveRoomCommandBody) (kickedUserIDValid types.NotEmptyText, err error) {
	kickedUserIDValid, err = types.NewNotEmptyText(leaveRoom.GetKickedUserId())
	if err != nil {
		return kickedUserIDValid, fmt.Errorf("kicked_user_id is empty")
	}
	return kickedUserIDValid, nil
}

// noRepeatCommandID - get, save and return commandID.
//
// commandID is valid if it's command hasn't been executed already
//
// if it's valid, it's saved, else -> error
//
// If commandID is empty, no checks are applied
func (s *RoomService) noRepeatCommandID(ctx context.Context, in *r.Command) (string, error) {
	commandID := in.GetCommandId()
	if len(commandID) > 0 {
		_, commandIDExists, err := s.commandIdShortCache.Get(ctx, commandID)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to get command_id from short cache", zap.String(commandIDZapKey, commandID), zap.Error(err))
			return "", fmt.Errorf("failed to get command_id from short cache: %w", err)
		}
		if commandIDExists {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "command_id exists in short cache, SKIPPED", zap.String(commandIDZapKey, commandID))
			return "", fmt.Errorf("command_id '%s' exists in short cache, SKIPPED", commandID)
		}
		if s.commandIdShortCache.Set(ctx, in.CommandId, struct{}{}) != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set command_id into short cache", zap.String(commandIDZapKey, commandID), zap.Error(err))
		}
	}
	return commandID, nil
}
