# errdef

[![Build Status](https://img.shields.io/github/actions/workflow/status/shiwano/errdef/test.yml?branch=main)](https://github.com/shiwano/errdef/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shiwano/errdef.svg)](https://pkg.go.dev/github.com/shiwano/errdef)
[![Go Report Card](https://goreportcard.com/badge/github.com/shiwano/errdef)](https://goreportcard.com/report/github.com/shiwano/errdef)

`errdef` splits error handling in Go into **Definitions** and **Error instances**, so you can keep errors typed, structured, and uniform.
It integrates cleanly with the standard ecosystem — `errors.Is` / `errors.As`, `fmt.Formatter`, `json.Marshaler`, and `slog.LogValuer` — while adding fields, stack traces, and flexible error composition.

> **Status:** The core API is stable, but minor breaking changes may occur before v1.0.0.

> **Requirements:** Go 1.25+

## Table of Contents

- [Getting Started](#getting-started)
- [Features](#features)
  - [Error Constructors](#error-constructors)
  - [Detailed Error Formatting](#detailed-error-formatting)
  - [JSON Marshaling](#json-marshaling)
  - [Structured Logging (`slog`)](#structured-logging-slog)
  - [Field Constructors](#field-constructors)
  - [Field Extractors](#field-extractors)
  - [Free-Form Details](#free-form-details)
  - [Context Integration](#context-integration)
  - [Redaction](#redaction)
  - [Joining Errors](#joining-errors)
  - [Panic Handling](#panic-handling)
  - [Error Resolution](#error-resolution)
  - [Error Deserialization](#error-deserialization)
  - [Ecosystem Integration](#ecosystem-integration)
  - [Built-in Options](#built-in-options)
- [Contributing](#contributing)
- [Appendix: Library Comparison](#appendix-library-comparison)
- [License](#license)

## Getting Started

```shell
go get github.com/shiwano/errdef
```

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/shiwano/errdef"
)

var (
    // Reusable error definition (sentinel-like, extensible).
    ErrNotFound = errdef.Define("not_found", errdef.HTTPStatus(404))

    // Type-safe field (constructor + extractor pair).
    UserID, UserIDFrom = errdef.DefineField[string]("user_id")
)

func findUser(ctx context.Context, id string) error {
    // Create an error; attach typed fields as needed.
    return ErrNotFound.With(ctx, UserID(id)).New("user not found")
}

func main() {
    err := findUser(context.TODO(), "u123")

    // Standard errors.Is still works.
    if errors.Is(err, ErrNotFound) {
        fmt.Println("user not found")
    }

    // Extract fields in a type-safe way.
    if userID, ok := UserIDFrom(err); ok {
        fmt.Println("user id:", userID)
    }
}
```

> **Note:** `errors.Is` compares the identity of a definition, not its kind string (e.g. `"not_found"`).

## Features

### Error Constructors

Choose how to construct an error depending on whether you create a new one or keep a cause.

- `New(msg)`: Create a new error.
- `Errorf(fmt, ...)`: Create with a formatted message.
- `Wrap(cause)`: Wrap and keep the cause (`errors.Is(err, cause)` stays true).
- `Wrapf(cause, fmt, ...)`: Wrap with a cause and a formatted message.
- `Join(causes...)`: Join multiple causes (`errors.Is(err, cause)` stays true).

```go
// New / Errorf
e1 := ErrNotFound.New("user not found")
e2 := ErrNotFound.Errorf("user %s not found", "u123")

// Wrap / Wrapf (keep the cause)
e3 := ErrNotFound.Wrap(sql.ErrNoRows)
e4 := ErrNotFound.Wrapf(sql.ErrNoRows, "lookup failed: %s", "u123")

errors.Is(e3, sql.ErrNoRows) // true
errors.Is(e4, sql.ErrNoRows) // true

// Join (keep multiple causes)
e5 := ErrNotFound.Join(sql.ErrNoRows, sql.ErrConnDone)

errors.Is(e5, sql.ErrNoRows)   // true
errors.Is(e5, sql.ErrConnDone) // true
```

#### Attaching Additional Options

- `With(ctx, ...opts)`: Requires ctx. Use when options need request-scoped data.
- `WithOptions(...opts)`: No ctx. Use for context-independent options.

```go
// Context-aware (requires ctx)
e1 := ErrNotFound.With(context.TODO(), UserID("u123")).New("user not found")

// Context-free (no ctx)
e2 := ErrNotFound.WithOptions(UserID("u123")).New("user not found")
```

### Detailed Error Formatting

Using the `%+v` format specifier will print the error message, kind, fields, stack trace, and any wrapped errors.

```go
err := findUser(ctx, "u-123")
fmt.Printf("%+v\n", err)
```

**Example Output:**

```
user not found
---
kind: not_found
fields:
  http_status: 404
  user_id: u-123
stack:
  main.findUser
    /path/to/your/project/main.go:23
  main.main
    /path/to/your/project/main.go:35
  runtime.main
    /usr/local/go/src/runtime/proc.go:250
causes: (1 error)
  [1] record not found
      ---
      stack:
        ...
```

### JSON Marshaling

`errdef.Error` implements `json.Marshaler` to produce structured JSON output.

**Example Output:**

```json
{
  "message": "user not found",
  "kind": "not_found",
  "fields": {
    "http_status": 404,
    "user_id": "u-123"
  },
  "stack": [
    { "function":"main.findUser","file":"/path/to/your/project/main.go","line":23 },
    { "function":"main.main","file":"/path/to/your/project/main.go","line":35 },
    { "function":"runtime.main","file":"/usr/local/go/src/runtime/proc.go","line":250 }
  ],
  "causes": [
    { "message": "record not found", "stack":[] }
  ]
}
```

> **Note:** If multiple fields have the same name, the last one in insertion order will be used in the JSON output.

### Structured Logging (`slog`)

`errdef.Error` implements `slog.LogValuer` out-of-the-box to provide structured logging with zero configuration.

```go
slog.Error("failed to find user", "error", err)
```

**Example Output:**

```json
{
  "level": "ERROR",
  "msg": "failed to find user",
  "error": {
    "message": "user not found",
    "kind": "not_found",
    "fields": {
      "http_status": 404,
      "user_id": "u-123"
    },
    "origin": {
      "file": "/path/to/your/project/main.go",
      "line": 23,
      "func": "main.findUser"
    },
    "causes": [
      "record not found"
    ]
  }
}
```

> **Note:** If multiple fields have the same name, the last one in insertion order will be used in the log output.

For more advanced control, you can:

- **Log only the structured fields**: The `Fields` type also implements `slog.LogValuer`.

  ```go
  fields := err.(errdef.Error).Fields()
  slog.Warn("...", "fields", fields)
  ```

- **Log the full stack trace**: The `Stack` type also implements `slog.LogValuer`.

  ```go
  stack := err.(errdef.Error).Stack()
  slog.Error("...", "stack", stack)
  ```

### Field Constructors

The field constructor can be chained with methods like `WithValue` or `WithValueFunc` to create new, simplified constructors.
This is useful for creating options with predefined or dynamically generated values.

```go
var (
    ErrorCodeAmountTooLarge = ErrorCode.WithValue(2002)

    errorUniqueID, _ = errdef.DefineField[string]("error_unique_id")
    ErrorUniqueID    = errorUniqueID.WithValueFunc(func() string {
        return generateRandomID()
    })
)

err := ErrPaymentFailed.With(
    ErrorCodeAmountTooLarge(),
    ErrorUniqueID(),
).New("amount too large")
```

### Field Extractors

The field extractor provides several helper methods for retrieving values from an error instance, especially for handling cases where a field might not exist.

```go
errWithCode := ErrPaymentFailed.New("payment failed")
errWithoutCode := ErrNotFound.New("not found")

code, ok := ErrorCodeFrom(errWithCode)
// code: 2001, ok: true

defaultCode := ErrorCodeFrom.OrDefault(errWithoutCode, 9999)
// defaultCode: 9999

fallbackCode := ErrorCodeFrom.OrFallback(errWithoutCode, func(err error) int {
    return 10000
})
// fallbackCode: 10000

codeWithDefault := ErrorCodeFrom.WithDefault(9999)
// codeWithDefault(errWithCode) -> 2001
// codeWithDefault(errWithoutCode) -> 9999
```

#### Extractor Search Policy

Extractors follow the same rules as `errors.As`.
They search the error chain and extract the value from the first matching `errdef.Error`, then stop searching.
If you need inner fields at the outer layer, prefer explicitly copying the needed fields when wrapping.

### Free-Form Details

You can attach free-form diagnostic details to an error under the `"details"` field.

```go
err := ErrNotFound.With(
  errdef.Details{"tenant_id": 1, "user_ids": []int{1,2,4}},
).Wrap(err)

details := errdef.DetailsFrom.OrZero(err)
// details: errdef.Details{
//   "tenant_id": 1,
//   "user_ids": []int{1,2,4},
// }
```

> **Note:** `Details` is a `map[string]any` type, allowing you to attach arbitrary key-value pairs.

### Context Integration

You can use `context.Context` to automatically attach request-scoped information to your errors.

```go
func tracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Attach the TraceID option to the context.
        ctx := errdef.ContextWithOptions(
            r.Context(),
            errdef.TraceID(r.Header.Get("X-Request-ID")),
        )
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

var ErrRateLimited = errdef.Define("rate_limited", errdef.HTTPStatus(429))

func someHandler(ctx context.Context) error {
    // The TraceID option is automatically attached from the context.
    return ErrRateLimited.With(ctx).New("too many requests")
}
```

### Redaction

Wrap secrets (tokens, emails, IDs, etc.) with `Redacted[T]` to ensure they always render as `"[REDACTED]"` in logs and serialized output (`fmt`, `json`, `slog`).
The original value remains accessible via `.Value()` for internal use.

```go
var UserEmail, UserEmailFrom = errdef.DefineField[errdef.Redacted[string]]("user_email")

err := ErrInvalidArgument.With(
  UserEmail(errdef.Redact("alice@example.com")),
).Wrap(err)

// fmt.Printf("%+v\n", err): user_email: [REDACTED]
// log/slog prints         : user_email="[REDACTED]"
// json.Marshal(err)       : { "user_email": "[REDACTED]" }
// internal access         : email, _ := UserEmailFrom(err); _ = email.Value()
```

### Joining Errors

You can join multiple errors into one using the `Join` method on a `Definition`.

```go
var (
  ErrLeft  = errdef.Define("left")
  ErrRight = errdef.Define("right")
  ErrTop   = errdef.Define("top")
)

l := ErrLeft.New("L")
r := ErrRight.New("R")
err := ErrTop.Join(l, r)
causes := err.(errdef.Error).Unwrap()
// causes: [l, r]
```

### Panic Handling

`errdef` provides a convenient way to convert panics into structured errors, ensuring that even unexpected failures are handled consistently.

```go
var ErrPanic = errdef.Define("panic", errdef.HTTPStatus(500))

func processRequest(w http.ResponseWriter, r *http.Request) (err error) {
    defer func() {
        if panicVal, captured := ErrPanic.CapturePanic(&err, recover()); captured {
           slog.Warn("a panic was captured", "panic_value", panicVal)
           // ...
        }
    }()
    maybePanic()
    return nil
}

if err := processRequest(w, r); err != nil {
    var pe errdef.PanicError
    if errors.As(err, &pe) {
        slog.Error("a panic occurred", "panic_value", pe.PanicValue())
    }
    // ...
}
```

### Error Resolution

For advanced use cases like mapping error codes from external APIs, use a `Resolver`.

```go
import (
    "github.com/shiwano/errdef"
    "github.com/shiwano/errdef/resolver"
)

var (
    ErrStripeCardDeclined = errdef.Define("card_declined", errdef.HTTPStatus(400))
    ErrStripeRateLimit    = errdef.Define("rate_limit", errdef.HTTPStatus(429))
    ErrStripeUnknown      = errdef.Define("stripe_unknown", errdef.HTTPStatus(500))

    // Order defines priority (first-wins).
    ErrStripe = resolver.New(
        ErrStripeCardDeclined,
        ErrStripeRateLimit,
    ).WithFallback(ErrStripeUnknown) // Remove if you want strict matching.
)

func handleStripeError(code, msg string) error {
    return ErrStripe.ResolveKind(errdef.Kind(code)).New(msg)
}

func handleStripeHTTPError(statusCode int, msg string) error {
    return ErrStripe.ResolveField(errdef.HTTPStatus.Key(), statusCode).New(msg)
}
```

> **Note:** If multiple definitions have the same Kind or field value, the first one in the resolver's definition order will be used.

### Error Deserialization

The `errdef/unmarshaler` package allows you to deserialize `errdef.Error` instances from JSON or other formats.
Use a `Resolver` to map kind strings to error definitions, and the unmarshaler will restore typed errors with their fields and stack traces.

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"

    "github.com/shiwano/errdef"
    "github.com/shiwano/errdef/resolver"
    "github.com/shiwano/errdef/unmarshaler"
)

var (
    ErrNotFound        = errdef.Define("not_found")
    UserID, UserIDFrom = errdef.DefineField[string]("user_id")
)

func main() {
    // Serialize an errdef.Error to JSON
    original := ErrNotFound.WithOptions(UserID("u123")).Wrapf(io.EOF, "user not found")
    data, _ := json.Marshal(original)

    // Deserialize JSON back into an errdef.Error
    r := resolver.New(ErrNotFound)
    u := unmarshaler.NewJSON(r, unmarshaler.WithStandardSentinelErrors())
    restored, _ := u.Unmarshal(data)

    fmt.Println(restored.Kind())             // "not_found"
    fmt.Println(restored.Error())            // "user not found"
    fmt.Println(UserIDFrom.OrZero(restored)) // "u123"
    fmt.Println(errors.Is(restored, io.EOF)) // true
}
```

### Ecosystem Integration

`errdef` is designed to work seamlessly with the broader Go ecosystem.

- **Structured Logging:** Implements `slog.LogValuer` for rich, structured logs out-of-the-box.
- **Error Reporting Services:**
  - **Sentry:** Compatible with the Sentry Go SDK by implementing the `stackTracer` interface.
  - **Google Cloud Error Reporting**: Integrates directly with the service by implementing the `DebugStacker` interface.
- **Legacy Error Handling:** Supports interoperability with `pkg/errors` by implementing the `causer` interface.

### Built-in Options

| Option                      | Description                                              | Extractor        |
|:----------------------------|:---------------------------------------------------------|:-----------------|
| `HTTPStatus(int)`           | Attaches an HTTP status code.                            | `HTTPStatusFrom` |
| `LogLevel(slog.Level)`      | Attaches a log level of type `slog.Level`.               | `LogLevelFrom`   |
| `TraceID(string)`           | Attaches a trace or request ID.                          | `TraceIDFrom`    |
| `Domain(string)`            | Labels the error with a service or subsystem name.       | `DomainFrom`     |
| `UserHint(string)`          | Provides a safe, user-facing hint message.               | `UserHintFrom`   |
| `Public()`                  | Marks the error as safe to expose externally.            | `IsPublic`       |
| `Retryable()`               | Marks the operation as retryable.                        | `IsRetryable`    |
| `RetryAfter(time.Duration)` | Recommends a delay to wait before retrying.              | `RetryAfterFrom` |
| `Unreportable()`            | Prevents the error from being sent to error tracking.    | `IsUnreportable` |
| `ExitCode(int)`             | Sets the exit code for a CLI application.                | `ExitCodeFrom`   |
| `HelpURL(string)`           | Provides a URL for documentation or help guides.         | `HelpURLFrom`    |
| `Details{}`                 | Attaches free-form diagnostic details to an error.       | `DetailsFrom`    |
| `NoTrace()`                 | Disables stack trace collection for the error.           | -                |
| `StackSkip(int)`            | Skips a specified number of frames during stack capture. | -                |
| `StackDepth(int)`           | Sets the depth of the stack capture (default: 32).       | -                |
| `Boundary()`                | Stops the error unwrapping chain at this point.          | -                |
| `Formatter(f)`              | Overrides the default `fmt.Formatter` behavior.          | -                |
| `JSONMarshaler(f)`          | Overrides the default `json.Marshaler` behavior.         | -                |
| `LogValuer(f)`              | Overrides the default `slog.LogValuer` behavior.         | -                |

#### Performance Knobs

- For hot paths where stack capture isn't necessary: Use `NoTrace()`
- To limit the number of frames captured in deep call stacks: Use `StackDepth(int)`
- To prevent deep error chains during error handling: Use `Boundary()`

## Appendix: Library Comparison

> **Last updated:** 2025-10-07
>
> If you spot inaccuracies or want another library included, please open an issue or PR.

### Comparison Table

| Feature                         | Go stdlib | pkg/errors | cockroach<br>db/errors | eris         | errorx | merry v2       | **errdef**          |
|---------------------------------|:---------:|:----------:|:----------------------:|:------------:|:------:|:--------------:|:-------------------:|
| `errors.Is`/`As` Compatibility  |    ✅     |     ✅     |           ✅           |      ✅      |   ✅   |       ✅       |       **✅**        |
| Def/Instance Separation         |    ❌     |     ❌     |           ❌           |      ❌      |   ❌   |       ❌       |       **✅**        |
| Automatic Stack Traces          |    ❌     |     ✅     |           ✅           |      ✅      |   ✅   |       ✅       |       **✅**        |
| Stack Control (Disable/Depth)   |    ❌     |     ❌     |           ⚠️           |      ❌      |   ❌   |       ✅       |       **✅**        |
| Structured Data                 |    ❌     |     ❌     |           ⚠️           |      ❌      |   ⚠️   |       ⚠️       | **✅ (Type-Safe)**  |
| Redaction                       |    ❌     |     ❌     |           ✅           |      ❌      |   ❌   |       ❌       |       **✅**        |
| Structured JSON                 |    ❌     |     ❌     |       ⚠️ (Proto)       | ⚠️ (Logging) |   ❌   | ⚠️ (Formatted) |       **✅**        |
| `slog` Integration              |    ❌     |     ❌     |           ❌           |      ❌      |   ❌   |       ❌       |       **✅**        |
| Panic Capture                   |    ❌     |     ❌     |           ✅           |      ❌      |   ❌   |       ❌       |       **✅**        |
| Multiple Causes (`errors.Join`) |    ✅     |     ❌     |           ✅           |      ✅      |   ✅   |       ✅       |       **✅**        |
| JSON Deserialization            |    ❌     |     ❌     |           ⚠️           |      ❌      |   ❌   |       ❌       |       **✅**        |
| Protobuf Deserialization        |    ❌     |     ❌     |           ✅           |      ❌      |   ❌   |       ❌       | **⚠️ (Extensible)** |

### Quick Notes

* **`pkg/errors`:** A historical library that added stack trace functionality, but is now archived.
* **`cockroachdb/errors`:** Specializes in distributed systems and has a very powerful Protobuf serialization/deserialization feature.
* **`eris`:** Provides good stack trace formatting but lacks structured field attachment feature.
* **`errorx` / `merry v2`:** Although not type-safe, they provide a feature to attach information to errors in a simple key-value format.
* **`errdef`:** Features a design that separates Definitions from Instances, enabling type-safe fields, native slog integration, and full JSON round-trip capabilities.

## Contributing

Contributions are welcome! Feel free to send issues or pull requests.

## License

This project is licensed under the [MIT](LICENSE) License.
