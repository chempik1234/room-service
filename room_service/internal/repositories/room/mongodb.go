package room

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/chempik1234/room-service/internal/errors"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	types2 "github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"strconv"
)

/*
MongoDB Schema:

Meanings:
{...} means map[string]string
<key> means string typed key
<val> means string, int, bool, float64, map[string]<val> or list of <val>

rooms Collection:
{
	_id: ObjectID("some-id"),
	id: "generated-id-golang",
	owner_user_id: "client-user-id",
	options: {...},
	users: [
		{
			id: "client-user-id",
			name: "client-user-name",
			metadata: {...}
		},
	],
	data: {
		<key>: <val>,
	}
}

service operations:

delete - remove completely from root "data" field
set -
	item_index nil -- set by key in root data -- room.data[data_id] = value
	item_index val -- set like this: room.data[data_id][item_index] = value
append - append(data[data_id], value), only list
remove - del room.data[data_id][item_index], if list then remove item by index

AI couldn't handle all that so
unfortunately, this isn't fully vibe coded...

*/

// MongoDBRepository - ports.RoomsPort impl with MongoDB
type MongoDBRepository struct {
	client          *mongo.Client
	db              *mongo.Database
	roomsCollection *mongo.Collection
}

// MongoRepoParams - params for initializing MongoDBRepository
type MongoRepoParams struct {
	Database       string
	RoomCollection string
	WriteConcern   *writeconcern.WriteConcern
	ReadConcern    *readconcern.ReadConcern
}

// NewMongoDBRepository - return new MongoDBRepository
//
// roomCollectionName default = "rooms"
func NewMongoDBRepository(client *mongo.Client, params MongoRepoParams) *MongoDBRepository {
	s := &MongoDBRepository{client: client}
	s.db = client.Database(
		params.Database,
		options.Database().SetReadConcern(params.ReadConcern),
		options.Database().SetWriteConcern(params.WriteConcern))
	if len(params.RoomCollection) == 0 {
		params.RoomCollection = "rooms"
	}
	s.roomsCollection = s.db.Collection(params.RoomCollection)
	return s
}

// CreateRoom - create room in MongoDB
//
// Create ID yourself
func (s *MongoDBRepository) CreateRoom(ctx context.Context, room *models.Room) (newRoom *models.Room, err error) {
	if room == nil {
		return nil, fmt.Errorf("internal error: CreateRoom - room is %v", room)
	}

	if err = s.errIfRoomExists(ctx, room.ID.String()); err != nil {
		return nil, err // already wrapped
	}

	// uses retry inside mongodb driver
	_, err = s.roomsCollection.InsertOne(ctx, bson.M{
		"id":            room.ID.String(),
		"owner_user_id": room.OwnerUserID.String(),
		"options":       room.Options,
	})

	if err != nil {
		return nil, fmt.Errorf("mongo insertOne room error: %w", err)
	}

	return room, nil
}

// DeleteRoom - delete room from MongoDB with all data inside
//
// Not found -> errors.ErrRoomDoesntExist
func (s *MongoDBRepository) DeleteRoom(ctx context.Context, params ports.DeleteRoomParams) (err error) {
	// uses retry inside mongodb driver
	res, err := s.roomsCollection.DeleteOne(ctx, bson.M{"id": params.RoomID.String()})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
		}
		return fmt.Errorf("mongo deleteOne room error: %w", err)
	}
	if res.DeletedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
	}
	return nil
}

