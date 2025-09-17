# errdef

[![Go Report Card](https://goreportcard.com/badge/github.com/shiwano/errdef)](https://goreportcard.com/report/github.com/shiwano/errdef)
[![Go Reference](https://pkg.go.dev/badge/github.com/shiwano/errdef.svg)](https://pkg.go.dev/github.com/shiwano/errdef)
[![Build Status](https://img.shields.io/github/actions/workflow/status/shiwano/errdef/test.yml?branch=main)](https://github.com/shiwano/errdef/actions)

`errdef` is a Go library for more structured, type-safe, and flexible error handling.

By clearly separating an error's **definition** from its runtime **instance**, `errdef` enables consistent error handling throughout your application.
It allows you to attach rich metadata to errors, simplifying debugging, logging, and API response generation.

## Features

- **✨ Intuitive & Reusable Definitions**: Separates an error's **definition** from its **instance** to enable consistent error handling across your application.
- **📦 Type-Safe Metadata**: Leverage generics to **type-safely** attach extra data like HTTP statuses, preventing runtime mistakes with IDE autocompletion.
- **🔧 Flexible Customization**: Easily add dynamic data, such as a trace ID, to errors using the option pattern and `context` integration.
- **🤝 Seamless Go Integration**: Fully compatible with the standard **`errors.Is` / `As`**. Implements `fmt.Formatter` (`%+v`) and `json.Marshaler` to simplify debugging and structured logging.
- **🚀 Rich, Built-in Options**: Comes with many practical, ready-to-use options like `HTTPStatus`, `LogLevel`, `Retryable`, and more.

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
    ErrNotFound = errdef.Define("not_found")

    ErrInvalidArgument = errdef.Define("invalid_argument",
        errdef.HTTPStatus(400),
        errdef.UserHint("Please check your input."),
    )
)
```

### 2. Create Error Instances

Next, create error instances from a `Definition` using methods like `New` or `Wrapf`.

```go
func findUser(ctx context.Context, id string) (*User, error) {
    user, err := db.Find(ctx, id)
    if err != nil {
        return nil, ErrNotFound.Wrapf(err, "user %s not found", id)
    }
    return user, nil
}

func updateUser(ctx context.Context, userID string, params UpdateParams) error {
    if err := params.Validate(); err != nil {
        return ErrInvalidArgument.With(ctx,
            errdef.Domain("user_service"), // Attach additional metadata
        ).New("validation failed")
    }
    // ...
}
```

### 3. Check and Use Errors

The created errors can be checked using the standard `errors.Is`.
You can also safely extract attached metadata using extractor functions (e.g., `HTTPStatusFrom`).

```go
func handler(w http.ResponseWriter, r *http.Request) {
    _, err := findUser(r.Context(), "user-123")
    if err != nil {
        if errors.Is(err, ErrNotFound) { // Or ErrNotFound.Is(err)
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }

        if status, ok := errdef.HTTPStatusFrom(err); ok {
            slog.Error("error with http status", "status", status, "err", err)
            http.Error(w, err.Error(), status)
            return
        }

        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
```

### 4. Detailed Error Formatting

Using the `%+v` format specifier will print the error message, fields, stack trace, and any wrapped errors.

```go
err := findUser(ctx, "user-123")
fmt.Printf("%+v\n", err)
```

**Example Output:**

```
user user-123 not found: record not found
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
        // Add the TraceID option to the context
        ctx := errdef.ContextWithOptions(r.Context(), errdef.TraceID(reqID))
        next.ServeHTTP(w, r.With(ctx))
    })
}

var ErrRateLimited = errdef.Define("rate_limited", errdef.HTTPStatus(429))

func someHandler(ctx context.Context) error {
    return ErrRateLimited.With(ctx).New("too many requests")
}
```

### Standard Options

| Option                 | Description                                                         | Extractor        |
|:-----------------------|:--------------------------------------------------------------------|:-----------------|
| `HTTPStatus(int)`      | Sets the HTTP status code.                                          | `HTTPStatusFrom` |
| `LogLevel(slog.Level)` | Sets the log level of type `slog.Level`.                            | `LogLevelFrom`   |
| `TraceID(string)`      | Sets a trace ID or request ID.                                      | `TraceIDFrom`    |
| `Domain(string)`       | Sets the service domain or subsystem name where the error occurred. | `DomainFrom`     |
| `UserHint(string)`     | Sets a hint message to be displayed to the user.                    | `UserHintFrom`   |
| `Public()`             | Marks the error as safe for external exposure (sets `true`).        | `IsPublic`       |
| `Retryable()`          | Marks the operation as retryable (sets `true`).                     | `IsRetryable`    |
| `RetryAfter(d)`        | Sets the duration (`time.Duration`) to wait before retrying.        | `RetryAfterFrom` |
| `ExitCode(int)`        | Sets the exit code for a CLI application.                           | `ExitCodeFrom`   |
| `HelpURL(string)`      | Sets a URL to documentation or troubleshooting guides.              | `HelpURLFrom`    |
| `NoTrace()`            | Disables stack trace collection.                                    | -                |
| `StackSkip(int)`       | Adds to the number of frames to skip during stack trace collection. | -                |
| `Boundary()`           | Marks this error as the end of an error chain, stopping `Unwrap`.   | -                |
| `Formatter(f)`         | Overrides the `fmt.Formatter` behavior with a custom function.      | -                |
| `JSONMarshaler(f)`     | Overrides the `json.Marshaler` behavior with a custom function.     | -                |

## Contributing

Contributions are welcome! Feel free to send issues or pull requests.

## License

This project is licensed under the [MIT](LICENSE) License.
