package errdef_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"maps"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
)

type (
	customFieldKey string

	customField struct {
		key   errdef.FieldKey
		value any
	}

	customFieldValue[T comparable] struct {
		value T
	}
)

func TestFields_Get(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", constructor("test_value"))
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		value, found := fields.Get(constructor.FieldKey())
		if !found {
			t.Fatal("want field to be found via Fields.Get")
		}
		if value.Value() != "test_value" {
			t.Errorf("want value %q, got %q", "test_value", value)
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		collected := maps.Collect(fields.Seq())
		if len(collected) != 0 {
			t.Errorf("want 0 fields, got %d", len(collected))
		}
	})

	t.Run("custom field", func(t *testing.T) {
		key := customFieldKey("test_field")
		def := errdef.Define("test_error", customField{
			key:   key,
			value: "test_value",
		})
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		value, found := fields.Get(key)
		if !found {
			t.Fatal("want field to be found via Fields.Get")
		}
		if value.Value() != "test_value" {
			t.Errorf("want value %q, got %q", "test_value", value)
		}
	})
}

func TestFields_FindKeys(t *testing.T) {
	constructor1, _ := errdef.DefineField[string]("common_field")
	constructor2, _ := errdef.DefineField[int]("common_field")
	constructor3, _ := errdef.DefineField[bool]("unique_field")

	def := errdef.Define("test_error",
		constructor1("string_value"),
		constructor2(42),
		constructor3(true),
	)
	err := def.New("test message")

	fields := err.(errdef.Error).Fields()
	keys := fields.FindKeys("common_field")
	if len(keys) != 2 {
		t.Fatalf("want 2 keys, got %d", len(keys))
	}

	value1, found1 := fields.Get(keys[0])
	if !found1 {
		t.Fatal("want first common_field to be found")
	}
	if value1.Value() != "string_value" && value1.Value() != 42 {
		t.Errorf("incorrect first common_field value, got %v", value1)
	}

	value2, found2 := fields.Get(keys[1])
	if !found2 {
		t.Fatal("want second common_field to be found")
	}
	if value1.Value() != "string_value" && value1.Value() != 42 {
		t.Errorf("incorrect second common_field value, got %v", value2)
	}
}

func TestFields_Seq(t *testing.T) {
	constructor1, _ := errdef.DefineField[string]("field1")
	constructor2, _ := errdef.DefineField[int]("field2")

	def := errdef.Define("test_error",
		constructor1("value1"),
		constructor2(123),
		customField{
			key:   customFieldKey("field3"),
			value: false,
		},
	)
	err := def.New("test message")

	fields := err.(errdef.Error).Fields()

	collected := make(map[string]any)
	for key, value := range fields.Seq() {
		collected[key.String()] = value.Value()
	}

	want := map[string]any{
		"field1": "value1",
		"field2": 123,
		"field3": false,
	}

	if !reflect.DeepEqual(collected, want) {
		t.Errorf("want fields %v, got %v", want, collected)
	}
}

func TestFields_SortedSeq(t *testing.T) {
	t.Run("basic sorting", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("c_field")
		constructor2, _ := errdef.DefineField[int]("a_field")

		def := errdef.Define("test_error",
			constructor1("value_c"),
			constructor2(123),
			customField{
				key:   customFieldKey("b_field"),
				value: true,
			},
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		var keys []string
		var values []any
		for key, value := range fields.SortedSeq() {
			keys = append(keys, key.String())
			values = append(values, value.Value())
		}

		wantKeys := []string{"c_field", "a_field", "b_field"}
		wantValues := []any{"value_c", 123, true}

		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("want keys %v, got %v", wantKeys, keys)
		}
		if !reflect.DeepEqual(values, wantValues) {
			t.Errorf("want values %v, got %v", wantValues, values)
		}
	})

	t.Run("same name keys", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("same_name")
		constructor2, _ := errdef.DefineField[int]("same_name")
		constructor3, _ := errdef.DefineField[bool]("same_name")

		def := errdef.Define("test_error",
			constructor1("string_value"),
			constructor2(42),
			constructor3(true),
			customField{
				key:   customFieldKey("same_name"),
				value: 3.14,
			},
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		var key1 []string
		var value1 []any
		for key, value := range fields.SortedSeq() {
			key1 = append(key1, key.String())
			value1 = append(value1, value.Value())
		}

		for range 10 {
			var key2 []string
			var value2 []any
			for key, value := range fields.SortedSeq() {
				key2 = append(key2, key.String())
				value2 = append(value2, value.Value())
			}

			if !reflect.DeepEqual(key1, key2) {
				t.Errorf("want keys %v, got %v", key1, key2)
			}
			if !reflect.DeepEqual(value1, value2) {
				t.Errorf("want values %v, got %v", value1, value2)
			}
		}
	})
}

