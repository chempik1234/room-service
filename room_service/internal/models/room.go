package models

import (
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
)

// Room - main model, stores users and their data
type Room struct {
	ID          RoomID
	OwnerUserID types.NotEmptyText
	Options     map[string]string
}

// NewRoom creates a new Room with given options
func NewRoom(ownerUserID types.NotEmptyText, options map[string]string) *Room {
	return &Room{
		ID:          RoomID(types.GenerateUUID()),
		OwnerUserID: ownerUserID,
		Options:     options,
	}
}

// RoomSnapshot - model of full room data, including User list, data & Room itself
type RoomSnapshot struct {
	Users  []*User
	Room   *Room
	Values map[string]Value
	// TODO: rest fields
}

// RoomID - type used for Room.ID
//
// rely on it when making params with it
type RoomID types.UUID

// String - convert RoomID into string
func (id *RoomID) String() string {
	return types.UUID(*id).String()
}
