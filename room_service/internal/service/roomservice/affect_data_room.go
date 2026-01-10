package roomservice

import (
	"context"
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"github.com/chempik1234/room-service/internal/ports"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/types"
	"github.com/wb-go/wbf/retry"
)

type affectDataParams struct {
	RoomID    *models.RoomID
	DataID    types.AnyText
	ItemIndex types.AnyText // optional
	Action    ports.Action
}

func (s *RoomService) affectDataInRoom(ctx context.Context, value *r.Value, dataEditMode r.DateEditMode, params *affectDataParams) (payload *r.Event_DataEdited, err error) {
	plainValue, err := ProtobufValueToValueObject(value)
	if err != nil {
		return nil, fmt.Errorf("error deserializing value: %w", err)
	}

	var resultValue *models.Value

	err = retry.Do(func() error {
		var errOp error
		resultValue, errOp = s.roomsRepo.AffectData(ctx, ports.AffectDataParams{
			RoomID: *params.RoomID,
			DataID: params.DataID,
			Action: params.Action,
			Value:  plainValue,
		})
		return errOp
	}, s.retryStrategy)
	if err != nil {
		return payload, fmt.Errorf("failed to affect data in room: %w", err)
	}

	//region result
	return &r.Event_DataEdited{
		DataEdited: &r.DataEditedEventBody{
			DataId:      params.DataID.String(),
			DataValue:   ValueObjectToProtobufValue(resultValue),
			CommandMode: dataEditMode,
			RoomId:      params.RoomID.String(),
		},
	}, nil
	//endregion
}
