package errdef_test

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shiwano/errdef"
)

func TestHTTPStatus(t *testing.T) {
	def := errdef.Define("test_error", errdef.HTTPStatus(404))
	err := def.New("test error")

	got, ok := errdef.HTTPStatusFrom(err)
	if !ok {
		t.Error("want HTTP status to be found")
	}
	if want := 404; got != want {
		t.Errorf("want status %d, got %d", want, got)
	}
}

func TestLogLevel(t *testing.T) {
	def := errdef.Define("test_error", errdef.LogLevel(slog.LevelError))
	err := def.New("test error")

	got, ok := errdef.LogLevelFrom(err)
	if !ok {
		t.Error("want log level to be found")
	}
	if want := slog.LevelError; got != want {
		t.Errorf("want level %v, got %v", want, got)
	}
}

func TestTraceID(t *testing.T) {
	want := "abc123-def456"
	def := errdef.Define("test_error", errdef.TraceID(want))
	err := def.New("test error")

	got, ok := errdef.TraceIDFrom(err)
	if !ok {
		t.Error("want trace ID to be found")
	}
	if got != want {
		t.Errorf("want trace ID %q, got %q", want, got)
	}
}

func TestDomain(t *testing.T) {
	want := "auth_service"
	def := errdef.Define("test_error", errdef.Domain(want))
	err := def.New("test error")

	got, ok := errdef.DomainFrom(err)
	if !ok {
		t.Error("want domain to be found")
	}
	if got != want {
		t.Errorf("want domain %q, got %q", want, got)
	}
}

func TestUserHint(t *testing.T) {
	want := "Please check your credentials"
	def := errdef.Define("test_error", errdef.UserHint(want))
	err := def.New("test error")

	got, ok := errdef.UserHintFrom(err)
	if !ok {
		t.Error("want user hint to be found")
	}
	if got != want {
		t.Errorf("want hint %q, got %q", want, got)
	}
}

func TestPublic(t *testing.T) {
	def := errdef.Define("test_error", errdef.Public())
	err := def.New("test error")

	if !errdef.IsPublic(err) {
		t.Error("want error to be public")
	}
}

func TestRetryable(t *testing.T) {
	def := errdef.Define("test_error", errdef.Retryable())
	err := def.New("test error")

	if !errdef.IsRetryable(err) {
		t.Error("want error to be retryable")
	}
}

func TestRetryAfter(t *testing.T) {
	want := 5 * time.Second
	def := errdef.Define("test_error", errdef.RetryAfter(want))
	err := def.New("test error")

	got, ok := errdef.RetryAfterFrom(err)
	if !ok {
		t.Error("want retry after to be found")
	}
	if got != want {
		t.Errorf("want duration %v, got %v", want, got)
	}
}

func TestUnreportable(t *testing.T) {
	def := errdef.Define("test_error", errdef.Unreportable())
	err := def.New("test error")

	if !errdef.IsUnreportable(err) {
		t.Error("want error to be retryable")
	}
}

func TestExitCode(t *testing.T) {
	want := 42
	def := errdef.Define("test_error", errdef.ExitCode(want))
	err := def.New("test error")

	got, ok := errdef.ExitCodeFrom(err)
	if !ok {
		t.Error("want exit code to be found")
	}
	if got != want {
		t.Errorf("want code %d, got %d", want, got)
	}
}

func TestHelpURL(t *testing.T) {
	want := "https://example.com/help"
	def := errdef.Define("test_error", errdef.HelpURL(want))
	err := def.New("test error")

	got, ok := errdef.HelpURLFrom(err)
	if !ok {
		t.Error("want help URL to be found")
	}
	if got != want {
		t.Errorf("want URL %q, got %q", want, got)
	}
}

func TestNoTrace(t *testing.T) {
	def := errdef.Define("test", errdef.NoTrace())
	err := def.New("test error")

	frames := err.(errdef.Error).Stack().Frames()
	if len(frames) != 0 {
		t.Errorf("want no stack frames, got %d", len(frames))
	}
}

