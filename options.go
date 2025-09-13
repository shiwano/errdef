package errdef

import (
	"log/slog"
	"time"
)

var (
	// HTTPStatus sets the HTTP status code.
	HTTPStatus, HTTPStatusFrom = DefineField[int]("http_status")

	// LogLevel sets the log level of type slog.Level.
	LogLevel, LogLevelFrom = DefineField[slog.Level]("log_level")

	// TraceID sets a trace ID or request ID.
	TraceID, TraceIDFrom = DefineField[string]("trace_id")

	// Domain sets the service domain or subsystem name where the error occurred.
	Domain, DomainFrom = DefineField[string]("domain")

	// UserHint sets a hint message to be displayed to the user.
	UserHint, UserHintFrom = DefineField[string]("user_hint")

	// Public marks the error as safe for external exposure (sets true).
	public, publicFrom = DefineField[bool]("public")
	Public, IsPublic   = public.Default(true), publicFrom.SingleReturn()

	// Retryable marks the operation as retryable (sets true).
	retryable, retryableFrom = DefineField[bool]("retryable")
	Retryable, IsRetryable   = retryable.Default(true), retryableFrom.SingleReturn()

	// RetryAfter sets the duration (time.Duration) to wait before retrying.
	RetryAfter, RetryAfterFrom = DefineField[time.Duration]("retry_after")

	// ExitCode sets the exit code for a CLI application.
	ExitCode, ExitCodeFrom = DefineField[int]("exit_code")

	// HelpURL sets a URL to documentation or troubleshooting guides.
	HelpURL, HelpURLFrom = DefineField[string]("help_url")
)

// NoTrace disables stack trace collection.
func NoTrace() Option {
	return &noTrace{}
}

// StackSkip adds to the number of frames to skip during stack trace collection.
func StackSkip(skip int) Option {
	return &stackSkip{skip: skip}
}

// Boundary marks this error as the end of an error chain, stopping Unwrap.
func Boundary() Option {
	return boundary{}
}

// Formatter overrides the fmt.Formatter behavior with a custom function.
func Formatter(f ErrorFormatter) Option {
	return formatter{formatter: f}
}

// JSONMarshaler overrides the json.Marshaler behavior with a custom function.
func JSONMarshaler(f ErrorJSONMarshaler) Option {
	return jsonMarshaler{marshaler: f}
}
