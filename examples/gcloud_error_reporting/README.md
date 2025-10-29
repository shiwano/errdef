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
  "stack_trace": "main.main\n  /path/to/main.go:24\nruntime.main\n  /usr/local/go/src/runtime/proc.go:250",
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
    },
    "origin": {
      "func": "main.main",
      "file": "/path/to/main.go",
      "line": 24
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
- **`error` field**: The structured error object from `errdef.Error` (includes message, kind, fields, origin, and causes)
- **`stack_trace` field**: Stack trace in Google Cloud format
- **`context.reportLocation` field**: Error origin location (filePath, lineNumber, functionName)
- **`context.httpRequest` field**: HTTP request context (if `gcerr.HTTPRequest()` is used)
- **`context.user` field**: User identifier (if `gcerr.User()` is used)

The `reportLocation` helps Error Reporting to display the exact location where the error occurred in the UI and enables proper error grouping.

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
  "stack_trace": "main.main.func1\n  /path/to/main.go:115\nnet/http.HandlerFunc.ServeHTTP\n  /usr/local/go/src/net/http/server.go:2136",
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
      "user_id": "u123",
      "gcerr.http_request": {
        "Method": "GET",
        "URL": "/users/u123",
        "UserAgent": "Mozilla/5.0",
        "Referrer": "https://example.com",
        "RemoteIP": "192.168.1.1:12345"
      },
      "gcerr.user": "u123"
    },
    "origin": {
      "func": "main.main.func1",
      "file": "/path/to/main.go",
      "line": 115
    },
    "causes": ["sql: no rows in result set"]
  }
}
```

This comprehensive example demonstrates:
- **HTTP Request Context**: Method, URL, user agent, referrer, and remote IP are automatically extracted from `*http.Request` and placed in both the `error.fields` (for structured access) and `context.httpRequest` (for Google Cloud Error Reporting)
- **User Context**: User identifier is placed in both `error.fields.gcerr.user` and `context.user`
- **Error Wrapping**: The original database error (`sql.ErrNoRows`) is preserved in the `error.causes` field
- **Error Grouping**: Error Reporting groups errors by stack trace, endpoint, status code, and error kind

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
