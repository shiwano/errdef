package unmarshaler

import (
	"cmp"
	"encoding/json"
	"iter"
	"log/slog"
	"slices"

	"github.com/shiwano/errdef"
)

type (
	fields struct {
		fields        map[errdef.FieldKey]errdef.FieldValue
		unknownFields map[string]any
	}

	unmarshaledFieldKey string

	unmarshaledFieldValue struct {
		value any
	}
)

var (
	_ errdef.Fields     = (*fields)(nil)
	_ slog.LogValuer    = (*fields)(nil)
	_ json.Marshaler    = (*fields)(nil)
	_ errdef.FieldKey   = unmarshaledFieldKey("")
	_ errdef.FieldValue = (*unmarshaledFieldValue)(nil)
)

func (f *fields) Get(key errdef.FieldKey) (errdef.FieldValue, bool) {
	if v, ok := f.fields[key]; ok {
		return v, true
	}
	if v, ok := f.unknownFields[key.String()]; ok {
		if _, ok := key.(unmarshaledFieldKey); ok {
			return &unmarshaledFieldValue{value: v}, true
		} else if tv, ok, err := tryConvertFieldValue(key, v); ok && err == nil {
			return tv, true
		}
	}
	return nil, false
}

func (f *fields) FindKeys(name string) []errdef.FieldKey {
	var keys []errdef.FieldKey
	for k := range f.fields {
		if k.String() == name {
			keys = append(keys, k)
		}
	}
	if _, ok := f.unknownFields[name]; ok {
		keys = append(keys, unmarshaledFieldKey(name))
	}
	return keys
}

func (f *fields) All() iter.Seq2[errdef.FieldKey, errdef.FieldValue] {
	return func(yield func(key errdef.FieldKey, value errdef.FieldValue) bool) {
		allKeys := make([]errdef.FieldKey, 0, len(f.fields)+len(f.unknownFields))
		for k := range f.fields {
			allKeys = append(allKeys, k)
		}
		for k := range f.unknownFields {
			allKeys = append(allKeys, unmarshaledFieldKey(k))
		}

		slices.SortFunc(allKeys, func(a, b errdef.FieldKey) int {
			return cmp.Compare(a.String(), b.String())
		})

		for _, k := range allKeys {
			if v, ok := f.fields[k]; ok {
				if !yield(k, v) {
					return
				}
			} else if v, ok := f.unknownFields[k.String()]; ok {
				if !yield(k, &unmarshaledFieldValue{value: v}) {
					return
				}
			}
		}
	}
}

func (f *fields) Len() int {
	return len(f.fields) + len(f.unknownFields)
}

func (f *fields) IsZero() bool {
	return len(f.fields)+len(f.unknownFields) == 0
}

func (f *fields) MarshalJSON() ([]byte, error) {
	result := make(map[string]any)
	for k, v := range f.All() {
		result[k.String()] = v.Value()
	}
	return json.Marshal(result)
}

func (f *fields) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, f.Len())
	for k, v := range f.All() {
		attrs = append(attrs, slog.Any(k.String(), v.Value()))
	}
	return slog.GroupValue(attrs...)
}

func (k unmarshaledFieldKey) String() string {
	return string(k)
}

func (k unmarshaledFieldKey) NewValue(value any) (errdef.FieldValue, bool) {
	return nil, false
}

func (k unmarshaledFieldKey) ZeroValue() errdef.FieldValue {
	return &unmarshaledFieldValue{value: nil}
}

func (v *unmarshaledFieldValue) Value() any {
	return v.value
}

func (v *unmarshaledFieldValue) Equal(other any) bool {
	return false
}
