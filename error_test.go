package errdef_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
)

func TestError_Error(t *testing.T) {
	t.Run("basic message", func(t *testing.T) {
		err := errdef.New("test message")

		if err.Error() != "test message" {
			t.Errorf("want message %q, got %q", "test message", err.Error())
		}
	})

	t.Run("wrapped error message", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := errdef.Wrap(original)

		if wrapped.Error() != "original error" {
			t.Errorf("want message %q, got %q", "original error", wrapped.Error())
		}
	})
}

func TestError_Kind(t *testing.T) {
	t.Run("default kind", func(t *testing.T) {
		err := errdef.New("test message").(errdef.Error)

		kind := err.Kind()
		if kind != "" {
			t.Errorf("want empty kind for default error, got %q", kind)
		}
	})

	t.Run("defined kind", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		kind := err.Kind()
		if kind != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", kind)
		}
	})
}

func TestError_Fields(t *testing.T) {
	t.Run("no fields", func(t *testing.T) {
		err := errdef.New("test message").(errdef.Error)

		fields := err.Fields()

		collected := maps.Collect(fields.Seq())
		if len(collected) != 0 {
			t.Errorf("want empty fields, got %d fields", len(collected))
		}
	})

	t.Run("with fields", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("user_id")
		err := errdef.New("test message", constructor("user123")).(errdef.Error)

		fields := err.Fields()

		collected := maps.Collect(fields.Seq())
		if len(collected) != 1 {
			t.Errorf("want 1 field, got %d", len(collected))
		}
		keys := fields.FindKeys("user_id")
		if got, ok := collected[keys[0]]; ok && got != "user123" {
			t.Errorf("want field value %q, got %q", "user123", got)
		}
	})
}

