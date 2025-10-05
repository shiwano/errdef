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

		got, ok := errdef.HTTPStatusFrom(err)
		if !ok {
			t.Fatal("want HTTP status to be found")
		}
		if want := 404; got != want {
			t.Errorf("want HTTP status %d, got %d", want, got)
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

		status, ok := errdef.HTTPStatusFrom(err)
		if !ok {
			t.Fatal("want HTTP status to be found")
		}
		if want := 400; status != want {
			t.Errorf("want HTTP status %d, got %d", want, status)
		}

		traceID, ok := errdef.TraceIDFrom(err)
		if !ok {
			t.Fatal("want TraceID to be found")
		}
		if want := "trace-123"; traceID != want {
			t.Errorf("want TraceID %q, got %q", want, traceID)
		}

		level, ok := errdef.LogLevelFrom(err)
		if !ok {
			t.Fatal("want LogLevel to be found")
		}
		if want := slog.LevelError; level != want {
			t.Errorf("want LogLevel %v, got %v", want, level)
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

		status, ok := errdef.HTTPStatusFrom(err)
		if !ok {
			t.Fatal("want HTTP status to be found")
		}
		if want := 500; status != want {
			t.Errorf("want HTTP status %d, got %d", want, status)
		}

		traceID, ok := errdef.TraceIDFrom(err)
		if !ok {
			t.Fatal("want TraceID to be found")
		}
		if want := "trace-456"; traceID != want {
			t.Errorf("want TraceID %q, got %q", want, traceID)
		}
	})
}
