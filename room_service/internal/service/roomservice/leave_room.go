package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
	"github.com/wb-go/wbf/retry"
)

type leaveRoomParams struct {
	roomID       *models.RoomID
	userID       types.NotEmptyText
	kickedUserID types.NotEmptyText
}

func (s *RoomService) leaveRoom(ctx context.Context, params *leaveRoomParams) (payload *r.Event_LeftRoom, err error) {
	err = retry.Do(func() error {
		return s.roomsRepo.LeaveRoom(ctx, ports.LeaveRoomParams{
			RoomID:              *params.roomID,
			CommandCallerUserID: params.userID,
			KickedUserID:        params.kickedUserID,
		})
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to leave room: %w", err)
	}

	return &r.Event_LeftRoom{
		LeftRoom: &r.LeftRoomEventBody{
			KickedUserId: params.kickedUserID.String(),
			RoomId:       params.roomID.String(),
		},
	}, nil
}
