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

	targetType := reflect.TypeOf(fk.ZeroValue().Value())
	if targetType == nil {
		return nil, false
	}

	if f64, ok := value.(float64); ok {
		if v, ok := tryConvertFloat64(fk, f64, targetType); ok {
			return v, true
		}
	}

	if m, ok := value.(map[string]any); ok {
		if v, ok := tryConvertMapToStruct(fk, m, targetType); ok {
			return v, true
		}
		if v, ok := tryConvertMap(fk, m, targetType); ok {
			return v, true
		}
	}

	if s, ok := value.([]any); ok {
		if v, ok := tryConvertSlice(fk, s, targetType); ok {
			return v, true
		}
	}

	valueType := reflect.TypeOf(value)

	if v, ok := tryConvertByUnderlyingType(fk, value, targetType, valueType); ok {
		return v, true
	}

	if v, ok := tryConvertPointer(fk, value, targetType, valueType); ok {
		return v, true
	}

	return nil, false
}

func tryConvertFloat64(fk errdef.FieldKey, f64 float64, targetType reflect.Type) (errdef.FieldValue, bool) {
	kind := targetType.Kind()

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		var min, max float64
		switch kind {
		case reflect.Int:
			min, max = math.MinInt, math.MaxInt
		case reflect.Int8:
			min, max = math.MinInt8, math.MaxInt8
		case reflect.Int16:
			min, max = math.MinInt16, math.MaxInt16
		case reflect.Int32:
			min, max = math.MinInt32, math.MaxInt32
		case reflect.Int64:
			min, max = math.MinInt64, math.MaxInt64
		}
		if f64 < min || f64 > max {
			return nil, false
		}
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		return fk.NewValue(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false
		}
		if f64 < 0 {
			return nil, false
		}
		var max float64
		switch kind {
		case reflect.Uint:
			max = math.MaxUint
		case reflect.Uint8:
			max = math.MaxUint8
		case reflect.Uint16:
			max = math.MaxUint16
		case reflect.Uint32:
			max = math.MaxUint32
		case reflect.Uint64:
			max = math.MaxUint64
		}
		if f64 > max {
			return nil, false
		}
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		return fk.NewValue(val)

	case reflect.Float32, reflect.Float64:
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		return fk.NewValue(val)
	}

	return nil, false
}

func tryConvertMapToStruct(fk errdef.FieldKey, m map[string]any, targetType reflect.Type) (errdef.FieldValue, bool) {
	isPtr := targetType.Kind() == reflect.Pointer
	structType := targetType
	if isPtr {
		if targetType.Elem().Kind() != reflect.Struct {
			return nil, false
		}
		structType = targetType.Elem()
	} else if targetType.Kind() != reflect.Struct {
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

func tryConvertMap(fk errdef.FieldKey, m map[string]any, targetType reflect.Type) (errdef.FieldValue, bool) {
	if targetType.Kind() != reflect.Map {
		return nil, false
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}

	mapPtr := reflect.New(targetType)
	if err := json.Unmarshal(jsonBytes, mapPtr.Interface()); err != nil {
		return nil, false
	}

	mapValue := mapPtr.Elem().Interface()
	return fk.NewValue(mapValue)
}

func tryConvertSlice(fk errdef.FieldKey, s []any, targetType reflect.Type) (errdef.FieldValue, bool) {
	if targetType.Kind() != reflect.Slice {
		return nil, false
	}

	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil, false
	}

	slicePtr := reflect.New(targetType)
	if err := json.Unmarshal(jsonBytes, slicePtr.Interface()); err != nil {
		return nil, false
	}

	sliceValue := slicePtr.Elem().Interface()
	return fk.NewValue(sliceValue)
}

func tryConvertByUnderlyingType(fk errdef.FieldKey, value any, targetType, valueType reflect.Type) (errdef.FieldValue, bool) {
	if valueType == nil {
		return nil, false
	}

	kind := targetType.Kind()
	if kind == reflect.Slice || kind == reflect.Map || kind == reflect.Struct || kind == reflect.Pointer {
		return nil, false
	}

	if targetType.Kind() == valueType.Kind() && targetType != valueType {
		converted := reflect.ValueOf(value).Convert(targetType).Interface()
		return fk.NewValue(converted)
	}

	return nil, false
}

func tryConvertPointer(fk errdef.FieldKey, value any, targetType, valueType reflect.Type) (errdef.FieldValue, bool) {
	if targetType.Kind() != reflect.Pointer {
		return nil, false
	}

	elemType := targetType.Elem()
	elemKind := elemType.Kind()

	if elemKind == reflect.Slice || elemKind == reflect.Map || elemKind == reflect.Struct {
		return nil, false
	}

	if valueType == nil {
		return nil, false
	}

	if elemKind != valueType.Kind() {
		return nil, false
	}

	ptrVal := reflect.New(elemType)
	converted := reflect.ValueOf(value).Convert(elemType)
	ptrVal.Elem().Set(converted)

	return fk.NewValue(ptrVal.Interface())
}