// JoinRoom - add user to room in MongoDB
//
// Not found -> errors.ErrRoomDoesntExist
// Idempotent - if user already in room, no error (just no-op)
func (s *MongoDBRepository) JoinRoom(ctx context.Context, params ports.JoinRoomParams) (err error) {
	err = s.errIfRoomDoesntExist(ctx, params.RoomID.String())
	if err != nil {
		return err // already wrapped
	}

	// serialize object to bson
	userDoc := bson.M{
		"id":       params.UserFull.ID.String(),
		"name":     params.UserFull.Name.String(),
		"metadata": params.UserFull.Metadata,
	}

	// Add user to room's users array if not already there
	// $addToSet = idempotency
	filter := bson.M{"id": params.RoomID.String()}
	update := bson.M{"$addToSet": bson.M{"users": userDoc}}

	// uses retry inside mongodb driver
	result, err := s.roomsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mongo updateOne add user to room error: %w", err)
	}

	// maybe (just maybe) room doesn't exist
	if result.MatchedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
	}

	return nil
}

// IsRoomOwner - check if room's owner is given user (MongoDB)
//
// Not found -> errors.ErrRoomDoesntExist
func (s *MongoDBRepository) IsRoomOwner(ctx context.Context, params ports.IsRoomOwnerParams) (bool, error) {
	// Find room and project only the owner_user_id field
	filter := bson.M{"id": params.RoomID.String()}
	projection := bson.M{"owner_user_id": 1, "_id": 0}

	var result struct {
		OwnerUserID string `bson:"owner_user_id"`
	}

	// uses retry inside mongodb driver
	err := s.roomsCollection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
		}
		return false, fmt.Errorf("mongo findOne room owner error: %w", err)
	}

	// Compare owner_user_id with provided user_id
	return result.OwnerUserID == params.UserID.String(), nil
}

// LeaveRoom - remove user from room (MongoDB)
//
// Room not found -> errors.ErrRoomDoesntExist
// User not found -> errors.ErrUserNotInRoom
func (s *MongoDBRepository) LeaveRoom(ctx context.Context, param ports.LeaveRoomParams) error {
	// check if room exists
	roomExists, err := s.roomExists(ctx, param.RoomID.String())
	if err != nil {
		return err // already wrapped
	}
	if !roomExists {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, param.RoomID.String())
	}

	// Build filter to find the specific user in the users array
	filter := bson.M{
		"id": param.RoomID.String(),
		"users": bson.M{
			"$elemMatch": bson.M{"id": param.KickedUserID.String()},
		},
	}

	// Check if user exists in room
	// uses retry inside mongodb driver
	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo countDocuments check user in room error: %w", err)
	}
	if count == 0 {
		return errors2.ErrQuick(errors2.ErrUserNotInRoom, param.KickedUserID.String())
	}

	// updateOne -> $pull -> users = remove
	updateFilter := bson.M{"id": param.RoomID.String()}
	update := bson.M{
		"$pull": bson.M{
			"users": bson.M{"id": param.KickedUserID.String()},
		},
	}

	result, err := s.roomsCollection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return fmt.Errorf("mongo updateOne remove user from room error: %w", err)
	}

	// maybe (just maybe) other transaction ruined our "user exists" check
	if result.MatchedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, param.RoomID.String())
	}

	return nil
}