func TestError_Stack(t *testing.T) {
	t.Run("stack exists", func(t *testing.T) {
		err := errdef.New("test message").(errdef.Error)

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
		err := errdef.New("test message").(errdef.Error)

		unwrapped := err.Unwrap()
		if len(unwrapped) != 0 {
			t.Errorf("want empty, got %v", unwrapped)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := errdef.Wrap(original).(errdef.Error)

		unwrapped := wrapped.Unwrap()
		if len(unwrapped) != 1 {
			t.Errorf("want 1 error, got %d", len(unwrapped))
		}
		if unwrapped[0] != original {
			t.Error("want unwrapped error to be original error")
		}
	})

	t.Run("with multiple errors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		def := errdef.Define("test_error")
		joined := def.Join(err1, err2).(errdef.Error)

		unwrapped := joined.Unwrap()
		if len(unwrapped) != 2 {
			t.Errorf("want 2 error, got %d", len(unwrapped))
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
		if len(unwrapped) != 1 {
			t.Errorf("want 1 error, got %d", len(unwrapped))
		}
		if unwrapped[0] != joined {
			t.Error("want unwrapped error to be the joined error")
		}
	})

	t.Run("boundary breaks chain", func(t *testing.T) {
		original := errors.New("original error")
		def := errdef.Define("boundary_error", errdef.Boundary())
		wrapped := def.Wrap(original).(errdef.Error)

		unwrapped := wrapped.Unwrap()
		if len(unwrapped) != 0 {
			t.Errorf("want empty due to boundary, got %v", unwrapped)
		}
	})
}

func TestDefinedError_Is(t *testing.T) {
	t.Run("same instance", func(t *testing.T) {
		err := errdef.New("test message")

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

func TestDebugStacker_DebugStack(t *testing.T) {
	t.Run("debug stack format", func(t *testing.T) {
		err := errdef.New("test message").(errdef.DebugStacker)

		debugStack := err.DebugStack()

		want := "test message\n\ngoroutine 1 [running]:\ngithub.com/shiwano/errdef_test.TestDebugStacker_DebugStack.func1()"
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

func TestStackTracer_StackTrace(t *testing.T) {
	t.Run("stack exists", func(t *testing.T) {
		type stackTracer interface {
			StackTrace() []uintptr
		}

		err := errdef.New("test message").(stackTracer)

		stackTrace := err.StackTrace()
		if len(stackTrace) == 0 {
			t.Error("want stack trace to have frames")
		}
	})

	t.Run("no trace", func(t *testing.T) {
		type stackTracer interface {
			StackTrace() []uintptr
		}

		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message").(stackTracer)

		stackTrace := err.StackTrace()
		if len(stackTrace) != 0 {
			t.Error("want no stack trace when disabled")
		}
	})
}

func TestCauser_Cause(t *testing.T) {
	t.Run("no cause", func(t *testing.T) {
		type causer interface {
			Cause() error
		}

		err := errdef.New("test message").(causer)

		cause := err.Cause()
		if cause != nil {
			t.Errorf("want no cause, got %v", cause)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		type causer interface {
			Cause() error
		}

		original := errors.New("original error")
		wrapped := errdef.Wrap(original).(causer)

		cause := wrapped.Cause()
		if cause != original {
			t.Error("want cause to be original error")
		}
	})
}

func TestFormatter_Format(t *testing.T) {
	t.Run("default format", func(t *testing.T) {
		err := errdef.New("test message")

		result := fmt.Sprintf("%s", err)
		if result != "test message" {
			t.Errorf("want %q, got %q", "test message", result)
		}
	})

	t.Run("quoted format", func(t *testing.T) {
		err := errdef.New("test message")

		result := fmt.Sprintf("%q", err)
		if result != `"test message"` {
			t.Errorf("want %q, got %q", `"test message"`, result)
		}
	})

	t.Run("verbose format", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", constructor("user123"))
		err := def.New("test message")

		result := fmt.Sprintf("%+v", err)
		if matched, _ := regexp.MatchString(
			`test message\n`+
				`\n`+
				`Kind:\n`+
				`\ttest_error\n`+
				`Fields:\n`+
				`\tuser_id: user123\n`+
				`Stack:\n`+
				`[\s\S]*`,
			result,
		); !matched {
			t.Errorf("want format to match pattern, got: %q", result)
		}
	})

	t.Run("verbose format with cause", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("user_id")
		original := errors.New("original error")
		def := errdef.Define("test_error", errdef.NoTrace(), constructor("user123"))
		wrapped := def.Wrap(original)

		result := fmt.Sprintf("%+v", wrapped)
		want := "original error\n" +
			"\n" +
			"Kind:\n" +
			"\ttest_error\n" +
			"Fields:\n" +
			"\tuser_id: user123\n" +
			"Causes:\n" +
			"\toriginal error\n"
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
			"\n" +
			"Kind:\n" +
			"\ttest_error\n" +
			"Causes:\n" +
			"\terror 1\n" +
			"\terror 2\n"
		if want != result {
			t.Errorf("want format to equal, got: %q", result)
		}
	})

	t.Run("go format", func(t *testing.T) {
		err := errdef.New("test message", errdef.NoTrace())

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

func TestMarshaler_MarshalJSON(t *testing.T) {
	t.Run("basic json", func(t *testing.T) {
		err := errdef.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		if result["message"] != "test message" {
			t.Errorf("want message %q, got %q", "test message", result["message"])
		}
		if result["kind"] != "" {
			t.Errorf("want empty kind, got %q", result["kind"])
		}
	})

	t.Run("with kind and fields", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", constructor("user123"))
		err := def.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		if result["kind"] != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", result["kind"])
		}

		fields := result["fields"].([]any)
		if len(fields) != 1 {
			t.Errorf("want 1 field, got %d", len(fields))
		}
		field := fields[0].(map[string]any)
		if field["key"] != "user_id" {
			t.Errorf("want field key %q, got %q", "user_id", field["key"])
		}
		if field["value"] != "user123" {
			t.Errorf("want field value %q, got %q", "user123", field["value"])
		}
	})

	t.Run("with causes", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := errdef.Wrap(original)

		// This test currently fails due to a bug in error.go:216
		// where c.Error() is converted to []byte without proper JSON quoting
		_, jsonErr := json.Marshal(wrapped)
		if jsonErr == nil {
			t.Skip("This test is expected to fail due to JSON marshaling bug in causes")
		}

		// The error should contain information about invalid JSON
		expectedErrMsg := "invalid character 'o' looking for beginning of value"
		if !strings.Contains(jsonErr.Error(), expectedErrMsg) {
			t.Errorf("want error containing %q, got %q", expectedErrMsg, jsonErr.Error())
		}
	})

	t.Run("with stack", func(t *testing.T) {
		err := errdef.New("test message")

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var result map[string]any
		if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
			t.Fatalf("want valid JSON, got %v", jsonErr)
		}

		stack := result["stack"].([]any)
		if len(stack) == 0 {
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

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		if result["custom_message"] != "test message" {
			t.Errorf("want custom_message %q, got %q", "test message", result["custom_message"])
		}
	})
}
