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
	err := errdef.New("test error", errdef.HTTPStatus(404))

	status, found := errdef.HTTPStatusFrom(err)
	if !found {
		t.Error("want HTTP status to be found")
	}
	if status != 404 {
		t.Errorf("want status 404, got %d", status)
	}
}

func TestLogLevel(t *testing.T) {
	err := errdef.New("test error", errdef.LogLevel(slog.LevelError))

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
	err := errdef.New("test error", errdef.TraceID(traceID))

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
	err := errdef.New("test error", errdef.Domain(domain))

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
	err := errdef.New("test error", errdef.UserHint(hint))

	h, found := errdef.UserHintFrom(err)
	if !found {
		t.Error("want user hint to be found")
	}
	if h != hint {
		t.Errorf("want hint %q, got %q", hint, h)
	}
}

func TestPublic(t *testing.T) {
	err := errdef.New("test error", errdef.Public())

	if !errdef.IsPublic(err) {
		t.Error("want error to be public")
	}
}

func TestRetryable(t *testing.T) {
	err := errdef.New("test error", errdef.Retryable())

	if !errdef.IsRetryable(err) {
		t.Error("want error to be retryable")
	}
}

func TestRetryAfter(t *testing.T) {
	duration := 5 * time.Second
	err := errdef.New("test error", errdef.RetryAfter(duration))

	d, found := errdef.RetryAfterFrom(err)
	if !found {
		t.Error("want retry after to be found")
	}
	if d != duration {
		t.Errorf("want duration %v, got %v", duration, d)
	}
}

func TestUnreportable(t *testing.T) {
	err := errdef.New("test error", errdef.Unreportable())

	if !errdef.IsUnreportable(err) {
		t.Error("want error to be retryable")
	}
}

func TestExitCode(t *testing.T) {
	code := 42
	err := errdef.New("test error", errdef.ExitCode(code))

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
	err := errdef.New("test error", errdef.HelpURL(url))

	u, found := errdef.HelpURLFrom(err)
	if !found {
		t.Error("want help URL to be found")
	}
	if u != url {
		t.Errorf("want URL %q, got %q", url, u)
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

func TestNoTrace(t *testing.T) {
	def := errdef.Define("test", errdef.NoTrace())
	err := def.New("test error")

	frames := err.(errdef.Error).Stack().Frames()
	if len(frames) != 0 {
		t.Errorf("want no stack frames, got %d", len(frames))
	}
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
