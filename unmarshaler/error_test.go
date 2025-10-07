package unmarshaler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
)

func TestUnmarshaledError_Error(t *testing.T) {
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
}

func TestUnmarshaledError_Kind(t *testing.T) {
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

	if unmarshaled.Kind() != "test_error" {
		t.Errorf("want kind %q, got %q", "test_error", unmarshaled.Kind())
	}
}

func TestUnmarshaledError_Fields(t *testing.T) {
	userID, _ := errdef.DefineField[string]("user_id")
	def := errdef.Define("test_error", userID("user123"))
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

	fields := unmarshaled.Fields()
	if fields.Len() != 1 {
		t.Errorf("want 1 field, got %d", fields.Len())
	}

	value, ok := fields.Get(userID.FieldKey())
	if !ok {
		t.Fatal("want user_id field to be found")
	}

	if value.Value() != "user123" {
		t.Errorf("want value %q, got %v", "user123", value.Value())
	}
}

func TestUnmarshaledError_Stack(t *testing.T) {
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

	stack := unmarshaled.Stack()
	if stack == nil {
		t.Fatal("want stack to exist")
	}

	if stack.Len() == 0 {
		t.Error("want stack to have frames")
	}
}

func TestUnmarshaledError_Unwrap(t *testing.T) {
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
}

func TestUnmarshaledError_Is(t *testing.T) {
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

	if !errors.Is(unmarshaled, def) {
		t.Error("want unmarshaled error to match definition")
	}

	otherDef := errdef.Define("other_error")
	if errors.Is(unmarshaled, otherDef) {
		t.Error("want unmarshaled error not to match different definition")
	}
}

func TestUnmarshaledError_MarshalJSON(t *testing.T) {
	t.Run("basic error", func(t *testing.T) {
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

		remarshaled, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to remarshal: %v", err)
		}

		var original_data, remarshaled_data map[string]any
		if err := json.Unmarshal(data, &original_data); err != nil {
			t.Fatalf("failed to unmarshal original: %v", err)
		}
		if err := json.Unmarshal(remarshaled, &remarshaled_data); err != nil {
			t.Fatalf("failed to unmarshal remarshaled: %v", err)
		}

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
			"stack":   original_data["stack"],
		}

		if !reflect.DeepEqual(remarshaled_data, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, remarshaled_data)
		}
	})

	t.Run("error with fields", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		requestID, _ := errdef.DefineField[int]("request_id")
		def := errdef.Define("test_error", userID("user123"), requestID(456))
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

		remarshaled, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to remarshal: %v", err)
		}

		var original_data, remarshaled_data map[string]any
		if err := json.Unmarshal(data, &original_data); err != nil {
			t.Fatalf("failed to unmarshal original: %v", err)
		}
		if err := json.Unmarshal(remarshaled, &remarshaled_data); err != nil {
			t.Fatalf("failed to unmarshal remarshaled: %v", err)
		}

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
			"fields": map[string]any{
				"request_id": float64(456),
				"user_id":    "user123",
			},
			"stack": original_data["stack"],
		}

		if !reflect.DeepEqual(remarshaled_data, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, remarshaled_data)
		}
	})

	t.Run("error with causes", func(t *testing.T) {
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

		remarshaled, err := json.Marshal(unmarshaled)
		if err != nil {
			t.Fatalf("failed to remarshal: %v", err)
		}

		var original_data, remarshaled_data map[string]any
		if err := json.Unmarshal(data, &original_data); err != nil {
			t.Fatalf("failed to unmarshal original: %v", err)
		}
		if err := json.Unmarshal(remarshaled, &remarshaled_data); err != nil {
			t.Fatalf("failed to unmarshal remarshaled: %v", err)
		}

		originalCauses := original_data["causes"].([]any)
		originalCause := originalCauses[0].(map[string]any)

		want := map[string]any{
			"message": "inner message",
			"kind":    "outer_error",
			"causes": []any{
				map[string]any{
					"message": "inner message",
					"kind":    "inner_error",
					"stack":   originalCause["stack"],
				},
			},
			"stack": original_data["stack"],
		}

		if !reflect.DeepEqual(remarshaled_data, want) {
			t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, remarshaled_data)
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("outer_error", userID("user123"))
		middleDef := errdef.Define("middle_error")
		innerDef := errdef.Define("inner_error")
		r := resolver.New(def, middleDef, innerDef)
		u := unmarshaler.NewJSON(r)

		inner := innerDef.New("inner message")
		middle := middleDef.Wrap(inner)
		outer := def.Wrap(middle)

		data1, err := json.Marshal(outer)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled1, err := u.Unmarshal(data1)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		data2, err := json.Marshal(unmarshaled1)
		if err != nil {
			t.Fatalf("failed to remarshal: %v", err)
		}

		var data1Map, data2Map map[string]any
		if err := json.Unmarshal(data1, &data1Map); err != nil {
			t.Fatalf("failed to unmarshal data1: %v", err)
		}
		if err := json.Unmarshal(data2, &data2Map); err != nil {
			t.Fatalf("failed to unmarshal data2: %v", err)
		}

		if !reflect.DeepEqual(data1Map, data2Map) {
			t.Errorf("JSON mismatch:\nwant: %+v\ngot:  %+v", data1Map, data2Map)
		}
	})
}

