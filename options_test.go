package errdef_test

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/shiwano/errdef"
)

func TestHTTPStatus(t *testing.T) {
	def := errdef.Define("test_error", errdef.HTTPStatus(404))
	err := def.New("test error")

	status, found := errdef.HTTPStatusFrom(err)
	if !found {
		t.Error("want HTTP status to be found")
	}
	if status != 404 {
		t.Errorf("want status 404, got %d", status)
	}
}

func TestLogLevel(t *testing.T) {
	def := errdef.Define("test_error", errdef.LogLevel(slog.LevelError))
	err := def.New("test error")

	level, found := errdef.LogLevelFrom(err)
	if !found {
		t.Error("want log level to be found")
	}
	if level != slog.LevelError {
		t.Errorf("want level %v, got %v", slog.LevelError, level)
	}
}

func TestTraceID(t *testing.T) {
	traceID := "abc123-def456"
	def := errdef.Define("test_error", errdef.TraceID(traceID))
	err := def.New("test error")

	id, found := errdef.TraceIDFrom(err)
	if !found {
		t.Error("want trace ID to be found")
	}
	if id != traceID {
		t.Errorf("want trace ID %q, got %q", traceID, id)
	}
}

func TestDomain(t *testing.T) {
	domain := "auth_service"
	def := errdef.Define("test_error", errdef.Domain(domain))
	err := def.New("test error")

	d, found := errdef.DomainFrom(err)
	if !found {
		t.Error("want domain to be found")
	}
	if d != domain {
		t.Errorf("want domain %q, got %q", domain, d)
	}
}

