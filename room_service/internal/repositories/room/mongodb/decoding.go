package mongodb

import (
	"fmt"
	"github.com/chempik1234/room-service/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// decodeBSONValueToModelValue - convert BSON RawValue to models.Value
//
// This helper handles all the different types that can be stored in MongoDB
// and converts them to the appropriate models.Value type
func decodeBSONValueToModelValue(raw bson.RawValue) (*models.Value, error) {
	val := models.EmptyValue()

	switch raw.Type {
	case bson.TypeInt32, bson.TypeInt64:
		var intVal int64
		if err := raw.Unmarshal(&intVal); err != nil {
			return nil, fmt.Errorf("unmarshal int error: %w", err)
		}
		val.SetInt(intVal)

	case bson.TypeString:
		var strVal string
		if err := raw.Unmarshal(&strVal); err != nil {
			return nil, fmt.Errorf("unmarshal string error: %w", err)
		}
		val.SetStr(strVal)

	case bson.TypeBoolean:
		var boolVal bool
		if err := raw.Unmarshal(&boolVal); err != nil {
			return nil, fmt.Errorf("unmarshal bool error: %w", err)
		}
		val.SetBool(boolVal)

	case bson.TypeDouble:
		var floatVal float64
		if err := raw.Unmarshal(&floatVal); err != nil {
			return nil, fmt.Errorf("unmarshal float error: %w", err)
		}
		val.SetFloat(floatVal)

	case bson.TypeBinary:
		var bytesVal []byte
		if err := raw.Unmarshal(&bytesVal); err != nil {
			return nil, fmt.Errorf("unmarshal bytes error: %w", err)
		}
		val.SetBytes(bytesVal)

	case bson.TypeArray:
		// Decode as array of values
		var rawArray []bson.RawValue
		if err := raw.Unmarshal(&rawArray); err != nil {
			return nil, fmt.Errorf("unmarshal array error: %w", err)
		}

		list := make([]models.Value, len(rawArray))
		for i, item := range rawArray {
			decodedItem, err := decodeBSONValueToModelValue(item)
			if err != nil {
				return nil, fmt.Errorf("decode array item %d error: %w", i, err)
			}
			list[i] = *decodedItem
		}
		val.SetList(list)

	case bson.TypeEmbeddedDocument:
		// Decode as map[string]Value
		var rawMap bson.M
		if err := raw.Unmarshal(&rawMap); err != nil {
			return nil, fmt.Errorf("unmarshal document error: %w", err)
		}

		resultMap := make(map[string]models.Value)
		for key, item := range rawMap {
			// For nested values, we need to recursively decode them
			// Marshal back to bytes and decode recursively
			decodedItem, err := decodeInterfaceToModelValue(item)
			if err != nil {
				return nil, fmt.Errorf("decode map value '%s' error: %w", key, err)
			}
			resultMap[key] = *decodedItem
		}
		val.SetMap(resultMap)

	case bson.TypeNull:
		// Null values - treat as empty string for now
		val.SetStr("")

	default:
		return nil, fmt.Errorf("unsupported BSON type: %v", raw.Type)
	}

	return val, nil
}

// encodeModelValueToBSON - convert models.Value to BSON-compatible value
//
// This helper handles all the different models.Value types and converts
// them to appropriate BSON types for MongoDB storage
func encodeModelValueToBSON(val *models.Value) (any, error) {
	if val == nil {
		return nil, nil
	}

	// Use type check methods to determine the actual value type

	switch {
	case val.IsInt():
		intVal, _ := val.GetInt()
		return intVal, nil

	case val.IsStr():
		strVal, _ := val.GetStr()
		return strVal, nil

	case val.IsBool():
		boolVal, _ := val.GetBool()
		return boolVal, nil

	case val.IsFloat():
		floatVal, _ := val.GetFloat()
		return floatVal, nil

	case val.IsBytes():
		bytesVal, _ := val.GetBytes()
		return bytesVal, nil

	case val.IsList():
		// Convert list items
		listVal, _ := val.GetList()
		result := make([]any, len(listVal))
		for i, item := range listVal {
			encodedItem, err := encodeModelValueToBSON(&item)
			if err != nil {
				return nil, fmt.Errorf("encode list item %d error: %w", i, err)
			}
			result[i] = encodedItem
		}
		return result, nil

	case val.IsMap():
		// Convert map items
		mapVal, _ := val.GetMap()
		result := make(map[string]any)
		for key, item := range mapVal {
			encodedItem, err := encodeModelValueToBSON(&item)
			if err != nil {
				return nil, fmt.Errorf("encode map value '%s' error: %w", key, err)
			}
			result[key] = encodedItem
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown value type")
	}
}

// decodeInterfaceToModelValue - convert any to models.Value recursively
//
// This helper handles nested structures from BSON unmarshaling
func decodeInterfaceToModelValue(item any) (*models.Value, error) {
	val := models.EmptyValue()

	switch v := item.(type) {
	case int32:
		val.SetInt(int64(v))
	case int64:
		val.SetInt(v)
	case int:
		val.SetInt(int64(v))
	case string:
		val.SetStr(v)
	case bool:
		val.SetBool(v)
	case float64:
		val.SetFloat(v)
	case []byte:
		val.SetBytes(v)
	case []any:
		// Handle nested arrays
		list := make([]models.Value, len(v))
		for i, elem := range v {
			decodedElem, err := decodeInterfaceToModelValue(elem)
			if err != nil {
				return nil, fmt.Errorf("decode array element %d error: %w", i, err)
			}
			list[i] = *decodedElem
		}
		val.SetList(list)
	case map[string]any:
		// Handle nested objects
		resultMap := make(map[string]models.Value)
		for key, elem := range v {
			decodedElem, err := decodeInterfaceToModelValue(elem)
			if err != nil {
				return nil, fmt.Errorf("decode map value '%s' error: %w", key, err)
			}
			resultMap[key] = *decodedElem
		}
		val.SetMap(resultMap)
		/*
			case primitive.D:
				// Handle BSON documents (ordered maps)
				resultMap := make(map[string]models.Value)
				for _, elem := range v {
					decodedElem, err := decodeInterfaceToModelValue(elem.Value)
					if err != nil {
						return nil, fmt.Errorf("decode bson element '%s' error: %w", elem.Key, err)
					}
					resultMap[elem.Key] = *decodedElem
				}
				val.SetMap(resultMap)
			case primitive.A:
				// Handle BSON arrays
				list := make([]models.Value, len(v))
				for i, elem := range v {
					decodedElem, err := decodeInterfaceToModelValue(elem)
					if err != nil {
						return nil, fmt.Errorf("decode bson array element %d error: %w", i, err)
					}
					list[i] = *decodedElem
				}
				val.SetList(list)
		*/
	case nil:
		// Null values
		val.SetStr("")
	default:
		return nil, fmt.Errorf("unsupported interface type: %T", item)
	}

	return val, nil
}
