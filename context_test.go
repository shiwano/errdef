package errdef_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/shiwano/errdef"
)

func TestContextWithOptions(t *testing.T) {
	t.Run("returns same context when no options", func(t *testing.T) {
		baseCtx := context.Background()
		ctx := errdef.ContextWithOptions(baseCtx)

		if ctx != baseCtx {
			t.Error("should return same context when no options provided")
		}
	})

	t.Run("adds single option", func(t *testing.T) {
		baseCtx := context.Background()
		ctx := errdef.ContextWithOptions(baseCtx, errdef.HTTPStatus(404))

		if ctx == baseCtx {
			t.Error("should return new context when options are added")
		}

		def := errdef.Define("test-error")
		err := def.With(ctx).New("error")

		if status, ok := errdef.HTTPStatusFrom(err); !ok || status != 404 {
			t.Errorf("want HTTP status 404, got %d", status)
		}
	})

	t.Run("adds multiple options", func(t *testing.T) {
		baseCtx := context.Background()
		ctx := errdef.ContextWithOptions(
			baseCtx,
			errdef.HTTPStatus(400),
			errdef.TraceID("trace-123"),
			errdef.LogLevel(slog.LevelError),
		)

		if ctx == baseCtx {
			t.Error("should return new context when options are added")
		}

		def := errdef.Define("test-error")
		err := def.With(ctx).New("error")

		if status, ok := errdef.HTTPStatusFrom(err); !ok || status != 400 {
			t.Errorf("want HTTP status 400, got %d", status)
		}
		if traceID, ok := errdef.TraceIDFrom(err); !ok || traceID != "trace-123" {
			t.Errorf("want TraceID 'trace-123', got '%s'", traceID)
		}
		if level, ok := errdef.LogLevelFrom(err); !ok || level != slog.LevelError {
			t.Errorf("want LogLevel Error, got %v", level)
		}
	})

	t.Run("accumulates options", func(t *testing.T) {
		baseCtx := context.Background()

		ctx1 := errdef.ContextWithOptions(baseCtx, errdef.HTTPStatus(400))
		ctx2 := errdef.ContextWithOptions(ctx1, errdef.TraceID("trace-456"))
		ctx3 := errdef.ContextWithOptions(ctx2, errdef.HTTPStatus(500))

		if ctx3 == baseCtx {
			t.Error("should create new context with accumulated options")
		}

		def := errdef.Define("test-error")
		err := def.With(ctx3).New("error")

		if status, ok := errdef.HTTPStatusFrom(err); !ok || status != 500 {
			t.Errorf("want HTTP status 500, got %d", status)
		}
		if traceID, ok := errdef.TraceIDFrom(err); !ok || traceID != "trace-456" {
			t.Errorf("want TraceID 'trace-456', got '%s'", traceID)
		}
	})
}