func TestUserHint(t *testing.T) {
	hint := "Please check your credentials"
	def := errdef.Define("test_error", errdef.UserHint(hint))
	err := def.New("test error")

	h, found := errdef.UserHintFrom(err)
	if !found {
		t.Error("want user hint to be found")
	}
	if h != hint {
		t.Errorf("want hint %q, got %q", hint, h)
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
	duration := 5 * time.Second
	def := errdef.Define("test_error", errdef.RetryAfter(duration))
	err := def.New("test error")

	d, found := errdef.RetryAfterFrom(err)
	if !found {
		t.Error("want retry after to be found")
	}
	if d != duration {
		t.Errorf("want duration %v, got %v", duration, d)
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
	code := 42
	def := errdef.Define("test_error", errdef.ExitCode(code))
	err := def.New("test error")

	c, found := errdef.ExitCodeFrom(err)
	if !found {
		t.Error("want exit code to be found")
	}
	if c != code {
		t.Errorf("want code %d, got %d", code, c)
	}
}

func TestHelpURL(t *testing.T) {
	url := "https://example.com/help"
	def := errdef.Define("test_error", errdef.HelpURL(url))
	err := def.New("test error")

	u, found := errdef.HelpURLFrom(err)
	if !found {
		t.Error("want help URL to be found")
	}
	if u != url {
		t.Errorf("want URL %q, got %q", url, u)
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
	original := errors.New("original error")
	def := errdef.Define("test", errdef.Boundary())
	wrapped := def.Wrap(original)

	if errors.Unwrap(wrapped) != nil {
		t.Error("want Unwrap to return nil when Boundary is set")
	}

	if errors.Is(wrapped, original) {
		t.Error("want Is relationship to be broken by boundary")
	}

	if wrapped.Error() != original.Error() {
		t.Errorf("want message %q, got %q", original.Error(), wrapped.Error())
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
	t.Run("string key-value pairs", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details("key1", "value1", "key2", 123))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 2 {
			t.Fatalf("want 2 details, got %d", len(details))
		}

		if details[0].Key != "key1" || details[0].Value != "value1" {
			t.Errorf("want key1=value1, got %s=%v", details[0].Key, details[0].Value)
		}

		if details[1].Key != "key2" || details[1].Value != 123 {
			t.Errorf("want key2=123, got %s=%v", details[1].Key, details[1].Value)
		}
	})

	t.Run("DetailKV struct", func(t *testing.T) {
		kv := errdef.Detail{Key: "custom_key", Value: "custom_value"}
		def := errdef.Define("test", errdef.Details(kv))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 1 {
			t.Fatalf("want 1 detail, got %d", len(details))
		}

		if details[0].Key != "custom_key" || details[0].Value != "custom_value" {
			t.Errorf("want custom_key=custom_value, got %s=%v", details[0].Key, details[0].Value)
		}
	})

	t.Run("DetailKV slice", func(t *testing.T) {
		kvs := []errdef.Detail{
			{Key: "k1", Value: "v1"},
			{Key: "k2", Value: "v2"},
		}
		def := errdef.Define("test", errdef.Details(kvs))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 2 {
			t.Fatalf("want 2 details, got %d", len(details))
		}

		if details[0].Key != "k1" || details[0].Value != "v1" {
			t.Errorf("want k1=v1, got %s=%v", details[0].Key, details[0].Value)
		}

		if details[1].Key != "k2" || details[1].Value != "v2" {
			t.Errorf("want k2=v2, got %s=%v", details[1].Key, details[1].Value)
		}
	})

	t.Run("empty args", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details())
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 0 {
			t.Errorf("want empty details, got %d items", len(details))
		}
	})

	t.Run("trailing string without value", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details("key1", "value1", "trailing"))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 2 {
			t.Fatalf("want 2 details, got %d", len(details))
		}

		if details[0].Key != "key1" || details[0].Value != "value1" {
			t.Errorf("want key1=value1, got %s=%v", details[0].Key, details[0].Value)
		}

		if details[1].Key != "__INVALID_TAIL__" || details[1].Value != "trailing" {
			t.Errorf("want __INVALID_TAIL__=trailing, got %s=%v", details[1].Key, details[1].Value)
		}
	})

	t.Run("non-string key element", func(t *testing.T) {
		def := errdef.Define("test", errdef.Details(123, "ignored"))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 2 {
			t.Fatalf("want 2 details, got %d", len(details))
		}

		if details[0].Key != "__INVALID_STANDALONE__" || details[0].Value != 123 {
			t.Errorf("want __INVALID_STANDALONE__=123, got %s=%v", details[0].Key, details[0].Value)
		}

		if details[1].Key != "__INVALID_TAIL__" || details[1].Value != "ignored" {
			t.Errorf("want __INVALID_TAIL__=ignored, got %s=%v", details[1].Key, details[1].Value)
		}
	})

	t.Run("mixed formats", func(t *testing.T) {
		kv := errdef.Detail{Key: "struct_key", Value: "struct_value"}
		kvs := []errdef.Detail{{Key: "slice_key", Value: "slice_value"}}
		def := errdef.Define("test", errdef.Details("string_key", "string_value", kv, kvs))
		err := def.New("test error")

		details, found := errdef.DetailsFrom(err)
		if !found {
			t.Fatal("want details to be found")
		}

		if len(details) != 3 {
			t.Fatalf("want 3 details, got %d", len(details))
		}

		if details[0].Key != "string_key" || details[0].Value != "string_value" {
			t.Errorf("want string_key=string_value, got %s=%v", details[0].Key, details[0].Value)
		}

		if details[1].Key != "struct_key" || details[1].Value != "struct_value" {
			t.Errorf("want struct_key=struct_value, got %s=%v", details[1].Key, details[1].Value)
		}

		if details[2].Key != "slice_key" || details[2].Value != "slice_value" {
			t.Errorf("want slice_key=slice_value, got %s=%v", details[2].Key, details[2].Value)
		}
	})

	t.Run("no details set", func(t *testing.T) {
		def := errdef.Define("test")
		err := def.New("test error")

		_, found := errdef.DetailsFrom(err)
		if found {
			t.Error("want details not to be found")
		}
	})
}
