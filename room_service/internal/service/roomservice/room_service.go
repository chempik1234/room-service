package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/logger"
	"github.com/wb-go/wbf/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	commandIdShortCache ports.CommandIDShortCache
	retryStrategy       retry.Strategy
}

// NewRoomService creates a new RoomService
func NewRoomService(roomsRepo ports.RoomsPort, commandIdShortCache ports.CommandIDShortCache, retryStrategy retry.Strategy) *RoomService {
	return &RoomService{
		roomsRepo:           roomsRepo,
		retryStrategy:       retryStrategy,
		commandIdShortCache: commandIdShortCache,
	}
}

// Stream - is the handler for life-cycle endpoint Stream
//
// Incoming commands - output Events with deltas or full snapshots (e.g. on room join)
func (s *RoomService) Stream(stream grpc.BidiStreamingServer[r.Command, r.Event]) error {
	var err error
	var received *r.Command

	// main cycle
	for {
		// 1) receive object

		// region try to receive
		received, err = stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("error receiving gRPC stream in_: %w", err)
		}
		// endregion

		// 2) ctx - commandScopeCtx stores command ID and logger
		commandScopeCtx, err := newCommandScopeCtx(context.Background())
		if err != nil {
			return fmt.Errorf("failed to init logger: %w", err)
		}

		// 3) execute command async-ly
		go func() {
			// 3.1) try to execute
			returnEvent, err := s.processCommand(commandScopeCtx, received)
			if err != nil {
				// if failed, send error
				logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "error processing command in stream", zap.Error(err))
				s.sendErrorToStream(commandScopeCtx, stream, returnEvent, err)
				return
			}

			// 3.2) send result if OK
			err = retry.Do(func() error { return stream.Send(returnEvent) }, s.retryStrategy)
			if err != nil {
				// if failed to send, then try to send error about step 2
				logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "failed to send event", zap.Error(err))
				s.sendErrorToStream(commandScopeCtx, stream, returnEvent, err)
			}
		}()
	}
}

// SingleCommand - is the handler for single command endpoint SingleCommand
//
// One incoming command - full room snapshot after command execution (or simple message about deleted room)
func (s *RoomService) SingleCommand(ctx context.Context, command *r.Command) (*r.Event, error) {
	commandScopeCtx, err := newCommandScopeCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	returnEvent, err := s.processCommand(commandScopeCtx, command)
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
	}

	return returnEvent, nil
}

// RoomsList - get rooms list with no inner data and no users list
func (s *RoomService) RoomsList(ctx context.Context, _ *r.EmptyMessage) (*r.RoomsShortList, error) {
	roomsList, err := s.roomsRepo.RoomsList(ctx)
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
