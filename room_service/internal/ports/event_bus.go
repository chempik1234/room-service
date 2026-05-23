package ports

import (
	"context"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/pkg/api/room_service"
)

// GrpcEvent - alias of *room_service.Event
type GrpcEvent *room_service.Event

// EventBusRepository - is a port for a bus that accepts events and allows to read them in all active connections.
//
// This way, we can easily send event to all clients in the same room.
//
// Using Redis pub/sub (or another outer service) would enable scaling.
type EventBusRepository interface {
	// EventToBus - add new incoming event to event bus
	EventToBus(ctx context.Context, event GrpcEvent) error
	// SubscribeToBusSpecificRoom - create chan to load events from pub/sub eventbus
	SubscribeToBusSpecificRoom(roomID models.RoomID) (chan GrpcEvent, any, error)
	// SubscribeToBusAllRooms - create chan to load events from pub/sub eventbus, but all events will be here
	SubscribeToBusAllRooms() (chan GrpcEvent, any, error)
	// UnsubscribeFromBus - stop receiving events and notify the subscriptions manager (e.g. close and remove channel)
	UnsubscribeFromBus(subscriptionResult any) error
}
