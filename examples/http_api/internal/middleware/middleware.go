package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/examples/http_api/internal/errdefs"
)

// RequestIDKey is the context key for storing request IDs.
type RequestIDKey struct{}

// Tracing adds a unique request ID to the context for distributed tracing.
// The request ID is extracted from the X-Request-ID header, or generated if not present.
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey{}, requestID)

		// Attach the request ID as an errdef option to the context
		ctx = errdef.ContextWithOptions(ctx, errdef.TraceID(requestID))

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging logs HTTP requests and responses using structured logging.
// It logs the request method, path, status code, duration, and any errors.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		requestID := ""
		if id, ok := r.Context().Value(RequestIDKey{}).(string); ok {
			requestID = id
		}

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID,
		)
	})
}

// Recovery recovers from panics and converts them to structured errors.
// This ensures that even unexpected failures are handled consistently.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := errdefs.ErrInternal.Recover(func() error {
			next.ServeHTTP(w, r)
			return nil
		})

		if err != nil {
			slog.Error("panic recovered", "error", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"an internal error occurred"}`))
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a simple request ID.
// In production, use a more robust ID generator (e.g., UUID, ULID).
func generateRequestID() string {
	return "req-" + time.Now().Format("20060102150405.000000")
}
