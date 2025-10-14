package errdef_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
)

func TestError_Error(t *testing.T) {
	t.Run("basic message", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		if want, got := "test message", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}
	})

	t.Run("wrapped error message", func(t *testing.T) {
		def := errdef.Define("test_error")
		cause := errors.New("original error")
		wrapped := def.Wrap(cause)

		if want, got := "original error", wrapped.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}
	})
}

func TestError_Kind(t *testing.T) {
	def := errdef.Define("test_error")
	err := def.New("test message").(errdef.Error)

	if got, want := err.Kind(), errdef.Kind("test_error"); got != want {
		t.Errorf("want kind %q, got %q", want, got)
	}
}

func TestError_Fields(t *testing.T) {
	t.Run("no fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		fields := err.Fields()
		collected := maps.Collect(fields.All())

		if want, got := 0, len(collected); got != want {
			t.Errorf("want %d fields, got %d", want, got)
		}
	})

	t.Run("with fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", ctor("user123"))
		err := def.New("test message").(errdef.Error)

		fields := err.Fields()
		collected := maps.Collect(fields.All())

		if want, got := 1, len(collected); got != want {
			t.Errorf("want %d field, got %d", want, got)
		}
		keys := fields.FindKeys("user_id")
		if got, ok := collected[keys[0]]; ok && got.Value() != "user123" {
			t.Errorf("want field value %q, got %q", "user123", got)
		}
	})
}

func TestError_Stack(t *testing.T) {
	t.Run("stack exists", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		stack := err.Stack()
		if stack == nil {
			t.Error("want stack trace to exist")
		}

		frames := stack.Frames()
		if len(frames) == 0 {
			t.Error("want stack frames to exist")
		}
	})

	t.Run("no trace", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message").(errdef.Error)

		stack := err.Stack()
		if stack != nil && len(stack.Frames()) > 0 {
			t.Error("want no stack trace when disabled")
		}
	})
}

func TestError_Unwrap(t *testing.T) {
	t.Run("no cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		if want, got := 0, len(err.Unwrap()); got != want {
			t.Errorf("want %d errors, got %d", want, got)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		cause := errors.New("original error")
		wrapped := def.Wrap(cause).(errdef.Error)

		unwrapped := wrapped.Unwrap()
		if want, got := 1, len(unwrapped); got != want {
			t.Errorf("want %d error, got %d", want, got)
		}
		if unwrapped[0] != cause {
			t.Error("want unwrapped error to be original error")
		}
	})

	t.Run("with multiple errors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		def := errdef.Define("test_error")
		joined := def.Join(err1, err2).(errdef.Error)

		unwrapped := joined.Unwrap()
		if want, got := 2, len(unwrapped); got != want {
			t.Errorf("want %d errors, got %d", want, got)
		}
		if unwrapped[0] != err1 {
			t.Error("want first unwrapped error to be error 1")
		}
		if unwrapped[1] != err2 {
			t.Error("want second unwrapped error to be error 2")
		}
	})

	t.Run("with joined error", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		def := errdef.Define("test_error")
		joined := def.Join(err1, err2)
		nestedJoined := def.Wrap(joined).(errdef.Error)

		unwrapped := nestedJoined.Unwrap()
		if want, got := 1, len(unwrapped); got != want {
			t.Errorf("want %d error, got %d", want, got)
		}
		if unwrapped[0] != joined {
			t.Error("want unwrapped error to be the joined error")
		}
	})
}

