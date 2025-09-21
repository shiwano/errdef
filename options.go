package errdef

import (
	"log/slog"
	"time"
)

var (
	public, publicFrom             = DefineField[bool]("public")
	retryable, retryableFrom       = DefineField[bool]("retryable")
	unreportable, unreportableFrom = DefineField[bool]("unreportable")

	// HTTPStatus attaches an HTTP status code.
	HTTPStatus, HTTPStatusFrom = DefineField[int]("http_status")

	// LogLevel attaches a log level of type `slog.Level`.
	LogLevel, LogLevelFrom = DefineField[slog.Level]("log_level")

	// TraceID attaches a trace or request ID.
	TraceID, TraceIDFrom = DefineField[string]("trace_id")

	// Domain labels the error with a service or subsystem name.
	Domain, DomainFrom = DefineField[string]("domain")

	// Provides a safe, user-facing hint message.
	UserHint, UserHintFrom = DefineField[string]("user_hint")

	// Public marks the error as safe to expose externally.
	Public, IsPublic = public.WithValue(true), publicFrom.WithZero()

	// Retryable marks the operation as retryable.
	Retryable, IsRetryable = retryable.WithValue(true), retryableFrom.WithZero()

	// RetryAfter recommends a delay to wait before retrying.
	RetryAfter, RetryAfterFrom = DefineField[time.Duration]("retry_after")

	// Unreportable prevents the error from being sent to error tracking.
	Unreportable, IsUnreportable = unreportable.WithValue(true), unreportableFrom.WithZero()

	// ExitCode sets the exit code for a CLI application.
	ExitCode, ExitCodeFrom = DefineField[int]("exit_code")

	// HelpURL provides a URL for documentation or help guides.
	HelpURL, HelpURLFrom = DefineField[string]("help_url")
)

// NoTrace disables stack trace collection for the error.
func NoTrace() Option {
	return &noTrace{}
}

// StackSkip skips a specified number of frames during stack capture.
func StackSkip(skip int) Option {
	return &stackSkip{skip: skip}
}

// StackDepth limits the depth of the stack trace capture.
func StackDepth(depth int) Option {
	return &stackDepth{depth: depth}
}

// Boundary stops the error unwrapping chain at this point.
func Boundary() Option {
	return &boundary{}
}

// Formatter overrides the default `fmt.Formatter` behavior.
func Formatter(f ErrorFormatter) Option {
	return &formatter{formatter: f}
}

// JSONMarshaler overrides the default `json.Marshaler` behavior.
func JSONMarshaler(f ErrorJSONMarshaler) Option {
	return &jsonMarshaler{marshaler: f}
}
