package sentry_test

import (
	"context"
	"log"
	"testing"
	"time"

	sentrygo "github.com/getsentry/sentry-go"
	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/examples/sentry"
)

var (
	ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404), sentry.Level(sentrygo.LevelInfo))
	ErrDatabase = errdef.Define("database", errdef.HTTPStatus(500), sentry.Level(sentrygo.LevelError))

	UserID, UserIDFrom = errdef.DefineField[string]("user_id")
	Email, EmailFrom   = errdef.DefineField[errdef.Redacted[string]]("email")
)

func TestCaptureError(t *testing.T) {
	ctx := context.Background()

	t.Run("unreportable error returns false", func(t *testing.T) {
		def := errdef.Define("unreportable_error", errdef.Unreportable())
		err := def.New("this should not be reported")
		captured := sentry.CaptureError(ctx, err)
		if captured {
			t.Error("expected CaptureError to return false for unreportable error, got true")
		}
	})

	t.Run("nil error returns false", func(t *testing.T) {
		captured := sentry.CaptureError(ctx, nil)
		if captured {
			t.Error("expected CaptureError to return false for nil error, got true")
		}
	})
}

func TestExtractStacktrace(t *testing.T) {
	ctx := context.Background()

	t.Run("errdef.Error has stack trace", func(t *testing.T) {
		err := ErrNotFound.With(ctx, UserID("u123")).New("user not found")
		stacktrace := sentrygo.ExtractStacktrace(err)

		if stacktrace == nil {
			t.Fatal("expected stack trace to be extracted, got nil")
		}

		if len(stacktrace.Frames) == 0 {
			t.Fatal("expected stack trace frames to be non-empty")
		}

		foundTestFunc := false
		for _, frame := range stacktrace.Frames {
			if frame.Module == "github.com/shiwano/errdef/examples/sentry_test" {
				foundTestFunc = true
				break
			}
		}
		if !foundTestFunc {
			t.Errorf("expected to find test function in stack trace frames")
		}
	})

	t.Run("wrapped errdef.Error has stack trace", func(t *testing.T) {
		innerErr := ErrDatabase.New("connection failed")
		err := ErrNotFound.Wrapf(innerErr, "user lookup failed")
		stacktrace := sentrygo.ExtractStacktrace(err)

		if stacktrace == nil {
			t.Fatal("expected stack trace to be extracted from wrapped error, got nil")
		}

		if len(stacktrace.Frames) == 0 {
			t.Fatal("expected stack trace frames to be non-empty for wrapped error")
		}

		foundTestFunc := false
		for _, frame := range stacktrace.Frames {
			if frame.Module == "github.com/shiwano/errdef/examples/sentry_test" {
				foundTestFunc = true
				break
			}
		}
		if !foundTestFunc {
			t.Errorf("expected to find test function in stack trace frames")
		}
	})
}

func ExampleCaptureError() {
	// Initialize Sentry
	if err := sentrygo.Init(sentrygo.ClientOptions{
		Dsn:            "https://examplePublicKey@o0.ingest.sentry.io/0",
		Debug:          true,
		SendDefaultPII: true,
	}); err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentrygo.Flush(2 * time.Second)

	ctx := context.Background()

	// Report error with context
	err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")
	sentry.CaptureError(ctx, err)
}
