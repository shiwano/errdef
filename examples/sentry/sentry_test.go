package sentry_test

import (
	"context"
	"log"
	"reflect"
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

	t.Run("success", func(t *testing.T) {
		transport := &mockTransport{}
		client, err := sentrygo.NewClient(sentrygo.ClientOptions{
			Transport: transport,
		})
		if err != nil {
			t.Fatalf("failed to create Sentry client: %v", err)
		}
		hub := sentrygo.NewHub(client, sentrygo.NewScope())

		// Create nested errors (3 levels)
		level0Err := ErrDatabase.With(ctx, UserID("u456")).New("connection failed")
		level1Err := ErrNotFound.Wrapf(level0Err, "user lookup failed")
		level2Err := ErrDatabase.Wrapf(level1Err, "query failed")

		// Capture error with custom hub
		testCtx := sentrygo.SetHubOnContext(ctx, hub)
		captured := sentry.CaptureError(testCtx, level2Err)

		if !captured {
			t.Fatal("expected CaptureError to return true")
		}

		if len(transport.events) == 0 {
			t.Fatal("expected at least one event to be sent to Sentry")
		}

		event := transport.events[0]
		errorContextRaw := event.Contexts["error"]
		errorContext := map[string]any(errorContextRaw)
		causes := errorContext["causes"].([]map[string]any)
		level1Causes := causes[0]["causes"].([]map[string]any)

		want := map[string]any{
			"fields": map[string]any{
				"http_status": 500,
			},
			"causes": []map[string]any{
				{
					"message": "user lookup failed: connection failed",
					"kind":    "not_found",
					"fields": map[string]any{
						"http_status": 404,
					},
					"stack": causes[0]["stack"],
					"causes": []map[string]any{
						{
							"message": "connection failed",
							"kind":    "database",
							"fields": map[string]any{
								"user_id":     "u456",
								"http_status": 500,
							},
							"stack": level1Causes[0]["stack"],
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(errorContext, want) {
			t.Errorf("error context mismatch:\ngot:  %#v\nwant: %#v", errorContext, want)
		}
	})

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

type mockTransport struct{ events []*sentrygo.Event }

func (t *mockTransport) SendEvent(event *sentrygo.Event)           { t.events = append(t.events, event) }
func (t *mockTransport) Flush(timeout time.Duration) bool          { return true }
func (t *mockTransport) FlushWithContext(ctx context.Context) bool { return true }
func (t *mockTransport) Configure(options sentrygo.ClientOptions)  {}
func (t *mockTransport) Close()                                    {}
