package unmarshaler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
)

func TestUnmarshaler_Unmarshal(t *testing.T) {
	t.Run("basic unmarshal", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		orig := def.New("test message")
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Error() != "test message" {
			t.Errorf("want message %q, got %q", "test message", unmarshaled.Error())
		}
		if unmarshaled.Kind() != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", unmarshaled.Kind())
		}
	})

	t.Run("with multiple definitions in resolver", func(t *testing.T) {
		def1 := errdef.Define("error_one")
		def2 := errdef.Define("error_two")
		r := resolver.New(def1, def2)
		u := unmarshaler.NewJSON(r)

		orig := def2.New("second error")
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Kind() != "error_two" {
			t.Errorf("want kind %q, got %q", "error_two", unmarshaled.Kind())
		}
	})
}

func TestUnmarshaler_ErrDecodeFailure(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	invalidJSON := []byte(`{invalid json}`)

	_, err := u.Unmarshal(invalidJSON)
	if err == nil {
		t.Fatal("want error for invalid JSON")
	}

	if !errors.Is(err, unmarshaler.ErrDecodeFailure) {
		t.Errorf("want ErrDecodeFailure, got %v", err)
	}
}

func TestUnmarshaler_ErrUnknownKind(t *testing.T) {
	def := errdef.Define("known_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	jsonData := `{
		"message": "test message",
		"kind": "unknown_error"
	}`

	_, err := u.Unmarshal([]byte(jsonData))
	if err == nil {
		t.Fatal("want error for unknown kind")
	}

	if !errors.Is(err, unmarshaler.ErrUnknownKind) {
		t.Errorf("want ErrUnknownKind, got %v", err)
	}

	if got := unmarshaler.KindFromError.OrZero(err); got != "unknown_error" {
		t.Errorf("want kind %q in error, got %q", "unknown_error", got)
	}
}

func TestUnmarshaler_Causes_Single(t *testing.T) {
	def := errdef.Define("outer_error")
	innerDef := errdef.Define("inner_error")
	r := resolver.New(def, innerDef)
	u := unmarshaler.NewJSON(r)

	inner := innerDef.New("inner message")
	outer := def.Wrap(inner)

	data, err := json.Marshal(outer)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 1 {
		t.Fatalf("want 1 cause, got %d", len(causes))
	}

	if causes[0].Error() != "inner message" {
		t.Errorf("want cause message %q, got %q", "inner message", causes[0].Error())
	}

	if causeErr, ok := causes[0].(errdef.Error); ok {
		if causeErr.Kind() != "inner_error" {
			t.Errorf("want cause kind %q, got %q", "inner_error", causeErr.Kind())
		}
	} else {
		t.Error("want cause to be errdef.Error")
	}
}

func TestUnmarshaler_Causes_Multiple(t *testing.T) {
	def := errdef.Define("outer_error")
	inner1Def := errdef.Define("inner1_error")
	inner2Def := errdef.Define("inner2_error")
	r := resolver.New(def, inner1Def, inner2Def)
	u := unmarshaler.NewJSON(r)

	inner1 := inner1Def.New("inner message 1")
	inner2 := inner2Def.New("inner message 2")
	outer := def.Join(inner1, inner2)

	data, err := json.Marshal(outer)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 2 {
		t.Fatalf("want 2 causes, got %d", len(causes))
	}

	if causes[0].Error() != "inner message 1" {
		t.Errorf("want first cause message %q, got %q", "inner message 1", causes[0].Error())
	}
	if causes[1].Error() != "inner message 2" {
		t.Errorf("want second cause message %q, got %q", "inner message 2", causes[1].Error())
	}
}

func TestUnmarshaler_Causes_Nested(t *testing.T) {
	def1 := errdef.Define("level1")
	def2 := errdef.Define("level2")
	def3 := errdef.Define("level3")
	r := resolver.New(def1, def2, def3)
	u := unmarshaler.NewJSON(r)

	level3 := def3.New("deepest error")
	level2 := def2.Wrap(level3)
	level1 := def1.Wrap(level2)

	data, err := json.Marshal(level1)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes1 := unmarshaled.Unwrap()
	if len(causes1) != 1 {
		t.Fatalf("want 1 cause at level 1, got %d", len(causes1))
	}

	level2Err, ok := causes1[0].(errdef.Error)
	if !ok {
		t.Fatal("want level 2 to be errdef.Error")
	}
	causes2 := level2Err.Unwrap()
	if len(causes2) != 1 {
		t.Fatalf("want 1 cause at level 2, got %d", len(causes2))
	}

	level3Err, ok := causes2[0].(errdef.Error)
	if !ok {
		t.Fatal("want level 3 to be errdef.Error")
	}
	if level3Err.Error() != "deepest error" {
		t.Errorf("want deepest error message %q, got %q", "deepest error", level3Err.Error())
	}
}

func TestUnmarshaler_Causes_Unmarshalable(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	t.Run("unknown kind with type and data", func(t *testing.T) {
		jsonData := `{
			"message": "outer message",
			"kind": "test_error",
			"causes": [
				{
					"message": "unknown error",
					"type": "CustomError"
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if causes[0].Error() != "unknown error" {
			t.Errorf("want cause message %q, got %q", "unknown error", causes[0].Error())
		}

		if causeErr, ok := causes[0].(errdef.ErrorTypeNamer); ok {
			if causeErr.TypeName() != "CustomError" {
				t.Errorf("want type %q, got %q", "CustomError", causeErr.TypeName())
			}
		} else {
			t.Errorf("want cause to be errdef.ErrorTypeNamer, got %T", causes[0])
		}
	})

	t.Run("unknown kind without type", func(t *testing.T) {
		jsonData := `{
			"message": "outer message",
			"kind": "test_error",
			"causes": [
				{
					"message": "unknown error",
					"kind": "unknown_kind"
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if causes[0].Error() != "unknown error" {
			t.Errorf("want cause message %q, got %q", "unknown error", causes[0].Error())
		}
	})

	t.Run("unknown kind without message", func(t *testing.T) {
		jsonData := `{
			"message": "outer message",
			"kind": "test_error",
			"causes": [
				{
					"kind": "unknown_kind"
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		causeMsg := causes[0].Error()
		if causeMsg != "<unknown: map[kind:unknown_kind]>" {
			t.Errorf("want cause message %q, got %q", "<unknown: map[kind:unknown_kind]>", causeMsg)
		}
	})
}

func TestUnmarshaler_Causes_Mixed(t *testing.T) {
	def1 := errdef.Define("outer_error")
	def2 := errdef.Define("known_error")
	r := resolver.New(def1, def2)
	u := unmarshaler.NewJSON(r)

	jsonData := `{
		"message": "outer message",
		"kind": "outer_error",
		"causes": [
			{
				"message": "known error",
				"kind": "known_error"
			},
			{
				"message": "unknown error",
				"kind": "unknown_kind"
			}
		]
	}`

	unmarshaled, err := u.Unmarshal([]byte(jsonData))
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 2 {
		t.Fatalf("want 2 causes, got %d", len(causes))
	}

	if knownErr, ok := causes[0].(errdef.Error); ok {
		if knownErr.Kind() != "known_error" {
			t.Errorf("want first cause kind %q, got %q", "known_error", knownErr.Kind())
		}
	} else {
		t.Error("want first cause to be errdef.Error")
	}

	if _, ok := causes[1].(errdef.Error); ok {
		t.Error("want second cause not to be errdef.Error")
	} else if causes[1].Error() != "unknown error" {
		t.Errorf("want second cause message %q, got %q", "unknown error", causes[1].Error())
	}
}

func TestUnmarshaler_Fields_TypeConversionFallback(t *testing.T) {
	userID, userIDFrom := errdef.DefineField[string]("user_id")
	count, countFrom := errdef.DefineField[int]("count")

	def := errdef.Define("test_error", userID("default_user"), count(99))
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	t.Run("use default value when type conversion fails", func(t *testing.T) {
		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"user_id": 12345,
				"count": "not a number"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := userIDFrom.OrZero(unmarshaled); got != "default_user" {
			t.Errorf("want user_id to fallback to %q, got %q", "default_user", got)
		}

		if got := countFrom.OrZero(unmarshaled); got != 99 {
			t.Errorf("want count to fallback to %d, got %d", 99, got)
		}
	})

	t.Run("use actual value when type conversion succeeds", func(t *testing.T) {
		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"user_id": "actual_user",
				"count": 42
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := userIDFrom.OrZero(unmarshaled); got != "actual_user" {
			t.Errorf("want user_id %q, got %q", "actual_user", got)
		}

		if got := countFrom.OrZero(unmarshaled); got != 42 {
			t.Errorf("want count %d, got %d", 42, got)
		}
	})

	t.Run("continue processing other fields after fallback", func(t *testing.T) {
		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"user_id": 12345,
				"count": 42
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := userIDFrom.OrZero(unmarshaled); got != "default_user" {
			t.Errorf("want user_id to fallback to %q, got %q", "default_user", got)
		}

		if got := countFrom.OrZero(unmarshaled); got != 42 {
			t.Errorf("want count %d, got %d", 42, got)
		}
	})
}

func TestUnmarshaler_Fields_Redacted(t *testing.T) {
	password, passwordFrom := errdef.DefineField[errdef.Redacted[string]]("password")

	def := errdef.Define("auth_error", password(errdef.Redact("secret123")))
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	orig := def.New("authentication failed")
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	t.Run("redacted field is accessible via Fields.Get", func(t *testing.T) {
		if value, ok := unmarshaled.Fields().Get(password.Key()); !ok {
			t.Error("want password to be accessible via Fields.Get")
		} else if value.Value() != "[REDACTED]" {
			t.Errorf("want password value %q, got %q", "[REDACTED]", value.Value())
		}
	})

	t.Run("redacted field appears in JSON marshaling", func(t *testing.T) {
		remarshaled, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to remarshal: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(remarshaled, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		fields, ok := result["fields"].(map[string]any)
		if !ok {
			t.Fatal("want fields in JSON")
		}

		if fields["password"] != "[REDACTED]" {
			t.Errorf("want password value %q in JSON, got %q", "[REDACTED]", fields["password"])
		}
	})

	t.Run("redacted field is not accessible via typed getter", func(t *testing.T) {
		if _, ok := passwordFrom(unmarshaled); ok {
			t.Error("want password field to be inaccessible via typed getter")
		}
	})
}

func TestUnmarshaler_WithStandardSentinelErrors(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r, unmarshaler.WithStandardSentinelErrors())

	t.Run("io.EOF", func(t *testing.T) {
		orig := def.Wrap(io.EOF)
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if !errors.Is(causes[0], io.EOF) {
			t.Errorf("want cause to be io.EOF, got %v", causes[0])
		}
	})

	t.Run("context.Canceled", func(t *testing.T) {
		orig := def.Wrap(context.Canceled)
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if !errors.Is(causes[0], context.Canceled) {
			t.Errorf("want cause to be context.Canceled, got %v", causes[0])
		}
	})

	t.Run("os.ErrNotExist", func(t *testing.T) {
		orig := def.Wrap(os.ErrNotExist)
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if !errors.Is(causes[0], os.ErrNotExist) {
			t.Errorf("want cause to be os.ErrNotExist, got %v", causes[0])
		}
	})
}

func TestUnmarshaler_SentinelErrors_Custom(t *testing.T) {
	customSentinel := errors.New("custom sentinel error")

	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r, unmarshaler.WithSentinelErrors(customSentinel))

	orig := def.Wrap(customSentinel)
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 1 {
		t.Fatalf("want 1 cause, got %d", len(causes))
	}

	if !errors.Is(causes[0], customSentinel) {
		t.Errorf("want cause to be customSentinel, got %v", causes[0])
	}
}

func TestUnmarshaler_SentinelErrors_WithoutOption(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	orig := def.Wrap(io.EOF)
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	causes := unmarshaled.Unwrap()
	if len(causes) != 1 {
		t.Fatalf("want 1 cause, got %d", len(causes))
	}

	if _, ok := causes[0].(errdef.Error); ok {
		t.Error("want cause not to be errdef.Error")
	} else if causes[0].Error() != io.EOF.Error() {
		t.Errorf("want cause message %q, got %q", io.EOF.Error(), causes[0].Error())
	}
}

func TestUnmarshaler_Causes_UnknownError_Recursive(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	t.Run("single nested unknown error", func(t *testing.T) {
		jsonData := `{
			"message": "outer message",
			"kind": "test_error",
			"causes": [
				{
					"message": "unknown outer",
					"kind": "unknown_kind",
					"type": "CustomError",
					"causes": [
						{
							"message": "unknown inner",
							"kind": "another_unknown",
							"type": "AnotherError"
						}
					]
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var gotMap map[string]any
		if err := json.Unmarshal(got, &gotMap); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"message": "outer message",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "unknown outer",
					"type":    "CustomError",
					"causes": []any{
						map[string]any{
							"message": "unknown inner",
							"type":    "AnotherError",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(gotMap, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", gotMap, want)
		}
	})

	t.Run("multiple nested unknown errors", func(t *testing.T) {
		jsonData := `{
			"message": "outer message",
			"kind": "test_error",
			"causes": [
				{
					"message": "unknown outer",
					"kind": "unknown_kind",
					"type": "CustomError",
					"causes": [
						{
							"message": "unknown inner 1",
							"kind": "another_unknown",
							"type": "AnotherError1"
						},
						{
							"message": "unknown inner 2",
							"kind": "yet_another_unknown",
							"type": "AnotherError2"
						}
					]
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var gotMap map[string]any
		if err := json.Unmarshal(got, &gotMap); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"message": "outer message",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "unknown outer",
					"type":    "CustomError",
					"causes": []any{
						map[string]any{
							"message": "unknown inner 1",
							"type":    "AnotherError1",
						},
						map[string]any{
							"message": "unknown inner 2",
							"type":    "AnotherError2",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(gotMap, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", gotMap, want)
		}
	})

	t.Run("deeply nested unknown errors", func(t *testing.T) {
		jsonData := `{
			"message": "level 1",
			"kind": "test_error",
			"causes": [
				{
					"message": "level 2",
					"kind": "unknown_kind_2",
					"type": "Error2",
					"causes": [
						{
							"message": "level 3",
							"kind": "unknown_kind_3",
							"type": "Error3",
							"causes": [
								{
									"message": "level 4",
									"kind": "unknown_kind_4",
									"type": "Error4"
								}
							]
						}
					]
				}
			]
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var gotMap map[string]any
		if err := json.Unmarshal(got, &gotMap); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"message": "level 1",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "level 2",
					"type":    "Error2",
					"causes": []any{
						map[string]any{
							"message": "level 3",
							"type":    "Error3",
							"causes": []any{
								map[string]any{
									"message": "level 4",
									"type":    "Error4",
								},
							},
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(gotMap, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", gotMap, want)
		}
	})

	t.Run("mixed known and unknown nested errors", func(t *testing.T) {
		knownDef := errdef.Define("known_error")
		mixedResolver := resolver.New(def, knownDef)
		mixedU := unmarshaler.NewJSON(mixedResolver)

		jsonData := `{
			"message": "outer",
			"kind": "test_error",
			"causes": [
				{
					"message": "unknown with known child",
					"kind": "unknown_kind",
					"type": "UnknownError",
					"causes": [
						{
							"message": "known error",
							"kind": "known_error"
						}
					]
				}
			]
		}`

		unmarshaled, err := mixedU.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var gotMap map[string]any
		if err := json.Unmarshal(got, &gotMap); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		want := map[string]any{
			"message": "outer",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "unknown with known child",
					"type":    "UnknownError",
					"causes": []any{
						map[string]any{
							"message": "known error",
							"kind":    "known_error",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(gotMap, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", gotMap, want)
		}
	})
}

func TestUnmarshaler_WithCustomFieldKeys(t *testing.T) {
	t.Run("basic custom field key", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		extraField, extraFieldFrom := errdef.DefineField[string]("extra")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(extraField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"extra": "extra value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got := extraFieldFrom.OrZero(unmarshaled)

		if got != "extra value" {
			t.Errorf("want %q, got %q", "extra value", got)
		}
	})

	t.Run("multiple custom field keys", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		field1, field1From := errdef.DefineField[string]("field1")
		field2, field2From := errdef.DefineField[int]("field2")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(
			field1.Key(),
			field2.Key(),
		))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"field1": "value1",
				"field2": 42
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got1 := field1From.OrZero(unmarshaled)
		if got1 != "value1" {
			t.Errorf("want field1 %q, got %q", "value1", got1)
		}

		got2 := field2From.OrZero(unmarshaled)
		if got2 != 42 {
			t.Errorf("want field2 %d, got %d", 42, got2)
		}
	})

	t.Run("type conversion with float64", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		intField, intFieldFrom := errdef.DefineField[int]("number")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(intField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"number": 100.0
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got := intFieldFrom.OrZero(unmarshaled)

		if got != 100 {
			t.Errorf("want %d, got %d", 100, got)
		}
	})

	t.Run("struct conversion", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		def := errdef.Define("test_error")
		r := resolver.New(def)

		addressField, addressFieldFrom := errdef.DefineField[Address]("address")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(addressField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"address": {
					"street": "123 Main St",
					"city": "Tokyo"
				}
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got := addressFieldFrom.OrZero(unmarshaled)

		if got.Street != "123 Main St" || got.City != "Tokyo" {
			t.Errorf("want {Street: \"123 Main St\", City: \"Tokyo\"}, got %+v", got)
		}
	})

	t.Run("slice conversion", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		idsField, idsFieldFrom := errdef.DefineField[[]int]("ids")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(idsField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"ids": [1, 2, 3, 4, 5]
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		got := idsFieldFrom.OrZero(unmarshaled)

		want := []int{1, 2, 3, 4, 5}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("conversion failure falls to unknownFields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		intField, intFieldFrom := errdef.DefineField[int]("number")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(intField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"number": "not a number"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if _, ok := intFieldFrom(unmarshaled); ok {
			t.Error("want number field not to be converted")
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("number")
		if len(keys) == 0 {
			t.Error("want number field to be in unknownFields")
		}
		if v, ok := fields.Get(keys[0]); !ok || v.Value() != "not a number" {
			t.Errorf("want unknown field %q, got %v", "not a number", v.Value())
		}
	})

	t.Run("field not in custom keys goes to unknownFields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		field1, _ := errdef.DefineField[string]("field1")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(field1.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"field1": "value1",
				"field2": "value2"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		fields := unmarshaled.Fields()
		keys := fields.FindKeys("field2")
		if len(keys) == 0 {
			t.Error("want field2 to be in unknownFields")
		}
	})

	t.Run("mix of defined and custom fields", func(t *testing.T) {
		definedField, definedFieldFrom := errdef.DefineField[string]("defined")
		def := errdef.Define("test_error", definedField("default"))
		r := resolver.New(def)

		customField, customFieldFrom := errdef.DefineField[string]("custom")
		u := unmarshaler.NewJSON(r, unmarshaler.WithCustomFields(customField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"defined": "defined value",
				"custom": "custom value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		gotDefined := definedFieldFrom.OrZero(unmarshaled)
		if gotDefined != "defined value" {
			t.Errorf("want defined field %q, got %q", "defined value", gotDefined)
		}

		gotCustom := customFieldFrom.OrZero(unmarshaled)
		if gotCustom != "custom value" {
			t.Errorf("want custom field %q, got %q", "custom value", gotCustom)
		}
	})
}

func TestUnmarshaler_DefinitionAsSentinel(t *testing.T) {
	t.Run("unmarshal definition as cause", func(t *testing.T) {
		def := errdef.Define("not_found")
		wrapper := errdef.Define("wrapper")
		r := resolver.New(def, wrapper)
		u := unmarshaler.NewJSON(r)

		wrappedDef := wrapper.Wrap(def)
		data, err := json.Marshal(wrappedDef)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if !errors.Is(causes[0], def) {
			t.Error("want cause to be the definition")
		}
	})

	t.Run("unmarshal definition with fields", func(t *testing.T) {
		ctor, extr := errdef.DefineField[int]("code")
		def := errdef.Define("not_found", ctor(404))
		wrapper := errdef.Define("wrapper")
		r := resolver.New(def, wrapper)
		u := unmarshaler.NewJSON(r)

		wrappedDef := wrapper.Wrap(def)
		data, err := json.Marshal(wrappedDef)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		code := extr.OrZero(causes[0])
		if code != 404 {
			t.Errorf("want code to be 404, got %d", code)
		}
	})
}

func TestUnmarshaler_Recover(t *testing.T) {
	t.Run("panic with error value", func(t *testing.T) {
		def := errdef.Define("panic_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		panicValue := errors.New("original panic error")
		recoveredErr := def.Recover(func() error {
			panic(panicValue)
		})

		data, err := json.Marshal(recoveredErr)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Kind() != "panic_error" {
			t.Errorf("want kind %q, got %q", "panic_error", unmarshaled.Kind())
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if causes[0].Error() != "original panic error" {
			t.Errorf("want cause message %q, got %q", "original panic error", causes[0].Error())
		}

		if unwrapper, ok := causes[0].(interface{ Unwrap() []error }); ok {
			unwrappedCauses := unwrapper.Unwrap()
			if len(unwrappedCauses) != 1 {
				t.Fatalf("want 1 unwrapped cause, got %d", len(unwrappedCauses))
			}
			if unwrappedCauses[0].Error() != "original panic error" {
				t.Errorf("want unwrapped message %q, got %q", "original panic error", unwrappedCauses[0].Error())
			}
		} else {
			t.Error("want panic cause to have Unwrap() []error")
		}
	})

	t.Run("panic with string value", func(t *testing.T) {
		def := errdef.Define("panic_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r)

		panicValue := "panic string message"
		recoveredErr := def.Recover(func() error {
			panic(panicValue)
		})

		data, err := json.Marshal(recoveredErr)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if causes[0].Error() != "panic string message" {
			t.Errorf("want cause message %q, got %q", "panic string message", causes[0].Error())
		}

		if unwrapper, ok := causes[0].(interface{ Unwrap() []error }); ok {
			unwrappedCauses := unwrapper.Unwrap()
			if len(unwrappedCauses) != 0 {
				t.Errorf("want no unwrapped causes for non-error panic value, got %d", len(unwrappedCauses))
			}
		}
	})

	t.Run("panic with errdef.Error value", func(t *testing.T) {
		def := errdef.Define("panic_error")
		innerDef := errdef.Define("inner_error")
		r := resolver.New(def, innerDef)
		u := unmarshaler.NewJSON(r)

		panicValue := innerDef.New("inner error message")
		recoveredErr := def.Recover(func() error {
			panic(panicValue)
		})

		data, err := json.Marshal(recoveredErr)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		causes := unmarshaled.Unwrap()
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		if causes[0].Error() != "inner error message" {
			t.Errorf("want cause message %q, got %q", "inner error message", causes[0].Error())
		}

		if unwrapper, ok := causes[0].(interface{ Unwrap() []error }); ok {
			unwrappedCauses := unwrapper.Unwrap()
			if len(unwrappedCauses) != 1 {
				t.Fatalf("want 1 unwrapped cause, got %d", len(unwrappedCauses))
			}

			if innerErrDef, ok := unwrappedCauses[0].(errdef.Error); ok {
				if innerErrDef.Kind() != "inner_error" {
					t.Errorf("want inner error kind %q, got %q", "inner_error", innerErrDef.Kind())
				}
				if innerErrDef.Error() != "inner error message" {
					t.Errorf("want inner error message %q, got %q", "inner error message", innerErrDef.Error())
				}
			} else {
				t.Error("want inner error to be errdef.Error")
			}
		} else {
			t.Error("want panic cause to have Unwrap() []error")
		}
	})
}

func TestUnmarshaler_WithStrictMode(t *testing.T) {
	t.Run("returns error for unknown field with strict mode enabled", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r, unmarshaler.WithStrictMode())

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"unknown_field": "value"
			}
		}`

		_, err := u.Unmarshal([]byte(jsonData))
		if err == nil {
			t.Fatal("want error for unknown field")
		}

		if !errors.Is(err, unmarshaler.ErrUnknownField) {
			t.Errorf("want ErrUnknownField, got %v", err)
		}

		if got := unmarshaler.FieldNameFromError.OrZero(err); got != "unknown_field" {
			t.Errorf("want field_name %q, got %q", "unknown_field", got)
		}

		if got := unmarshaler.KindFromError.OrZero(err); got != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", got)
		}
	})

	t.Run("allows known fields with strict mode enabled", func(t *testing.T) {
		knownField, knownFieldFrom := errdef.DefineField[string]("known_field")
		def := errdef.Define("test_error", knownField("default"))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r, unmarshaler.WithStrictMode())

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"known_field": "value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := knownFieldFrom.OrZero(unmarshaled); got != "value" {
			t.Errorf("want %q, got %q", "value", got)
		}
	})

	t.Run("allows custom fields with strict mode enabled", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		customField, customFieldFrom := errdef.DefineField[string]("custom")
		u := unmarshaler.NewJSON(r,
			unmarshaler.WithStrictMode(),
			unmarshaler.WithCustomFields(customField.Key()),
		)

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"custom": "value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := customFieldFrom.OrZero(unmarshaled); got != "value" {
			t.Errorf("want %q, got %q", "value", got)
		}
	})

	t.Run("returns error for unknown kind with strict mode enabled and FallbackResolver", func(t *testing.T) {
		knownDef := errdef.Define("known_error")
		fallbackDef := errdef.Define("")
		r := resolver.New(knownDef).WithFallback(fallbackDef)
		u := unmarshaler.NewJSON(r, unmarshaler.WithStrictMode())

		jsonData := `{
			"message": "test message",
			"kind": "unknown_error"
		}`

		_, err := u.Unmarshal([]byte(jsonData))
		if err == nil {
			t.Fatal("want error for unknown kind with strict mode enabled")
		}

		if !errors.Is(err, unmarshaler.ErrUnknownKind) {
			t.Errorf("want ErrUnknownKind, got %v", err)
		}

		if got := unmarshaler.KindFromError.OrZero(err); got != "unknown_error" {
			t.Errorf("want kind %q in error, got %q", "unknown_error", got)
		}
	})
}

func TestUnmarshaler_WithBuiltinFields(t *testing.T) {
	t.Run("unmarshal all built-in fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r, unmarshaler.WithBuiltinFields())

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"http_status": 500,
				"trace_id": "trace-123",
				"domain": "api",
				"user_hint": "Please try again later",
				"public": true,
				"retryable": true,
				"retry_after": 5000000000,
				"unreportable": true,
				"exit_code": 1,
				"help_url": "https://example.com/help",
				"details": { "key": "value" }
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := errdef.HTTPStatusFrom.OrZero(unmarshaled); got != 500 {
			t.Errorf("want http_status %d, got %d", 500, got)
		}

		if got := errdef.TraceIDFrom.OrZero(unmarshaled); got != "trace-123" {
			t.Errorf("want trace_id %q, got %q", "trace-123", got)
		}

		if got := errdef.DomainFrom.OrZero(unmarshaled); got != "api" {
			t.Errorf("want domain %q, got %q", "api", got)
		}

		if got := errdef.UserHintFrom.OrZero(unmarshaled); got != "Please try again later" {
			t.Errorf("want user_hint %q, got %q", "Please try again later", got)
		}

		if got := errdef.IsPublic(unmarshaled); !got {
			t.Error("want public to be true")
		}

		if got := errdef.IsRetryable(unmarshaled); !got {
			t.Error("want retryable to be true")
		}

		if got := errdef.RetryAfterFrom.OrZero(unmarshaled); got != 5000000000 {
			t.Errorf("want retry_after %d, got %d", 5000000000, got)
		}

		if got := errdef.IsUnreportable(unmarshaled); !got {
			t.Error("want unreportable to be true")
		}

		if got := errdef.ExitCodeFrom.OrZero(unmarshaled); got != 1 {
			t.Errorf("want exit_code %d, got %d", 1, got)
		}

		if got := errdef.HelpURLFrom.OrZero(unmarshaled); got != "https://example.com/help" {
			t.Errorf("want help_url %q, got %q", "https://example.com/help", got)
		}

		if got := errdef.DetailsFrom.OrZero(unmarshaled); !reflect.DeepEqual(got, errdef.Details{"key": "value"}) {
			t.Errorf("want details %v, got %v", errdef.Details{"key": "value"}, got)
		}
	})

	t.Run("works with strict fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r,
			unmarshaler.WithStrictMode(),
			unmarshaler.WithBuiltinFields(),
		)

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"http_status": 404,
				"public": true
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := errdef.HTTPStatusFrom.OrZero(unmarshaled); got != 404 {
			t.Errorf("want http_status %d, got %d", 404, got)
		}

		if got := errdef.IsPublic(unmarshaled); !got {
			t.Error("want public to be true")
		}
	})

	t.Run("predefined field wins built-in field", func(t *testing.T) {
		customHTTPStatus, customHTTPStatusFrom := errdef.DefineField[int]("http_status")
		def := errdef.Define("test_error", customHTTPStatus(404))
		r := resolver.New(def)
		u := unmarshaler.NewJSON(r, unmarshaler.WithBuiltinFields())

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"http_status": 404
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := customHTTPStatusFrom.OrZero(unmarshaled); got != 404 {
			t.Errorf("want custom http_status %d, got %d", 404, got)
		}

		if _, ok := errdef.HTTPStatusFrom(unmarshaled); ok {
			t.Error("want http_status not to be from built-in field")
		}
	})
}
