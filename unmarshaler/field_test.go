package unmarshaler_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/unmarshaler"
)

func TestFields_Defined(t *testing.T) {
	userID, userIDFrom := errdef.DefineField[string]("user_id")
	count, countFrom := errdef.DefineField[int]("count")

	def := errdef.Define("test_error", userID("user123"), count(42))
	resolver := errdef.NewResolver(def)
	u := unmarshaler.NewJSON(resolver)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got, ok := userIDFrom(unmarshaled); !ok || got != "user123" {
		t.Errorf("want user_id %q, got %q (ok=%v)", "user123", got, ok)
	}

	if got, ok := countFrom(unmarshaled); !ok || got != 42 {
		t.Errorf("want count %d, got %d (ok=%v)", 42, got, ok)
	}
}

func TestFields_Unknown(t *testing.T) {
	def := errdef.Define("test_error")
	resolver := errdef.NewResolver(def)
	u := unmarshaler.NewJSON(resolver)

	jsonData := `{
		"message": "test message",
		"kind": "test_error",
		"fields": {
			"unknown_field": "unknown_value",
			"another_field": 123
		}
	}`

	unmarshaled, err := u.Unmarshal([]byte(jsonData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	fields := unmarshaled.Fields()
	if fields.Len() != 2 {
		t.Errorf("want 2 unknown fields, got %d", fields.Len())
	}

	keys := fields.FindKeys("unknown_field")
	if len(keys) != 1 {
		t.Fatalf("want 1 key for unknown_field, got %d", len(keys))
	}

	value, ok := fields.Get(keys[0])
	if !ok {
		t.Fatal("want unknown_field to be found")
	}

	if value.Value() != "unknown_value" {
		t.Errorf("want value %q, got %v", "unknown_value", value.Value())
	}
}

func TestFields_Mixed(t *testing.T) {
	userID, userIDFrom := errdef.DefineField[string]("user_id")
	def := errdef.Define("test_error", userID("user123"))
	resolver := errdef.NewResolver(def)
	u := unmarshaler.NewJSON(resolver)

	jsonData := `{
		"message": "test message",
		"kind": "test_error",
		"fields": {
			"user_id": "user123",
			"unknown_field": "value"
		}
	}`

	unmarshaled, err := u.Unmarshal([]byte(jsonData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got, ok := userIDFrom(unmarshaled); !ok || got != "user123" {
		t.Errorf("want user_id %q, got %q (ok=%v)", "user123", got, ok)
	}

	fields := unmarshaled.Fields()
	keys := fields.FindKeys("unknown_field")
	if len(keys) != 1 {
		t.Fatalf("want 1 key for unknown_field, got %d", len(keys))
	}

	value, ok := fields.Get(keys[0])
	if !ok {
		t.Fatal("want unknown_field to be found")
	}

	if value.Value() != "value" {
		t.Errorf("want value %q, got %v", "value", value.Value())
	}
}

func TestFields_Seq(t *testing.T) {
	userID, _ := errdef.DefineField[string]("user_id")
	countField, _ := errdef.DefineField[int]("count")
	def := errdef.Define("test_error", userID("user123"), countField(42))
	resolver := errdef.NewResolver(def)
	u := unmarshaler.NewJSON(resolver)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	fields := unmarshaled.Fields()
	fieldCount := 0
	for k, v := range fields.Seq() {
		if k.String() == "user_id" {
			if v.Value() != "user123" {
				t.Errorf("want user_id %q, got %v", "user123", v.Value())
			}
		}
		fieldCount++
	}

	if fieldCount != 2 {
		t.Errorf("want 2 fields in seq, got %d", fieldCount)
	}
}

func TestFields_SortedSeq(t *testing.T) {
	userID, _ := errdef.DefineField[string]("user_id")
	countField, _ := errdef.DefineField[int]("count")
	def := errdef.Define("test_error", userID("user123"), countField(42))
	resolver := errdef.NewResolver(def)
	u := unmarshaler.NewJSON(resolver)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	fields := unmarshaled.Fields()
	var keys []string
	for k := range fields.SortedSeq() {
		keys = append(keys, k.String())
	}

	if len(keys) != 2 {
		t.Errorf("want 2 fields, got %d", len(keys))
	}

	if keys[0] != "user_id" && keys[0] != "count" {
		t.Errorf("unexpected key: %s", keys[0])
	}
}

func TestFields_MarshalJSON(t *testing.T) {
	t.Run("defined fields", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		count, _ := errdef.DefineField[int]("count")
		def := errdef.Define("test_error", userID("user123"), count(42))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		original := def.New("test message")
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		fields := unmarshaled.Fields()
		marshaled, err := json.Marshal(fields)
		if err != nil {
			t.Fatalf("failed to marshal fields: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(marshaled, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"count":   float64(42),
			"user_id": "user123",
		}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, result)
		}
	})

	t.Run("unknown fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"unknown_field": "unknown_value",
				"another_field": 123
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		fields := unmarshaled.Fields()
		marshaled, err := json.Marshal(fields)
		if err != nil {
			t.Fatalf("failed to marshal fields: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(marshaled, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"another_field": float64(123),
			"unknown_field": "unknown_value",
		}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, result)
		}
	})

	t.Run("mixed fields", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", userID("user123"))
		resolver := errdef.NewResolver(def)
		u := unmarshaler.NewJSON(resolver)

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"user_id": "user123",
				"unknown_field": "value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		fields := unmarshaled.Fields()
		marshaled, err := json.Marshal(fields)
		if err != nil {
			t.Fatalf("failed to marshal fields: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(marshaled, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"unknown_field": "value",
			"user_id":       "user123",
		}

		if !reflect.DeepEqual(result, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, result)
		}
	})
}
