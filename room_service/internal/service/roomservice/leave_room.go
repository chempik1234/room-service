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

type leaveRoomParams struct {
	roomID       *models.RoomID
	userID       types.NotEmptyText
	kickedUserID types.NotEmptyText
}

func (s *RoomService) leaveRoom(ctx context.Context, params *leaveRoomParams) (payload *r.Event_LeftRoom, err error) {
	err = retry.Do(func() error {
		errRepo := s.roomsRepo.LeaveRoom(ctx, ports.LeaveRoomParams{
			RoomID:              *params.roomID,
			CommandCallerUserID: params.userID,
			KickedUserID:        params.kickedUserID,
		})
		if errRepo != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to leave a room, retrying", zap.Error(err))
		}
		return errRepo
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to leave room: %w", err)
	}

	// Check if the user who left was the owner, and if so, assign a new random owner
	isOwner, err := s.roomsRepo.IsRoomOwner(ctx, ports.IsRoomOwnerParams{
		RoomID: *params.roomID,
		UserID: params.kickedUserID,
	})
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "failed to check if user is owner, skipping owner reassignment", zap.Error(err))
		return &r.Event_LeftRoom{
			LeftRoom: &r.LeftRoomEventBody{
				KickedUserId: params.kickedUserID.String(),
				RoomId:       params.roomID.String(),
			},
		}, nil
	}

	if isOwner {
		snapshot, err := s.roomsRepo.RoomSnapshot(ctx, ports.RoomSnapshotParams{
			RoomID: *params.roomID,
		})
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Warn(ctx, "failed to get room snapshot, cannot assign new owner", zap.Error(err))
		} else if len(snapshot.Users) > 0 {
			// Pick a random user from the remaining users as the new owner
			newOwner := snapshot.Users[0]

			_, err = s.roomsRepo.SetOwnerUserID(ctx, ports.SetOwnerUserIDParams{
				RoomID:     *params.roomID,
				NewOwnerID: newOwner.ID,
			})
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set new owner", zap.Error(err))
			} else {
				logger.GetLoggerFromCtx(ctx).Info(ctx, "assigned new owner after previous owner left",
					zap.String("new_owner_id", newOwner.ID.String()),
					zap.String("room_id", params.roomID.String()))
			}
		}
	}

	return &r.Event_LeftRoom{
		LeftRoom: &r.LeftRoomEventBody{
			KickedUserId: params.kickedUserID.String(),
			RoomId:       params.roomID.String(),
		},
	}, nil
}
