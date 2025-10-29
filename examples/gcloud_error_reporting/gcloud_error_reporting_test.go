package gcerr_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
	gcerr "github.com/shiwano/errdef/examples/gcloud_error_reporting"
)

var (
	ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
	ErrDatabase = errdef.Define("database", errdef.HTTPStatus(500))

	UserID, _ = errdef.DefineField[string]("user_id")
	Email, _  = errdef.DefineField[errdef.Redacted[string]]("email")
	Count, _  = errdef.DefineField[int]("count")
)

func TestError(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/users/u123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://example.com")
	req.RemoteAddr = "192.168.1.1:12345"

	tests := []struct {
		name string
		err  error
		want map[string]any
	}{
		{
			name: "basic error with fields",
			err:  ErrNotFound.WithOptions(UserID("u123")).New("user not found"),
			want: map[string]any{
				"error": map[string]any{
					"message": "user not found",
					"kind":    "not_found",
					"fields": map[string]any{
						"user_id":     "u123",
						"http_status": float64(404),
					},
				},
				"stack_trace": "",
				"context": map[string]any{
					"reportLocation": nil,
					"httpRequest": map[string]any{
						"responseStatusCode": float64(404),
					},
				},
			},
		},
		{
			name: "error with HTTPRequest and User",
			err: ErrNotFound.WithOptions(
				UserID("u123"),
				gcerr.HTTPRequest(req),
				gcerr.User("u123"),
			).New("user not found"),
			want: map[string]any{
				"error": map[string]any{
					"message": "user not found",
					"kind":    "not_found",
					"fields": map[string]any{
						"user_id":     "u123",
						"http_status": float64(404),
					},
				},
				"stack_trace": "",
				"context": map[string]any{
					"reportLocation": nil,
					"httpRequest": map[string]any{
						"method":             "GET",
						"url":                "https://example.com/users/u123",
						"userAgent":          "Mozilla/5.0",
						"referrer":           "https://example.com",
						"responseStatusCode": float64(404),
						"remoteIp":           "192.168.1.1:12345",
					},
					"user": "u123",
				},
			},
		},
		{
			name: "wrapped error",
			err:  ErrNotFound.Wrapf(ErrDatabase.New("connection failed"), "user lookup failed"),
			want: map[string]any{
				"error": map[string]any{
					"message": "user lookup failed: connection failed",
					"kind":    "not_found",
					"fields": map[string]any{
						"http_status": float64(404),
					},
					"causes": []any{"connection failed"},
				},
				"stack_trace": "",
				"context": map[string]any{
					"reportLocation": nil,
					"httpRequest": map[string]any{
						"responseStatusCode": float64(404),
					},
				},
			},
		},
		{
			name: "error with redacted field",
			err: ErrNotFound.WithOptions(
				UserID("u123"),
				Email(errdef.Redact("user@example.com")),
			).New("user not found"),
			want: map[string]any{
				"error": map[string]any{
					"message": "user not found",
					"kind":    "not_found",
					"fields": map[string]any{
						"user_id":     "u123",
						"email":       "[REDACTED]",
						"http_status": float64(404),
					},
				},
				"stack_trace": "",
				"context": map[string]any{
					"reportLocation": nil,
					"httpRequest": map[string]any{
						"responseStatusCode": float64(404),
					},
				},
			},
		},
		{
			name: "non-errdef error",
			err:  errors.New("standard error"),
			want: map[string]any{
				"@type":   "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
				"message": "standard error",
			},
		},
		{
			name: "error with NoTrace option",
			err:  errdef.Define("no_trace", errdef.NoTrace()).New("error without stack trace"),
			want: map[string]any{
				"error": map[string]any{
					"message": "error without stack trace",
					"kind":    "no_trace",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{}))

			logger.Error("operation failed", gcerr.Error(tt.err))

			var got map[string]any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			// Remove log metadata
			delete(got, "time")
			delete(got, "level")
			delete(got, "msg")

			want := tt.want
			if _, ok := want["stack_trace"]; ok {
				want["stack_trace"] = got["stack_trace"]
			}
			if context, ok := want["context"].(map[string]any); ok {
				if _, ok := context["reportLocation"]; ok {
					if resultContext, ok := got["context"].(map[string]any); ok {
						context["reportLocation"] = resultContext["reportLocation"]
					}
				}
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("result mismatch\ngot:  %#v\nwant: %#v", got, want)
			}

			// Verify stack_trace format if present
			if stackTrace, ok := got["stack_trace"].(string); ok && stackTrace != "" {
				if !strings.Contains(stackTrace, "\n") {
					t.Errorf("want stack_trace to contain newlines, got %q", stackTrace)
				}
			}

			// Verify reportLocation format if present
			if context, ok := got["context"].(map[string]any); ok {
				if reportLocation, ok := context["reportLocation"].(map[string]any); ok {
					if v, ok := reportLocation["filePath"].(string); !ok || v == "" {
						t.Errorf("want reportLocation.filePath to be a non-empty string, got %v", reportLocation["filePath"])
					}
					if v, ok := reportLocation["lineNumber"].(float64); !ok || v <= 0 {
						t.Errorf("want reportLocation.lineNumber to be a positive number, got %v", reportLocation["lineNumber"])
					}
					if v, ok := reportLocation["functionName"].(string); !ok || v == "" {
						t.Errorf("want reportLocation.functionName to be a non-empty string, got %v", reportLocation["functionName"])
					}
				}
			}
		})
	}
}
