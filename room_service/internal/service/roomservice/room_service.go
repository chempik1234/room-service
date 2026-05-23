package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"github.com/wb-go/wbf/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
)

const commandIDZapKey = "command_id"

// RoomService is the grpc handler class (without handler abstraction)
//
// Implements RoomServiceServer
type RoomService struct {
	r.RoomServiceServer
	// execute commands and store data
	roomsRepo ports.RoomsPort
	// no-repeat
	commandIDShortCache ports.CommandIDShortCache
	retryStrategy       retry.Strategy

	eventBus ports.EventBusRepository
}

// NewRoomService creates a new RoomService
func NewRoomService(
	roomsRepo ports.RoomsPort, commandIDShortCache ports.CommandIDShortCache,
	eventBus ports.EventBusRepository,
	retryStrategy retry.Strategy,
) *RoomService {
	return &RoomService{
		roomsRepo:           roomsRepo,
		retryStrategy:       retryStrategy,
		commandIDShortCache: commandIDShortCache,
		eventBus:            eventBus,
	}
}

// Stream - is the handler for life-cycle endpoint Stream
//
// Incoming commands - output Events with deltas or full snapshots (e.g. on room join)
// Use metadata "room-id" to optimize for single room:
// - empty/missing "room-id" = all rooms (default)
// - specific "room-id" = only that room, validates all commands match this room
//
// 1. get room-id filter
// 2. subscribe to events
//
// 3. Read life cycle:
// 1) receive object
// 2) validate room ID matches metadata filter (if set)
// 3) ctx - commandScopeCtx stores command ID and logger
// 4) execute command async-ly
// 4.1) try to execute (if failed, send error)
// 4.2) send result if OK (if failed to send, then try to send error about step 2)
//
// 4. Write life cycle:
// 1) receive event from event bus
// 2) get only related to selected room if any (defined on stream creating)
func (s *RoomService) Stream(stream grpc.BidiStreamingServer[r.Command, r.Event]) error {
	var received *r.Command

	resultErr := make(chan error)

	// Check metadata for room-specific streaming
	roomIDFilter, err := getRoomIDFilterForStream(stream.Context())
	if err != nil {
		return err // already wrapped
	}

	//region Subscribe to appropriate events channel
	var eventsToSendChan chan ports.GrpcEvent
	var subscription any

	if roomIDFilter == nil {
		eventsToSendChan, subscription, err = s.eventBus.SubscribeToBusAllRooms()
	} else {
		eventsToSendChan, subscription, err = s.eventBus.SubscribeToBusSpecificRoom(*roomIDFilter)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to event bus: %w", err)
	}
	//endregion

	// we don't quit, until reading is stopped
	go func() {
		// main cycle
		for {
			// 1) receive object

			// region try to receive
			received, err = stream.Recv()
			if err == io.EOF {
				resultErr <- nil
				break
			} else if err != nil {
				resultErr <- fmt.Errorf("error receiving gRPC stream in_: %w", err)
				break
			}
			// endregion

			// 2) validate room ID matches metadata filter (if set)
			if roomIDFilter != nil {
				if received.RoomId == nil {
					resultErr <- fmt.Errorf("command has no room_id but metadata room-id is set to '%s'", roomIDFilter.String())
					break
				}
				if *received.RoomId != roomIDFilter.String() {
					resultErr <- fmt.Errorf("command room_id '%s' doesn't match metadata room-id '%s'", *received.RoomId, roomIDFilter.String())
					break
				}
			}

			// 3) ctx - commandScopeCtx stores command ID and logger
			var commandScopeCtx context.Context
			commandScopeCtx, err = newCommandScopeCtx(context.Background())
			if err != nil {
				resultErr <- fmt.Errorf("failed to init logger: %w", err)
				break
			}

			// 4) execute command async-ly
			go func() {
				// 4.1) try to execute
				var returnEvent *r.Event
				var ok bool
				returnEvent, ok, err = s.processCommand(commandScopeCtx, received)
				if err != nil {
					// if failed, send error
					logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "error processing command in stream", zap.Error(err))
					s.sendErrorToStream(commandScopeCtx, stream, returnEvent, err)
					return
				}

				// 4.2) send result if OK

				err = retry.Do(func() error {
					if ok {
						return s.eventBus.EventToBus(commandScopeCtx, returnEvent)
					}
					// errors aren't sent into bus because why would one read them?
					return stream.Send(returnEvent)
				}, s.retryStrategy)

				if err != nil {
					// if failed to send, then try to send error about step 2
					logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "failed to send event", zap.Error(err))
					s.sendErrorToStream(commandScopeCtx, stream, returnEvent, err)
				}
			}()
		}
	}()

	go func() {
		var sendErr error

		ctx, _ := logger.New(context.Background())

		// loop incoming events
		for e := range eventsToSendChan {
			// try to send
			sendErr = retry.Do(func() error { return stream.Send(e) }, s.retryStrategy)
			if sendErr != nil {
				// if failed to send, then try to send error
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to send event", zap.Error(sendErr))
				s.sendErrorToStream(ctx, stream, e, sendErr)
			}
		}
	}()

	<-resultErr

	err = s.eventBus.UnsubscribeFromBus(subscription)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from events: %w", err)
	}

	return nil
}

