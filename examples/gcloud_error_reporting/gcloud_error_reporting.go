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
//   - message: The error message
//   - stack_trace: Stack trace in string format (if present)
//   - context.reportLocation: Error origin location (if stack trace is present)
//   - context.httpRequest: HTTP request context (if HTTPRequestField is present)
//   - context.user: User identifier (if UserField is present)
//   - kind: Error kind (if present)
//   - fields: Custom fields (if present, excluding gcer_http_request and gcer_user)
//   - causes: Array of cause error messages (if present)
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

	attrs := []any{
		slog.String("message", e.Error()),
	}

	if stackTrace, ok := buildStackTrace(err, e); ok {
		attrs = append(attrs, stackTrace)
	}

	if context, ok := buildContext(e); ok {
		attrs = append(attrs, context)
	}

	if e.Kind() != "" {
		attrs = append(attrs, slog.String("kind", string(e.Kind())))
	}

	if fields, ok := buildFields(e); ok {
		attrs = append(attrs, fields)
	}

	if causes, ok := buildCauses(e); ok {
		attrs = append(attrs, causes)
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

func buildFields(e errdef.Error) (slog.Attr, bool) {
	if e.Fields().Len() == 0 {
		return slog.Attr{}, false
	}

	attrs := make([]any, 0, e.Fields().Len())
	for k, v := range e.Fields().All() {
		// Skip gcer-specific fields as they're already in context
		if k.String() == "gcerr.http_request" || k.String() == "gcerr.user" {
			continue
		}
		attrs = append(attrs, slog.Any(k.String(), v.Value()))
	}

	if len(attrs) > 0 {
		return slog.Group("fields", attrs...), true
	}
	return slog.Attr{}, false
}

func buildCauses(e errdef.Error) (slog.Attr, bool) {
	causes := e.Unwrap()
	if len(causes) == 0 {
		return slog.Attr{}, false
	}

	causeMessages := make([]string, len(causes))
	for i, cause := range causes {
		causeMessages[i] = cause.Error()
	}
	return slog.Any("causes", causeMessages), true
}
