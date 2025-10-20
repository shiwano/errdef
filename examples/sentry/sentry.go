package sentry

import (
	"context"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/shiwano/errdef"
)

// Level sets the Sentry severity level for the error.
var Level, levelFrom = errdef.DefineField[sentry.Level]("sentry.level")

// CaptureError reports an error to Sentry with context from errdef data.
//
// This function:
//   - Returns false if the error is nil or unreportable (errdef.IsUnreportable)
//   - Retrieves the Sentry hub from the context
//   - Configures the Sentry scope with error metadata:
//   - Level (defaults to sentry.LevelError)
//   - Kind as a tag
//   - Domain as a tag
//   - HTTPStatus as a tag
//   - TraceID as a tag
//   - All custom fields as context data
//   - Captures the error exception
func CaptureError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	} else if errdef.IsUnreportable(err) {
		return false
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetLevel(levelFrom.OrDefault(err, sentry.LevelError))

		if kind, ok := errdef.KindFrom(err); ok {
			scope.SetTag("error.kind", string(kind))
		}

		if fields, ok := errdef.FieldsFrom(err); ok {
			data := make(map[string]any)
			for k, v := range fields.All() {
				data[k.String()] = v.Value()
			}
			scope.SetContext("error.fields", data)
		}

		if domain, ok := errdef.DomainFrom(err); ok {
			scope.SetTag("error.domain", domain)
		}

		if status, ok := errdef.HTTPStatusFrom(err); ok {
			scope.SetTag("http.status", strconv.Itoa(status))
		}

		if traceID, ok := errdef.TraceIDFrom(err); ok {
			scope.SetTag("trace.id", traceID)
		}
	})

	hub.CaptureException(err)
	return true
}
