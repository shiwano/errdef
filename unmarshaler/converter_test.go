package unmarshaler_test

import (
	"encoding/json"
	"maps"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
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
		{"float32 overflow positive", float32(0), 3.5e38, false},
		{"float32 overflow negative", float32(0), -3.5e38, false},
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

			resolverWithField := resolver.New(def)
			u := unmarshaler.NewJSON(resolverWithField)

			unmarshaled, err := u.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if tt.wantOk {
				fields := unmarshaled.Fields()
				keys := fields.FindKeys("test")
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

				if _, ok := maps.Collect(unmarshaled.UnknownFields())["test"]; ok {
					t.Error("want field not to be in unknownFields")
				}
			} else {
				val, ok := maps.Collect(unmarshaled.UnknownFields())["test"]
				if !ok {
					t.Error("want field to be in unknownFields")
				} else {
					if reflect.TypeOf(val) != reflect.TypeOf(tt.input) {
						t.Errorf("want original value type %T, got %T", tt.input, val)
					}
					if val != tt.input {
						t.Errorf("want original value %v, got %v", tt.input, val)
					}
				}
			}
		})
	}

	t.Run("int derived type", func(t *testing.T) {
		type Age int

		ageCtor, ageFrom := errdef.DefineField[Age]("age")
		def := errdef.Define("test_error", ageCtor(0))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"age": 30
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := ageFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != 30 {
			t.Errorf("want %d, got %d", 30, got)
		}
	})

	t.Run("float64 derived type", func(t *testing.T) {
		type Score float64

		scoreCtor, scoreFrom := errdef.DefineField[Score]("score")
		def := errdef.Define("test_error", scoreCtor(0))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"score": 98.5
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := scoreFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != 98.5 {
			t.Errorf("want %v, got %v", 98.5, got)
		}
	})
}

