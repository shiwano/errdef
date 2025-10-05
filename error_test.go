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

		if err.Error() != "test message" {
			t.Errorf("want message %q, got %q", "test message", err.Error())
		}
	})

	t.Run("wrapped error message", func(t *testing.T) {
		def := errdef.Define("test_error")
		original := errors.New("original error")
		wrapped := def.Wrap(original)

		if wrapped.Error() != "original error" {
			t.Errorf("want message %q, got %q", "original error", wrapped.Error())
		}
	})
}

func TestError_Kind(t *testing.T) {
	def := errdef.Define("test_error")
	err := def.New("test message").(errdef.Error)

	kind := err.Kind()
	if kind != "test_error" {
		t.Errorf("want kind %q, got %q", "test_error", kind)
	}
}

func TestError_Fields(t *testing.T) {
	t.Run("no fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.Error)

		fields := err.Fields()

		collected := maps.Collect(fields.Seq())
		if len(collected) != 0 {
			t.Errorf("want empty fields, got %d fields", len(collected))
		}
	})

	t.Run("with fields", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", constructor("user123"))
		err := def.New("test message").(errdef.Error)

		fields := err.Fields()

		collected := maps.Collect(fields.Seq())
		if len(collected) != 1 {
			t.Errorf("want 1 field, got %d", len(collected))
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

		unwrapped := err.Unwrap()
		if len(unwrapped) != 0 {
			t.Errorf("want empty, got %v", unwrapped)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		original := errors.New("original error")
		wrapped := def.Wrap(original).(errdef.Error)

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

func TestDebugStacker_DebugStack(t *testing.T) {
	t.Run("debug stack format", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(errdef.DebugStacker)

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

func TestCauser_Cause(t *testing.T) {
	type causer interface {
		Cause() error
	}

	t.Run("no cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message").(causer)

		cause := err.Cause()
		if cause != nil {
			t.Errorf("want no cause, got %v", cause)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		def := errdef.Define("test_error")
		original := errors.New("original error")
		wrapped := def.Wrap(original).(causer)

		cause := wrapped.Cause()
		if cause != original {
			t.Error("want cause to be original error")
		}
	})

	t.Run("boundary breaks chain", func(t *testing.T) {
		original := errors.New("original error")
		def := errdef.Define("boundary_error", errdef.Boundary())
		wrapped := def.Wrap(original).(causer)

		cause := wrapped.Cause()
		if cause != nil {
			t.Errorf("want no cause due to boundary, got %v", cause)
		}
	})
}

func TestFormatter_Format(t *testing.T) {
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
		userIDCtor, _ := errdef.DefineField[string]("user_id")
		passwordCtor, _ := errdef.DefineField[errdef.Redacted[string]]("password")
		def := errdef.Define("test_error", userIDCtor("user123"), passwordCtor(errdef.Redact("secret")))
		err := def.New("test message")

		result := fmt.Sprintf("%+v", err)
		if matched, _ := regexp.MatchString(
			`test message\n`+
				`\n`+
				`Kind:\n`+
				`\ttest_error\n`+
				`Fields:\n`+
				`\tuser_id: user123\n`+
				`\tpassword: \[REDACTED\]\n`+
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
			"\toriginal error"
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
			"\t---\n" +
			"\terror 2"
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

func TestMarshaler_MarshalJSON(t *testing.T) {
	t.Run("basic json", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

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
		if result["kind"] != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", result["kind"])
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

		fields := result["fields"].(map[string]any)
		if len(fields) != 1 {
			t.Errorf("want 1 field, got %d", len(fields))
		}
		if fields["user_id"] != "user123" {
			t.Errorf("want field %q, got %q", "user123", fields["user_id"])
		}
	})

	t.Run("with causes", func(t *testing.T) {
		def := errdef.Define("test_error")
		original := errors.New("original error")
		wrapped := def.Wrap(original)

		data, jsonErr := json.Marshal(wrapped)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		causes := result["causes"].([]any)
		if len(causes) != 1 {
			t.Fatalf("want 1 cause, got %d", len(causes))
		}

		cause := causes[0].(map[string]any)
		if cause["message"] != "original error" {
			t.Errorf("want cause message %q, got %q", "original error", cause["message"])
		}
	})

	t.Run("with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

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

	t.Run("comprehensive json structure", func(t *testing.T) {
		userIDCtor, _ := errdef.DefineField[string]("user_id")
		passwordCtor, _ := errdef.DefineField[errdef.Redacted[string]]("password")

		def := errdef.Define("auth_error", userIDCtor("user123"), passwordCtor(errdef.Redact("secret")))
		original := errors.New("connection failed")
		err := def.Wrap(original)

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
				map[string]any{"message": "connection failed"},
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
		if !strings.Contains(funcName, "TestMarshaler_MarshalJSON") {
			t.Errorf("want function name to contain TestMarshaler_MarshalJSON, got %q", funcName)
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

	t.Run("causes with json.Marshaler", func(t *testing.T) {
		customErr := &customError{Code: 500, Msg: "custom error"}

		def := errdef.Define("wrapper_error", errdef.NoTrace())
		err := def.Wrap(customErr)

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "error code 500: custom error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "error code 500: custom error",
					"data": map[string]any{
						"code": float64(500),
						"msg":  "custom error",
					},
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
		customErr := &customError{Code: 404, Msg: "not found"}
		stdErr := errors.New("standard error")

		joined := def2.Join(definedErr, customErr, stdErr)

		data, jsonErr := json.Marshal(joined)
		if jsonErr != nil {
			t.Fatalf("want no error, got %v", jsonErr)
		}

		var got map[string]any
		if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
			t.Fatalf("want valid JSON, got %v", unmarshalErr)
		}

		want := map[string]any{
			"message": "defined error\nerror code 404: not found\nstandard error",
			"kind":    "wrapper_error",
			"causes": []any{
				map[string]any{
					"message": "defined error",
					"kind":    "defined_error",
				},
				map[string]any{
					"message": "error code 404: not found",
					"data": map[string]any{
						"code": float64(404),
						"msg":  "not found",
					},
				},
				map[string]any{
					"message": "standard error",
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
		constructor, _ := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error", constructor("user123"), errdef.NoTrace())
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
		original := errors.New("original error")
		wrapped := def.Wrap(original)

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

		userIDConstructor, _ := errdef.DefineField[string]("user_id")
		passwordConstructor, _ := errdef.DefineField[errdef.Redacted[string]]("password")
		def := errdef.Define(
			"auth_error",
			userIDConstructor("user123"),
			passwordConstructor(errdef.Redact("my-secret-password")),
		)

		original := errors.New("connection failed")
		err := def.Wrap(original)

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

type customError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *customError) Error() string {
	return fmt.Sprintf("error code %d: %s", e.Code, e.Msg)
}

func (e *customError) MarshalJSON() ([]byte, error) {
	type alias customError
	return json.Marshal((*alias)(e))
}
