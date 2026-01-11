package mongodb

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/chempik1234/room-service/internal/errors"
	"github.com/chempik1234/room-service/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDBRepository) roomExists(ctx context.Context, id string) (bool, error) {
	var res bson.M
	err := s.roomsCollection.FindOne(ctx, bson.M{"id": id}).Decode(&res)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error while checking if room exists: %w", err)
	}
	return true, nil
}

// errIfRoomExists - return err if room ID already exists or DB error, else nil
//
// Returned error is already wrapped
func (s *MongoDBRepository) errIfRoomExists(ctx context.Context, id string) error {
	idExists, err := s.roomExists(ctx, id)
	if err != nil {
		return err
	}

	if idExists {
		return errors2.ErrQuick(errors2.ErrRoomIDAlreadyExists, id)
	}

	return nil
}

// errIfRoomDoesntExist - return err if room ID doesn't exist or DB error, else nil
//
// Returned error is already wrapped
func (s *MongoDBRepository) errIfRoomDoesntExist(ctx context.Context, id string) error {
	idExists, err := s.roomExists(ctx, id)
	if err != nil {
		return err
	}

	if !idExists {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, id)
	}

	return nil
}

// valueInRoom - return whole value by data_id
//
// errors are wrapped already
func (s *MongoDBRepository) valueInRoom(ctx context.Context, roomID string, dataID string) (*models.Value, error) {
	valuePathInDB := "data." + dataID
	raw, err := s.roomsCollection.FindOne(ctx, bson.M{"id": roomID}, options.FindOne().SetProjection(bson.M{valuePathInDB: 1})).Raw()
	if err != nil {
		return nil, fmt.Errorf("error while querying room value (data.%s): %w", dataID, err)
	}
	var val bson.RawValue
	if val, err = raw.LookupErr(valuePathInDB); err != nil {
		return nil, fmt.Errorf("error while querying room value (data.%s): queried data_id not in result", dataID)
	}
	result, err := decodeBSONValueToModelValue(val)
	if err != nil {
		return nil, fmt.Errorf("error while decoding room value (data.%s): %w", dataID, err)
	}
	return result, nil
}
