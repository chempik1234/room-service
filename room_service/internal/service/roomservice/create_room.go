package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
	"github.com/wb-go/wbf/retry"
)

func (s *RoomService) createRoom(ctx context.Context, userID types.NotEmptyText, payload *r.Command_CreateRoom) (roomID models.RoomID, roomCreatedPayload *r.Event_RoomCreated, err error) {
	newRoom := models.NewRoom(userID, payload.CreateRoom.GetRoomOptions())

	//region create room logic
	err = retry.Do(func() error {
		var err error
		newRoom, err = s.roomsRepo.CreateRoom(ctx, newRoom)
		if err != nil {
			return err
		}
		return nil
	}, s.retryStrategy)
	if err != nil {
		return roomID, roomCreatedPayload, fmt.Errorf("failed to create room: %w", err)
	}
	//endregion

	roomID = newRoom.ID

	// result
	return roomID, &r.Event_RoomCreated{
		RoomCreated: &r.RoomCreatedEventBody{
			RoomOptions: newRoom.Options,
			RoomId:      newRoom.ID.String(),
		},
	}, nil
}
