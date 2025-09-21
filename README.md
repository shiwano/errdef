# errdef

[![Go Report Card](https://goreportcard.com/badge/github.com/shiwano/errdef)](https://goreportcard.com/report/github.com/shiwano/errdef)
[![Go Reference](https://pkg.go.dev/badge/github.com/shiwano/errdef.svg)](https://pkg.go.dev/github.com/shiwano/errdef)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shiwano/errdef/test.yml?branch=main)](https://github.com/shiwano/errdef/actions)

`errdef` splits error handling in Go into **Definitions** and **Error instances**, so you can keep errors typed, structured, and uniform.
It integrates cleanly with the standard ecosystem — `errors.Is` / `errors.As`, `fmt.Formatter`, `json.Marshaler`, and `slog.LogValuer` — while adding fields, stack traces, and flexible error composition.

> **Status:** The core API is stable, but minor breaking changes may occur before v1.0.0.

> **Requirements:** Go 1.25+

## Features

- **✨ Consistent by Design**: Achieve consistent error handling application-wide by separating error definitions from instances.
- **🔧 Structured Metadata**: Attach type-safe metadata as options or automatically from context.
- **🤝 Works with Go Standard**: Integrates seamlessly with standard interfaces like `errors.Is` / `errors.As`, `fmt.Formatter`, `json.Marshaler`, and `slog.LogValuer`.
- **🚀 Rich, Built-in Options**: Provides a rich set of ready-to-use options for common use cases like web services and CLIs (e.g., `NoTrace`, `HTTPStatus`, and `LogLevel`).

## Installation

```shell
go get github.com/shiwano/errdef
```

## Getting Started

### 1. Define Your Errors

First, define the error kinds used in your application with `errdef.Define`.
You can also define fields to attach structured data to errors using `errdef.DefineField`.
This is typically done once at the package's global scope.

```go
package myapp

import "github.com/shiwano/errdef"

var (
    // Define error kinds
    ErrNotFound      = errdef.Define("not_found", errdef.HTTPStatus(http.StatusNotFound))
    ErrInvalidArgument = errdef.Define("invalid_argument", errdef.HTTPStatus(http.StatusBadRequest))

    // Define fields to attach to errors (constructor + extractor pair)
    UserID, UserIDFrom = errdef.DefineField[string]("user_id")
)
```

### 2. Create Error Instances

Next, create error instances from a `Definition` using methods like `New`, `Errorf`, `Wrap`, or `Wrapf`.
The `With` method allows you to attach information from a `context` or apply additional options.

```go
func findUser(ctx context.Context, userID string) (*User, error) {
    user, err := db.Find(ctx, userID)
    if err != nil {
        return nil, ErrNotFound.With(ctx, UserID(userID)).Wrapf(err, "user not found")
    }
    return user, nil
}
```

### 3. Check and Use Errors

The created errors can be checked using the standard `errors.Is`.
You can also safely extract field values using the extractor function (e.g., `UserIDFrom`) created by `DefineField`.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    _, err := findUser(r.Context(), "u-123")
    if err != nil {
        // You can use ErrNotFound.Is(err) as well
        if errors.Is(err, ErrNotFound) {
            userID := UserIDFrom.OrZero(err)
            slog.Warn("User not found", "user_id", userID)
            // ...
            return
        }
        // ...
    }
}
```

## Advanced Usage

### Detailed Error Formatting

Using the `%+v` format specifier will print the error message, kind, fields, stack trace, and any wrapped errors.

```go
err := findUser(ctx, "u-123")
fmt.Printf("%+v\n", err)
```

**Example Output:**

```
user not found

Kind:
	not_found
Fields:
	http_status: 404
	user_id: u-123
Stack:
	main.findUser
		/path/to/your/project/main.go:23
	main.main
		/path/to/your/project/main.go:35
	runtime.main
		/usr/local/go/src/runtime/proc.go:250
Causes:
	record not found

	Stack:
		...
```

### JSON Marshaling

`errdef.Error` implements `json.Marshaler` to produce structured JSON output.
The causes are structured for `errdef.Error`, and best-effort for others (string if needed).

**Example Output:**

```json
{
  "message": "user not found",
  "kind": "not_found",
  "fields": [
    { "key": "http_status",  "value": 404 },
    { "key": "user_id", "value": "u-123" }
  ],
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

### Structured Logging (`slog`)

`errdef.Error` implements `slog.LogValuer` out-of-the-box to provide structured logging with zero configuration.

When you pass an `errdef.Error` to `slog`, it automatically formats into a structured group containing the message, kind, custom fields, error origin, and causal chain.

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

For more advanced control, you can:

- **Log only the structured fields**: The `Fields` type also implements `slog.LogValuer`, allowing you to log just the custom fields from an error.

  ```go
  fields := err.(errdef.Error).Fields()
  slog.Warn("Request rejected due to invalid argument", "fields", fields)
  ```

  **Output:**

  ```json
  {
    "level": "WARN",
    "msg": "Request rejected due to invalid argument",
    "fields": {
      "http_status": 400,
      "invalid_param": "email"
    }
  }
  ```

- **Log the full stack trace**: The `Stack` type also implements `slog.LogValuer`.

  ```go
  stack := err.(errdef.Error).Stack()
  slog.Error("...", "stack", stack)
  ```

- **Completely override the format**: Use the `errdef.LogValuer(...)` option.

  ```go
  var customFormat = errdef.LogValuer(func(err errdef.Error) slog.Value { ... })
  var ErrCustom = errdef.Define("...", customFormat)
  slog.Error("...", "error", ErrCustom.New("error"))
  ```

### Simplified Field Constructors

The field constructor can be chained with methods like `WithValue` or `WithValueFunc` to create new, simplified constructors.
This is useful for creating options with predefined or dynamically generated values.

```go
var (
    ErrorCodeAmountTooLarge = ErrorCode.WithValue(2002)

    errorGroupID, _ = errdef.DefineField[string]("error_group_id")
    ErrorGroupID = errorGroupID.WithValueFunc(func() string {
        return generateRandomID()
    })
)

err := ErrPaymentFailed.With(
    ErrorCodeAmountTooLarge(),
    ErrorGroupID(),
).New("amount too large")
```

### Extracting Field Values

The field extractor provides several convenient, chainable methods for safely retrieving values from an error instance, especially for handling cases where a field might not exist.

```go
errWithCode := ErrPaymentFailed.New("payment failed")
errWithoutCode := errdef.New("another error")

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
They search the error chain and extract the value from the **first matching `errdef.Error`**, then stop searching.

### Context Integration

You can use `context.Context` to automatically attach request-scoped information to your errors.

```go
func tracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        reqID := r.Header.Get("X-Request-ID")
        // Attach the TraceID option to the context
        ctx := errdef.ContextWithOptions(r.Context(), errdef.TraceID(reqID))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

var ErrRateLimited = errdef.Define("rate_limited", errdef.HTTPStatus(429))

func someHandler(ctx context.Context) error {
    // With(ctx, …) applies context options first, then explicit options (last-write-wins)
    return ErrRateLimited.With(ctx).New("too many requests")
}
```

### Joining Errors

You can join multiple errors into one using `errdef.Join` or the `Join` method on a `Definition`.

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
        ErrPanic.CapturePanic(&err, recover())
    }()
    maybePanic()
    return nil
}

func main() {
    if err := processRequest(w, r); err != nil {
        var pe errdef.PanicError
        if errors.As(err, &pe) {
            slog.Error("recovered from panic", 
                "panic_value", pe.PanicValue(),
                "error", fmt.Sprintf("%+v", err),
            )
        }
        // ...
    }
}
```

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
| `NoTrace()`                 | Disables stack trace collection for the error.           | -                |
| `StackSkip(int)`            | Skips a specified number of frames during stack capture. | -                |
| `StackDepth(int)`           | Limits the depth of the stack trace capture.             | -                |
| `Boundary()`                | Stops the error unwrapping chain at this point.          | -                |
| `Formatter(f)`              | Overrides the default `fmt.Formatter` behavior.          | -                |
| `JSONMarshaler(f)`          | Overrides the default `json.Marshaler` behavior.         | -                |
| `LogValuer(f)`              | Overrides the default `slog.LogValuer` behavior.         | -                |

#### Performance Knobs

- For hot paths where stack capture isn't necessary: Use `NoTrace()`
- To limit the number of frames captured in deep call stacks: Use `StackDepth(int)`
- To prevent deep error chains during error handling: Use `Boundary()`

## Contributing

Contributions are welcome! Feel free to send issues or pull requests.

## License

This project is licensed under the [MIT](LICENSE) License.
