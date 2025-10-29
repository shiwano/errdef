# Google Cloud Error Reporting Integration Example

This example demonstrates how to format `errdef` errors for Google Cloud Error Reporting using Go's standard `log/slog` package. When running on Google Cloud (Cloud Run, Cloud Functions, GKE, Compute Engine), errors logged in this format are automatically recognized and grouped by Error Reporting.

For more information about error formatting requirements, see the [official documentation](https://cloud.google.com/error-reporting/docs/formatting-error-messages).

## Basic Example

```go
package main

import (
  "log/slog"
  "os"

  "github.com/shiwano/errdef"
  gcerr "github.com/shiwano/errdef/examples/gcloud_error_reporting"
)

var (
  ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
  UserID, _   = errdef.DefineField[string]("user_id")
)

func main() {
  // Setup JSON logger for Google Cloud
  // In production, this will be automatically picked up by Cloud Logging
  logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
  }))

  err := ErrNotFound.WithOptions(
    UserID("u123"),
  ).New("user not found")

  // Log the error - Error Reporting will automatically detect it
  logger.Error("failed to find user", gcerr.Error(err))
}
```

**Output:**

```json
{
  "time": "2024-01-01T00:00:00Z",
  "level": "ERROR",
  "msg": "failed to find user",
  "stack_trace": "user not found\n\ngoroutine 1 [running]:\nmain.main()\n\t/path/to/main.go:24 +0x...\nruntime.main()\n\t/usr/local/go/src/runtime/proc.go:250 +0x...",
  "context": {
    "reportLocation": {
      "filePath": "/path/to/main.go",
      "lineNumber": 24,
      "functionName": "main.main"
    },
    "httpRequest": {
      "responseStatusCode": 404
    }
  },
  "error": {
    "message": "user not found",
    "kind": "not_found",
    "fields": {
      "http_status": 404,
      "user_id": "u123"
    }
  }
}
```

## How It Works

Google Cloud Error Reporting automatically detects errors in log entries when they contain at least one of:

1. **`message` field**: The error message
2. **`stack_trace` field**: Stack trace in string format (highest priority)
3. **`exception` field**: Exception information

This implementation provides **`stack_trace`**, which takes the highest priority for error detection.

The `gcerr.Error()` function formats `errdef` errors with:
- **`error.message` field**: The error message
- **`error.kind` field**: Error kind for classification (if present)
- **`error.fields` field**: Custom fields excluding gcerr-specific fields (if present)
- **`error.causes` field**: Array of cause error messages (if present)
- **`stack_trace` field**: Stack trace in Google Cloud format
- **`context.reportLocation` field**: Error origin location (filePath, lineNumber, functionName)
- **`context.httpRequest` field**: HTTP request context (if `gcerr.HTTPRequest()` is used)
- **`context.user` field**: User identifier (if `gcerr.User()` is used)

The `reportLocation` helps Error Reporting to display the exact location where the error occurred in the UI and enables proper error grouping. Note that gcerr-specific fields like `gcerr.http_request` and `gcerr.user` are excluded from `error.fields` as they are already included in the `context` section.

## HTTP Request, User Context, and Wrapped Errors

This example shows how to use HTTP request information, user context, and error wrapping together:

```go
package main

import (
  "database/sql"
  "log/slog"
  "net/http"
  "os"

  "github.com/shiwano/errdef"
  gcerr "github.com/shiwano/errdef/examples/gcloud_error_reporting"
)

var (
  ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))
  UserID, _   = errdef.DefineField[string]("user_id")
)

func main() {
  logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

  http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Path[len("/users/"):]

    // Simulate database query
    dbErr := sql.ErrNoRows

    // Wrap the database error with HTTP context
    err := ErrNotFound.WithOptions(
      UserID(userID),
      gcerr.HTTPRequest(r),
      gcerr.User(userID),
    ).Wrap(dbErr)

    logger.Error("request failed", gcerr.Error(err))
    http.Error(w, "Not Found", http.StatusNotFound)
  })
}
```

**Output:**

```json
{
  "time": "2024-01-01T00:00:00Z",
  "level": "ERROR",
  "msg": "request failed",
  "stack_trace": "sql: no rows in result set\n\ngoroutine 1 [running]:\nmain.main.func1()\n\t/path/to/main.go:115 +0x...\nnet/http.HandlerFunc.ServeHTTP()\n\t/usr/local/go/src/net/http/server.go:2136 +0x...",
  "context": {
    "reportLocation": {
      "filePath": "/path/to/main.go",
      "lineNumber": 115,
      "functionName": "main.main.func1"
    },
    "httpRequest": {
      "method": "GET",
      "url": "/users/u123",
      "userAgent": "Mozilla/5.0",
      "referrer": "https://example.com",
      "responseStatusCode": 404,
      "remoteIp": "192.168.1.1:12345"
    },
    "user": "u123"
  },
  "error": {
    "message": "sql: no rows in result set",
    "kind": "not_found",
    "fields": {
      "http_status": 404,
      "user_id": "u123"
    },
    "causes": ["sql: no rows in result set"]
  }
}
```

This comprehensive example demonstrates:
- **HTTP Request Context**: Method, URL, user agent, referrer, and remote IP are automatically extracted from `*http.Request` and placed in `context.httpRequest` (for Google Cloud Error Reporting)
- **User Context**: User identifier is placed in `context.user`
- **Error Wrapping**: The original database error (`sql.ErrNoRows`) is preserved in the `error.causes` field
- **Error Grouping**: Error Reporting groups errors by stack trace, endpoint, status code, and error kind
- **Field Filtering**: gcerr-specific fields (`gcerr.http_request`, `gcerr.user`) are automatically excluded from `error.fields` to avoid duplication with the `context` section

## Sensitive Data

Use `errdef.Redacted[T]` to ensure sensitive data is never exposed in logs:

```go
var Email, _ = errdef.DefineField[errdef.Redacted[string]]("email")

err := ErrNotFound.WithOptions(
  UserID("u123"),
  Email(errdef.Redact("user@example.com")),
).New("user not found")

logger.Error("operation failed", gcerr.Error(err))
// Email will appear as "[REDACTED]" in logs
```

## Running the Example

```bash
go test -v
```
