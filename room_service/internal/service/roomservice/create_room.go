package roomservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"github.com/wb-go/wbf/retry"
)

func (s *RoomService) createRoom(ctx context.Context, userID types.NotEmptyText, payload *r.Command_CreateRoom) (roomID models.RoomID, roomCreatedPayload *r.Event_RoomCreated, err error) {
	newRoom := models.NewRoom(userID, payload.CreateRoom.GetRoomOptions())

	var resultRoom *models.Room

	//region create room logic
	err = retry.Do(func() error {
		if newRoom == nil {
			return errors.New("room creation failed (internal error): newRoom is nil")
		}

		var err error

		resultRoom, err = s.roomsRepo.CreateRoom(ctx, newRoom) // don't rewrite like "newRoom, err = ..." !!!
		return err
	}, s.retryStrategy)
	if err != nil {
		return roomID, roomCreatedPayload, fmt.Errorf("failed to create room: %w", err)
	}
	//endregion

	roomID = resultRoom.ID

	// result
	return roomID, &r.Event_RoomCreated{
		RoomCreated: &r.RoomCreatedEventBody{
			RoomOptions: resultRoom.Options,
			RoomId:      resultRoom.ID.String(),
		},
	}, nil
}
