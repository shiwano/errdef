package zap_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
	zaphelper "github.com/shiwano/errdef/examples/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var (
	ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
	ErrDatabase = errdef.Define("database", errdef.HTTPStatus(500))

	UserID, _ = errdef.DefineField[string]("user_id")
	Email, _  = errdef.DefineField[errdef.Redacted[string]]("email")
	Count, _  = errdef.DefineField[int]("count")
)

var tests = []struct {
	name string
	err  error
	want map[string]any
}{
	{
		name: "basic error with fields",
		err:  ErrNotFound.WithOptions(UserID("u123")).New("user not found"),
		want: map[string]any{
			"message": "user not found",
			"kind":    "not_found",
			"fields": map[string]any{
				"user_id":     "u123",
				"http_status": 404,
			},
			"origin": nil,
		},
	},
	{
		name: "error with redacted field",
		err: ErrNotFound.WithOptions(
			UserID("u123"),
			Email(errdef.Redact("user@example.com")),
		).New("user not found"),
		want: map[string]any{
			"message": "user not found",
			"kind":    "not_found",
			"fields": map[string]any{
				"user_id":     "u123",
				"email":       errdef.Redact("user@example.com"),
				"http_status": 404,
			},
			"origin": nil,
		},
	},
	{
		name: "wrapped error",
		err:  ErrNotFound.Wrapf(ErrDatabase.New("connection failed"), "user lookup failed"),
		want: map[string]any{
			"message": "user lookup failed: connection failed",
			"kind":    "not_found",
			"fields": map[string]any{
				"http_status": 404,
			},
			"origin": nil,
		},
	},
	{
		name: "multiple fields",
		err: ErrNotFound.WithOptions(
			UserID("u123"),
			Count(42),
		).New("user not found"),
		want: map[string]any{
			"message": "user not found",
			"kind":    "not_found",
			"fields": map[string]any{
				"user_id":     "u123",
				"count":       42,
				"http_status": 404,
			},
			"origin": nil,
		},
	},
	{
		name: "non-errdef error",
		err:  errors.New("standard error"),
		want: map[string]any{
			"message": "standard error",
		},
	},
}

func TestErrorInline(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			logger.Info("test", zaphelper.ErrorInline(tt.err))

			if recorded.Len() != 1 {
				t.Fatalf("want 1 log entry, got %d", recorded.Len())
			}

			entry := recorded.All()[0]
			got := entry.ContextMap()

			want := tt.want
			if _, hasOrigin := want["origin"]; hasOrigin {
				want["origin"] = got["origin"]
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("fields mismatch\ngot:  %#v\nwant: %#v", got, want)
			}

			if _, hasOrigin := tt.want["origin"]; hasOrigin {
				if origin, ok := got["origin"].(map[string]any); !ok {
					t.Errorf("want origin to be a map, got %T", got["origin"])
				} else {
					if v, ok := origin["func"].(string); !ok || !strings.Contains(v, "init") {
						t.Errorf("want origin.func to contain 'init', got %v", origin["func"])
					}
					if v, ok := origin["file"].(string); !ok || !strings.Contains(v, "zap_test.go") {
						t.Errorf("want origin.file to contain 'zap_test.go', got %v", origin["file"])
					}
					if v, ok := origin["line"].(int); !ok || v <= 0 {
						t.Errorf("want origin.line to be a positive int, got %v", origin["line"])
					}
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			logger.Info("operation failed", zaphelper.Error(tt.err))

			if recorded.Len() != 1 {
				t.Fatalf("want 1 log entry, got %d", recorded.Len())
			}

			entry := recorded.All()[0]
			got := entry.ContextMap()

			errorObj, ok := got["error"].(map[string]any)
			if !ok {
				t.Fatalf("want error to be a nested object, got %T", got["error"])
			}

			want := tt.want
			if _, hasOrigin := want["origin"]; hasOrigin {
				want["origin"] = errorObj["origin"]
			}

			if !reflect.DeepEqual(errorObj, want) {
				t.Errorf("error object mismatch\ngot:  %#v\nwant: %#v", errorObj, want)
			}

			if _, hasOrigin := tt.want["origin"]; hasOrigin {
				if origin, ok := errorObj["origin"].(map[string]any); !ok {
					t.Errorf("want origin to be a map, got %T", errorObj["origin"])
				} else {
					if v, ok := origin["func"].(string); !ok || !strings.Contains(v, "init") {
						t.Errorf("want origin.func to contain 'init', got %v", origin["func"])
					}
					if v, ok := origin["file"].(string); !ok || !strings.Contains(v, "zap_test.go") {
						t.Errorf("want origin.file to contain 'zap_test.go', got %v", origin["file"])
					}
					if v, ok := origin["line"].(int); !ok || v <= 0 {
						t.Errorf("want origin.line to be a positive int, got %v", origin["line"])
					}
				}
			}
		})
	}
}

func ExampleError() {
	ctx := context.Background()

	config := zap.NewProductionConfig()
	logger, _ := config.Build()
	defer func() { _ = logger.Sync() }()

	err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

	// Error nests error information under "error" key
	logger.Info("operation failed", zaphelper.Error(err))

	innerErr := ErrDatabase.New("connection timeout")
	wrappedErr := ErrNotFound.Wrapf(innerErr, "failed to find user")
	logger.Error("database operation failed", zaphelper.Error(wrappedErr))
}

func ExampleErrorInline() {
	ctx := context.Background()

	config := zap.NewProductionConfig()
	logger, _ := config.Build()
	defer func() { _ = logger.Sync() }()

	err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

	// ErrorInline expands all error information at the top level
	logger.Info("operation failed", zaphelper.ErrorInline(err))

	innerErr := ErrDatabase.New("connection timeout")
	wrappedErr := ErrNotFound.Wrapf(innerErr, "failed to find user")
	logger.Error("database operation failed", zaphelper.ErrorInline(wrappedErr))
}
