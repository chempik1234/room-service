package ports

import (
	"context"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
)

// RoomsPort - - port for "Room" and everything in it (userID, users' data managing)
type RoomsPort interface {
	// CreateRoom - "Create" method for "Room", generates and returns ID
	CreateRoom(ctx context.Context, params *models.Room) (room *models.Room, err error)
	// DeleteRoom - "Delete" method for "Room", error on not found
	DeleteRoom(ctx context.Context, params DeleteRoomParams) (err error)
	// JoinRoom - adds user to visitors of existing room (if room exists, error on not found), idempotent
	JoinRoom(ctx context.Context, params JoinRoomParams) (err error)
	// IsRoomOwner - returns true if user is owner of given room, used for security checks
	IsRoomOwner(ctx context.Context, params IsRoomOwnerParams) (bool, error)
	// LeaveRoom - kick user from one's room, either by himself or by admin
	//
	// When making a repo, use IsRoomOwner to check it
	LeaveRoom(ctx context.Context, param LeaveRoomParams) error
	// RoomSnapshot - return a whole sight on room - ownerID, room data KV, roomID...
	RoomSnapshot(ctx context.Context, params RoomSnapshotParams) (*models.RoomSnapshot, error)
	// AffectData - set/delete whole data field or item in dict/list (depends on what models.Value is stored)
	//
	// The whole data storage is a KV storage that can store different values, including lists and dicts
	AffectData(ctx context.Context, params AffectDataParams) error
}

// DeleteRoomParams - param set for RoomsPort.DeleteRoom method
type DeleteRoomParams struct {
	RoomID models.RoomID
	UserID types.NotEmptyText
}

// JoinRoomParams - param set for RoomsPort.JoinRoom method
type JoinRoomParams struct {
	RoomID   models.RoomID
	UserFull models.User
}

// LeaveRoomParams - param set for RoomsPort.LeaveRoom method
type LeaveRoomParams struct {
	RoomID              models.RoomID
	CommandCallerUserID types.NotEmptyText
	KickedUserID        types.NotEmptyText
}

// RoomSnapshotParams - param set for RoomsPort.RoomSnapshot method
type RoomSnapshotParams struct {
	RoomID models.RoomID
}

// IsRoomOwnerParams - param set for RoomsPort.JoinRoom method
type IsRoomOwnerParams struct {
	RoomID models.RoomID
	UserID types.NotEmptyText
}

// Action is type for data affection modes ENUM
//
// SET, DELETE, APPEND, REMOVE
type Action uint8

const (
	ActionSet    Action = iota
	ActionDelete Action = iota
	ActionAppend Action = iota
	ActionRemove Action = iota
)

// AffectDataParams - params for RoomsPort.AffectData
type AffectDataParams struct {
	RoomID models.RoomID
	DataID types.AnyText
	Action Action
	Value  *models.Value
}
