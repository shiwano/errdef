package unmarshaler

import (
	"encoding/json"
	"math"
	"reflect"

	"github.com/shiwano/errdef"
)

func tryConvertFieldValue(fk errdef.FieldKey, value any) (errdef.FieldValue, bool, error) {
	if v, ok := fk.NewValue(value); ok {
		return v, true, nil
	}

	targetType := reflect.TypeOf(fk.ZeroValue().Value())
	if targetType == nil {
		return nil, false, nil
	}

	if f64, ok := value.(float64); ok {
		if v, ok, err := tryConvertFloat64(fk, f64, targetType); err != nil {
			return nil, false, err
		} else if ok {
			return v, true, nil
		}
	}

	if _, ok := value.(map[string]any); ok {
		if v, ok, err := tryConvertViaJSON(fk, value, targetType); err != nil {
			return nil, false, err
		} else if ok {
			return v, true, nil
		}
	}

	if _, ok := value.([]any); ok {
		if v, ok, err := tryConvertViaJSON(fk, value, targetType); err != nil {
			return nil, false, err
		} else if ok {
			return v, true, nil
		}
	}

	valueType := reflect.TypeOf(value)

	if v, ok, err := tryConvertByUnderlyingType(fk, value, targetType, valueType); err != nil {
		return nil, false, err
	} else if ok {
		return v, true, nil
	}

	if v, ok, err := tryConvertPointer(fk, value, targetType, valueType); err != nil {
		return nil, false, err
	} else if ok {
		return v, true, nil
	}

	return nil, false, nil
}

func tryConvertFloat64(fk errdef.FieldKey, f64 float64, targetType reflect.Type) (errdef.FieldValue, bool, error) {
	kind := targetType.Kind()

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false, nil
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
			return nil, false, nil
		}
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		v, ok := fk.NewValue(val)
		return v, ok, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, frac := math.Modf(f64); frac != 0 {
			return nil, false, nil
		}
		if f64 < 0 {
			return nil, false, nil
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
			return nil, false, nil
		}
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		v, ok := fk.NewValue(val)
		return v, ok, nil

	case reflect.Float32:
		if math.Abs(f64) > math.MaxFloat32 {
			return nil, false, nil
		}
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		v, ok := fk.NewValue(val)
		return v, ok, nil

	case reflect.Float64:
		val := reflect.ValueOf(f64).Convert(targetType).Interface()
		v, ok := fk.NewValue(val)
		return v, ok, nil
	}

	return nil, false, nil
}

func tryConvertViaJSON(fk errdef.FieldKey, value any, targetType reflect.Type) (errdef.FieldValue, bool, error) {
	kind := targetType.Kind()

	if kind == reflect.Pointer {
		if targetType.Elem().Kind() != reflect.Struct {
			return nil, false, nil
		}
	} else if kind != reflect.Struct && kind != reflect.Map && kind != reflect.Slice {
		return nil, false, nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, false, ErrInternal.Wrapf(err, "failed to marshal value")
	}

	targetPtr := reflect.New(targetType)
	if err := json.Unmarshal(jsonBytes, targetPtr.Interface()); err != nil {
		return nil, false, ErrFieldUnmarshalFailure.Wrapf(err, "failed to unmarshal to %s", targetType)
	}

	targetValue := targetPtr.Elem().Interface()
	v, ok := fk.NewValue(targetValue)
	return v, ok, nil
}

func tryConvertByUnderlyingType(fk errdef.FieldKey, value any, targetType, valueType reflect.Type) (errdef.FieldValue, bool, error) {
	if valueType == nil {
		return nil, false, nil
	}

	kind := targetType.Kind()
	if kind == reflect.Slice || kind == reflect.Map || kind == reflect.Struct || kind == reflect.Pointer {
		return nil, false, nil
	}

	if targetType.Kind() == valueType.Kind() && targetType != valueType {
		converted := reflect.ValueOf(value).Convert(targetType).Interface()
		v, ok := fk.NewValue(converted)
		return v, ok, nil
	}

	return nil, false, nil
}

func tryConvertPointer(fk errdef.FieldKey, value any, targetType, valueType reflect.Type) (errdef.FieldValue, bool, error) {
	if targetType.Kind() != reflect.Pointer {
		return nil, false, nil
	}

	elemType := targetType.Elem()
	elemKind := elemType.Kind()

	if elemKind == reflect.Slice || elemKind == reflect.Map || elemKind == reflect.Struct {
		return nil, false, nil
	}

	if valueType == nil {
		return nil, false, nil
	}

	if elemKind != valueType.Kind() {
		return nil, false, nil
	}

	ptrVal := reflect.New(elemType)
	converted := reflect.ValueOf(value).Convert(elemType)
	ptrVal.Elem().Set(converted)

	v, ok := fk.NewValue(ptrVal.Interface())
	return v, ok, nil
}