// RoomSnapshot - return a whole sight on room - ownerID, room data KV, roomID... (MongoDB)
//
// Room not found -> errors.ErrRoomDoesntExist
func (s *MongoDBRepository) RoomSnapshot(ctx context.Context, params ports.RoomSnapshotParams) (*models.RoomSnapshot, error) {
	// in snapshot, we need all fields
	var mongoRoom struct {
		ID          string            `bson:"id"`
		OwnerUserID string            `bson:"owner_user_id"`
		Options     map[string]string `bson:"options"`
		Users       []struct {
			ID       string            `bson:"id"`
			Name     string            `bson:"name"`
			Metadata map[string]string `bson:"metadata"`
		} `bson:"users"`
		Data map[string]bson.RawValue `bson:"data"`
	}

	// uses retry inside mongodb driver
	filter := bson.M{"id": params.RoomID.String()}
	err := s.roomsCollection.FindOne(ctx, filter).Decode(&mongoRoom)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
		}
		return nil, fmt.Errorf("mongo findOne room snapshot error: %w", err)
	}

	// bson users to models.User
	users := make([]*models.User, 0, len(mongoRoom.Users))
	for _, u := range mongoRoom.Users {
		userID, err := types2.NewNotEmptyText(u.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID '%s': %w", u.ID, err)
		}
		userName, err := types2.NewNotEmptyText(u.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid user name '%s': %w", u.Name, err)
		}
		users = append(users, &models.User{
			ID:       userID,
			Name:     userName,
			Metadata: u.Metadata,
		})
	}

	// bson data to map
	// nil = empty
	values := make(map[string]models.Value)
	if mongoRoom.Data != nil {
		for key, rawValue := range mongoRoom.Data {
			val, err := decodeBSONValueToModelValue(rawValue)
			if err != nil {
				return nil, fmt.Errorf("failed to decode data field '%s': %w", key, err)
			}
			values[key] = *val
		}
	}

	// validate roomUUID from DB
	roomUUID, err := types2.NewUUID(mongoRoom.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid room ID '%s': %w", mongoRoom.ID, err)
	}
	roomID := models.RoomID(roomUUID)

	// validate room owner ID from DB
	ownerUserID, err := types2.NewNotEmptyText(mongoRoom.OwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner user ID '%s': %w", mongoRoom.OwnerUserID, err)
	}

	// Build the snapshot
	snapshot := &models.RoomSnapshot{
		Users: users,
		Room: &models.Room{
			ID:          roomID,
			OwnerUserID: ownerUserID,
			Options:     mongoRoom.Options,
		},
		Values: values,
	}

	return snapshot, nil
}

// AffectData - set/delete whole data field or item in dict/list (depends on what models.Value is stored)
//
// # The whole data storage is a KV storage that can store different values, including lists and dicts
//
// Room not found -> errors.ErrRoomDoesntExist
// Data not found -> errors.ErrDataPieceDoesntExist
func (s *MongoDBRepository) AffectData(ctx context.Context, params ports.AffectDataParams) error {
	// First, check if room exists
	roomExists, err := s.roomExists(ctx, params.RoomID.String())
	if err != nil {
		return err // already wrapped
	}
	if !roomExists {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
	}

	dataID := params.DataID.String()

	switch params.Action {
	case ports.ActionSet:
		// SET mode: set data[data_id] = value
		return s.setData(ctx, params.RoomID.String(), dataID, params.Value, params.ItemIndex.String())

	case ports.ActionDelete:
		// DELETE mode: delete data[data_id]
		return s.deleteData(ctx, params.RoomID.String(), dataID)

	case ports.ActionAppend:
		// APPEND mode: append to data[data_id] (must be list or map)
		return s.appendData(ctx, params.RoomID.String(), dataID, params.Value)

	case ports.ActionRemove:
		// REMOVE mode: remove from data[data_id] by index/key
		itemIndexValid, err := types2.NewNotEmptyText(params.ItemIndex.String())
		if err != nil {
			return fmt.Errorf("invalid item_index: %w", err)
		}

		return s.removeData(ctx, params.RoomID.String(), dataID, itemIndexValid)

	default:
		return fmt.Errorf("unknown action: %d", params.Action)
	}
}

// setData - set data[data_id] = value in MongoDB
//
// if itemIndex != "" --> set data[data_id][itemIndex] = value
func (s *MongoDBRepository) setData(ctx context.Context, roomID, dataID string, value *models.Value, itemIndex string) error {
	// Convert models.Value to BSON value
	bsonValue, err := encodeModelValueToBSON(value)
	if err != nil {
		return fmt.Errorf("failed to encode value to BSON: %w", err)
	}

	filter := bson.M{"id": roomID}

	var update bson.M
	if len(itemIndex) == 0 {
		update = bson.M{"$set": bson.M{fmt.Sprintf("data.%s", dataID): bsonValue}}
	} else {
		// TODO: does is work for array?
		update = bson.M{"$set": bson.M{fmt.Sprintf("data.%s.%s", dataID, itemIndex): bsonValue}}
	}

	result, err := s.roomsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mongo updateOne set data error: %w", err)
	}
	if result.MatchedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, roomID)
	}

	return nil
}

