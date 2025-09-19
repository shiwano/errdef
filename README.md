# errdef

[![Go Report Card](https://goreportcard.com/badge/github.com/shiwano/errdef)](https://goreportcard.com/report/github.com/shiwano/errdef)
[![Go Reference](https://pkg.go.dev/badge/github.com/shiwano/errdef.svg)](https://pkg.go.dev/github.com/shiwano/errdef)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shiwano/errdef/test.yml?branch=main)](https://github.com/shiwano/errdef/actions)

`errdef` is a Go library for more structured, type-safe, and flexible error handling.

By clearly separating an error's **definition** from its runtime **instance**, `errdef` enables consistent error handling across your application.
It allows you to attach rich metadata to errors, simplifying debugging, structured logging, and API response generation.

## Features

- **✨ Consistent by Design**: Achieve consistent error handling application-wide by separating error **definitions** from **instances**.
- **🔧 Structured Metadata**: Attach type-safe metadata as options or automatically from context. Generics ensure compile-time safety.
- **🤝 Works with Go Standard**: Integrates seamlessly with standard interfaces like `errors.Is`, `fmt.Formatter`, and `json.Marshaler`.
- **🚀 Rich, Built-in Options**: Provides a rich set of ready-to-use options for common use cases like web services and CLIs (e.g., `HTTPStatus`).

## Installation

```shell
go get github.com/shiwano/errdef
```

## Getting Started

### 1. Define Your Errors

First, define the error types used in your application with `errdef.Define`.
This is typically done once at the package's global scope.

```go
package myapp

import "github.com/shiwano/errdef"

var (
    ErrNotFound = errdef.Define("not_found",
        errdef.HTTPStatus(404),
    )

    ErrInvalidArgument = errdef.Define("invalid_argument",
        errdef.HTTPStatus(400),
        errdef.UserHint("Please check your input."),
    )
)
```

### 2. Create Error Instances

Next, create error instances from a `Definition` using methods like `New`, `Wrap` or `Wrapf`.

```go
func findUser(id int64) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, ErrNotFound.Wrapf(err, "user %d not found", id)
    }
    return user, nil
}

func updateUser(userID string, params UpdateParams) error {
    if err := params.Validate(); err != nil {
        return ErrInvalidArgument.New("validation failed")
    }
    // ...
}
```

### 3. Check and Use Errors

The created errors can be checked using the standard `errors.Is`.
You can also safely extract attached metadata using extractor functions (e.g., `HTTPStatusFrom`).

```go
func handler(w http.ResponseWriter, r *http.Request) {
    _, err := findUser(123)
    if err != nil {
        if errors.Is(err, ErrNotFound) { // You can use ErrNotFound.Is(err) as well
            slog.Warn("User not found", "error", err)
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }

        slog.Error("Unhandled error", "error", fmt.Sprintf("%+v", err))

        status := errdef.HTTPStatusFrom.OrDefault(err, http.StatusInternalServerError)
        message := errdef.UserHintFrom.OrDefault(err, "An error occurred")
        http.Error(w, message, status)
    }
}
```

### 4. Detailed Error Formatting

Using the `%+v` format specifier will print the error message, kind, fields, stack trace, and any wrapped errors.

```go
err := findUser(ctx, 123)
fmt.Printf("%+v\n", err)
```

**Example Output:**

```
user 123 not found: record not found

Kind:
	not_found
Fields:
	http_status: 404
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
		database.Find
			/path/to/your/project/db.go:12
		...
```

## Advanced Usage

### Defining Custom Fields

You can easily define project-specific, type-safe fields using `errdef.DefineField`.

```go
package myapp

import "github.com/shiwano/errdef"

var ErrorCode, ErrorCodeFrom = errdef.DefineField[int]("error_code")

var ErrPaymentFailed = errdef.Define("payment_failed",
    errdef.HTTPStatus(500),
    ErrorCode(2001), // Attach a custom error code
)
```

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
    return ErrRateLimited.With(ctx).New("too many requests")
}
```

### Built-in Options

| Option                 | Description                                                                     | Extractor        |
|:-----------------------|:--------------------------------------------------------------------------------|:-----------------|
| `HTTPStatus(int)`      | Sets the HTTP status code.                                                      | `HTTPStatusFrom` |
| `LogLevel(slog.Level)` | Sets the log level of type `slog.Level`.                                        | `LogLevelFrom`   |
| `TraceID(string)`      | Sets a trace ID or request ID.                                                  | `TraceIDFrom`    |
| `Domain(string)`       | Sets the service domain or subsystem name where the error occurred.             | `DomainFrom`     |
| `UserHint(string)`     | Sets a hint message to be displayed to the user.                                | `UserHintFrom`   |
| `Public()`             | Marks the error as safe for external exposure (default `false`).                | `IsPublic`       |
| `Retryable()`          | Marks the operation as retryable (default `false`).                             | `IsRetryable`    |
| `RetryAfter(d)`        | Sets the duration (`time.Duration`) to wait before retrying.                    | `RetryAfterFrom` |
| `NotReportable()`      | Marks the error as not reportable to an error tracking system (default `true`). | `IsReportable`   |
| `ExitCode(int)`        | Sets the exit code for a CLI application.                                       | `ExitCodeFrom`   |
| `HelpURL(string)`      | Sets a URL to documentation or troubleshooting guides.                          | `HelpURLFrom`    |
| `NoTrace()`            | Disables stack trace collection.                                                | -                |
| `StackSkip(int)`       | Adds to the number of frames to skip during stack trace collection.             | -                |
| `Boundary()`           | Marks this error as the end of an error chain, stopping `errors.Unwrap`.        | -                |
| `Formatter(f)`         | Overrides the `fmt.Formatter` behavior with a custom function.                  | -                |
| `JSONMarshaler(f)`     | Overrides the `json.Marshaler` behavior with a custom function.                 | -                |

## Contributing

Contributions are welcome! Feel free to send issues or pull requests.

## License

This project is licensed under the [MIT](LICENSE) License.
