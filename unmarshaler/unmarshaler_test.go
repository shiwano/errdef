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

func TestUnmarshaler_Error_DecodeFailure(t *testing.T) {
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

func TestUnmarshaler_Error_KindNotFound(t *testing.T) {
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

	if !errors.Is(err, unmarshaler.ErrKindNotFound) {
		t.Errorf("want ErrKindNotFound, got %v", err)
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

		if causeErr, ok := causes[0].(interface{ Type() string }); ok {
			if causeErr.Type() != "CustomError" {
				t.Errorf("want type %q, got %q", "CustomError", causeErr.Type())
			}
		} else {
			t.Errorf("want cause to be interface with Type(), got %T", causes[0])
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

		if got, ok := userIDFrom(unmarshaled); !ok || got != "default_user" {
			t.Errorf("want user_id to fallback to %q, got %q (ok=%v)", "default_user", got, ok)
		}

		if got, ok := countFrom(unmarshaled); !ok || got != 99 {
			t.Errorf("want count to fallback to %d, got %d (ok=%v)", 99, got, ok)
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

		if got, ok := userIDFrom(unmarshaled); !ok || got != "actual_user" {
			t.Errorf("want user_id %q, got %q (ok=%v)", "actual_user", got, ok)
		}

		if got, ok := countFrom(unmarshaled); !ok || got != 42 {
			t.Errorf("want count %d, got %d (ok=%v)", 42, got, ok)
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

		if got, ok := userIDFrom(unmarshaled); !ok || got != "default_user" {
			t.Errorf("want user_id to fallback to %q, got %q (ok=%v)", "default_user", got, ok)
		}

		if got, ok := countFrom(unmarshaled); !ok || got != 42 {
			t.Errorf("want count %d, got %d (ok=%v)", 42, got, ok)
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

func TestUnmarshaler_WithAdditionalFieldKeys(t *testing.T) {
	t.Run("basic additional field key", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		extraField, extraFieldFrom := errdef.DefineField[string]("extra")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(extraField.Key()))

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

		got, ok := extraFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want extra field to be found")
		}

		if got != "extra value" {
			t.Errorf("want %q, got %q", "extra value", got)
		}
	})

	t.Run("multiple additional field keys", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		field1, field1From := errdef.DefineField[string]("field1")
		field2, field2From := errdef.DefineField[int]("field2")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(
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

		got1, ok1 := field1From(unmarshaled)
		if !ok1 {
			t.Fatal("want field1 to be found")
		}
		if got1 != "value1" {
			t.Errorf("want field1 %q, got %q", "value1", got1)
		}

		got2, ok2 := field2From(unmarshaled)
		if !ok2 {
			t.Fatal("want field2 to be found")
		}
		if got2 != 42 {
			t.Errorf("want field2 %d, got %d", 42, got2)
		}
	})

	t.Run("type conversion with float64", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		intField, intFieldFrom := errdef.DefineField[int]("number")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(intField.Key()))

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

		got, ok := intFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want number field to be found")
		}

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
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(addressField.Key()))

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

		got, ok := addressFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want address field to be found")
		}

		if got.Street != "123 Main St" || got.City != "Tokyo" {
			t.Errorf("want {Street: \"123 Main St\", City: \"Tokyo\"}, got %+v", got)
		}
	})

	t.Run("slice conversion", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		idsField, idsFieldFrom := errdef.DefineField[[]int]("ids")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(idsField.Key()))

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

		got, ok := idsFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want ids field to be found")
		}

		want := []int{1, 2, 3, 4, 5}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("conversion failure falls to unknownFields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		intField, intFieldFrom := errdef.DefineField[int]("number")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(intField.Key()))

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

	t.Run("field not in additional keys goes to unknownFields", func(t *testing.T) {
		def := errdef.Define("test_error")
		r := resolver.New(def)

		field1, _ := errdef.DefineField[string]("field1")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(field1.Key()))

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

	t.Run("mix of defined and additional fields", func(t *testing.T) {
		definedField, definedFieldFrom := errdef.DefineField[string]("defined")
		def := errdef.Define("test_error", definedField("default"))
		r := resolver.New(def)

		additionalField, additionalFieldFrom := errdef.DefineField[string]("additional")
		u := unmarshaler.NewJSON(r, unmarshaler.WithAdditionalFields(additionalField.Key()))

		jsonData := `{
			"message": "test message",
			"kind": "test_error",
			"fields": {
				"defined": "defined value",
				"additional": "additional value"
			}
		}`

		unmarshaled, err := u.Unmarshal([]byte(jsonData))
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		gotDefined, ok := definedFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want defined field to be found")
		}
		if gotDefined != "defined value" {
			t.Errorf("want defined field %q, got %q", "defined value", gotDefined)
		}

		gotAdditional, ok := additionalFieldFrom(unmarshaled)
		if !ok {
			t.Fatal("want additional field to be found")
		}
		if gotAdditional != "additional value" {
			t.Errorf("want additional field %q, got %q", "additional value", gotAdditional)
		}
	})
}
