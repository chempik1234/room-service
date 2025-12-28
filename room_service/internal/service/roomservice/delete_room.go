package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"github.com/wb-go/wbf/retry"
)

func (s *RoomService) deleteRoom(ctx context.Context, userID types.NotEmptyText, roomID *models.RoomID) (payload *r.Event_RoomDeleted, err error) {
	//region check if userID is owner of room
	var isRoomOwner bool
	err = retry.Do(func() error {
		var errRepo error
		isRoomOwner, errRepo = s.roomsRepo.IsRoomOwner(ctx, ports.IsRoomOwnerParams{
			RoomID: *roomID,
			UserID: userID,
		})
		return errRepo
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to check if user is room owner: %w", err)
	}
	if !isRoomOwner {
		return payload, fmt.Errorf("user '%s' is not a room owner (%s)",
			userID.String(),
			roomID.String())
	}
	//endregion

	//region delete room logic
	err = retry.Do(func() error {
		return s.roomsRepo.DeleteRoom(ctx, ports.DeleteRoomParams{
			RoomID: *roomID,
			UserID: userID,
		})
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to join room: %w", err)
	}
	//endregion

	return &r.Event_RoomDeleted{
		RoomDeleted: &r.RoomDeletedEventBody{
			DeletedRoomId: roomID.String(),
		},
	}, nil
}
