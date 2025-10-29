package gcerr

import (
	"log/slog"
	"net/http"

	"github.com/shiwano/errdef"
)

// HTTPRequest represents the HTTP request context for Error Reporting.
type httpRequestData struct {
	Method    string
	URL       string
	UserAgent string
	Referrer  string
	RemoteIP  string
}

var (
	httpRequest, httpRequestFrom = errdef.DefineField[httpRequestData]("gcerr.http_request")

	// HTTPRequest is a field constructor for HTTP request context.
	// It accepts *http.Request and automatically extracts relevant information.
	// The response status code should be set using errdef.HTTPStatus separately.
	HTTPRequest = httpRequest.WithHTTPRequestFunc(func(req *http.Request) httpRequestData {
		return httpRequestData{
			Method:    req.Method,
			URL:       req.URL.String(),
			UserAgent: req.UserAgent(),
			Referrer:  req.Referer(),
			RemoteIP:  req.RemoteAddr,
		}
	})

	// User is a field constructor for user context.
	// Use this to attach user identifier to errors.
	User, userFrom = errdef.DefineField[string]("gcerr.user")
)

// Error wraps an errdef.Error for Google Cloud Error Reporting.
// It returns a slog.Attr that formats the error in a way that Error Reporting
// can automatically recognize and group errors.
//
// The error is formatted with the following fields:
//   - error.message: The error message
//   - error.kind: Error kind (if present)
//   - error.fields: Custom fields excluding gcerr.* fields (if present)
//   - error.causes: Array of cause error messages (if present)
//   - stack_trace: Stack trace in string format (if present)
//   - context.reportLocation: Error origin location (if stack trace is present)
//   - context.httpRequest: HTTP request context (if HTTPRequest is present)
//   - context.user: User identifier (if User is present)
//
// Google Cloud Error Reporting requires at least one of message, stack_trace,
// or exception fields. This implementation provides stack_trace which takes
// the highest priority for error detection.
//
// See https://cloud.google.com/error-reporting/docs/formatting-error-messages
//
// When used with slog.JSONHandler, the output will be automatically
// recognized by Google Cloud Error Reporting when running on:
//   - Cloud Run
//   - Cloud Functions
//   - GKE (Google Kubernetes Engine)
//   - Compute Engine with Cloud Logging agent
func Error(err error) slog.Attr {
	e, ok := err.(errdef.Error)
	if !ok {
		return slog.Group("",
			slog.String("@type", "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"),
			slog.String("message", err.Error()),
		)
	}

	// Build error group manually to control its structure
	errorAttrs := []any{
		slog.String("message", e.Error()),
	}

	if e.Kind() != "" {
		errorAttrs = append(errorAttrs, slog.String("kind", string(e.Kind())))
	}

	if e.Fields().Len() > 0 {
		filteredFields := filterGCloudFields(e.Fields())
		if len(filteredFields) > 0 {
			errorAttrs = append(errorAttrs, slog.Any("fields", filteredFields))
		}
	}

	causes := e.Unwrap()
	if len(causes) > 0 {
		causeMessages := make([]string, len(causes))
		for i, c := range causes {
			causeMessages[i] = c.Error()
		}
		errorAttrs = append(errorAttrs, slog.Any("causes", causeMessages))
	}

	attrs := []any{
		slog.Group("error", errorAttrs...),
	}

	if stackTrace, ok := buildStackTrace(err, e); ok {
		attrs = append(attrs, stackTrace)
	}

	if context, ok := buildContext(e); ok {
		attrs = append(attrs, context)
	}

	return slog.Group("", attrs...)
}

func buildStackTrace(err error, e errdef.Error) (slog.Attr, bool) {
	if e.Stack().Len() > 0 {
		if ds, ok := err.(errdef.DebugStacker); ok {
			return slog.String("stack_trace", ds.DebugStack()), true
		}
	}
	return slog.Attr{}, false
}

func buildContext(e errdef.Error) (slog.Attr, bool) {
	var attrs []any

	if reportLocation, ok := buildReportLocation(e); ok {
		attrs = append(attrs, reportLocation)
	}

	if httpRequest, ok := buildHTTPRequest(e); ok {
		attrs = append(attrs, httpRequest)
	}

	if user, ok := userFrom(e); ok {
		attrs = append(attrs, slog.String("user", user))
	}

	if len(attrs) > 0 {
		return slog.Group("context", attrs...), true
	}
	return slog.Attr{}, false
}

func filterGCloudFields(fields errdef.Fields) map[string]any {
	filtered := make(map[string]any)
	for k, v := range fields.All() {
		// Skip gcerr-specific fields as they're already in context
		if k.String() == "gcerr.http_request" || k.String() == "gcerr.user" {
			continue
		}
		filtered[k.String()] = v.Value()
	}
	return filtered
}

func buildReportLocation(e errdef.Error) (slog.Attr, bool) {
	if e.Stack().Len() > 0 {
		if frame, ok := e.Stack().HeadFrame(); ok {
			return slog.Group("reportLocation",
				slog.String("filePath", frame.File),
				slog.Int("lineNumber", frame.Line),
				slog.String("functionName", frame.Func),
			), true
		}
	}
	return slog.Attr{}, false
}

func buildHTTPRequest(e errdef.Error) (slog.Attr, bool) {
	var attrs []any

	if req, ok := httpRequestFrom(e); ok {
		attrs = append(attrs,
			slog.String("method", req.Method),
			slog.String("url", req.URL),
			slog.String("userAgent", req.UserAgent),
			slog.String("referrer", req.Referrer),
			slog.String("remoteIp", req.RemoteIP),
		)
	}

	if code, ok := errdef.HTTPStatusFrom(e); ok {
		attrs = append(attrs, slog.Int("responseStatusCode", code))
	}

	if len(attrs) > 0 {
		return slog.Group("httpRequest", attrs...), true
	}
	return slog.Attr{}, false
}
