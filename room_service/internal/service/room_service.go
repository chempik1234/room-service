package service

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/room-service/pkg/utils"
	"github.com/chempik1234/super-danis-library-golang/pkg/logger"
	"github.com/chempik1234/super-danis-library-golang/pkg/types"
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
	commandIdShortCache ports.CommandIdShortCache
	retryStrategy       retry.Strategy
}

// NewRoomService creates a new RoomService
func NewRoomService(roomsRepo ports.RoomsPort, commandIdShortCache ports.CommandIdShortCache, retryStrategy retry.Strategy) *RoomService {
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
		// region try to receive
		received, err = stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("error receiving gRPC stream in_: %w", err)
		}
		// endregion

		// commandScopeCtx stores command ID
		var commandScopeCtx context.Context
		commandScopeCtx = context.WithValue(context.Background(), logger.KeyForRequestID, utils.GenerateRequestID())
		commandScopeCtx, err = logger.New(commandScopeCtx)
		if err != nil {
			return fmt.Errorf("failed to init logger: %w", err)
		}

		// execute command async-ly
		go func(ctx context.Context, in *r.Command) {
			//region check command id no-repeat
			commandID := in.GetCommandId()
			if len(commandID) > 0 {
				_, commandIDExists, err := s.commandIdShortCache.Get(ctx, commandID)
				if err != nil {
					logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to get command_id from short cache", zap.String(commandIDZapKey, commandID), zap.Error(err))
					return
				}
				if commandIDExists {
					logger.GetLoggerFromCtx(ctx).Info(ctx, "command_id exists in short cache, SKIPPED", zap.String(commandIDZapKey, commandID))
					return
				}
			}
			//endregion

			logger.GetLoggerFromCtx(ctx).Info(ctx, "command received, processing", zap.String(commandIDZapKey, commandID))

			returnEvent := &r.Event{
				Timestamp: utils.NowTimestamp(),
				RoomId:    in.GetRoomId(),
				UserId:    in.GetUserId(),
				Payload:   nil,
			}

			skipCommandExecution := false

			// user id is always required, so we validate it before switch
			// room id isn't really required in all commands, so we check it only when really need
			var userIDValid types.NotEmptyText
			userIDValid, err = types.NewNotEmptyText(in.GetUserId())
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "someone entered empty userID")
				s.sendError(ctx, stream, returnEvent, fmt.Errorf("userID is empty"))
				skipCommandExecution = true
			}

			if !skipCommandExecution {
				switch payload := in.Payload.(type) {
				case *r.Command_CreateRoom:
					//region validate room
					var newRoom *models.Room
					newRoom = models.NewRoom(userIDValid, payload.CreateRoom.GetRoomOptions())
					//endregion

					returnEvent.RoomId = newRoom.ID.String()

					//region create room logic
					err = retry.Do(func() error {
						newRoom, err = s.roomsRepo.CreateRoom(ctx, newRoom)
						if err != nil {
							return err
						}
						return nil
					}, s.retryStrategy)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to create room", zap.Error(err))
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("failed to create room: %w", err))
						break
					}
					//endregion

					//region result
					returnEvent.Payload = &r.Event_RoomCreated{
						RoomCreated: &r.RoomCreatedEventBody{
							RoomOptions: newRoom.Options,
						},
					}
					//endregion
				case *r.Command_DeleteRoom:
					//region validate room
					var roomIDValidated types.UUID
					roomIDValidated, err = types.NewUUID(in.GetRoomId())
					if err != nil {
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("room id - invalid uuid"))
						break
					}
					//endregion

					//region check if userIDValid is owner of room
					var isRoomOwner bool
					err = retry.Do(func() error {
						var errRepo error
						isRoomOwner, errRepo = s.roomsRepo.IsRoomOwner(ctx, ports.IsRoomOwnerParams{
							RoomID: roomIDValidated,
							UserID: userIDValid,
						})
						return errRepo
					}, s.retryStrategy)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to check if user is room owner", zap.Error(err))
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("failed to check if user is room owner: %w", err))
						break
					}
					if !isRoomOwner {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "user is not a room owner",
							zap.String("room_id", roomIDValidated.String()),
							zap.String("user_id", userIDValid.String()))
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("user '%s' is not a room owner (%s)",
							userIDValid.String(),
							roomIDValidated.String()))
						break
					}
					//endregion

					//region delete room logic
					err = retry.Do(func() error {
						return s.roomsRepo.DeleteRoom(ctx, ports.DeleteRoomParams{
							RoomID: roomIDValidated,
							UserID: userIDValid,
						})
					}, s.retryStrategy)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to delete room", zap.Error(err))
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("failed to join room: %w", err))
						break
					}
					//endregion

					//region result
					returnEvent.Payload = &r.Event_RoomDeleted{
						RoomDeleted: &r.RoomDeletedEventBody{
							RoomDeleted: true,
						},
					}
					//endregion
				case *r.Command_JoinRoom:
					//region validate room
					var roomIDValidated types.UUID
					roomIDValidated, err = types.NewUUID(in.GetRoomId())
					if err != nil {
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("room id - invalid uuid"))
						break
					}
					//endregion

					//region validate user full
					var joinedUserIDValid types.NotEmptyText
					var joinedUserNameValid types.NotEmptyText

					joinedUserIDValid, err = types.NewNotEmptyText(payload.JoinRoom.UserFull.GetId())
					if err != nil {
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("user_full userID is empty"))
						break
					}
					joinedUserNameValid, err = types.NewNotEmptyText(payload.JoinRoom.UserFull.GetName())
					if err != nil {
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("user_full name is empty"))
						break
					}
					//endregion

					userModel := models.User{
						Metadata: payload.JoinRoom.UserFull.GetMetadata(),
						ID:       joinedUserIDValid,
						Name:     joinedUserNameValid,
					}

					//region join room logic
					err = retry.Do(func() error {
						return s.roomsRepo.JoinRoom(ctx, ports.JoinRoomParams{
							RoomID:   roomIDValidated,
							UserFull: userModel,
						})
					}, s.retryStrategy)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to join room", zap.Error(err))
						s.sendError(ctx, stream, returnEvent, fmt.Errorf("failed to join room: %w", err))
						break
					}
					//endregion

					//region result
					returnEvent.Payload = &r.Event_JoinedRoom{
						JoinedRoom: &r.JoinedRoomEventBody{
							UserFull: &r.User{
								Id:       userModel.ID.String(),
								Name:     userModel.Name.String(),
								Metadata: userModel.Metadata,
							},
						},
					}
					//endregion
					// TODO: LeaveRoomCommandBody, SetAppendDeleteDataCommandBody, RefreshRoomCommandBody
				}
			}

			err = retry.Do(func() error { return stream.Send(returnEvent) }, s.retryStrategy)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to send event", zap.Error(err))
				s.sendError(ctx, stream, returnEvent, err)
			}

			if s.commandIdShortCache.Set(ctx, in.CommandId, struct{}{}) != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set command_id into short cache", zap.String(commandIDZapKey, commandID), zap.Error(err))
			}
		}(commandScopeCtx, received)
	}
}

// SingleCommand - is the handler for single command endpoint SingleCommand
//
// One incoming command - full room snapshot after command execution (or simple message about deleted room)
func (s *RoomService) SingleCommand(ctx context.Context, command *r.Command) (*r.SingleEvent, error) {
	return nil, nil
}

func (s *RoomService) sendError(ctx context.Context, stream grpc.BidiStreamingServer[r.Command, r.Event], baseEvent *r.Event, err error) {
	err2 := stream.Send(&r.Event{
		Timestamp: baseEvent.Timestamp,
		RoomId:    baseEvent.RoomId,
		UserId:    baseEvent.UserId,
		Payload:   &r.Event_ErrorMessage{ErrorMessage: &r.ErrorMessage{Error: err.Error()}},
	})
	if err2 != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to send error", zap.Error(err), zap.String("original_error", err.Error()))
	}
}
