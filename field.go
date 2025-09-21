package errdef

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"slices"
)

type (
	// Fields represents a collection of structured error fields.
	Fields interface {
		json.Marshaler
		// Get retrieves the value associated with the given key.
		Get(key FieldKey) (any, bool)
		// FindKeys finds all keys that match the given name.
		FindKeys(name string) []FieldKey
		// Seq returns an iterator over all key-value pairs.
		Seq() iter.Seq2[FieldKey, any]
		// SortedSeq returns an iterator over all key-value pairs sorted by key.
		SortedSeq() iter.Seq2[FieldKey, any]
		// Len returns the number of fields.
		Len() int
	}

	// FieldKey represents a key for structured error fields.
	FieldKey interface {
		fmt.Stringer
	}

	fields map[FieldKey]any

	fieldKey struct {
		name string
	}
)

var (
	_ Fields         = fields{}
	_ slog.LogValuer = (*fields)(nil)
)

func newFields() fields {
	return make(fields)
}

func (f fields) Get(key FieldKey) (any, bool) {
	v, ok := f[key]
	return v, ok
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

func (f fields) Seq() iter.Seq2[FieldKey, any] {
	return func(yield func(key FieldKey, value any) bool) {
		for k, v := range f {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (f fields) SortedSeq() iter.Seq2[FieldKey, any] {
	return func(yield func(key FieldKey, value any) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(f), func(a, b FieldKey) int {
			if a.String() == b.String() {
				vA := f[a]
				vB := f[b]
				vAT := fmt.Sprintf("%T", vA)
				vBT := fmt.Sprintf("%T", vB)
				if vAT == vBT {
					return cmp.Compare(fmt.Sprintf("%v", vA), fmt.Sprintf("%v", vB))
				}
				return cmp.Compare(vAT, vBT)
			}
			return cmp.Compare(a.String(), b.String())
		}) {
			v := f[k]
			if !yield(k, v) {
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
		fields = append(fields, field{Key: k.String(), Value: v})
	}
	return json.Marshal(fields)
}

func (f fields) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, f.Len())
	for k, v := range f.Seq() {
		attrs = append(attrs, slog.Any(k.String(), v))
	}
	return slog.GroupValue(attrs...)
}

func (f fields) set(key FieldKey, value any) {
	f[key] = value
}

func (f fields) clone() fields {
	return maps.Clone(f)
}

func (k fieldKey) String() string {
	return k.name
}

func fieldValueFrom[T any](err error, key FieldKey) (T, bool) {
	var e *definedError
	if errors.As(err, &e) {
		if v, found := e.def.fields.Get(key); found {
			if tv, ok := v.(T); ok {
				return tv, true
			}
		}
	}
	var zero T
	return zero, false
}
