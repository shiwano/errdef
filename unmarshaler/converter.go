package unmarshaler

import (
	"encoding/json"
	"math"
	"reflect"

	"github.com/shiwano/errdef"
)

func tryConvertFieldValue(fk errdef.FieldKey, value any) (errdef.FieldValue, bool) {
	if v, ok := fk.NewValue(value); ok {
		return v, true
	}

	if f64, ok := value.(float64); ok {
		if v, ok := tryConvertFloat64(fk, f64); ok {
			return v, true
		}
	}

	if m, ok := value.(map[string]any); ok {
		if v, ok := tryConvertMapToStruct(fk, m); ok {
			return v, true
		}
	}

	if s, ok := value.([]any); ok {
		if v, ok := tryConvertSlice(fk, s); ok {
			return v, true
		}
	}

	return nil, false
}

func tryConvertFloat64(fk errdef.FieldKey, f64 float64) (errdef.FieldValue, bool) {
	switch fk.ZeroValue().Value().(type) {
	case int:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < math.MinInt || f64 > math.MaxInt {
			return nil, false
		}
		return fk.NewValue(int(f64))
	case int8:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < math.MinInt8 || f64 > math.MaxInt8 {
			return nil, false
		}
		return fk.NewValue(int8(f64))
	case int16:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < math.MinInt16 || f64 > math.MaxInt16 {
			return nil, false
		}
		return fk.NewValue(int16(f64))
	case int32:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < math.MinInt32 || f64 > math.MaxInt32 {
			return nil, false
		}
		return fk.NewValue(int32(f64))
	case int64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < math.MinInt64 || f64 > math.MaxInt64 {
			return nil, false
		}
		return fk.NewValue(int64(f64))
	case uint:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 || f64 > math.MaxUint {
			return nil, false
		}
		return fk.NewValue(uint(f64))
	case uint8:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 || f64 > math.MaxUint8 {
			return nil, false
		}
		return fk.NewValue(uint8(f64))
	case uint16:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 || f64 > math.MaxUint16 {
			return nil, false
		}
		return fk.NewValue(uint16(f64))
	case uint32:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 || f64 > math.MaxUint32 {
			return nil, false
		}
		return fk.NewValue(uint32(f64))
	case uint64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 || f64 > math.MaxUint64 {
			return nil, false
		}
		return fk.NewValue(uint64(f64))
	case float32:
		return fk.NewValue(float32(f64))
	case float64:
		return fk.NewValue(f64)
	}
	return nil, false
}

func tryConvertMapToStruct(fk errdef.FieldKey, m map[string]any) (errdef.FieldValue, bool) {
	valueType := reflect.TypeOf(fk.ZeroValue().Value())
	if valueType == nil {
		return nil, false
	}

	isPtr := valueType.Kind() == reflect.Pointer
	structType := valueType
	if isPtr {
		if valueType.Elem().Kind() != reflect.Struct {
			return nil, false
		}
		structType = valueType.Elem()
	} else if valueType.Kind() != reflect.Struct {
		return nil, false
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}

	structPtr := reflect.New(structType)
	if err := json.Unmarshal(jsonBytes, structPtr.Interface()); err != nil {
		return nil, false
	}

	if isPtr {
		return fk.NewValue(structPtr.Interface())
	}

	structValue := structPtr.Elem().Interface()
	return fk.NewValue(structValue)
}

func tryConvertSlice(fk errdef.FieldKey, s []any) (errdef.FieldValue, bool) {
	valueType := reflect.TypeOf(fk.ZeroValue().Value())
	if valueType == nil {
		return nil, false
	}

	if valueType.Kind() != reflect.Slice {
		return nil, false
	}

	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil, false
	}

	slicePtr := reflect.New(valueType)
	if err := json.Unmarshal(jsonBytes, slicePtr.Interface()); err != nil {
		return nil, false
	}

	sliceValue := slicePtr.Elem().Interface()
	return fk.NewValue(sliceValue)
}
