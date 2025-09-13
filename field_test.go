package errdef_test

import (
	"encoding/json"
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
)

func (k customFieldKey) String() string {
	return string(k)
}

func (o customField) ApplyOption(a errdef.OptionApplier) {
	a.SetField(o.key, o.value)
}

func TestFields_Get(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", constructor("test_value"))
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		keys := fields.FindKeys("test_field")
		if len(keys) != 1 {
			t.Fatalf("want 1 key, got %d", len(keys))
		}
		actualKey := keys[0]

		value, found := fields.Get(actualKey)
		if !found {
			t.Fatal("want field to be found via Fields.Get")
		}
		if value != "test_value" {
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
		if value != "test_value" {
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
	if value1 != "string_value" && value1 != 42 {
		t.Errorf("incorrect first common_field value, got %v", value1)
	}

	value2, found2 := fields.Get(keys[1])
	if !found2 {
		t.Fatal("want second common_field to be found")
	}
	if value1 != "string_value" && value1 != 42 {
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
		collected[key.String()] = value
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
			values = append(values, value)
		}

		wantKeys := []string{"a_field", "b_field", "c_field"}
		wantValues := []any{123, true, "value_c"}

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
			value1 = append(value1, value)
		}

		for range 10 {
			var key2 []string
			var value2 []any
			for key, value := range fields.SortedSeq() {
				key2 = append(key2, key.String())
				value2 = append(value2, value)
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

		jsonData, err := fields.MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		want := `[{"key":"a_field","value":42},{"key":"b_field","value":"string_value"},{"key":"c_field","value":true}]`

		if string(jsonData) != want {
			t.Errorf("want JSON %s, got %s", want, string(jsonData))
		}

		var result []map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("want 3 fields, got %d", len(result))
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields := err.(errdef.Error).Fields()

		jsonData, err := fields.MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		want := `[]`

		if string(jsonData) != want {
			t.Errorf("want JSON %s, got %s", want, string(jsonData))
		}

		var result []map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("want 0 fields, got %d", len(result))
		}
	})
}
