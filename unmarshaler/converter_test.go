package unmarshaler_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/unmarshaler"
)

func TestTryConvertFloat64(t *testing.T) {
	tests := []struct {
		name      string
		fieldType any
		input     float64
		wantOk    bool
	}{
		{"int valid", int(0), 100.0, true},
		{"int8 valid", int8(0), 127.0, true},
		{"int16 valid", int16(0), 32767.0, true},
		{"int32 valid", int32(0), 2147483647.0, true},
		{"int64 valid", int64(0), 9223372036854775807.0, true},

		{"int fractional", int(0), 100.5, false},
		{"int8 fractional", int8(0), 10.5, false},
		{"int16 fractional", int16(0), 100.5, false},
		{"int32 fractional", int32(0), 100.5, false},
		{"int64 fractional", int64(0), 100.5, false},

		{"int8 overflow positive", int8(0), 128.0, false},
		{"int8 overflow negative", int8(0), -129.0, false},
		{"int16 overflow positive", int16(0), 32768.0, false},
		{"int16 overflow negative", int16(0), -32769.0, false},

		{"uint valid", uint(0), 100.0, true},
		{"uint8 valid", uint8(0), 255.0, true},
		{"uint16 valid", uint16(0), 65535.0, true},
		{"uint32 valid", uint32(0), 4294967295.0, true},

		{"uint fractional", uint(0), 100.5, false},
		{"uint8 fractional", uint8(0), 10.5, false},
		{"uint16 fractional", uint16(0), 100.5, false},
		{"uint32 fractional", uint32(0), 100.5, false},
		{"uint64 fractional", uint64(0), 100.5, false},

		{"uint negative", uint(0), -1.0, false},
		{"uint8 negative", uint8(0), -1.0, false},

		{"uint8 overflow", uint8(0), 256.0, false},
		{"uint16 overflow", uint16(0), 65536.0, false},

		{"float32 valid", float32(0), 3.14, true},
		{"float64 valid", float64(0), 3.14159, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := map[string]any{
				"message": "test",
				"kind":    "test_error",
				"fields": map[string]any{
					"test": tt.input,
				},
			}

			data, err := json.Marshal(jsonData)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var def *errdef.Definition
			switch tt.fieldType.(type) {
			case int:
				ctor, _ := errdef.DefineField[int]("test")
				def = errdef.Define("test_error", ctor(0))
			case int8:
				ctor, _ := errdef.DefineField[int8]("test")
				def = errdef.Define("test_error", ctor(0))
			case int16:
				ctor, _ := errdef.DefineField[int16]("test")
				def = errdef.Define("test_error", ctor(0))
			case int32:
				ctor, _ := errdef.DefineField[int32]("test")
				def = errdef.Define("test_error", ctor(0))
			case int64:
				ctor, _ := errdef.DefineField[int64]("test")
				def = errdef.Define("test_error", ctor(0))
			case uint:
				ctor, _ := errdef.DefineField[uint]("test")
				def = errdef.Define("test_error", ctor(0))
			case uint8:
				ctor, _ := errdef.DefineField[uint8]("test")
				def = errdef.Define("test_error", ctor(0))
			case uint16:
				ctor, _ := errdef.DefineField[uint16]("test")
				def = errdef.Define("test_error", ctor(0))
			case uint32:
				ctor, _ := errdef.DefineField[uint32]("test")
				def = errdef.Define("test_error", ctor(0))
			case uint64:
				ctor, _ := errdef.DefineField[uint64]("test")
				def = errdef.Define("test_error", ctor(0))
			case float32:
				ctor, _ := errdef.DefineField[float32]("test")
				def = errdef.Define("test_error", ctor(0))
			case float64:
				ctor, _ := errdef.DefineField[float64]("test")
				def = errdef.Define("test_error", ctor(0))
			}

			resolverWithField := errdef.NewResolver(def)
			u := unmarshaler.NewJSON(resolverWithField)

			unmarshaled, err := u.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			fields := unmarshaled.Fields()
			keys := fields.FindKeys("test")

			if tt.wantOk {
				val, ok := fields.Get(keys[0])
				if !ok {
					t.Error("want field to be accessible")
				}
				if reflect.TypeOf(val.Value()) != reflect.TypeOf(tt.fieldType) {
					t.Errorf("want type %T, got %T", tt.fieldType, val.Value())
				}
				if reflect.ValueOf(val.Value()).IsZero() {
					t.Errorf("want value to be non-zero, got zero value %v", val.Value())
				}
			} else {
				val, ok := fields.Get(keys[0])
				if !ok {
					t.Error("want field to be accessible")
				}
				if reflect.TypeOf(val.Value()) != reflect.TypeOf(tt.fieldType) {
					t.Errorf("want default value with type %T, got %T", tt.fieldType, val.Value())
				}
				if !reflect.ValueOf(val.Value()).IsZero() {
					t.Errorf("want default value to be zero value, got %v", val.Value())
				}
			}
		})
	}
}

func TestTryConvertMapToStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	jsonData := `{
		"message": "test message",
		"kind": "test_error",
		"fields": {
			"address": {
				"street": "123 Main St",
				"city": "New York"
			}
		}
	}`

	t.Run("not pointer", func(t *testing.T) {
		addressCtor, addressFrom := errdef.DefineField[Address]("address")
		def := errdef.Define("test_error", addressCtor(Address{}))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		addr, ok := addressFrom(unmarshaled)
		if !ok {
			t.Fatal("want address field to be found")
		}

		if addr.Street != "123 Main St" {
			t.Errorf("want street %q, got %q", "123 Main St", addr.Street)
		}
		if addr.City != "New York" {
			t.Errorf("want city %q, got %q", "New York", addr.City)
		}
	})

	t.Run("pointer", func(t *testing.T) {
		addressCtor, addressFrom := errdef.DefineField[*Address]("address")
		def := errdef.Define("test_error", addressCtor(&Address{}))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		addr, ok := addressFrom(unmarshaled)
		if !ok {
			t.Fatal("want address field to be found")
		}

		if addr.Street != "123 Main St" {
			t.Errorf("want street %q, got %q", "123 Main St", addr.Street)
		}
		if addr.City != "New York" {
			t.Errorf("want city %q, got %q", "New York", addr.City)
		}
	})
}

func TestTryConvertSlice(t *testing.T) {
	t.Run("slice of primitives", func(t *testing.T) {
		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"ids": [1, 2, 3, 4, 5]
			}
		}`

		idsCtor, idsFrom := errdef.DefineField[[]int]("ids")
		def := errdef.Define("test_error", idsCtor([]int{}))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		ids, ok := idsFrom(unmarshaled)
		if !ok {
			t.Fatal("want ids field to be found")
		}

		want := []int{1, 2, 3, 4, 5}
		if !reflect.DeepEqual(ids, want) {
			t.Errorf("want ids %v, got %v", want, ids)
		}
	})

	t.Run("slice of structs", func(t *testing.T) {
		type Item struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"items": [
					{"id": 1, "name": "item1"},
					{"id": 2, "name": "item2"}
				]
			}
		}`

		t.Run("not pointer", func(t *testing.T) {
			itemsCtor, itemsFrom := errdef.DefineField[[]Item]("items")
			def := errdef.Define("test_error", itemsCtor([]Item{}))
			resolver := errdef.NewResolver(def)
			u := unmarshaler.NewJSON(resolver)

			unmarshaled, err := u.Unmarshal([]byte(jsonData))
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			items, ok := itemsFrom(unmarshaled)
			if !ok {
				t.Fatal("want items field to be found")
			}

			if len(items) != 2 {
				t.Fatalf("want 2 items, got %d", len(items))
			}

			if items[0].ID != 1 || items[0].Name != "item1" {
				t.Errorf("want items[0] {ID: 1, Name: \"item1\"}, got %+v", items[0])
			}
			if items[1].ID != 2 || items[1].Name != "item2" {
				t.Errorf("want items[1] {ID: 2, Name: \"item2\"}, got %+v", items[1])
			}
		})

		t.Run("pointer", func(t *testing.T) {
			itemsCtor, itemsFrom := errdef.DefineField[[]*Item]("items")
			def := errdef.Define("test_error", itemsCtor([]*Item{}))
			resolver := errdef.NewResolver(def)
			u := unmarshaler.NewJSON(resolver)

			unmarshaled, err := u.Unmarshal([]byte(jsonData))
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			items, ok := itemsFrom(unmarshaled)
			if !ok {
				t.Fatal("want items field to be found")
			}

			if len(items) != 2 {
				t.Fatalf("want 2 items, got %d", len(items))
			}

			if items[0].ID != 1 || items[0].Name != "item1" {
				t.Errorf("want items[0] {ID: 1, Name: \"item1\"}, got %+v", items[0])
			}
			if items[1].ID != 2 || items[1].Name != "item2" {
				t.Errorf("want items[1] {ID: 2, Name: \"item2\"}, got %+v", items[1])
			}
		})
	})
}

func TestTryConvertFieldValue(t *testing.T) {
	t.Run("direct conversion success", func(t *testing.T) {
		stringCtor, stringFrom := errdef.DefineField[string]("test")
		def := errdef.Define("test_error", stringCtor(""))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"test": "hello"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := stringFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != "hello" {
			t.Errorf("want %q, got %q", "hello", got)
		}
	})

	t.Run("float64 conversion", func(t *testing.T) {
		intCtor, intFrom := errdef.DefineField[int]("test")
		def := errdef.Define("test_error", intCtor(0))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"test": 42.0
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := intFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != 42 {
			t.Errorf("want %d, got %d", 42, got)
		}
	})

	t.Run("map to struct conversion", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		addressCtor, addressFrom := errdef.DefineField[Address]("test")
		def := errdef.Define("test_error", addressCtor(Address{}))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"test": {
					"street": "123 Main St",
					"city": "Tokyo"
				}
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := addressFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got.Street != "123 Main St" || got.City != "Tokyo" {
			t.Errorf("want {Street: \"123 Main St\", City: \"Tokyo\"}, got %+v", got)
		}
	})

	t.Run("slice conversion", func(t *testing.T) {
		sliceCtor, sliceFrom := errdef.DefineField[[]int]("test")
		def := errdef.Define("test_error", sliceCtor([]int{}))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"test": [1, 2, 3]
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := sliceFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		want := []int{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("conversion failure", func(t *testing.T) {
		intCtor, intFrom := errdef.DefineField[int]("test")
		def := errdef.Define("test_error", intCtor(99))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"test": "not a number"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := intFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != 99 {
			t.Errorf("want default value %d, got %d", 99, got)
		}
	})
}
