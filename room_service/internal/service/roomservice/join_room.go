package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"github.com/wb-go/wbf/retry"
	"go.uber.org/zap"
)

type roomServiceJoinRoomParams struct {
	roomID             *models.RoomID
	joinedUserID       types.NotEmptyText
	joinedUserName     types.NotEmptyText
	joinedUserMetadata map[string]string
}

func (s *RoomService) joinRoom(ctx context.Context, params *roomServiceJoinRoomParams) (payload *r.Event_JoinedRoom, err error) {
	userModel := models.User{
		Metadata: params.joinedUserMetadata,
		ID:       params.joinedUserID,
		Name:     params.joinedUserName,
	}

	//region join room logic
	err = retry.Do(func() error {
		return s.roomsRepo.JoinRoom(ctx, ports.JoinRoomParams{
			RoomID:   *params.roomID,
			UserFull: userModel,
		})
	}, s.retryStrategy)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to join room", zap.Error(err))
		return payload, fmt.Errorf("failed to join room: %w", err)
	}
	//endregion

	return &r.Event_JoinedRoom{
		JoinedRoom: &r.JoinedRoomEventBody{
			UserFull: &r.User{
				Id:       userModel.ID.String(),
				Name:     userModel.Name.String(),
				Metadata: userModel.Metadata,
			},
			RoomId: types.UUID(*params.roomID).String(),
		},
	}, nil
}
