package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/wb-go/wbf/retry"
	"go.uber.org/zap"
)

func (s *RoomService) refreshRoom(ctx context.Context, roomID *models.RoomID) (payload *r.Event_FullRoom, err error) {
	//region snapshot room logic
	var room *models.RoomSnapshot
	err = retry.Do(func() error {
		var errSnapshot error
		room, errSnapshot = s.roomsRepo.RoomSnapshot(ctx, ports.RoomSnapshotParams{
			RoomID: *roomID,
		})
		return errSnapshot
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to get room snapshot: %w", err)
	}
	//endregion

	//region result
	roomUsers := make([]*r.User, len(room.Users))
	for i, user := range room.Users {
		roomUsers[i] = &r.User{
			Id:       user.ID.String(),
			Name:     user.Name.String(),
			Metadata: user.Metadata,
		}
	}

	roomValues := make(map[string]*r.Value, len(room.Values))

	for key, value := range room.Values {
		roomValues[key], err = PlainObjectToProtobufValue(value)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to get room snapshot", zap.Error(err))
			return payload, fmt.Errorf("failed to get room snapshot: %w", err)
		}
	}

	return &r.Event_FullRoom{
		FullRoom: &r.FullRoomSnapshotEventBody{
			Room:        &r.RoomData{Values: roomValues},
			Users:       roomUsers,
			RoomOptions: room.Room.Options,
			RoomId:      room.Room.ID.String(),
		},
	}, nil
}
