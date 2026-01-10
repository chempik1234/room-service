package roomservice

import (
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	r "github.com/chempik1234/room-service/pkg/api/room_service"
)

// ValueObjectToProtobufValue - convert models.Value object to room_service.Value
func ValueObjectToProtobufValue(value *models.Value) *r.Value {
	var result *r.Value
	if value.IsBytes() {
		bytes, _ := value.GetBytes()
		result = &r.Value{Value: &r.Value_BinaryValue{BinaryValue: bytes}}
	} else if value.IsBool() {
		boolValue, _ := value.GetBool()
		result = &r.Value{Value: &r.Value_BoolValue{BoolValue: boolValue}}
	} else if value.IsFloat() {
		floatValue, _ := value.GetFloat()
		result = &r.Value{Value: &r.Value_FloatValue{FloatValue: floatValue}}
	} else if value.IsInt() {
		intValue, _ := value.GetInt()
		result = &r.Value{Value: &r.Value_IntValue{IntValue: intValue}}
	} else if value.IsStr() {
		strValue, _ := value.GetStr()
		result = &r.Value{Value: &r.Value_StringValue{StringValue: strValue}}
	} else if value.IsList() {
		listValue, _ := value.GetList()
		protoList := make([]*r.Value, len(listValue))
		for index, element := range listValue {
			protoList[index] = ValueObjectToProtobufValue(&element)
		}
		result = &r.Value{Value: &r.Value_ListValue{ListValue: &r.ListValue{Values: protoList}}}
	} else if value.IsMap() {
		mapValue, _ := value.GetMap()
		protoMap := make(map[string]*r.Value, len(mapValue))
		for key, element := range mapValue {
			protoMap[key] = ValueObjectToProtobufValue(&element)
		}
		result = &r.Value{Value: &r.Value_MapValue{MapValue: &r.MapValue{Values: protoMap}}}
	}

	return result
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
