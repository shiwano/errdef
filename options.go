package errdef

import (
	"log/slog"
	"time"
)

// Detail represents a single key-value pair of diagnostic information.
type Detail struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

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

	// DetailsFrom extracts the free-form diagnostic details from an error chain.
	// See Details function for how the field is attached.
	detailsField, DetailsFrom = DefineField[[]Detail]("details")
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

// LogValuer overrides the default `slog.LogValuer` behavior.
func LogValuer(f ErrorLogValuer) Option {
	return &logValuer{valuer: f}
}

// Details attaches free-form diagnostic details to an error under the "details" field.
// Arguments are normalized according to the following rules (never panics):
//   - string + any           : treated as a (key,value) pair
//   - Detail / []Detail  : added directly
//   - trailing string        : stored as {Key:"__INVALID_TAIL__", Value:string}
//   - non-string key element : stored as {Key:"__INVALID_STANDALONE__", Value:value}
func Details(args ...any) Option {
	if len(args) == 0 {
		return detailsField([]Detail{})
	}

	const (
		invalidTailKey       = "__INVALID_TAIL__"
		invalidStandaloneKey = "__INVALID_STANDALONE__"
	)

	out := make([]Detail, 0, len(args)/2+2)
	for i := 0; i < len(args); {
		switch v := args[i].(type) {
		case Detail:
			out = append(out, v)
			i++
			continue
		case []Detail:
			out = append(out, v...)
			i++
			continue
		case string:
			if i+1 < len(args) {
				out = append(out, Detail{Key: v, Value: args[i+1]})
				i += 2
			} else {
				out = append(out, Detail{Key: invalidTailKey, Value: v})
				i++
			}
		default:
			out = append(out, Detail{Key: invalidStandaloneKey, Value: v})
			i++
		}
	}
	return detailsField(out)
}