func getRoomIDFilterForStream(ctx context.Context) (*models.RoomID, error) {
	var roomIDFilter *models.RoomID
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if roomIDs := md.Get("room-id"); len(roomIDs) > 0 && roomIDs[0] != "" {
			roomID, err := types.NewUUID(roomIDs[0])
			if err != nil {
				return nil, fmt.Errorf("invalid room-id in metadata: %w", err)
			}
			convertedRoomID := models.RoomID(roomID)
			roomIDFilter = &convertedRoomID
		}
	}
	return roomIDFilter, nil
}

// SingleCommand - is the handler for single command endpoint SingleCommand
//
// One incoming command - full room snapshot after command execution (or simple message about deleted room)
func (s *RoomService) SingleCommand(ctx context.Context, command *r.Command) (*r.Event, error) {
	commandScopeCtx, err := newCommandScopeCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	returnEvent, ok, err := s.processCommand(commandScopeCtx, command)
	if err != nil {
		// if failed, show (and later return) error
		logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "error processing single command", zap.Error(err))
	}

	if returnEvent == nil {
		roomID := ""
		if command.RoomId != nil {
			roomID = *command.RoomId
		}
		returnEvent = quickErrorEvent(roomID, command.UserId, "internal error: returned event is nil!")

		ok = false
	}

	if ok {
		err = retry.Do(func() error { return s.eventBus.EventToBus(commandScopeCtx, returnEvent) }, s.retryStrategy)
		if err != nil {
			err = fmt.Errorf("failed to send to eventBus: %w", err)
		}
	}

	return returnEvent, err
}

// RoomsList - get rooms list with no inner data and no users list
func (s *RoomService) RoomsList(ctx context.Context, _ *r.EmptyMessage) (*r.RoomsShortList, error) {
	roomsList, err := s.roomsRepo.RoomsList(ctx, "")
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to list rooms", zap.Error(err))
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}

	protoRoomsList := make([]*r.RoomShort, len(roomsList))
	for i, room := range roomsList {
		protoRoomsList[i] = &r.RoomShort{
			RoomOptions: room.Options,
			RoomId:      room.ID.String(),
			RoomOwnerId: room.OwnerUserID.String(),
		}
	}

	return &r.RoomsShortList{Rooms: protoRoomsList}, err
}

// UserActiveRoomsList - get list of rooms that user is currently in (with no inner data and no users list)
func (s *RoomService) UserActiveRoomsList(ctx context.Context, userIDMessage *r.UserIDMessage) (*r.RoomsShortList, error) {
	roomsList, err := s.roomsRepo.RoomsList(ctx, types.NewAnyText(userIDMessage.GetUserId()))
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to list rooms", zap.Error(err))
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}

	protoRoomsList := make([]*r.RoomShort, len(roomsList))
	for i, room := range roomsList {
		protoRoomsList[i] = &r.RoomShort{
			RoomOptions: room.Options,
			RoomId:      room.ID.String(),
			RoomOwnerId: room.OwnerUserID.String(),
		}
	}

	return &r.RoomsShortList{Rooms: protoRoomsList}, err
}