func TestUnmarshaledError_Format(t *testing.T) {
	t.Run("default format", func(t *testing.T) {
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

		result := fmt.Sprintf("%s", unmarshaled)
		if result != "test message" {
			t.Errorf("want %q, got %q", "test message", result)
		}
	})

	t.Run("quoted format", func(t *testing.T) {
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

		result := fmt.Sprintf("%q", unmarshaled)
		if result != `"test message"` {
			t.Errorf("want %q, got %q", `"test message"`, result)
		}
	})

	t.Run("verbose format", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", errdef.NoTrace(), userID("user123"))
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

		result := fmt.Sprintf("%+v", unmarshaled)
		want := "test message\n" +
			"---\n" +
			"kind: test_error\n" +
			"fields:\n" +
			"  user_id: user123"
		if want != result {
			t.Errorf("want format to equal:\n%q\ngot:\n%q", want, result)
		}
	})

	t.Run("verbose format with cause", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		innerDef := errdef.Define("inner_error", errdef.NoTrace())
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

		result := fmt.Sprintf("%+v", unmarshaled)
		want := "inner message\n" +
			"---\n" +
			"kind: test_error\n" +
			"causes: (1 error)\n" +
			"  [1] inner message\n" +
			"      ---\n" +
			"      kind: inner_error"
		if want != result {
			t.Errorf("want format to equal:\n%q\ngot:\n%q", want, result)
		}
	})

	t.Run("verbose format with causes", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err1Def := errdef.Define("error_1", errdef.NoTrace())
		err2Def := errdef.Define("error_2", errdef.NoTrace())
		r := resolver.New(def, err1Def, err2Def)
		u := unmarshaler.NewJSON(r)

		err1 := err1Def.New("error 1")
		err2 := err2Def.New("error 2")
		joined := def.Join(err1, err2)

		data, err := json.Marshal(joined)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		unmarshaled, err := u.Unmarshal(data)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		result := fmt.Sprintf("%+v", unmarshaled)
		want := "error 1\n" +
			"error 2\n" +
			"---\n" +
			"kind: test_error\n" +
			"causes: (2 errors)\n" +
			"  [1] error 1\n" +
			"      ---\n" +
			"      kind: error_1\n" +
			"  [2] error 2\n" +
			"      ---\n" +
			"      kind: error_2"
		if want != result {
			t.Errorf("want format to equal:\n%q\ngot:\n%q", want, result)
		}
	})

	t.Run("go format", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
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

		result := fmt.Sprintf("%#v", unmarshaled)
		if !strings.Contains(result, "unmarshaler.unmarshaledError") {
			t.Errorf("want format to contain type name, got: %q", result)
		}
	})
}

func TestUnmarshaledError_LogValue(t *testing.T) {
	userID, _ := errdef.DefineField[string]("user_id")
	def := errdef.Define("test_error")
	innerDef := errdef.Define("inner_error")
	r := resolver.New(def, innerDef)
	u := unmarshaler.NewJSON(r)

	inner := innerDef.New("inner message")
	outer := def.WithOptions(userID("user123")).Wrap(inner)

	data, err := json.Marshal(outer)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	logValue := unmarshaled.(slog.LogValuer).LogValue()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("test", "error", logValue)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal log output: %v", err)
	}

	errorData := result["error"].(map[string]any)

	want := map[string]any{
		"message": "inner message",
		"kind":    "test_error",
		"fields": map[string]any{
			"user_id": "user123",
		},
		"causes": []any{"inner message"},
		"origin": errorData["origin"],
	}

	if !reflect.DeepEqual(errorData, want) {
		t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, errorData)
	}
}

func TestUnmarshaledError_DebugStack(t *testing.T) {
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

	debugStacker, ok := unmarshaled.(errdef.DebugStacker)
	if !ok {
		t.Fatal("want unmarshaled error to implement DebugStacker")
	}

	debugStack := debugStacker.DebugStack()

	if !strings.Contains(debugStack, "test message") {
		t.Errorf("want debug stack to contain error message, got: %q", debugStack)
	}

	if !strings.Contains(debugStack, "goroutine 1 [running]:") {
		t.Errorf("want debug stack to contain goroutine info, got: %q", debugStack)
	}

	if unmarshaled.Stack().Len() > 0 {
		frame, _ := unmarshaled.Stack().HeadFrame()
		if !strings.Contains(debugStack, frame.Func) {
			t.Errorf("want debug stack to contain function name %q, got: %q", frame.Func, debugStack)
		}
		if !strings.Contains(debugStack, frame.File) {
			t.Errorf("want debug stack to contain file name %q, got: %q", frame.File, debugStack)
		}
	}
}

func TestUnmarshaledError_Cause(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
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

		causer, ok := unmarshaled.(interface{ Cause() error })
		if !ok {
			t.Fatal("want unmarshaled error to implement causer")
		}

		cause := causer.Cause()
		if cause == nil {
			t.Fatal("want cause to exist")
		}

		if cause.Error() != "inner message" {
			t.Errorf("want cause message %q, got %q", "inner message", cause.Error())
		}
	})

	t.Run("without cause", func(t *testing.T) {
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

		causer, ok := unmarshaled.(interface{ Cause() error })
		if !ok {
			t.Fatal("want unmarshaled error to implement causer")
		}

		cause := causer.Cause()
		if cause != nil {
			t.Errorf("want cause to be nil, got: %v", cause)
		}
	})
}
