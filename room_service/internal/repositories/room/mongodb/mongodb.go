package mongodb

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/chempik1234/room-service/internal/errors"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	types2 "github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.uber.org/zap"
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

const (
	roomOwnerUserIDField = "owner_user_id"
	roomIDField          = "id"
	roomOptionsField     = "options"
	roomDataField        = "data"
)

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
		roomIDField:          room.ID.String(),
		roomOwnerUserIDField: room.OwnerUserID.String(),
		roomOptionsField:     room.Options,
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
	res, err := s.roomsCollection.DeleteOne(ctx, bson.M{roomIDField: params.RoomID.String()})
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
		roomIDField: params.UserFull.ID.String(),
		"name":      params.UserFull.Name.String(),
		"metadata":  params.UserFull.Metadata,
	}

	// Add user to room's users array if not already there
	// $addToSet = idempotency
	filter := bson.M{roomIDField: params.RoomID.String()}
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
	filter := bson.M{roomIDField: params.RoomID.String()}
	projection := bson.M{roomOwnerUserIDField: 1}

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
		roomIDField: param.RoomID.String(),
		"users": bson.M{
			"$elemMatch": bson.M{roomIDField: param.KickedUserID.String()},
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
	updateFilter := bson.M{roomIDField: param.RoomID.String()}
	update := bson.M{
		"$pull": bson.M{
			"users": bson.M{roomIDField: param.KickedUserID.String()},
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

// SetOwnerUserID - try to set new owner ID (MongoDB)
//
// errors.ErrUserNotInRoom if userID isn't in room
func (s *MongoDBRepository) SetOwnerUserID(ctx context.Context, params ports.SetOwnerUserIDParams) (bool, error) {
	// we need these fields
	var mongoRoom struct {
		OwnerUserID string `bson:"owner_user_id"`
		Users       []struct {
			ID string `bson:"id"`
		} `bson:"users"`
	}

	// uses retry inside mongodb driver
	filter := bson.M{roomIDField: params.RoomID.String()}
	err := s.roomsCollection.FindOne(ctx, filter).Decode(&mongoRoom)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
		}
		return false, fmt.Errorf("mongo findOne room snapshot error: %w", err)
	}

	oldOwnerID := mongoRoom.OwnerUserID

	// early exit
	if oldOwnerID == params.NewOwnerID.String() {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "someone tried to set owner id that is already there",
			zap.Stringer("new_owner_id", params.NewOwnerID),
			zap.String("old_owner_id", oldOwnerID))
		return false, nil
	}

	// check if user is in the room
	userExists := false
	for _, user := range mongoRoom.Users {
		if user.ID == params.NewOwnerID.String() {
			userExists = true
			break
		}
	}

	if !userExists {
		return false, errors2.ErrQuick(errors2.ErrUserNotInRoom,
			fmt.Sprintf("(roomID: '%s' userID: '%s')",
				params.RoomID.String(),
				params.NewOwnerID.String()))
	}

	res, err := s.roomsCollection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{roomOwnerUserIDField: params.NewOwnerID.String()}})
	if err != nil {
		return false, fmt.Errorf("mongo updateOne update user in room error: %w", err)
	}
	if res.MatchedCount == 0 {
		return false, fmt.Errorf("internal error: room was queried for validation, but not found when updating")
	}

	return oldOwnerID != params.NewOwnerID.String(), nil
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
	filter := bson.M{roomIDField: params.RoomID.String()}
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
func (s *MongoDBRepository) AffectData(ctx context.Context, params ports.AffectDataParams) (*models.Value, error) {
	// First, check if room exists
	roomExists, err := s.roomExists(ctx, params.RoomID.String())
	if err != nil {
		return nil, err // already wrapped
	}
	if !roomExists {
		return nil, errors2.ErrQuick(errors2.ErrRoomDoesntExist, params.RoomID.String())
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
			return nil, fmt.Errorf("invalid item_index: %w", err)
		}

		return s.removeData(ctx, params.RoomID.String(), dataID, itemIndexValid)

	default:
		return nil, fmt.Errorf("unknown action: %d", params.Action)
	}
}

// RoomsList - return list of all rooms with no inner data and no users list
func (s *MongoDBRepository) RoomsList(ctx context.Context) ([]*models.Room, error) {
	// step 1. query rooms
	projection := bson.M{roomDataField: 0}
	cursor, err := s.roomsCollection.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		return nil, fmt.Errorf("failed to Find() rooms list: %w", err)
	}

	// step 1.1 close cursor
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		if err := cursor.Close(ctx); err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "mongo cursor close", zap.Error(err))
		}
	}(cursor, ctx)

	// step 2. fill the result list from cursor
	resultList := make([]*models.Room, 0)

	// reused variables
	var roomID types2.UUID // the chain is: bson.Raw -> UUID -> models.RoomID(UUID), so we can't use models.RoomID!
	var ownerUserID types2.NotEmptyText

	var rawValue bson.RawValue

	for cursor.Next(ctx) {
		// roomID
		rawValue, err = cursor.Current.LookupErr(roomIDField)
		if err != nil {
			return nil, fmt.Errorf("invalid room '%s' - no room \"%s\": %w", roomIDField, err)
		}
		roomID, err = types2.NewUUID(rawValue.StringValue())
		if err != nil {
			return nil, fmt.Errorf("invalid room '%s' - \"%s\" isn't a UUID: %w", roomIDField, err)
		}

		// ownerUserID
		rawValue, err = cursor.Current.LookupErr(roomOwnerUserIDField)
		if err != nil {
			return nil, fmt.Errorf("invalid room '%s' - no \"%s\": %w", roomID.String(), roomOwnerUserIDField, err)
		}
		ownerUserID, err = types2.NewNotEmptyText(rawValue.StringValue())
		if err != nil {
			return nil, fmt.Errorf("invalid room '%s' - \"%s\" is empty: %w", roomID.String(), roomOwnerUserIDField, err)
		}

		// options
		rawValue, err = cursor.Current.LookupErr(roomOptionsField)
		if err != nil {
			return nil, fmt.Errorf("invalid room '%s' - no \"%s\": %w", roomID.String(), roomOptionsField, err)
		}
		var roomOptions map[string]string
		if err = bson.Unmarshal(rawValue.Value, &roomOptions); err != nil {
			// return nil, fmt.Errorf("invalid room '%s' - \"%s\" is not a dict {str: str}: %w", roomID.String(), roomOptionsField, err)
			logger.GetLoggerFromCtx(ctx).Warn(ctx, "failed to unmarshal room options", zap.Error(err))
			roomOptions = make(map[string]string)
		}

		resultList = append(resultList, &models.Room{
			ID:          models.RoomID(roomID),
			OwnerUserID: ownerUserID,
			Options:     roomOptions,
		})
	}

	return resultList, nil
}