func TestError_UnwrapTree(t *testing.T) {
	t.Run("no causes", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		tree := err.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		// When there are no causes, JSON should be null
		if got != nil {
			t.Errorf("want null, got %+v", got)
		}
	})

	t.Run("single cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		cause := errors.New("original error")
		wrapped := def.Wrap(cause).(errdef.Error)

		tree := wrapped.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "original error",
				"type":    "*errors.errorString",
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("nested causes", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		baseErr := errors.New("base error")
		wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
		err := def.Wrap(wrappedErr).(errdef.Error)

		tree := err.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "wrapped: base error",
				"type":    "*fmt.wrapError",
				"causes": []any{
					map[string]any{
						"message": "base error",
						"type":    "*errors.errorString",
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("multiple causes with Join", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		def := errdef.Define("test_error")
		joined := def.Join(err1, err2).(errdef.Error)

		tree := joined.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "error 1",
				"type":    "*errors.errorString",
			},
			map[string]any{
				"message": "error 2",
				"type":    "*errors.errorString",
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("circular reference detection", func(t *testing.T) {
		var ce1, ce2 *circularError
		ce1 = &circularError{msg: "error 1"}
		ce2 = &circularError{msg: "error 2", cause: ce1}
		ce1.cause = ce2

		def := errdef.Define("test_error")
		wrapped := def.Wrap(ce1).(errdef.Error)

		tree := wrapped.UnwrapTree()
		if !tree.HasCycle() {
			t.Error("want HasCycle to be true when circular reference detected")
		}
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "error 1",
				"type":    "*errdef_test.circularError",
				"causes": []any{
					map[string]any{
						"message": "error 2",
						"type":    "*errdef_test.circularError",
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("mixed error types", func(t *testing.T) {
		def1 := errdef.Define("inner_error", errdef.NoTrace())
		def2 := errdef.Define("outer_error", errdef.NoTrace())

		innerErr := def1.New("inner message")
		stdErr := errors.New("standard error")
		joined := errors.Join(innerErr, stdErr)
		outerErr := def2.Wrap(joined).(errdef.Error)

		tree := outerErr.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "inner message\nstandard error",
				"type":    "*errors.joinError",
				"causes": []any{
					map[string]any{
						"message": "inner message",
						"kind":    "inner_error",
					},
					map[string]any{
						"message": "standard error",
						"type":    "*errors.errorString",
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("duplicate sentinel errors with Join", func(t *testing.T) {
		sentinelErr := errors.New("sentinel error")
		def := errdef.Define("test_error", errdef.NoTrace())

		joined := def.Join(sentinelErr, sentinelErr, errors.New("other error")).(errdef.Error)

		tree := joined.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "sentinel error",
				"type":    "*errors.errorString",
			},
			map[string]any{
				"message": "sentinel error",
				"type":    "*errors.errorString",
			},
			map[string]any{
				"message": "other error",
				"type":    "*errors.errorString",
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("nil receiver error", func(t *testing.T) {
		var nilErr *nilReceiverError
		def := errdef.Define("test_error", errdef.NoTrace())
		wrapped := def.Wrap(nilErr).(errdef.Error)

		tree := wrapped.UnwrapTree()
		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "nil receiver error",
				"type":    "*errdef_test.nilReceiverError",
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("shared error (not circular)", func(t *testing.T) {
		err1 := errors.New("err1")
		err2 := fmt.Errorf("err2: %w", err1)
		wrap1 := fmt.Errorf("wrap1: %w", err2)
		wrap2 := fmt.Errorf("wrap2: %w", err2)

		def := errdef.Define("test_error", errdef.NoTrace())
		joined := def.Join(wrap1, wrap2).(errdef.Error)

		tree := joined.UnwrapTree()
		if tree.HasCycle() {
			t.Error("want HasCycle to be false")
		}

		data, jsonErr := json.Marshal(tree)
		if jsonErr != nil {
			t.Fatalf("failed to marshal tree: %v", jsonErr)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		want := []any{
			map[string]any{
				"message": "wrap1: err2: err1",
				"type":    "*fmt.wrapError",
				"causes": []any{
					map[string]any{
						"message": "err2: err1",
						"type":    "*fmt.wrapError",
						"causes": []any{
							map[string]any{
								"message": "err1",
								"type":    "*errors.errorString",
							},
						},
					},
				},
			},
			map[string]any{
				"message": "wrap2: err2: err1",
				"type":    "*fmt.wrapError",
				"causes": []any{
					map[string]any{
						"message": "err2: err1",
						"type":    "*fmt.wrapError",
						"causes": []any{
							map[string]any{
								"message": "err1",
								"type":    "*errors.errorString",
							},
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("mismatch:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

func TestError_Is(t *testing.T) {
	t.Run("same instance", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		if !errors.Is(err, err) {
			t.Error("want error to be equal to itself")
		}
	})

	t.Run("same definition", func(t *testing.T) {
		def := errdef.Define("test_error")
		err1 := def.New("message 1")
		err2 := def.New("message 2")

		if !errors.Is(err1, def) {
			t.Error("want error to match its definition")
		}
		if !errors.Is(err2, def) {
			t.Error("want error to match its definition")
		}
	})

	t.Run("different definitions", func(t *testing.T) {
		def1 := errdef.Define("error_1")
		def2 := errdef.Define("error_2")
		err1 := def1.New("message")
		err2 := def2.New("message")

		if errors.Is(err1, def2) {
			t.Error("want error not to match different definition")
		}
		if errors.Is(err1, err2) {
			t.Error("want errors from different definitions not to match")
		}
	})
}

func TestError_DebugStack(t *testing.T) {
	t.Run("debug stack format", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.DebugStacker)

		debugStack := err.DebugStack()

		want := "test message\n\ngoroutine 1 [running]:\ngithub.com/shiwano/errdef_test.TestError_DebugStack.func1()"
		if !strings.HasPrefix(debugStack, want) {
			t.Errorf("want debug stack to start with %q, but got: %q", want, debugStack)
		}
	})

	t.Run("no trace when disabled", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message").(errdef.DebugStacker)

		debugStack := err.DebugStack()
		if !strings.Contains(debugStack, "test message") {
			t.Error("want debug stack to contain error message even without trace")
		}
		if !strings.Contains(debugStack, "goroutine 1 [running]:") {
			t.Error("want debug stack to contain goroutine header even without trace")
		}
	})
}

func TestError_StackTrace(t *testing.T) {
	type stackTracer interface {
		StackTrace() []uintptr
	}

	t.Run("stack exists", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(stackTracer)

		stackTrace := err.StackTrace()
		if len(stackTrace) == 0 {
			t.Error("want stack trace to have frames")
		}
	})

	t.Run("no trace", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message").(stackTracer)

		stackTrace := err.StackTrace()
		if len(stackTrace) != 0 {
			t.Error("want no stack trace when disabled")
		}
	})
}

func TestError_Cause(t *testing.T) {
	type causer interface {
		Cause() error
	}

	t.Run("no cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(causer)

		if got := err.Cause(); got != nil {
			t.Errorf("want no cause, got %v", got)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		orig := errors.New("original error")
		wrapped := def.Wrap(orig).(causer)

		if got := wrapped.Cause(); got != orig {
			t.Error("want cause to be original error")
		}
	})
}

func TestError_Format(t *testing.T) {
	t.Run("default format", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		result := fmt.Sprintf("%s", err)
		if result != "test message" {
			t.Errorf("want %q, got %q", "test message", result)
		}
	})

	t.Run("quoted format", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		result := fmt.Sprintf("%q", err)
		if result != `"test message"` {
			t.Errorf("want %q, got %q", `"test message"`, result)
		}
	})

	t.Run("verbose format", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		password, _ := errdef.DefineField[errdef.Redacted[string]]("password")
		def := errdef.Define("test_error",
			userID("user123"),
			password(errdef.Redact("secret")),
			errdef.Details{"additional": "info", "count": 42},
		)
		err := def.New("test message")

		result := fmt.Sprintf("%+v", err)
		if matched, _ := regexp.MatchString(
			`test message\n`+
				`---\n`+
				`kind: test_error\n`+
				`fields:\n`+
				`  user_id: user123\n`+
				`  password: \[REDACTED\]\n`+
				`  details: map\[additional:info count:42\]\n`+
				`stack:\n`+
				`[\s\S]*`,
			result,
		); !matched {
			t.Errorf("want format to match pattern, got: %q", result)
		}
	})

	t.Run("verbose format with cause", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("user_id")
		cause := errors.New("original error")
		def := errdef.Define("test_error", errdef.NoTrace(), ctor("user123"))
		wrapped := def.Wrap(cause)

		result := fmt.Sprintf("%+v", wrapped)
		want := "original error\n" +
			"---\n" +
			"kind: test_error\n" +
			"fields:\n" +
			"  user_id: user123\n" +
			"causes: (1 error)\n" +
			"  [1] original error"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("verbose format with causes", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		def := errdef.Define("test_error", errdef.NoTrace())
		joined := def.Join(err1, err2)

		result := fmt.Sprintf("%+v", joined)

		want := "error 1\n" +
			"error 2\n" +
			"---\n" +
			"kind: test_error\n" +
			"causes: (2 errors)\n" +
			"  [1] error 1\n" +
			"  [2] error 2"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("verbose format with circular reference", func(t *testing.T) {
		var ce1, ce2 *circularError
		ce1 = &circularError{msg: "error 1"}
		ce2 = &circularError{msg: "error 2", cause: ce1}
		ce1.cause = ce2

		def := errdef.Define("test_error", errdef.NoTrace())
		wrapped := def.Wrap(ce1)

		result := fmt.Sprintf("%+v", wrapped)

		want := "error 1\n" +
			"---\n" +
			"kind: test_error\n" +
			"causes: (1 error)\n" +
			"  [1] error 1\n" +
			"      ---\n" +
			"      causes: (1 error)\n" +
			"      [1] error 2"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("verbose format with nested causes", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		baseErr := errors.New("base error")
		wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
		err := def.Wrap(wrappedErr)

		result := fmt.Sprintf("%+v", err)

		want := "wrapped: base error\n" +
			"---\n" +
			"kind: test_error\n" +
			"causes: (1 error)\n" +
			"  [1] wrapped: base error\n" +
			"      ---\n" +
			"      causes: (1 error)\n" +
			"      [1] base error"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("verbose format with nested Error types", func(t *testing.T) {
		def1 := errdef.Define("inner_error", errdef.NoTrace())
		def2 := errdef.Define("outer_error", errdef.NoTrace())

		baseErr := errors.New("base error")
		innerErr := def1.Wrap(baseErr)
		outerErr := def2.Wrap(innerErr)

		result := fmt.Sprintf("%+v", outerErr)

		want := "base error\n" +
			"---\n" +
			"kind: outer_error\n" +
			"causes: (1 error)\n" +
			"  [1] base error\n" +
			"      ---\n" +
			"      kind: inner_error\n" +
			"      causes: (1 error)\n" +
			"      [1] base error"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("go format", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message")

		result := fmt.Sprintf("%#v", err)
		if matched, _ := regexp.MatchString(
			`&errdef\.definedError\{`+
				`def:\(\*errdef\.Definition\)\(0x[0-9a-f]+\), `+
				`msg:"test message", `+
				`cause:error\(nil\), `+
				`stack:errdef\.stack\(nil\), `+
				`joined:false`+
				`\}`,
			result,
		); !matched {
			t.Errorf("want format to match pattern, got: %q", result)
		}
	})

	t.Run("custom formatter", func(t *testing.T) {
		customFormatter := func(err errdef.Error, s fmt.State, verb rune) {
			_, _ = fmt.Fprintf(s, "CUSTOM: %s", err.Error())
		}

		def := errdef.Define("test_error", errdef.Formatter(customFormatter))
		err := def.New("test message")

		result := fmt.Sprintf("%s", err)
		if result != "CUSTOM: test message" {
			t.Errorf("want %q, got %q", "CUSTOM: test message", result)
		}
	})
}

func TestError_MarshalJSON(t *testing.T) {
	t.Run("basic json", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("with kind and fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", ctor("user123"), errdef.NoTrace())
		err := def.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
			"fields": map[string]any{
				"user_id": "user123",
			},
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("with causes", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		orig := errors.New("original error")
		wrapped := def.Wrap(orig)

		data, jsonErr := json.Marshal(wrapped)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "original error",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "original error",
					"type":    "*errors.errorString",
				},
			},
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("with circular reference", func(t *testing.T) {
		var ce1, ce2 *circularError
		ce1 = &circularError{msg: "error 1"}
		ce2 = &circularError{msg: "error 2", cause: ce1}
		ce1.cause = ce2

		def := errdef.Define("test_error", errdef.NoTrace())
		wrapped := def.Wrap(ce1)

		data, jsonErr := json.Marshal(wrapped)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "error 1",
			"kind":    "test_error",
			"causes": []any{
				map[string]any{
					"message": "error 1",
					"type":    "*errdef_test.circularError",
					"causes": []any{
						map[string]any{
							"message": "error 2",
							"type":    "*errdef_test.circularError",
						},
					},
				},
			},
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if jsonErr := json.Unmarshal(data, &got); jsonErr != nil {
			t.Fatalf("want valid JSON, got %v", jsonErr)
		}

		stack, ok := got["stack"].([]any)
		if !ok || len(stack) == 0 {
			t.Error("want stack frames to exist")
		}
	})

	t.Run("custom json marshaler", func(t *testing.T) {
		customMarshaler := func(err errdef.Error) ([]byte, error) {
			return json.Marshal(map[string]string{
				"custom_message": err.Error(),
			})
		}

		def := errdef.Define("test_error", errdef.JSONMarshaler(customMarshaler))
		err := def.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"custom_message": "test message",
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("comprehensive json structure", func(t *testing.T) {
		userID, _ := errdef.DefineField[string]("user_id")
		password, _ := errdef.DefineField[errdef.Redacted[string]]("password")

		def := errdef.Define("auth_error", userID("user123"), password(errdef.Redact("secret")))
		orig := errors.New("connection failed")
		err := def.Wrap(orig)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if jsonErr := json.Unmarshal(data, &got); jsonErr != nil {
			t.Fatalf("want valid JSON, got %v", jsonErr)
		}

		stack, _ := got["stack"].([]any)
		if len(stack) == 0 {
			t.Fatal("want stack frames to exist")
		}
		frame0 := stack[0].(map[string]any)

		want := map[string]any{
			"message": "connection failed",
			"kind":    "auth_error",
			"fields": map[string]any{
				"user_id":  "user123",
				"password": "[REDACTED]",
			},
			"causes": []any{
				map[string]any{
					"message": "connection failed",
					"type":    "*errors.errorString",
				},
			},
			"stack": stack,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON structure mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}

		file, _ := frame0["file"].(string)
		if !strings.Contains(file, "error_test.go") {
			t.Errorf("want file to contain error_test.go, got %q", file)
		}

		funcName, _ := frame0["func"].(string)
		if !strings.Contains(funcName, "TestError_MarshalJSON") {
			t.Errorf("want function name to contain TestError_MarshalJSON, got %q", funcName)
		}
	})

	t.Run("causes with definedError", func(t *testing.T) {
		def1 := errdef.Define("inner_error", errdef.NoTrace())
		def2 := errdef.Define("outer_error", errdef.NoTrace())

		innerErr := def1.New("inner message")
		outerErr := def2.Wrap(innerErr)

		data, jsonErr := json.Marshal(outerErr)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "inner message",
			"kind":    "outer_error",
			"causes": []any{
				map[string]any{
					"message": "inner message",
					"kind":    "inner_error",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("causes with standard error", func(t *testing.T) {
		def := errdef.Define("wrapper_error", errdef.NoTrace())
		stdErr := errors.New("standard error")
		err := def.Wrap(stdErr)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "standard error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "standard error",
					"type":    "*errors.errorString",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("causes with mixed error types", func(t *testing.T) {
		def1 := errdef.Define("defined_error", errdef.NoTrace())
		def2 := errdef.Define("wrapper_error", errdef.NoTrace())

		definedErr := def1.New("defined error")
		stdErr := errors.New("standard error")

		joined := def2.Join(definedErr, stdErr)

		data, jsonErr := json.Marshal(joined)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "defined error\nstandard error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "defined error",
					"kind":    "defined_error",
				},
				map[string]any{
					"message": "standard error",
					"type":    "*errors.errorString",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("nested causes with fmt.Errorf wrap", func(t *testing.T) {
		def := errdef.Define("wrapper_error", errdef.NoTrace())

		baseErr := errors.New("base error")
		wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
		err := def.Wrap(wrappedErr)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "wrapped: base error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "wrapped: base error",
					"type":    "*fmt.wrapError",
					"causes": []any{
						map[string]any{
							"message": "base error",
							"type":    "*errors.errorString",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("deeply nested error chain", func(t *testing.T) {
		def := errdef.Define("wrapper_error", errdef.NoTrace())

		err1 := errors.New("level 3 error")
		err2 := fmt.Errorf("level 2: %w", err1)
		err3 := fmt.Errorf("level 1: %w", err2)
		err := def.Wrap(err3)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "level 1: level 2: level 3 error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "level 1: level 2: level 3 error",
					"type":    "*fmt.wrapError",
					"causes": []any{
						map[string]any{
							"message": "level 2: level 3 error",
							"type":    "*fmt.wrapError",
							"causes": []any{
								map[string]any{
									"message": "level 3 error",
									"type":    "*errors.errorString",
								},
							},
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("with value type error", func(t *testing.T) {
		def := errdef.Define("wrapper_error", errdef.NoTrace())

		valueErr := valueTypeError{msg: "value error"}
		err := def.Wrap(valueErr)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "value error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "value error",
					"type":    "errdef_test.valueTypeError",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}

func TestError_LogValue(t *testing.T) {
	t.Run("message only", func(t *testing.T) {
		def := errdef.Define("", errdef.NoTrace())
		err := def.New("test message")

		value := err.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		want := map[string]any{
			"message": "test message",
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("with kind and fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", ctor("user123"), errdef.NoTrace())
		err := def.New("test message")

		value := err.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
			"fields": map[string]any{
				"user_id": "user123",
			},
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		value := err.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		frames := err.(errdef.Error).Stack().Frames()
		if len(frames) == 0 {
			t.Fatal("want non-empty frames")
		}

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
			"origin": map[string]any{
				"func": frames[0].Func,
				"file": frames[0].File,
				"line": float64(frames[0].Line),
			},
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("with causes", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		cause := errors.New("original error")
		wrapped := def.Wrap(cause)

		value := wrapped.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		want := map[string]any{
			"message": "original error",
			"kind":    "test_error",
			"causes":  []any{"original error"},
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("no trace when disabled", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message")

		value := err.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("custom log valuer", func(t *testing.T) {
		customLogValuer := func(err errdef.Error) slog.Value {
			return slog.GroupValue(
				slog.String("custom_message", err.Error()),
				slog.String("custom_kind", string(err.Kind())),
			)
		}

		def := errdef.Define("test_error", errdef.LogValuer(customLogValuer))
		err := def.New("test message")

		value := err.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("error", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		errorData := result["error"].(map[string]any)

		want := map[string]any{
			"custom_message": "test message",
			"custom_kind":    "test_error",
		}

		if !reflect.DeepEqual(errorData, want) {
			t.Errorf("want error %+v, got %+v", want, errorData)
		}
	})

	t.Run("actual JSON logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		userID, _ := errdef.DefineField[string]("user_id")
		password, _ := errdef.DefineField[errdef.Redacted[string]]("password")
		def := errdef.Define(
			"auth_error",
			userID("user123"),
			password(errdef.Redact("my-secret-password")),
		)

		cause := errors.New("connection failed")
		err := def.Wrap(cause)

		logger.Error("authentication error", "error", err)

		if strings.Contains(buf.String(), "my-secret-password") {
			t.Fatal("want secret to be redacted, but found in log output")
		}

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("failed to unmarshal log output: %v", err)
		}

		errorGroup := got["error"].(map[string]any)
		origin := errorGroup["origin"].(map[string]any)

		want := map[string]any{
			"time":  got["time"],
			"level": "ERROR",
			"msg":   "authentication error",
			"error": map[string]any{
				"message": "connection failed",
				"kind":    "auth_error",
				"fields": map[string]any{
					"user_id":  "user123",
					"password": "[REDACTED]",
				},
				"origin": map[string]any{
					"file": origin["file"],
					"line": origin["line"],
					"func": origin["func"],
				},
				"causes": []any{"connection failed"},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("log output mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}

		file, _ := origin["file"].(string)
		if !strings.Contains(file, "error_test.go") {
			t.Errorf("want file to contain error_test.go, got %q", file)
		}

		funcName, _ := origin["func"].(string)
		if !strings.Contains(funcName, "TestError_LogValue") {
			t.Errorf("want function name to contain TestError_LogValue, got %q", funcName)
		}
	})
}

type circularError struct {
	msg   string
	cause error
}

func (e *circularError) Error() string {
	return e.msg
}

func (e *circularError) Unwrap() error {
	return e.cause
}

type nilReceiverError struct {
	msg string
}

func (e *nilReceiverError) Error() string {
	if e == nil {
		return "nil receiver error"
	}
	return e.msg
}

type valueTypeError struct {
	msg string
}

func (e valueTypeError) Error() string {
	return e.msg
}
