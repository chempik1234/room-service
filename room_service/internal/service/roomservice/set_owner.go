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
)

type roomServiceSetOwnerParams struct {
	roomID     *models.RoomID
	newOwnerID types.NotEmptyText
}

func (s *RoomService) setOwner(ctx context.Context, params *roomServiceSetOwnerParams) (payload *r.Event_OwnerChanged, err error) {
	ownerChanged := false

	//region logic
	err = retry.Do(func() error {
		var errInternal error
		ownerChanged, err = s.roomsRepo.SetOwnerUserID(ctx, ports.SetOwnerUserIDParams{
			RoomID:     *params.roomID,
			NewOwnerID: params.newOwnerID,
		})
		return errInternal
	}, s.retryStrategy)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to set owner for room", zap.Error(err))
		return payload, fmt.Errorf("failed to set owner for room: %w", err)
	}
	//endregion

	return &r.Event_OwnerChanged{
		OwnerChanged: &r.OwnerChangedEventBody{
			NewOwnerId:      params.newOwnerID.String(),
			OwnerHasChanged: ownerChanged,
		},
	}, nil
}
