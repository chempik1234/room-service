package ports

import (
	"context"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
)

// RoomsPort - is the port for "Room" and everything in it (userID, users' data managing)
type RoomsPort interface {
	// CreateRoom is the "Create" method for "Room", generates and returns ID
	CreateRoom(ctx context.Context, params *models.Room) (room *models.Room, err error)
	// DeleteRoom is the "Delete" method for "Room", error on not found
	DeleteRoom(ctx context.Context, params DeleteRoomParams) (err error)
	// JoinRoom adds user to visitors of existing room (if room exists, error on not found), idempotent
	JoinRoom(ctx context.Context, params JoinRoomParams) (err error)
	IsRoomOwner(ctx context.Context, params IsRoomOwnerParams) (bool, error)
}

// DeleteRoomParams is the param set for RoomsPort.DeleteRoom method
type DeleteRoomParams struct {
	RoomID types.UUID
	UserID types.NotEmptyText
}

// JoinRoomParams is the param set for RoomsPort.JoinRoom method
type JoinRoomParams struct {
	RoomID   types.UUID
	UserFull models.User
}

// IsRoomOwnerParams is the param set for RoomsPort.JoinRoom method
type IsRoomOwnerParams struct {
	RoomID types.UUID
	UserID types.NotEmptyText
}