func TestStackSkip(t *testing.T) {
	t.Run("stack skip with positive value", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackSkip(1))
		err := def.New("test error")

		f := err.(errdef.Error).Stack().Frames()[0]
		if strings.Contains(f.Func, "TestStackSkip") {
			t.Errorf("want stack to skip TestStackSkip frame, got %s", f.Func)
		}
	})

	t.Run("stack skip with zero value", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackSkip(0))
		err := def.New("test error")

		f := err.(errdef.Error).Stack().Frames()[0]
		if !strings.Contains(f.Func, "TestStackSkip") {
			t.Errorf("want stack to include TestStackSkip frame, got %s", f.Func)
		}
	})

	t.Run("stack skip with negative value", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackSkip(1), errdef.StackSkip(-1))
		err := def.New("test error")

		f := err.(errdef.Error).Stack().Frames()[0]
		if !strings.Contains(f.Func, "TestStackSkip") {
			t.Errorf("want stack to include TestStackSkip frame, got %s", f.Func)
		}
	})

	t.Run("stack skip with large value", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackSkip(100))
		err := def.New("test error")

		frames := err.(errdef.Error).Stack().Frames()
		if len(frames) != 0 {
			t.Errorf("want no stack frames, got %d", len(frames))
		}
	})
}

func TestStackDepth(t *testing.T) {
	t.Run("stack depth with zero value (uses default)", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackDepth(0))
		err := def.New("test error")

		frames := err.(errdef.Error).Stack().Frames()
		if len(frames) != 3 {
			t.Errorf("want 3 stack frames, got %d", len(frames))
		}
	})

	t.Run("stack depth with small positive value", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackDepth(1))
		err := def.New("test error")

		frames := err.(errdef.Error).Stack().Frames()
		if len(frames) != 1 {
			t.Errorf("want 1 stack frames, got %d", len(frames))
		}
	})

	t.Run("stack depth with multiple values (last one wins)", func(t *testing.T) {
		def := errdef.Define("test", errdef.StackDepth(10), errdef.StackDepth(2))
		err := def.New("test error")

		frames := err.(errdef.Error).Stack().Frames()
		if len(frames) != 2 {
			t.Errorf("want 2 stack frames, got %d", len(frames))
		}
	})
}

func TestBoundary(t *testing.T) {
	orig := errors.New("original error")
	def := errdef.Define("test", errdef.Boundary())
	wrapped := def.Wrap(orig)

	if errors.Unwrap(wrapped) != nil {
		t.Error("want Unwrap to return nil when Boundary is set")
	}

	if errors.Is(wrapped, orig) {
		t.Error("want Is relationship to be broken by boundary")
	}

	if wrapped.Error() != orig.Error() {
		t.Errorf("want message %q, got %q", orig.Error(), wrapped.Error())
	}
}

func TestFormatter(t *testing.T) {
	customFormatter := func(err errdef.Error, s fmt.State, verb rune) {
		_, _ = fmt.Fprintf(s, "CUSTOM: %s", err.Error())
	}

	def := errdef.Define("test", errdef.Formatter(customFormatter))
	err := def.New("test error")

	formatted := fmt.Sprintf("%v", err)
	if formatted != "CUSTOM: test error" {
		t.Errorf("want custom formatted output, got %q", formatted)
	}
}