func TestTryConvertInt64(t *testing.T) {
	tests := []struct {
		name      string
		fieldType any
		input     int64
		wantOk    bool
	}{
		{"int valid", int(0), 100, true},
		{"int8 valid", int8(0), 127, true},
		{"int16 valid", int16(0), 32767, true},
		{"int32 valid", int32(0), 2147483647, true},
		{"int64 valid", int64(0), 9223372036854775807, true},

		{"int8 overflow positive", int8(0), 128, false},
		{"int8 overflow negative", int8(0), -129, false},
		{"int16 overflow positive", int16(0), 32768, false},
		{"int16 overflow negative", int16(0), -32769, false},
		{"int32 overflow positive", int32(0), 2147483648, false},
		{"int32 overflow negative", int32(0), -2147483649, false},

		{"uint valid", uint(0), 100, true},
		{"uint8 valid", uint8(0), 255, true},
		{"uint16 valid", uint16(0), 65535, true},
		{"uint32 valid", uint32(0), 4294967295, true},
		{"uint64 valid", uint64(0), 9223372036854775807, true},

		{"uint negative", uint(0), -1, false},
		{"uint8 negative", uint8(0), -1, false},
		{"uint16 negative", uint16(0), -1, false},
		{"uint32 negative", uint32(0), -1, false},
		{"uint64 negative", uint64(0), -1, false},

		{"uint8 overflow", uint8(0), 256, false},
		{"uint16 overflow", uint16(0), 65536, false},
		{"uint32 overflow", uint32(0), 4294967296, false},

		{"float32 valid small", float32(0), 100, true},
		{"float64 valid", float64(0), 123456789, true},
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

			resolverWithField := resolver.New(def)
			u := unmarshaler.NewJSON(resolverWithField)

			unmarshaled, err := u.Unmarshal(data)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if tt.wantOk {
				fields := unmarshaled.Fields()
				keys := fields.FindKeys("test")
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

				if _, ok := maps.Collect(unmarshaled.UnknownFields())["test"]; ok {
					t.Error("want field not to be in unknownFields")
				}
			} else {
				val, ok := maps.Collect(unmarshaled.UnknownFields())["test"]
				if !ok {
					t.Error("want field to be in unknownFields")
				} else {
					if val != float64(tt.input) && val != tt.input {
						t.Errorf("want original value %v, got %v", tt.input, val)
					}
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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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

func TestTryConvertMap(t *testing.T) {
	t.Run("map derived type", func(t *testing.T) {
		type Metadata map[string]string

		metadataCtor, metadataFrom := errdef.DefineField[Metadata]("metadata")
		def := errdef.Define("test_error", metadataCtor(Metadata{}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"metadata": {
					"key1": "value1",
					"key2": "value2"
				}
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := metadataFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		want := Metadata{"key1": "value1", "key2": "value2"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
			r := resolver.New(def)
			u := unmarshaler.NewJSON(r)

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
			r := resolver.New(def)
			u := unmarshaler.NewJSON(r)

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

	t.Run("slice derived type", func(t *testing.T) {
		type IDs []int

		idsCtor, idsFrom := errdef.DefineField[IDs]("ids")
		def := errdef.Define("test_error", idsCtor(IDs{}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"ids": [1, 2, 3]
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := idsFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		want := IDs{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})
}

func TestTryConvertByUnderlyingType(t *testing.T) {
	t.Run("string derived type", func(t *testing.T) {
		type UserID string

		userIDCtor, userIDFrom := errdef.DefineField[UserID]("user_id")
		def := errdef.Define("test_error", userIDCtor(""))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"user_id": "user123"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := userIDFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != "user123" {
			t.Errorf("want %q, got %q", "user123", got)
		}
	})

	t.Run("bool derived type", func(t *testing.T) {
		type Flag bool

		flagCtor, flagFrom := errdef.DefineField[Flag]("flag")
		def := errdef.Define("test_error", flagCtor(false))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"flag": true
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := flagFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got != true {
			t.Errorf("want %v, got %v", true, got)
		}
	})
}

func TestTryConvertFieldValue(t *testing.T) {
	t.Run("direct conversion success", func(t *testing.T) {
		stringCtor, stringFrom := errdef.DefineField[string]("test")
		def := errdef.Define("test_error", stringCtor(""))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

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

		_, ok := intFrom(unmarshaled)
		if ok {
			t.Error("want field not to be converted")
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("test")
		if len(keys) == 0 {
			t.Fatal("want field to be in unknownFields")
		}
		val, ok := fields.Get(keys[0])
		if !ok {
			t.Fatal("want field to be accessible")
		}
		if val.Value() != "not a number" {
			t.Errorf("want original value %q, got %v", "not a number", val.Value())
		}
	})
}

func TestTryConvertViaJSONError(t *testing.T) {
	t.Run("type mismatch in struct field", func(t *testing.T) {
		type User struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		userCtor, _ := errdef.DefineField[User]("user")
		def := errdef.Define("test_error", userCtor(User{ID: 999, Name: "default"}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"user": {
					"id": "not_a_number",
					"name": "John"
				}
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err == nil {
			t.Fatal("want unmarshal error")
		}

		if unmarshaled != nil {
			t.Errorf("want nil result on error, got %v", unmarshaled)
		}
	})

	t.Run("invalid map structure", func(t *testing.T) {
		type Config struct {
			Value string `json:"value"`
		}

		configCtor, configFrom := errdef.DefineField[Config]("config")
		def := errdef.Define("test_error", configCtor(Config{Value: "default"}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"config": {
					"value": ["array", "not", "string"]
				}
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err == nil {
			t.Fatal("want unmarshal error")
		}

		if unmarshaled != nil {
			got, ok := configFrom(unmarshaled)
			if ok && got.Value == "default" {
				return
			}
			t.Errorf("want default value on error")
		}
	})
}

func TestTryConvertNilValue(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		type Config struct {
			Value string `json:"value"`
		}

		configCtor, configFrom := errdef.DefineField[Config]("config")
		def := errdef.Define("test_error", configCtor(Config{Value: "default"}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"config": null
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		_, ok := configFrom(unmarshaled)
		if ok {
			t.Error("want field not to be converted from nil")
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("config")
		if len(keys) == 0 {
			t.Fatal("want field to be in unknownFields")
		}
		val, ok := fields.Get(keys[0])
		if !ok {
			t.Fatal("want field to be accessible")
		}
		if val.Value() != nil {
			t.Errorf("want nil value, got %v", val.Value())
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		idsCtor, idsFrom := errdef.DefineField[[]int]("ids")
		def := errdef.Define("test_error", idsCtor([]int{1, 2, 3}))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"ids": null
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		_, ok := idsFrom(unmarshaled)
		if ok {
			t.Error("want field not to be converted from nil")
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("ids")
		if len(keys) == 0 {
			t.Fatal("want field to be in unknownFields")
		}
		val, ok := fields.Get(keys[0])
		if !ok {
			t.Fatal("want field to be accessible")
		}
		if val.Value() != nil {
			t.Errorf("want nil value, got %v", val.Value())
		}
	})

	t.Run("nil pointer field", func(t *testing.T) {
		type UserID string

		userIDCtor, userIDFrom := errdef.DefineField[*UserID]("user_id")
		defaultID := UserID("default")
		def := errdef.Define("test_error", userIDCtor(&defaultID))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"user_id": null
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		_, ok := userIDFrom(unmarshaled)
		if ok {
			t.Error("want field not to be converted from nil")
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("user_id")
		if len(keys) == 0 {
			t.Fatal("want field to be in unknownFields")
		}
		val, ok := fields.Get(keys[0])
		if !ok {
			t.Fatal("want field to be accessible")
		}
		if val.Value() != nil {
			t.Errorf("want nil value, got %v", val.Value())
		}
	})
}

func TestTryConvertPointer(t *testing.T) {
	t.Run("pointer to string derived type", func(t *testing.T) {
		type UserID string

		userIDCtor, userIDFrom := errdef.DefineField[*UserID]("user_id")
		def := errdef.Define("test_error", userIDCtor(nil))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"user_id": "user123"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := userIDFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got == nil {
			t.Fatal("want non-nil pointer")
		}

		if *got != "user123" {
			t.Errorf("want %q, got %q", "user123", *got)
		}
	})

	t.Run("pointer to bool derived type", func(t *testing.T) {
		type Flag bool

		flagCtor, flagFrom := errdef.DefineField[*Flag]("flag")
		def := errdef.Define("test_error", flagCtor(nil))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"flag": true
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, ok := flagFrom(unmarshaled)
		if !ok {
			t.Fatal("want field to be found")
		}

		if got == nil {
			t.Fatal("want non-nil pointer")
		}

		wantFlag := Flag(true)
		if *got != wantFlag {
			t.Errorf("want %v, got %v", wantFlag, *got)
		}
	})

	t.Run("pointer to struct is converted via tryConvertMapToStruct", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
		}

		addressCtor, addressFrom := errdef.DefineField[*Address]("address")
		def := errdef.Define("test_error", addressCtor(nil))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		jsonData := `{
			"message": "test",
			"kind": "test_error",
			"fields": {
				"address": {
					"street": "123 Main St"
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

		if got.Street != "123 Main St" {
			t.Errorf("want converted struct, got %+v", got)
		}
	})
}
