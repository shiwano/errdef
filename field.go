package errdef

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"reflect"
	"slices"
)

type (
	// Fields represents a collection of structured error fields.
	Fields interface {
		json.Marshaler
		// Get retrieves the value associated with the given key.
		Get(key FieldKey) (FieldValue, bool)
		// FindKeys finds all keys that match the given name.
		FindKeys(name string) []FieldKey
		// Seq returns an iterator over all key-value pairs.
		Seq() iter.Seq2[FieldKey, FieldValue]
		// SortedSeq returns an iterator over all key-value pairs sorted by key.
		SortedSeq() iter.Seq2[FieldKey, FieldValue]
		// Len returns the number of fields.
		Len() int
	}

	// FieldKey represents a key for structured error fields.
	FieldKey interface {
		fmt.Stringer
	}

	// FieldValue represents a value for structured error fields.
	FieldValue interface {
		// Value returns the underlying value.
		Value() any
		// Equals checks if the value is equal to another value.
		Equals(other any) bool
	}

	fields map[FieldKey]indexedFieldValue

	fieldKey struct {
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
	_ Fields         = fields{}
	_ slog.LogValuer = fields{}
	_ FieldKey       = (*fieldKey)(nil)
	_ FieldValue     = (*fieldValue[string])(nil)
)

func newFields() fields {
	return make(fields)
}

func (f fields) Get(key FieldKey) (FieldValue, bool) {
	v, ok := f[key]
	return v.value, ok
}

func (f fields) FindKeys(name string) []FieldKey {
	var keys []FieldKey
	for k := range f {
		if k.String() == name {
			keys = append(keys, k)
		}
	}
	return keys
}

func (f fields) Seq() iter.Seq2[FieldKey, FieldValue] {
	return func(yield func(key FieldKey, value FieldValue) bool) {
		for k, v := range f {
			if !yield(k, v.value) {
				return
			}
		}
	}
}

func (f fields) SortedSeq() iter.Seq2[FieldKey, FieldValue] {
	return func(yield func(key FieldKey, value FieldValue) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(f), func(a, b FieldKey) int {
			return cmp.Compare(f[a].index, f[b].index)
		}) {
			v := f[k]
			if !yield(k, v.value) {
				return
			}
		}
	}
}

func (f fields) Len() int {
	return len(f)
}

func (f fields) MarshalJSON() ([]byte, error) {
	type field struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	fields := make([]field, 0, len(f))
	for k, v := range f.SortedSeq() {
		fields = append(fields, field{Key: k.String(), Value: v.Value()})
	}
	return json.Marshal(fields)
}

func (f fields) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, f.Len())
	for k, v := range f.SortedSeq() {
		attrs = append(attrs, slog.Any(k.String(), v.Value()))
	}
	return slog.GroupValue(attrs...)
}

func (f fields) set(key FieldKey, value FieldValue) {
	f[key] = indexedFieldValue{
		value: value,
		index: len(f),
	}
}

func (f fields) clone() fields {
	return maps.Clone(f)
}

func (k *fieldKey) String() string {
	return k.name
}

func (v *fieldValue[T]) Value() any {
	return v.value
}

func (v *fieldValue[T]) Equals(other any) bool {
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
		default:
			vVal := reflect.ValueOf(v.value)
			otherVal := reflect.ValueOf(tOther)
			if vVal.IsValid() && otherVal.IsValid() && vVal.Comparable() && otherVal.Comparable() {
				return vVal.Interface() == otherVal.Interface()
			}
			return reflect.DeepEqual(v.value, tOther)
		}
	} else if fv, ok := other.(FieldValue); ok {
		return v.Equals(fv.Value())
	}
	return false
}

func fieldValueFrom[T any](err error, key FieldKey) (T, bool) {
	var e *definedError
	if errors.As(err, &e) {
		if v, found := e.def.fields.Get(key); found {
			vv := v.Value()

			if tv, ok := vv.(T); ok {
				return tv, true
			}
		}
	}
	var zero T
	return zero, false
}