func TestFields_Len(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		if got := fields.Len(); got != 0 {
			t.Errorf("want 0, got %d", got)
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("field1")
		constructor2, _ := errdef.DefineField[int]("field2")
		constructor3, _ := errdef.DefineField[bool]("same_name_field")
		constructor4, _ := errdef.DefineField[float64]("same_name_field")

		def := errdef.Define("test_error",
			constructor1("value1"),
			constructor2(123),
			constructor3(true),
			constructor4(3.14),
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		if got := fields.Len(); got != 4 {
			t.Errorf("want 3, got %d", got)
		}
	})
}

func TestFields_MarshalJSON(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("b_field")
		constructor2, _ := errdef.DefineField[int]("a_field")

		def := errdef.Define("test_error",
			constructor1("string_value"),
			constructor2(42),
			customField{
				key:   customFieldKey("c_field"),
				value: true,
			},
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		jsonData, err := fields.(json.Marshaler).MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{
			"b_field": "string_value",
			"a_field": float64(42),
			"c_field": true,
		}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("want %+v, got %+v", want, result)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		jsonData, err := fields.(json.Marshaler).MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("want %+v, got %+v", want, result)
		}
	})

	t.Run("overwrite same name fields", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("field")
		constructor2, _ := errdef.DefineField[int]("field")
		constructor3, _ := errdef.DefineField[bool]("field")

		def := errdef.Define("test_error",
			constructor1("first"),
			constructor2(42),
			constructor3(true),
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		jsonData, err := fields.(json.Marshaler).MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{
			"field": true,
		}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("want %+v, got %+v", want, result)
		}
	})
}

func TestFields_LogValue(t *testing.T) {
	t.Run("with fields", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("user_id")
		constructor2, _ := errdef.DefineField[int]("status_code")
		def := errdef.Define("test_error", constructor1("user123"), constructor2(404))
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()
		value := fields.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("fields", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{
			"user_id":     "user123",
			"status_code": float64(404),
		}

		if !reflect.DeepEqual(result["fields"], want) {
			t.Errorf("want fields %+v, got %+v", want, result["fields"])
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()
		value := fields.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("fields", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if result["fields"] != nil {
			t.Errorf("want fields to be nil for empty fields, got %+v", result["fields"])
		}
	})

	t.Run("overwrite same name fields", func(t *testing.T) {
		constructor1, _ := errdef.DefineField[string]("field")
		constructor2, _ := errdef.DefineField[int]("field")
		constructor3, _ := errdef.DefineField[bool]("field")

		def := errdef.Define("test_error",
			constructor1("first"),
			constructor2(42),
			constructor3(true),
		)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()
		value := fields.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("fields", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{
			"field": true,
		}

		if !reflect.DeepEqual(result["fields"], want) {
			t.Errorf("want fields %+v, got %+v", want, result["fields"])
		}
	})

	t.Run("custom field key", func(t *testing.T) {
		customKey := customFieldKey("custom_field")
		customOpt := customField{key: customKey, value: "custom_value"}
		def := errdef.Define("test_error", customOpt)
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()
		value := fields.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("fields", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		want := map[string]any{
			"custom_field": "custom_value",
		}

		if !reflect.DeepEqual(result["fields"], want) {
			t.Errorf("want fields %+v, got %+v", want, result["fields"])
		}
	})
}

func (o customField) ApplyOption(a errdef.OptionApplier) {
	a.SetField(o.key, &customFieldValue[any]{value: o.value})
}

func (k customFieldKey) String() string {
	return string(k)
}

func (k customFieldKey) NewValue(value any) (errdef.FieldValue, bool) {
	return nil, false
}

func (k customFieldKey) ZeroValue() errdef.FieldValue {
	return &customFieldValue[any]{value: nil}
}

func (v *customFieldValue[T]) Value() any {
	return v.value
}

func (v *customFieldValue[T]) Equals(other any) bool {
	if tv, ok := other.(T); ok {
		return v.value == tv
	}
	return false
}
