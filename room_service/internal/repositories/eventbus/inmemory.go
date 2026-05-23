package eventbus

import (
	"context"
	"errors"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"sync"
	"time"
)

// How it works?
//
// Sending event:
// 1) proxy it to all subscribers in same room
// 2) proxy it to room_id = "" (for broadcasters who want all events)
//
// Drawing:
//
// event -> channel --|
//                    |       roomId = 1
//            |---room_id?----|
//            |               |
//           ...              |
//                            |
//                            |
//           subscriber_2 <------> subscriber_1
//
// Subscribing:
// 1) add subscriber to map by smallest available key
// 2) return a inMemoryEventBusSubscription with roomID and subscriber key in the map
//
// Unsubscribing:
// 1) delete from map by EventBusSubscription (subscribers[roomID][subscriptionKey]
// 2) map instead of list for persistent subscriptionKey (delete from list -> everything moves)

// inMemoryEventBusSubscription - subscription data needed to know what channel to unsubscribe
type inMemoryEventBusSubscription struct {
	roomIDString string
	keyInMap     int
}

type roomSubscribersBlock struct {
	mu          *sync.RWMutex
	subscribers map[int]chan ports.GrpcEvent
}

func (s *roomSubscribersBlock) addSubscriber() (channel chan ports.GrpcEvent, keyInMap int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyInMap = 0
	for {
		if _, ok := s.subscribers[keyInMap]; !ok {
			break
		}
		keyInMap++
	}

	channel = make(chan ports.GrpcEvent)
	s.subscribers[keyInMap] = channel

	return channel, keyInMap
}

// InMemoryEventBus - implementation of EventBusRepository with in-memory channels only
type InMemoryEventBus struct {
	mu          *sync.RWMutex
	subscribers map[string]*roomSubscribersBlock
}

// NewEventBusInMemory - create new InMemoryEventBus
func NewEventBusInMemory() *InMemoryEventBus {
	return &InMemoryEventBus{
		mu:          new(sync.RWMutex),
		subscribers: make(map[string]*roomSubscribersBlock),
	}
}

// EventToBus - add new incoming event to event bus
//
// Copy event to all subscribers in same room
func (b *InMemoryEventBus) EventToBus(ctx context.Context, event ports.GrpcEvent) error {
	s := event.RoomId

	b.mu.RLock()
	if _, ok := b.subscribers[s]; !ok {
		return fmt.Errorf("can't send event to room '%s': no subscribers are there", s)
	}

	go func() {
		b.subscribers[s].mu.RLock()
		defer b.subscribers[s].mu.RUnlock()

		// send to room-specific subscribers
		for _, subscriber := range b.subscribers[s].subscribers {
			select {
			case subscriber <- event:
				// Sent successfully
			case <-time.After(50 * time.Millisecond):
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "subscriber channel timeout")
			}
		}

		// send to broadcasters if any - they get all events
		if _, ok := b.subscribers[""]; ok {
			for _, broadcastSubscriber := range b.subscribers[""].subscribers {
				select {
				case broadcastSubscriber <- event:
					// Sent successfully
				case <-time.After(50 * time.Millisecond):
					logger.GetLoggerFromCtx(ctx).Warn(ctx, "broadcastSubscriber channel timeout")
				}
			}
		}
	}()

	return nil
}

// SubscribeToBusSpecificRoom - create a new channel that will receive events related to single room
//
// returns the subscription object - use it when unsubscribing
func (b *InMemoryEventBus) SubscribeToBusSpecificRoom(roomID models.RoomID) (chan ports.GrpcEvent, any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := roomID.String()

	if _, ok := b.subscribers[s]; !ok {
		b.subscribers[s] = &roomSubscribersBlock{
			mu:          new(sync.RWMutex),
			subscribers: make(map[int]chan ports.GrpcEvent),
		}
	}

	channel, keyInMap := b.subscribers[s].addSubscriber()

	return channel, inMemoryEventBusSubscription{
		roomIDString: s,
		keyInMap:     keyInMap,
	}, nil
}

// SubscribeToBusAllRooms - create a new channel that will receive events for all rooms
//
// returns the subscription object - use it when unsubscribing
func (b *InMemoryEventBus) SubscribeToBusAllRooms() (chan ports.GrpcEvent, any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[""]; !ok {
		b.subscribers[""] = &roomSubscribersBlock{
			mu:          new(sync.RWMutex),
			subscribers: make(map[int]chan ports.GrpcEvent),
		}
	}

	channel, keyInMap := b.subscribers[""].addSubscriber()

	return channel, inMemoryEventBusSubscription{
		roomIDString: "",
		keyInMap:     keyInMap,
	}, nil
}

// UnsubscribeFromBus - unsubscribe from bus
//
// Use the saved subscription object!
func (b *InMemoryEventBus) UnsubscribeFromBus(subscriptionResult any) error {
	subscription, ok := subscriptionResult.(inMemoryEventBusSubscription)
	if !ok {
		return errors.New("wrong subscription result type! Excepted: UnsubscribeToBus")
	}

	// lock root + subscribersBlock mutexes
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.subscribers[subscription.roomIDString].mu.Lock()
	defer b.subscribers[subscription.roomIDString].mu.Unlock()

	// close the channel
	var channelToUnsubscribe chan ports.GrpcEvent
	channelToUnsubscribe, ok = b.subscribers[subscription.roomIDString].subscribers[subscription.keyInMap]
	if !ok {
		return errors.New("cannot unsubscribe: key in map not exist (no subscriber here)")
	}
	close(channelToUnsubscribe)

	// mark channel as deleted so it gets destroyed soon
	delete(b.subscribers[subscription.roomIDString].subscribers, subscription.keyInMap)

	return nil
}
