package models

import (
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
)

// Room - main model, stores users and their data
type Room struct {
	ID          types.UUID
	OwnerUserId types.NotEmptyText
	Options     map[string]string
}

// NewRoom creates a new Room with given options
func NewRoom(ownerUserID types.NotEmptyText, options map[string]string) *Room {
	return &Room{
		ID:          types.GenerateUUID(),
		OwnerUserId: ownerUserID,
		Options:     options,
	}
}

// RoomSnapshot - model of full room data, including User list, data & Room itself
type RoomSnapshot struct {
	Room *Room
	// TODO: rest fields
}
