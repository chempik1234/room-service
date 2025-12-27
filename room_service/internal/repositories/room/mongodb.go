package room

import (
	"context"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
)

// MongoDBRepository - ports.RoomsPort impl with MongoDB
type MongoDBRepository struct {
}

// NewMongoDBRepository - return new MongoDBRepository
func NewMongoDBRepository() *MongoDBRepository {
	return &MongoDBRepository{}
}

// CreateRoom - create room in MongoDB
//
// Create ID yourself
func (m MongoDBRepository) CreateRoom(ctx context.Context, params *models.Room) (room *models.Room, err error) {
	//TODO implement me
	panic("implement me")
}

// DeleteRoom - delete room from MongoDB with all data inside
//
// Not found -> errors.ErrRoomDoesntExist
func (m MongoDBRepository) DeleteRoom(ctx context.Context, params ports.DeleteRoomParams) (err error) {
	//TODO implement me
	panic("implement me")
}

// JoinRoom - add user to room in MongoDB
//
// Not found -> errors.ErrRoomDoesntExist
func (m MongoDBRepository) JoinRoom(ctx context.Context, params ports.JoinRoomParams) (err error) {
	//TODO implement me
	panic("implement me")
}

// IsRoomOwner - check if room's owner is given user (MongoDB)
//
// Not found -> errors.ErrRoomDoesntExist
func (m MongoDBRepository) IsRoomOwner(ctx context.Context, params ports.IsRoomOwnerParams) (bool, error) {
	//TODO implement me
	panic("implement me")
}

// LeaveRoom - remove user from room (MongoDB)
//
// Room not found -> errors.ErrRoomDoesntExist
// User not found -> errors.ErrUserNotInRoom
func (m MongoDBRepository) LeaveRoom(ctx context.Context, param ports.LeaveRoomParams) error {
	//TODO implement me
	panic("implement me")
}

// RoomSnapshot - return a whole sight on room - ownerID, room data KV, roomID... (MongoDB)
//
// Room not found -> errors.ErrRoomDoesntExist
func (m MongoDBRepository) RoomSnapshot(ctx context.Context, params ports.RoomSnapshotParams) (*models.RoomSnapshot, error) {
	//TODO implement me
	panic("implement me")
}

// AffectData - set/delete whole data field or item in dict/list (depends on what models.Value is stored)
//
// # The whole data storage is a KV storage that can store different values, including lists and dicts
//
// Room not found -> errors.ErrRoomDoesntExist
// Data not found -> errors.ErrDataPieceDoesntExist
func (m MongoDBRepository) AffectData(ctx context.Context, params ports.AffectDataParams) error {
	//TODO implement me
	panic("implement me")
}