// deleteData - delete data[data_id] from MongoDB
func (s *MongoDBRepository) deleteData(ctx context.Context, roomID, dataID string) error {
	// check if "data" field exists
	filter := bson.M{
		"id":                           roomID,
		fmt.Sprintf("data.%s", dataID): bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
	}

	// delete the value from room.data
	updateFilter := bson.M{"id": roomID}
	update := bson.M{"$unset": bson.M{fmt.Sprintf("data.%s", dataID): ""}}

	result, err := s.roomsCollection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return fmt.Errorf("mongo updateOne delete data error: %w", err)
	}
	if result.MatchedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, roomID)
	}

	return nil
}

// appendData - append value to data[data_id] (only array)
func (s *MongoDBRepository) appendData(ctx context.Context, roomID, dataID string, value *models.Value) error {
	// models.Value to BSON
	bsonValue, err := encodeModelValueToBSON(value)
	if err != nil {
		return fmt.Errorf("failed to encode value to BSON: %w", err)
	}

	// check if room."data" exists
	filter := bson.M{
		"id":                           roomID,
		fmt.Sprintf("data.%s", dataID): bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
	}

	// Determine if we're appending to array or map
	// try $push (for arrays) first
	filter = bson.M{"id": roomID}
	update := bson.M{"$push": bson.M{fmt.Sprintf("data.%s", dataID): bsonValue}}

	result, err := s.roomsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		// GPT note:
		// If $push fails, it might be a map, try $mergeObjects or set for nested map
		// For now, just return the error
		return fmt.Errorf("mongo updateOne append data error: %w", err)
	}
	if result.MatchedCount == 0 {
		return errors2.ErrQuick(errors2.ErrRoomDoesntExist, roomID)
	}

	return nil
}

// removeData - remove item from data[data_id] by index/key
//
// if itemIndex != "" --> remove item data[data_id][itemIndex]
func (s *MongoDBRepository) removeData(ctx context.Context, roomID, dataID string, itemIndex types2.NotEmptyText) error {
	// check if data_id in "data" field exists
	removedItemPath := fmt.Sprintf("data.%s", dataID)

	filter := bson.M{
		"id":            roomID,
		removedItemPath: bson.M{"$exists": true},
	}

	count, err := s.roomsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo countDocuments check data exists error: %w", err)
	}
	if count == 0 {
		return errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, dataID)
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
				fmt.Sprintf("data.%s", dataID): 1,
			},
		)).Decode(&doc)
	if err != nil {
		return fmt.Errorf("failed to get room's data by roomID: %w", err)
	}

	// step 1.
	if value, ok := doc["data"].(bson.M)[dataID]; ok {
		switch valueType := value.(type) {
		case bson.A:
			// we need int for indexing in-memory: we find item by value, not by key, which is awful!
			indexInt, err := strconv.Atoi(itemIndex.String())
			if err != nil {
				return fmt.Errorf("failed to convert itemIndex to int (item is array): %w", err)
			}

			// step 2.
			// uses retry inside
			res, err := s.roomsCollection.UpdateOne(ctx, filter,
				bson.M{"$pull": bson.M{fmt.Sprintf("data.%s", dataID): bson.M{"$eq": value.([]any)[indexInt]}}})
			// {"$pull": {"data.<data_id>": {"eq": <val data[<data_id>][<itemIndex>]> } } }
			if err != nil {
				return fmt.Errorf("mongo updateOne $pull data error (dataId: '%s', array_index: '%d'): %w", dataID, indexInt, err)
			}
			if res.ModifiedCount == 0 {
				return errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, fmt.Errorf("%s (array_index='%d')", dataID, indexInt))
			}

		case bson.M:
			// step 2.
			// uses retry inside
			res, err := s.roomsCollection.UpdateOne(ctx, filter,
				bson.M{"$unset": bson.M{fmt.Sprintf("data.%s.%s", dataID, itemIndex): ""}})
			if err != nil {
				return fmt.Errorf("mongo updateOne $unset data error (dataId: '%s', array_index: '%d'): %w", dataID, itemIndex.String(), err)
			}
			if res.ModifiedCount == 0 {
				return errors2.ErrQuick(errors2.ErrDataPieceDoesntExist, fmt.Errorf("%s (dict_key='%d')", dataID, itemIndex.String()))
			}
		default:
			return errors2.ErrQuick(errors.ErrUnsupported, fmt.Sprintf("remove for type %T", valueType))
		}
	}

	return nil
}

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

