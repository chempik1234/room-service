package roomservice

import (
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
	"reflect"
)

// PlainObjectToProtobufValue - convert regular object to room_service.Value
func PlainObjectToProtobufValue(value any) (*r.Value, error) {
	var curValue *r.Value
	switch reflect.ValueOf(value).Kind() {
	case reflect.Int64:
		curValue = &r.Value{Value: &r.Value_IntValue{IntValue: value.(int64)}}
	case reflect.Float64:
		curValue = &r.Value{Value: &r.Value_FloatValue{FloatValue: value.(float64)}}
	case reflect.String:
		curValue = &r.Value{Value: &r.Value_StringValue{StringValue: value.(string)}}
	case reflect.Bool:
		curValue = &r.Value{Value: &r.Value_BoolValue{BoolValue: value.(bool)}}
	case reflect.Map:
		mapObjects := value.(map[string]any)
		resultMap := make(map[string]*r.Value)
		var err error
		for key, valObj := range mapObjects {
			resultMap[key], err = PlainObjectToProtobufValue(valObj)
			if err != nil {
				return nil, fmt.Errorf("error serializing map to Value: %w", err)
			}
		}
	case reflect.Slice:
		listObjects := value.([]any)
		list := make([]*r.Value, len(listObjects))
		var err error
		for index, item := range listObjects {
			list[index], err = PlainObjectToProtobufValue(item)
			if err != nil {
				return nil, fmt.Errorf("error serializing list to Value: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unknown type: %v", reflect.TypeOf(value).Kind())
	}
	return curValue, nil
}

// ProtobufValueToPlainObject - convert room_service.Value to regular object
func ProtobufValueToPlainObject(value *r.Value) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("nil value provided")
	}

	switch v := value.Value.(type) {
	case *r.Value_IntValue:
		return v.IntValue, nil
	case *r.Value_FloatValue:
		return v.FloatValue, nil
	case *r.Value_StringValue:
		return v.StringValue, nil
	case *r.Value_BoolValue:
		return v.BoolValue, nil
	case *r.Value_BinaryValue:
		return v.BinaryValue, nil
	case *r.Value_MapValue:
		resultMap := make(map[string]any)
		for key, val := range v.MapValue.GetValues() {
			obj, err := ProtobufValueToPlainObject(val)
			if err != nil {
				return nil, fmt.Errorf("error deserializing map value for key '%s': %w", key, err)
			}
			resultMap[key] = obj
		}
		return resultMap, nil
	case *r.Value_ListValue:
		resultList := make([]any, len(v.ListValue.GetValues()))
		for index, val := range v.ListValue.GetValues() {
			obj, err := ProtobufValueToPlainObject(val)
			if err != nil {
				return nil, fmt.Errorf("error deserializing list value at index %d: %w", index, err)
			}
			resultList[index] = obj
		}
		return resultList, nil
	default:
		return nil, fmt.Errorf("unknown value type: %T", v)
	}
}

// ProtobufValueToValueObject - convert room_service.Value to models.Value
func ProtobufValueToValueObject(protoValue *r.Value) (val *models.Value, err error) {
	if protoValue == nil {
		return val, fmt.Errorf("nil protoValue provided")
	}

	switch v := protoValue.Value.(type) {
	case *r.Value_IntValue:
		val = models.IntValue(v.IntValue)
	case *r.Value_FloatValue:
		val = models.FloatValue(v.FloatValue)
	case *r.Value_StringValue:
		val = models.StrValue(v.StringValue)
	case *r.Value_BoolValue:
		val = models.BoolValue(v.BoolValue)
	case *r.Value_BinaryValue:
		val = models.BytesValue(v.BinaryValue)
	case *r.Value_MapValue:
		resultMap := make(map[string]models.Value, len(v.MapValue.GetValues()))

		for key, protoItem := range v.MapValue.GetValues() {
			valueItem, err := ProtobufValueToValueObject(protoItem)
			if err != nil {
				return val, fmt.Errorf("error deserializing map value for key '%s' (%v): %w", key, protoItem, err)
			}
			resultMap[key] = *valueItem
		}

		val = models.MapValue(resultMap)
	case *r.Value_ListValue:
		resultList := make([]models.Value, len(v.ListValue.GetValues()))

		for index, protoItem := range v.ListValue.GetValues() {
			obj, err := ProtobufValueToValueObject(protoItem)
			if err != nil {
				return nil, fmt.Errorf("error deserializing list protoValue at index %d (%v): %w", index, protoItem, err)
			}
			resultList[index] = *obj
		}

		val = models.ListValue(resultList)
	default:
		return nil, fmt.Errorf("unknown protoValue type: %T", v)
	}

	return val, nil
}
