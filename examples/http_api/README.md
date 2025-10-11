# HTTP API Error Handling Example

This example demonstrates practical HTTP API error handling using `errdef` in a real-world web application structure.

## Overview

This example showcases best practices for:

- **Error Definition Management**: Centralized error definitions in a dedicated package
- **Context Integration**: Request ID injection and propagation via `context.Context`
- **Type-Safe Fields**: Custom fields for structured error context (UserID, Email, etc.)
- **HTTP Status Codes**: Automatic status code mapping with `HTTPStatus`
- **Sensitive Data Protection**: Email redaction using `Redacted[T]`
- **Structured Logging**: Integration with `log/slog` for rich error logs
- **Error Propagation**: Best practices for error handling across layers (Repository → Service → Handler)
- **JSON Error Responses**: Converting errors to user-friendly JSON responses
- **Public vs Internal Errors**: Controlling what information is exposed to clients

## Project Structure

```
http_api/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, server setup, routing
├── internal/
│   ├── errdefs/
│   │   └── errdefs.go           # Error definitions and field definitions
│   ├── handler/
│   │   └── handler.go           # HTTP handler layer (request/response)
│   ├── service/
│   │   └── service.go           # Business logic layer
│   ├── repository/
│   │   └── repository.go        # Data access layer (in-memory mock)
│   └── middleware/
│       └── middleware.go        # HTTP middleware (tracing, logging, recovery)
├── go.mod
└── README.md
```

## Running the Example

```bash
cd examples/http_api
go run cmd/server/main.go
```

The server will start on `http://localhost:8080` and display example curl commands.

## Example Requests

### 1. Get User (Success)

```bash
curl http://localhost:8080/users/1
```

**Response:**

```json
{
  "ID": "1",
  "Name": "Alice",
  "Email": "alice@example.com"
}
```

### 2. Get Non-Existent User (Not Found)

```bash
curl http://localhost:8080/users/999
```

**Response:**

```json
{
  "error": "user not found",
  "kind": "not_found",
  "trace_id": "req-20240101120000.000000"
}
```

**Log Output:**

```json
{
  "level": "ERROR",
  "msg": "request failed",
  "error": {
    "message": "user not found",
    "kind": "not_found",
    "fields": {
      "http_status": 404,
      "trace_id": "req-20240101120000.000000",
      "user_id": "999"
    },
    "origin": {
      "file": "/path/to/repository.go",
      "line": 55,
      "func": "repository.(*inMemoryRepository).FindByID"
    }
  }
}
```

### 3. Create User with Invalid Email (Validation Error)

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"David","email":"invalid-email"}'
```

**Response:**

```json
{
  "error": "validation failed",
  "kind": "validation",
  "trace_id": "req-20240101120001.000000",
  "validation_errors": {
    "email": "email is invalid"
  }
}
```

### 4. Create User with Duplicate Email (Conflict)

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice2","email":"alice@example.com"}'
```

**Response:**

```json
{
  "error": "email already exists",
  "kind": "conflict",
  "trace_id": "req-20240101120002.000000"
}
```

**Log Output (note the redacted email):**

```json
{
  "level": "ERROR",
  "msg": "request failed",
  "error": {
    "message": "email already exists",
    "kind": "conflict",
    "fields": {
      "email": "[REDACTED]",
      "http_status": 409,
      "trace_id": "req-20240101120002.000000"
    }
  }
}
```

### 5. Update Another User's Data (Forbidden)

```bash
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 2" \
  -d '{"name":"Alice Hacked","email":"hacked@example.com"}'
```

**Response:**

```json
{
  "error": "an internal error occurred",
  "kind": "forbidden",
  "trace_id": "req-20240101120003.000000"
}
```

> **Note:** The error message is generic because `ErrForbidden` is not marked with `Public()`.

**Log Output:**

```json
{
  "level": "ERROR",
  "msg": "request failed",
  "error": {
    "message": "cannot update another user's data",
    "kind": "forbidden",
    "fields": {
      "details": {
        "target_user_id": "1"
      },
      "http_status": 403,
      "resource_type": "user",
      "trace_id": "req-20240101120003.000000",
      "user_id": "2"
    }
  }
}
```

## Best Practices

### 1. Organize Errors in a Dedicated Package

**Why?**
- Provides a single source of truth for all application errors
- Avoids circular dependencies between layers
- Allows sharing error definitions with external clients when needed

```go
import (
    "errors"
    "yourapp/errdefs"
)

if errors.Is(err, errdefs.ErrNotFound) {
    // Handle not found
}
```

> **Note:** Using a distinct package name like `errdefs` helps avoid naming conflicts with the standard `errors` package.

### 2. Use Context for Request-Scoped Data

Inject trace IDs and other request-scoped options into the context using middleware:

```go
func Tracing(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := generateRequestID()
        ctx := errdef.ContextWithOptions(r.Context(), errdef.TraceID(requestID))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Then errors created with `With(ctx, ...)` will automatically include the trace ID:

```go
return errdefs.ErrNotFound.With(ctx, errdefs.UserID(id)).New("user not found")
```

### 3. Redact Sensitive Information

Use `Redacted[T]` to ensure sensitive data is never exposed in logs or responses:

```go
errdefs.Email(errdef.Redact("alice@example.com"))
```

This will appear as `[REDACTED]` in all outputs (logs, JSON, fmt), but you can still access the original value internally:

```go
if email, ok := errdefs.EmailFrom(err); ok {
    originalValue := email.Value() // "alice@example.com"
}
```

### 4. Control Public Error Messages

Mark errors as public only when safe to expose to external clients:

```go
// Safe to show to users
ErrValidation = errdef.Define("validation", errdef.HTTPStatus(400), errdef.Public())

// Should be hidden from users
ErrForbidden = errdef.Define("forbidden", errdef.HTTPStatus(403))
```

In your handler:

```go
message := err.Error()
if !errdef.IsPublic(err) {
    message = "an internal error occurred"
}
```

### 5. Propagate Errors Across Layers

Each layer should add its own context while preserving the original error:

**Repository Layer:**

```go
return errdefs.ErrNotFound.With(ctx, errdefs.UserID(id)).New("user not found")
```

**Service Layer:**

```go
if errors.Is(err, errdefs.ErrNotFound) {
    return nil, err // Pass through
}
return nil, errdefs.ErrDatabase.With(ctx).Wrap(err) // Wrap with context
```

**Handler Layer:**

```go
h.writeError(w, r, err) // Convert to JSON response
```

### 6. Use Structured Logging

`errdef` integrates seamlessly with `log/slog`:

```go
slog.Error("request failed", "error", err)
```

This automatically logs all error fields in a structured format.