// decodeBSONValueToModelValue - convert BSON RawValue to models.Value
//
// This helper handles all the different types that can be stored in MongoDB
// and converts them to the appropriate models.Value type
func decodeBSONValueToModelValue(raw bson.RawValue) (*models.Value, error) {
	val := models.EmptyValue()

	switch raw.Type {
	case bson.TypeInt32, bson.TypeInt64:
		var intVal int64
		if err := raw.Unmarshal(&intVal); err != nil {
			return nil, fmt.Errorf("unmarshal int error: %w", err)
		}
		val.SetInt(intVal)

	case bson.TypeString:
		var strVal string
		if err := raw.Unmarshal(&strVal); err != nil {
			return nil, fmt.Errorf("unmarshal string error: %w", err)
		}
		val.SetStr(strVal)

	case bson.TypeBoolean:
		var boolVal bool
		if err := raw.Unmarshal(&boolVal); err != nil {
			return nil, fmt.Errorf("unmarshal bool error: %w", err)
		}
		val.SetBool(boolVal)

	case bson.TypeDouble:
		var floatVal float64
		if err := raw.Unmarshal(&floatVal); err != nil {
			return nil, fmt.Errorf("unmarshal float error: %w", err)
		}
		val.SetFloat(floatVal)

	case bson.TypeBinary:
		var bytesVal []byte
		if err := raw.Unmarshal(&bytesVal); err != nil {
			return nil, fmt.Errorf("unmarshal bytes error: %w", err)
		}
		val.SetBytes(bytesVal)

	case bson.TypeArray:
		// Decode as array of values
		var rawArray []bson.RawValue
		if err := raw.Unmarshal(&rawArray); err != nil {
			return nil, fmt.Errorf("unmarshal array error: %w", err)
		}

		list := make([]models.Value, len(rawArray))
		for i, item := range rawArray {
			decodedItem, err := decodeBSONValueToModelValue(item)
			if err != nil {
				return nil, fmt.Errorf("decode array item %d error: %w", i, err)
			}
			list[i] = *decodedItem
		}
		val.SetList(list)

	case bson.TypeEmbeddedDocument:
		// Decode as map[string]Value
		var rawMap bson.M
		if err := raw.Unmarshal(&rawMap); err != nil {
			return nil, fmt.Errorf("unmarshal document error: %w", err)
		}

		resultMap := make(map[string]models.Value)
		for key, item := range rawMap {
			// For nested values, we need to recursively decode them
			// Marshal back to bytes and decode recursively
			decodedItem, err := decodeInterfaceToModelValue(item)
			if err != nil {
				return nil, fmt.Errorf("decode map value '%s' error: %w", key, err)
			}
			resultMap[key] = *decodedItem
		}
		val.SetMap(resultMap)

	case bson.TypeNull:
		// Null values - treat as empty string for now
		val.SetStr("")

	default:
		return nil, fmt.Errorf("unsupported BSON type: %v", raw.Type)
	}

	return val, nil
}

