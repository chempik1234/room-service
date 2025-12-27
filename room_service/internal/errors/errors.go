package errors

import "errors"

// ErrRoomDoesntExist - when no room found with given filter
var ErrRoomDoesntExist = errors.New("room does not exist")

// ErrRoomIDAlreadyExists - when roomID is not unique
var ErrRoomIDAlreadyExists = errors.New("room already exists")

// ErrUserNotInRoom - when user you're trying to remove isn't in room
var ErrUserNotInRoom = errors.New("user not in room")

// ErrDataPieceDoesntExist - when data item by key you're trying to read/update/delete doesn't exist
var ErrDataPieceDoesntExist = errors.New("data piece does not exist")
