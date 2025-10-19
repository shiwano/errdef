package errdef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
)

type (
	// Error extends the built-in error interface with additional functionality
	// for structured error handling including kinds, fields, and stack traces.
	//
	// Error instances are created from a Definition and remain immutable after creation.
	// They provide rich context through Kind (error classification), Fields (structured data),
	// and Stack (call stack information), while maintaining compatibility with standard
	// Go error handling via errors.Is and errors.As.
	//
	// Error chains are supported through Unwrap() for standard error unwrapping,
	// and UnwrapTree() for accessing the full error tree with cycle detection.
	Error interface {
		error

		// Kind returns the type of this error.
		Kind() Kind
		// Fields returns the structured fields associated with this error.
		Fields() Fields
		// Stack returns the stack trace where this error was created.
		Stack() Stack
		// Unwrap returns the errors that this error wraps.
		Unwrap() []error
		// UnwrapTree returns all causes as a tree structure.
		// This method includes cycle detection: when a circular reference is detected,
		// the node that would create the cycle is excluded, ensuring the result remains acyclic.
		// While circular references are rare in practice, this check serves as a defensive
		// programming measure.
		UnwrapTree() Nodes
	}

	// DebugStacker returns a string that resembles the output of debug.Stack().
	// This is useful for integrating with Google Cloud Error Reporting.
	// See: https://cloud.google.com/error-reporting/reference/rest/v1beta1/projects.events/report#ReportedErrorEvent
	//
	// NOTE: The goroutine ID and state may differ from the actual one.
	DebugStacker interface {
		DebugStack() string
	}

	// stackTracer is used by Sentry SDK to extract stack traces from errors.
	// See: https://github.com/getsentry/sentry-go/blob/54a69e05ea609d3fc32fb1393770258dde6796c1/stacktrace.go#L84-L87
	stackTracer interface {
		StackTrace() []uintptr
	}

	// causer is used by pkg/errors to extract the cause of an error.
	// See: https://github.com/golang/go/issues/31778
	causer interface {
		Cause() error
	}

	definedError struct {
		def    *definition
		msg    string
		cause  error
		stack  stack
		joined bool
	}

	jsonErrorData struct {
		Message string  `json:"message"`
		Kind    string  `json:"kind,omitempty"`
		Fields  Fields  `json:"fields,omitempty,omitzero"`
		Stack   []Frame `json:"stack,omitempty"`
		Causes  Nodes   `json:"causes,omitempty"`
	}
)

var (
	_ Error          = (*definedError)(nil)
	_ DebugStacker   = (*definedError)(nil)
	_ fmt.GoStringer = (*definedError)(nil)
	_ fmt.Formatter  = (*definedError)(nil)
	_ json.Marshaler = (*definedError)(nil)
	_ slog.LogValuer = (*definedError)(nil)
	_ stackTracer    = (*definedError)(nil)
	_ causer         = (*definedError)(nil)
	_ fieldsGetter   = (*definedError)(nil)
)

func newError(d *definition, cause error, msg string, joined bool, stackSkip int) error {
	var stack stack
	if !d.noTrace {
		depth := callersDepth
		if d.stackDepth > 0 {
			depth = d.stackDepth
		}
		stack = newStack(depth, d.stackSkip+stackSkip)
	}
	return &definedError{
		def:    d,
		msg:    msg,
		cause:  cause,
		stack:  stack,
		joined: joined,
	}
}

func (e *definedError) Error() string {
	return e.msg
}

func (e *definedError) Kind() Kind {
	return e.def.kind
}

func (e *definedError) Fields() Fields {
	return e.def.fields
}

func (e *definedError) Stack() Stack {
	return e.stack
}

func (e *definedError) Unwrap() []error {
	if e.cause == nil {
		return nil
	}
	if e.joined {
		if u, ok := e.cause.(interface{ Unwrap() []error }); ok {
			return u.Unwrap()
		}
	}
	return []error{e.cause}
}

func (e *definedError) UnwrapTree() Nodes {
	return e.def.BuildCauseTree(e)
}

func (e *definedError) Is(target error) bool {
	if e == target {
		return true
	}
	if d, ok := target.(*definition); ok {
		return e.def.root() == d.root()
	}
	return false
}

func (e *definedError) GoString() string {
	type (
		definedError_ definedError
		definedError  definedError_
	)
	return fmt.Sprintf("%#v", (*definedError)(e))
}

func (e *definedError) Format(s fmt.State, verb rune) {
	e.def.FormatError(e, s, verb)
}

func (e *definedError) MarshalJSON() ([]byte, error) {
	return e.def.MarshalErrorJSON(e)
}

func (e *definedError) LogValue() slog.Value {
	return e.def.MakeErrorLogValue(e)
}

func (e *definedError) DebugStack() string {
	buf := bytes.NewBufferString(e.Error())
	// hard-coded cause we can't get it in pure Go.
	buf.WriteString("\n\ngoroutine 1 [running]:")

	for _, pc := range e.stack.StackTrace() {
		if fn := runtime.FuncForPC(pc); fn != nil {
			buf.WriteByte('\n')
			file, line := fn.FileLine(pc)
			fmt.Fprintf(buf, "%s()\n\t%s:%d +%#x", fn.Name(), file, line, fn.Entry())
		}
	}
	return buf.String()
}

func (e *definedError) StackTrace() []uintptr {
	return e.stack.StackTrace()
}

func (e *definedError) Cause() error {
	return e.cause
}