// encodeModelValueToBSON - convert models.Value to BSON-compatible value
//
// This helper handles all the different models.Value types and converts
// them to appropriate BSON types for MongoDB storage
func encodeModelValueToBSON(val *models.Value) (any, error) {
	if val == nil {
		return nil, nil
	}

	// Use type check methods to determine the actual value type

	switch {
	case val.IsInt():
		intVal, _ := val.GetInt()
		return intVal, nil

	case val.IsStr():
		strVal, _ := val.GetStr()
		return strVal, nil

	case val.IsBool():
		boolVal, _ := val.GetBool()
		return boolVal, nil

	case val.IsFloat():
		floatVal, _ := val.GetFloat()
		return floatVal, nil

	case val.IsBytes():
		bytesVal, _ := val.GetBytes()
		return bytesVal, nil

	case val.IsList():
		// Convert list items
		listVal, _ := val.GetList()
		result := make([]any, len(listVal))
		for i, item := range listVal {
			encodedItem, err := encodeModelValueToBSON(&item)
			if err != nil {
				return nil, fmt.Errorf("encode list item %d error: %w", i, err)
			}
			result[i] = encodedItem
		}
		return result, nil

	case val.IsMap():
		// Convert map items
		mapVal, _ := val.GetMap()
		result := make(map[string]any)
		for key, item := range mapVal {
			encodedItem, err := encodeModelValueToBSON(&item)
			if err != nil {
				return nil, fmt.Errorf("encode map value '%s' error: %w", key, err)
			}
			result[key] = encodedItem
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown value type")
	}
}

// decodeInterfaceToModelValue - convert any to models.Value recursively
//
// This helper handles nested structures from BSON unmarshaling
func decodeInterfaceToModelValue(item any) (*models.Value, error) {
	val := models.EmptyValue()

	switch v := item.(type) {
	case int32:
		val.SetInt(int64(v))
	case int64:
		val.SetInt(v)
	case int:
		val.SetInt(int64(v))
	case string:
		val.SetStr(v)
	case bool:
		val.SetBool(v)
	case float64:
		val.SetFloat(v)
	case []byte:
		val.SetBytes(v)
	case []any:
		// Handle nested arrays
		list := make([]models.Value, len(v))
		for i, elem := range v {
			decodedElem, err := decodeInterfaceToModelValue(elem)
			if err != nil {
				return nil, fmt.Errorf("decode array element %d error: %w", i, err)
			}
			list[i] = *decodedElem
		}
		val.SetList(list)
	case map[string]any:
		// Handle nested objects
		resultMap := make(map[string]models.Value)
		for key, elem := range v {
			decodedElem, err := decodeInterfaceToModelValue(elem)
			if err != nil {
				return nil, fmt.Errorf("decode map value '%s' error: %w", key, err)
			}
			resultMap[key] = *decodedElem
		}
		val.SetMap(resultMap)
		/*
			case primitive.D:
				// Handle BSON documents (ordered maps)
				resultMap := make(map[string]models.Value)
				for _, elem := range v {
					decodedElem, err := decodeInterfaceToModelValue(elem.Value)
					if err != nil {
						return nil, fmt.Errorf("decode bson element '%s' error: %w", elem.Key, err)
					}
					resultMap[elem.Key] = *decodedElem
				}
				val.SetMap(resultMap)
			case primitive.A:
				// Handle BSON arrays
				list := make([]models.Value, len(v))
				for i, elem := range v {
					decodedElem, err := decodeInterfaceToModelValue(elem)
					if err != nil {
						return nil, fmt.Errorf("decode bson array element %d error: %w", i, err)
					}
					list[i] = *decodedElem
				}
				val.SetList(list)
		*/
	case nil:
		// Null values
		val.SetStr("")
	default:
		return nil, fmt.Errorf("unsupported interface type: %T", item)
	}

	return val, nil
}
