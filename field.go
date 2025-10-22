package errdef

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"net/url"
	"reflect"
	"slices"
	"time"
)

type (
	// Fields represents a collection of structured error fields.
	Fields interface {
		// Get retrieves the value associated with the given key.
		Get(key FieldKey) (FieldValue, bool)
		// FindKeys finds all keys that match the given name.
		FindKeys(name string) []FieldKey
		// All returns an iterator over all key-value pairs sorted by insertion order.
		All() iter.Seq2[FieldKey, FieldValue]
		// Len returns the number of fields.
		Len() int
		// IsZero checks if there are no fields.
		IsZero() bool
	}

	// FieldKey represents a key for structured error fields.
	FieldKey interface {
		fmt.Stringer
		// From creates a new FieldValue with the same type from the given value.
		NewValue(value any) (FieldValue, bool)
		// ZeroValue returns a FieldValue representing the zero value for the key's type.
		ZeroValue() FieldValue
	}

	// FieldValue represents a value for structured error fields.
	FieldValue interface {
		// Value returns the underlying value.
		Value() any
		// Equal checks if the value is equal to another value.
		Equal(other any) bool
	}

	fieldsGetter interface {
		Fields() Fields
	}

	fields struct {
		data      map[FieldKey]indexedFieldValue
		lastIndex int
	}

	fieldKey[T any] struct {
		name string
	}

	fieldValue[T any] struct {
		value T
	}

	indexedFieldValue struct {
		value FieldValue
		index int
	}
)

var (
	_ Fields         = (*fields)(nil)
	_ json.Marshaler = (*fields)(nil)
	_ slog.LogValuer = (*fields)(nil)
	_ FieldKey       = (*fieldKey[string])(nil)
	_ FieldValue     = (*fieldValue[string])(nil)
)

func newFields() *fields {
	return &fields{
		data:      nil,
		lastIndex: 0,
	}
}

func (f *fields) Get(key FieldKey) (FieldValue, bool) {
	v, ok := f.data[key]
	if !ok {
		return nil, false
	}
	return v.value, true
}

func (f *fields) FindKeys(name string) []FieldKey {
	var keys []FieldKey
	for k := range f.data {
		if k.String() == name {
			keys = append(keys, k)
		}
	}
	return keys
}

func (f *fields) All() iter.Seq2[FieldKey, FieldValue] {
	return func(yield func(key FieldKey, value FieldValue) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(f.data), func(a, b FieldKey) int {
			return cmp.Compare(f.data[a].index, f.data[b].index)
		}) {
			v := f.data[k]
			if !yield(k, v.value) {
				return
			}
		}
	}
}

func (f *fields) Len() int {
	return len(f.data)
}

func (f *fields) IsZero() bool {
	return len(f.data) == 0
}

func (f *fields) MarshalJSON() ([]byte, error) {
	fields := make(map[string]any, len(f.data))
	for k, v := range f.All() {
		// If multiple fields have the same name,
		// the last one in insertion order will be used.
		fields[k.String()] = v.Value()
	}
	return json.Marshal(fields)
}

func (f *fields) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, f.Len())
	for k, v := range f.All() {
		// If multiple fields have the same name,
		// the last one in insertion order will be used.
		attrs = append(attrs, slog.Any(k.String(), v.Value()))
	}
	return slog.GroupValue(attrs...)
}

func (f *fields) set(key FieldKey, value FieldValue) {
	if f.data == nil {
		f.data = make(map[FieldKey]indexedFieldValue)
	}
	f.lastIndex++
	f.data[key] = indexedFieldValue{
		value: value,
		index: f.lastIndex,
	}
}

func (f *fields) clone() *fields {
	return &fields{
		data:      maps.Clone(f.data),
		lastIndex: f.lastIndex,
	}
}

func (k *fieldKey[T]) String() string {
	return k.name
}

func (k *fieldKey[T]) NewValue(value any) (FieldValue, bool) {
	if tv, ok := value.(T); ok {
		return &fieldValue[T]{value: tv}, true
	}
	return nil, false
}

func (k *fieldKey[T]) ZeroValue() FieldValue {
	var zero T
	return &fieldValue[T]{value: zero}
}

func (v *fieldValue[T]) Value() any {
	return v.value
}

func (v *fieldValue[T]) Equal(other any) bool {
	if tOther, ok := other.(T); ok {
		switch tv := any(v.value).(type) {
		case string:
			return tv == other.(string)
		case int:
			return tv == other.(int)
		case int8:
			return tv == other.(int8)
		case int16:
			return tv == other.(int16)
		case int32:
			return tv == other.(int32)
		case int64:
			return tv == other.(int64)
		case uint:
			return tv == other.(uint)
		case uint8:
			return tv == other.(uint8)
		case uint16:
			return tv == other.(uint16)
		case uint32:
			return tv == other.(uint32)
		case uint64:
			return tv == other.(uint64)
		case float32:
			return tv == other.(float32)
		case float64:
			return tv == other.(float64)
		case bool:
			return tv == other.(bool)
		case complex64:
			return tv == other.(complex64)
		case complex128:
			return tv == other.(complex128)
		case time.Time:
			return tv.Equal(other.(time.Time))
		case time.Duration:
			return tv == other.(time.Duration)
		case []byte:
			return bytes.Equal(tv, other.([]byte))
		case *url.URL:
			return tv.String() == other.(*url.URL).String()
		default:
			vVal := reflect.ValueOf(v.value)
			otherVal := reflect.ValueOf(tOther)
			if vVal.IsValid() && otherVal.IsValid() && vVal.Comparable() && otherVal.Comparable() {
				return vVal.Interface() == otherVal.Interface()
			}
			return reflect.DeepEqual(v.value, tOther)
		}
	} else if fv, ok := other.(FieldValue); ok {
		return v.Equal(fv.Value())
	}
	return false
}

func fieldValueFrom[T any](err error, key FieldKey) (T, bool) {
	var e fieldsGetter
	if errors.As(err, &e) {
		if fv, ok := e.Fields().Get(key); ok {
			v := fv.Value()

			if tv, ok := v.(T); ok {
				return tv, true
			}
		}
	}

	var zero T
	return zero, false
}
