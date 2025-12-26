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

func (m MongoDBRepository) CreateRoom(ctx context.Context, params *models.Room) (room *models.Room, err error) {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) DeleteRoom(ctx context.Context, params ports.DeleteRoomParams) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) JoinRoom(ctx context.Context, params ports.JoinRoomParams) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) IsRoomOwner(ctx context.Context, params ports.IsRoomOwnerParams) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) LeaveRoom(ctx context.Context, param ports.LeaveRoomParams) error {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) RoomSnapshot(ctx context.Context, params ports.RoomSnapshotParams) (*models.RoomSnapshot, error) {
	//TODO implement me
	panic("implement me")
}

func (m MongoDBRepository) AffectData(ctx context.Context, params ports.AffectDataParams) error {
	//TODO implement me
	panic("implement me")
}
