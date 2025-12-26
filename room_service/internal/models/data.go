package models

type valueType uint8

const (
	typeNotSet valueType = iota
	typeInt    valueType = iota
	typeStr    valueType = iota
	typeBool   valueType = iota
	typeFloat  valueType = iota
	typeBytes  valueType = iota
	typeList   valueType = iota
	typeMap    valueType = iota
)

// Value - universal value type for storing
type Value struct {
	intValue   *int64
	strValue   *string
	boolValue  *bool
	floatValue *float64
	bytesValue *[]byte
	listValue  *[]Value
	mapValue   *map[string]Value

	valueType valueType
}

// EmptyValue - create empty Value
func EmptyValue() *Value {
	return &Value{valueType: typeNotSet}
}

// IntValue - create value that stores given int64 value
func IntValue(value int64) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetInt(value)
	return val
}

// StrValue - create value that stores given string value
func StrValue(value string) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetStr(value)
	return val
}

// BoolValue - create value that stores given bool value
func BoolValue(value bool) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetBool(value)
	return val
}

// FloatValue - create value that stores given float value
func FloatValue(value float64) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetFloat(value)
	return val
}

// BytesValue - create value that stores given bytes value
func BytesValue(value []byte) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetBytes(value)
	return val
}

// ListValue - create value that stores given slice value
func ListValue(value []Value) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetList(value)
	return val
}

// MapValue - create value that stores given map[string]Value value
func MapValue(value map[string]Value) *Value {
	val := &Value{valueType: typeNotSet}
	val.SetMap(value)
	return val
}

// SetInt - set value type as int64 and assign it a value
func (v *Value) SetInt(i int64) {
	v.valueType = typeInt
	copyVal := i
	v.intValue = &copyVal
}

// SetStr - set value type as str and assign it a value
func (v *Value) SetStr(s string) {
	v.valueType = typeStr
	copyVal := s
	v.strValue = &copyVal
}

// SetBool - set value type as bool and assign it a value
func (v *Value) SetBool(b bool) {
	v.valueType = typeBool
	copyVal := b
	v.boolValue = &copyVal
}

// SetFloat - set value type as float and assign it a value
func (v *Value) SetFloat(f float64) {
	v.valueType = typeFloat
	copyVal := f
	v.floatValue = &copyVal
}

// SetBytes - set value type as bytes and assign it a value
func (v *Value) SetBytes(b []byte) {
	v.valueType = typeBytes
	copyVal := make([]byte, len(b))
	copy(copyVal, b)

	v.resetValues()
	v.bytesValue = &copyVal
}

// SetList - set value type as list and assign it a value
func (v *Value) SetList(l []Value) {
	v.valueType = typeList
	copyVal := make([]Value, len(l))
	copy(copyVal, l)

	v.resetValues()
	v.listValue = &copyVal
}

// SetMap - set value type as map and assign it a value
func (v *Value) SetMap(m map[string]Value) {
	v.valueType = typeMap
	copyVal := make(map[string]Value)
	for key, val := range m {
		copyVal[key] = val
	}

	v.resetValues()
	v.mapValue = &copyVal
}

// Equal - check if values are equal
func (v *Value) Equal(v2 *Value) bool {
	if v.valueType != v2.valueType {
		return false
	}

	switch v.valueType {
	case typeInt, typeStr, typeBool, typeFloat:
		return v.intValue == v2.intValue
	case typeBytes:
		if len(*v2.bytesValue) != len(*v.bytesValue) {
			return false
		}
		for i := range *v.bytesValue {
			if (*v.bytesValue)[i] != (*v2.bytesValue)[i] {
				return false
			}
		}
	case typeList:
		if len(*v2.listValue) != len(*v.listValue) {
			return false
		}
		for i := range *v.listValue {
			if !(*v.listValue)[i].Equal(&(*v2.listValue)[i]) {
				return false
			}
		}
	case typeMap:
		if len(*v2.mapValue) != len(*v.mapValue) {
			return false
		}
		for k, m1 := range *v.mapValue {
			if m2, ok := (*v2.mapValue)[k]; !m2.Equal(&m1) || !ok {
				return false
			}
		}
		for k, m2 := range *v2.mapValue {
			if m1, ok := (*v.mapValue)[k]; !m1.Equal(&m2) || !ok {
				return false
			}
		}
	default:
		panic("unhandled default case")
	}

	return true
}

func (v *Value) resetValues() {
	v.intValue = nil
	v.strValue = nil
	v.boolValue = nil
	v.floatValue = nil
	v.bytesValue = nil
	v.listValue = nil
	v.mapValue = nil
}
