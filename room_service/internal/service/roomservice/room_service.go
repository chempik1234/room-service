package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/ports"
	"github.com/chempik1234/room-service/internal/projectutils"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/pkg/logger"
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
		commandScopeCtx, err := logger.New(context.WithValue(context.Background(), logger.KeyForRequestID, projectutils.GenerateRequestID()))
		if err != nil {
			return fmt.Errorf("failed to init logger: %w", err)
		}

		// 3) execute command async-ly
		go func() {
			// 3.1) try to execute
			returnEvent, err := s.processCommand(commandScopeCtx, received)
			if err != nil {
				// if failed, send error
				logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "error processing command", zap.Error(err))
				s.sendError(commandScopeCtx, stream, returnEvent, err)
				return
			}

			// 3.2) send result if OK
			err = retry.Do(func() error { return stream.Send(returnEvent) }, s.retryStrategy)
			if err != nil {
				// if failed to send, then try to send error about step 2
				logger.GetLoggerFromCtx(commandScopeCtx).Error(commandScopeCtx, "failed to send event", zap.Error(err))
				s.sendError(commandScopeCtx, stream, returnEvent, err)
			}
		}()
	}
}

// SingleCommand - is the handler for single command endpoint SingleCommand
//
// One incoming command - full room snapshot after command execution (or simple message about deleted room)
func (s *RoomService) SingleCommand(ctx context.Context, command *r.Command) (*r.SingleEvent, error) {
	return nil, nil
}
