/*
Package errdef is a Go library for more structured, type-safe, and flexible error handling.

errdef clearly separates an error's "definition" from its runtime "instance".
This enables consistent error handling throughout your application.
It allows you to attach rich metadata to errors, simplifying debugging, logging, and API response generation.

# Basic Usage

First, define the error types used in your application with `errdef.Define`.
This is typically done once at the package's global scope.

	package myapp

	import "github.com/shiwano/errdef"

	var (
		ErrNotFound = errdef.Define("not_found")

		ErrInvalidArgument = errdef.Define("invalid_argument",
			errdef.HTTPStatus(400),
			errdef.UserHint("Please check your input."),
		)
	)

Next, create error instances from a `Definition` using methods like `New` or `Wrapf`.

	func findUser(ctx context.Context, id string) (*User, error) {
		user, err := db.Find(ctx, id)
		if err != nil {
			return nil, ErrNotFound.Wrapf(err, "user %s not found", id)
		}
		return user, nil
	}

The created errors can be checked using the standard `errors.Is`.
You can also safely extract attached metadata using extractor functions like `HTTPStatusFrom`.

	func handler(w http.ResponseWriter, r *http.Request) {
		_, err := findUser(r.Context(), "user-123")
		if err != nil {
			// Check the error type with errors.Is
			if errors.Is(err, ErrNotFound) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			// Extract the HTTP status code from the error
			if status, ok := errdef.HTTPStatusFrom(err); ok {
				slog.Error("error with http status", "status", status, "err", err)
				http.Error(w, err.Error(), status)
				return
			}

			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}

# Custom Fields

You can easily define project-specific, type-safe fields using `errdef.DefineField`.

	package myapp

	import "github.com/shiwano/errdef"

	// Define a field named "error_code" of type int
	var ErrorCode, ErrorCodeFrom = errdef.DefineField[int]("error_code")

	// Use the custom field in an error definition
	var ErrPaymentFailed = errdef.Define("payment_failed",
		errdef.HTTPStatus(500),
		ErrorCode(2001), // Attach a custom error code
	)

# Context Integration

You can use `context.Context` to automatically attach request-scoped information,
such as a request ID, to your errors.

	func tracingMiddleware(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			// Add errdef options to the context
			ctx := errdef.ContextWithOptions(r.Context(), errdef.TraceID(reqID))
			next.ServeHTTP(w, r.With(ctx))
		})
	}

	func someHandler(ctx context.Context) error {
		return ErrRateLimited.With(ctx).New("too many requests")
	}
*/
package errdef
