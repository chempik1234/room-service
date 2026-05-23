package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	errors2 "github.com/chempik1234/room-service/internal/errors"
	"github.com/chempik1234/room-service/internal/models"
	types2 "github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// setData - set data[data_id] = value in MongoDB
//
// if itemIndex != "" --> set data[data_id][itemIndex] = value
func (s *RoomsMongoDBRepository) setData(ctx context.Context, roomID, dataID string, value *models.Value, itemIndex string) (*models.Value, error) {
	// Convert models.Value to BSON value
	bsonValue, err := encodeModelValueToBSON(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value to BSON: %w", err)
	}

	filter := bson.M{roomIDField: roomID}

	// DEBUG: Verify room exists before update
	var roomDoc bson.M
	err = s.roomsCollection.FindOne(ctx, filter, options.FindOne().SetProjection(bson.M{roomIDField: 1})).Decode(&roomDoc)
	if err != nil {
		return nil, fmt.Errorf("room not found before update: %s (filter: %v)", err, filter)
	}
	fmt.Printf("🐛 Room exists check: found room %s in collection %s\n", roomID, s.roomsCollection.Name())

	var update bson.M
	if len(itemIndex) == 0 {
		update = bson.M{"$set": bson.M{fmt.Sprintf("%s.%s", roomDataField, dataID): bsonValue}}
	} else {
		// TODO: does is work for array?
		update = bson.M{"$set": bson.M{fmt.Sprintf("%s.%s.%s", roomDataField, dataID, itemIndex): bsonValue}}
	}

	result, err := s.roomsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("mongo updateOne set data error: %w", err)
	}

	// Verify the update actually happened by reading it back
	var verificationDoc bson.M
	err = s.roomsCollection.FindOne(ctx, filter).Decode(&verificationDoc)
	if err == nil {
		fmt.Printf("🐛 Verification: room exists after update, data field: %v\n", verificationDoc[roomDataField])
	}

	if result.MatchedCount == 0 {
		return nil, errors2.ErrQuick(errors2.ErrNoDataUpdated, map[string]bson.M{
			"filter": filter,
			"update": update,
		})
	}

	var resultValue *models.Value
	if len(itemIndex) == 0 {
		resultValue = value
	} else {
		if resultValue, err = s.valueInRoom(ctx, roomID, dataID); err != nil {
			return nil, err // already wrapped
		}
	}

	return resultValue, nil
}

// deleteData - delete data[data_id] from MongoDB
func (s *RoomsMongoDBRepository) deleteData(ctx context.Context, roomID, dataID string) (*models.Value, error) {
	// check if "data" field exists
	filter := bson.M{
		"id": roomID,
		fmt.Sprintf("%s.%s", roomDataField, dataID): bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return nil, errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
	}

	// delete the value from room.data
	updateFilter := bson.M{"id": roomID}
	update := bson.M{"$unset": bson.M{fmt.Sprintf("%s.%s", roomDataField, dataID): ""}}

	result, err := s.roomsCollection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return nil, fmt.Errorf("mongo updateOne delete data error: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, errors2.ErrQuick(errors2.ErrRoomDoesntExist, roomID)
	}

	return nil, nil
}

// appendData - append value to data[data_id] (only array)
func (s *RoomsMongoDBRepository) appendData(ctx context.Context, roomID, dataID string, value *models.Value) (*models.Value, error) {
	// models.Value to BSON
	bsonValue, err := encodeModelValueToBSON(value)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value to BSON: %w", err)
	}

	// check if room."data" exists
	filter := bson.M{
		"id": roomID,
		fmt.Sprintf("%s.%s", roomDataField, dataID): bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return nil, errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
	}

	// Determine if we're appending to array or map
	// try $push (for arrays) first
	filter = bson.M{"id": roomID}
	update := bson.M{"$push": bson.M{fmt.Sprintf("%s.%s", roomDataField, dataID): bsonValue}}

	result, err := s.roomsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		// GPT note:
		// If $push fails, it might be a map, try $mergeObjects or set for nested map
		// For now, just return the error
		return nil, fmt.Errorf("mongo updateOne append data error: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, errors2.ErrQuick(errors2.ErrRoomDoesntExist, roomID)
	}

	var resultValue *models.Value
	if resultValue, err = s.valueInRoom(ctx, roomID, dataID); err != nil {
		return nil, err // already wrapped
	}

	return resultValue, nil
}

// removeData - remove item from data[data_id] by index/key
//
// if itemIndex != "" --> remove item data[data_id][itemIndex]
func (s *RoomsMongoDBRepository) removeData(ctx context.Context, roomID, dataID string, itemIndex types2.NotEmptyText) (*models.Value, error) {
	// check if data_id in "data" field exists
	removedItemPath := fmt.Sprintf("%s.%s", roomDataField, dataID)

	filter := bson.M{
		"id":            roomID,
		removedItemPath: bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return nil, errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
	}

	// For REMOVE, we need to:
	// 0. Get data[data_id] value
	// 1. Determine if data[data_id] is a list or map
	// 2. Remove the element at that index/key

	// step 0.
	var doc bson.M // { "data.%s": <val> }
	err = s.roomsCollection.FindOne(ctx,
		bson.M{"id": roomID},
		options.FindOne().SetProjection(
			bson.M{
				fmt.Sprintf("%s.%s", roomDataField, dataID): 1,
			},
		)).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("failed to get room's data by roomID: %w", err)
	}

	// step 1.
	if value, ok := doc[roomDataField].(bson.M)[dataID]; ok {
		switch valueType := value.(type) {
		case bson.A:
			// we need int for indexing in-memory: we find item by value, not by key, which is awful!
			indexInt, err := strconv.Atoi(itemIndex.String())
			if err != nil {
				return nil, fmt.Errorf("failed to convert itemIndex to int (item is array): %w", err)
			}

			// step 2.
			// uses retry inside
			res, err := s.roomsCollection.UpdateOne(ctx, filter,
				bson.M{"$pull": bson.M{fmt.Sprintf("%s.%s", roomDataField, dataID): bson.M{"$eq": value.([]any)[indexInt]}}})
			// {"$pull": {"data.<data_id>": {"eq": <val data[<data_id>][<itemIndex>]> } } }
			if err != nil {
				return nil, fmt.Errorf("mongo updateOne $pull data error (dataId: '%s', array_index: '%d'): %w", dataID, indexInt, err)
			}
			if res.ModifiedCount == 0 {
				return nil, errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, fmt.Errorf("%s (array_index='%d')", dataID, indexInt))
			}

		case bson.M:
			// step 2.
			// uses retry inside
			res, err := s.roomsCollection.UpdateOne(ctx, filter,
				bson.M{"$unset": bson.M{fmt.Sprintf("%s.%s.%s", roomDataField, dataID, itemIndex): ""}})
			if err != nil {
				return nil, fmt.Errorf("mongo updateOne $unset data error (dataId: '%s', dict_key: '%s'): %w", dataID, itemIndex.String(), err)
			}
			if res.ModifiedCount == 0 {
				return nil, errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, fmt.Errorf("%s (dict_key='%s')", dataID, itemIndex.String()))
			}
		default:
			return nil, errors2.ErrQuick(errors.ErrUnsupported, fmt.Sprintf("remove for type %T", valueType))
		}
	}

	var resultValue *models.Value
	if len(itemIndex) == 0 {
		resultValue = nil
	} else {
		if resultValue, err = s.valueInRoom(ctx, roomID, dataID); err != nil {
			return nil, err // already wrapped
		}
	}

	return resultValue, nil
}
