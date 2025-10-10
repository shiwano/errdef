package errdef

import (
	"fmt"
	"log/slog"
	"maps"
	"time"
)

var (
	detailsFieldKey = &fieldKey[Details]{name: "details"}

	public, publicFrom             = DefineField[bool]("public")
	retryable, retryableFrom       = DefineField[bool]("retryable")
	unreportable, unreportableFrom = DefineField[bool]("unreportable")
)

var (
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

	// DetailsFrom extracts diagnostic details from an error.
	DetailsFrom FieldExtractor[Details] = detailsFrom
)

// NoTrace disables stack trace collection for the error.
func NoTrace() Option {
	return &noTrace{}
}

// StackSkip skips a specified number of frames during stack capture.
func StackSkip(skip int) Option {
	return &stackSkip{skip: skip}
}

// StackDepth sets the depth of the stack trace capture (default: 32).
func StackDepth(depth int) Option {
	return &stackDepth{depth: depth}
}

// Formatter overrides the default `fmt.Formatter` behavior.
func Formatter(f func(err Error, s fmt.State, verb rune)) Option {
	return &formatter{formatter: f}
}

// JSONMarshaler overrides the default `json.Marshaler` behavior.
func JSONMarshaler(f func(err Error) ([]byte, error)) Option {
	return &jsonMarshaler{marshaler: f}
}

// LogValuer overrides the default `slog.LogValuer` behavior.
func LogValuer(f func(err Error) slog.Value) Option {
	return &logValuer{valuer: f}
}

// Details represents a map of diagnostic details that can be attached to an error.
type Details map[string]any

// Key returns the field key for Details.
func (d Details) Key() FieldKey {
	return detailsFieldKey
}

func (d Details) applyOption(def *Definition) {
	def.fields.set(detailsFieldKey, &fieldValue[Details]{value: maps.Clone(d)})
}

func detailsFrom(err error) (Details, bool) {
	return fieldValueFrom[Details](err, detailsFieldKey)
}