func TestLogValuer(t *testing.T) {
	t.Run("custom log valuer", func(t *testing.T) {
		customLogValuer := func(err errdef.Error) slog.Value {
			return slog.GroupValue(
				slog.String("custom_message", err.Error()),
				slog.String("custom_kind", string(err.Kind())),
				slog.String("custom_field", "custom_value"),
			)
		}

		def := errdef.Define("test_error", errdef.LogValuer(customLogValuer))
		err := def.New("test message")

		logValuer := err.(slog.LogValuer)
		value := logValuer.LogValue()
		attrs := value.Group()

		attrMap := make(map[string]slog.Value)
		for _, attr := range attrs {
			attrMap[attr.Key] = attr.Value
		}

		if customMessage := attrMap["custom_message"]; customMessage.String() != "test message" {
			t.Errorf("want custom_message %q, got %q", "test message", customMessage.String())
		}

		if customKind := attrMap["custom_kind"]; customKind.String() != "test_error" {
			t.Errorf("want custom_kind %q, got %q", "test_error", customKind.String())
		}

		if customField := attrMap["custom_field"]; customField.String() != "custom_value" {
			t.Errorf("want custom_field %q, got %q", "custom_value", customField.String())
		}

		if msg := attrMap["message"]; msg.Any() != nil {
			t.Error("want no default message when custom log valuer is used")
		}
	})

	t.Run("nil log valuer uses default", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.LogValuer(nil))
		err := def.New("test message")

		logValuer := err.(slog.LogValuer)
		value := logValuer.LogValue()
		attrs := value.Group()

		attrMap := make(map[string]slog.Value)
		for _, attr := range attrs {
			attrMap[attr.Key] = attr.Value
		}

		if msg := attrMap["message"]; msg.String() != "test message" {
			t.Errorf("want message %q, got %q", "test message", msg.String())
		}

		if kind := attrMap["kind"]; kind.String() != "test_error" {
			t.Errorf("want kind %q, got %q", "test_error", kind.String())
		}
	})

	t.Run("overriding log valuer", func(t *testing.T) {
		firstLogValuer := func(err errdef.Error) slog.Value {
			return slog.GroupValue(slog.String("first", "value"))
		}

		secondLogValuer := func(err errdef.Error) slog.Value {
			return slog.GroupValue(slog.String("second", "value"))
		}

		def := errdef.Define("test_error", errdef.LogValuer(firstLogValuer), errdef.LogValuer(secondLogValuer))
		err := def.New("test message")

		logValuer := err.(slog.LogValuer)
		value := logValuer.LogValue()
		attrs := value.Group()

		attrMap := make(map[string]slog.Value)
		for _, attr := range attrs {
			attrMap[attr.Key] = attr.Value
		}

		if second := attrMap["second"]; second.String() != "value" {
			t.Errorf("want second log valuer to override first, got %v", second.Any())
		}

		if first := attrMap["first"]; first.Any() != nil {
			t.Error("want first log valuer to be overridden")
		}
	})
}

func TestDetails(t *testing.T) {
	t.Run("map with multiple values", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details{
			"key1": "value1",
			"key2": 123,
		})
		err := def.New("test error")

		details, ok := errdef.DetailsFrom(err)
		if !ok {
			t.Fatal("want details to be found")
		}

		want := errdef.Details{
			"key1": "value1",
			"key2": 123,
		}
		if !reflect.DeepEqual(details, want) {
			t.Errorf("want details=%v, got details=%v", want, details)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details{})
		err := def.New("test error")

		details, ok := errdef.DetailsFrom(err)
		if !ok {
			t.Fatal("want details to be found")
		}

		want := errdef.Details{}
		if !reflect.DeepEqual(details, want) {
			t.Errorf("want details=%v, got details=%v", want, details)
		}
	})

	t.Run("complex values", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details{
			"string_key": "string_value",
			"int_key":    42,
			"slice_key":  []int{1, 2, 3},
		})
		err := def.New("test error")

		details, ok := errdef.DetailsFrom(err)
		if !ok {
			t.Fatal("want details to be found")
		}

		want := errdef.Details{
			"string_key": "string_value",
			"int_key":    42,
			"slice_key":  []int{1, 2, 3},
		}
		if !reflect.DeepEqual(details, want) {
			t.Errorf("want details=%v, got details=%v", want, details)
		}
	})

	t.Run("no details set", func(t *testing.T) {
		def := errdef.Define("test")
		err := def.New("test error")

		_, ok := errdef.DetailsFrom(err)
		if ok {
			t.Error("want details not to be found")
		}
	})

	t.Run("Key method returns key that can retrieve value", func(t *testing.T) {
		want := errdef.Details{
			"key1": "value1",
			"key2": 123,
		}
		def := errdef.Define("test", want)
		err := def.New("test error")

		key := want.Key()
		fields := err.(errdef.Error).Fields()
		fieldValue, ok := fields.Get(key)
		if !ok {
			t.Fatal("want field value to be found using Key()")
		}

		got := fieldValue.Value().(errdef.Details)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("want details=%v, got details=%v", want, got)
		}
	})
}
