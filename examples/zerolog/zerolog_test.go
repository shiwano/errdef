package zerolog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/shiwano/errdef"
	zerologhelper "github.com/shiwano/errdef/examples/zerolog"
)

var (
	ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
	ErrDatabase = errdef.Define("database", errdef.HTTPStatus(500))

	UserID, _ = errdef.DefineField[string]("user_id")
	Email, _  = errdef.DefineField[errdef.Redacted[string]]("email")
	Count, _  = errdef.DefineField[int]("count")
)

type errorTestCase struct {
	name string
	err  error
	want map[string]any
}

var errorTestCases = []errorTestCase{
	{
		name: "basic error with fields",
		err:  ErrNotFound.WithOptions(UserID("u123")).New("user not found"),
		want: map[string]any{
			"message": "user not found",
			"kind":    "not_found",
			"fields": map[string]any{
				"user_id":     "u123",
				"http_status": float64(404),
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
				"email":       "[REDACTED]",
				"http_status": float64(404),
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
				"http_status": float64(404),
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
				"count":       float64(42),
				"http_status": float64(404),
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

func TestEmbedObject(t *testing.T) {
	for _, tt := range errorTestCases {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := zerolog.New(buf)
			logger.Info().EmbedObject(zerologhelper.Error(tt.err)).Msg("test")

			var fields map[string]any
			if err := json.Unmarshal(buf.Bytes(), &fields); err != nil {
				t.Fatalf("failed to unmarshal log: %v", err)
			}

			delete(fields, "level")

			want := make(map[string]any)
			maps.Copy(want, tt.want)
			if _, hasOrigin := want["origin"]; hasOrigin {
				want["origin"] = fields["origin"]
			}

			// Remove message field from fields map for comparison with want
			delete(fields, "message")
			delete(want, "message")

			if !reflect.DeepEqual(fields, want) {
				t.Errorf("fields mismatch\ngot:  %#v\nwant: %#v", fields, want)
			}

			if _, hasOrigin := tt.want["origin"]; hasOrigin {
				if origin, ok := fields["origin"].(map[string]any); !ok {
					t.Errorf("want origin to be a map, got %T", fields["origin"])
				} else {
					if v, ok := origin["func"].(string); !ok || !strings.Contains(v, "init") {
						t.Errorf("want origin.func to contain 'init', got %v", origin["func"])
					}
					if v, ok := origin["file"].(string); !ok || !strings.Contains(v, "zerolog_test.go") {
						t.Errorf("want origin.file to contain 'zerolog_test.go', got %v", origin["file"])
					}
					if v, ok := origin["line"].(float64); !ok || v <= 0 {
						t.Errorf("want origin.line to be a positive number, got %v", origin["line"])
					}
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	for _, tt := range errorTestCases {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := zerolog.New(buf)
			logger.Info().Object("error", zerologhelper.Error(tt.err)).Msg("operation failed")

			var fields map[string]any
			if err := json.Unmarshal(buf.Bytes(), &fields); err != nil {
				t.Fatalf("failed to unmarshal log: %v", err)
			}

			errorObj, ok := fields["error"].(map[string]any)
			if !ok {
				t.Fatalf("want error to be a nested object, got %T", fields["error"])
			}

			want := make(map[string]any)
			maps.Copy(want, tt.want)
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
					if v, ok := origin["file"].(string); !ok || !strings.Contains(v, "zerolog_test.go") {
						t.Errorf("want origin.file to contain 'zerolog_test.go', got %v", origin["file"])
					}
					if v, ok := origin["line"].(float64); !ok || v <= 0 {
						t.Errorf("want origin.line to be a positive number, got %v", origin["line"])
					}
				}
			}
		})
	}
}

func ExampleError() {
	ctx := context.Background()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

	// Error nests error information under "error" key
	logger.Info().Object("error", zerologhelper.Error(err)).Msg("operation failed")

	innerErr := ErrDatabase.New("connection timeout")
	wrappedErr := ErrNotFound.Wrapf(innerErr, "failed to find user")
	logger.Error().Object("error", zerologhelper.Error(wrappedErr)).Msg("database operation failed")
}

func ExampleError_embedObject() {
	ctx := context.Background()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	err := ErrNotFound.With(ctx, UserID("user123")).New("user not found")

	// EmbedObject expands all error information at the top level
	logger.Info().EmbedObject(zerologhelper.Error(err)).Msg("operation failed")

	innerErr := ErrDatabase.New("connection timeout")
	wrappedErr := ErrNotFound.Wrapf(innerErr, "failed to find user")
	logger.Error().EmbedObject(zerologhelper.Error(wrappedErr)).Msg("database operation failed")
}
