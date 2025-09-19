package errdef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
)

type (
	// Error extends the built-in error interface with additional functionality
	// for structured error handling including kinds, fields, and stack traces.
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
	}

	// ErrorFormatter is a function type for custom error formatting.
	ErrorFormatter func(err Error, s fmt.State, verb rune)

	// ErrorJSONMarshaler is a function type for custom JSON marshaling of errors.
	ErrorJSONMarshaler func(err Error) ([]byte, error)

	// DebugStacker returns a string that resembles the output of debug.Stack().
	// This is useful for integrating with Google Cloud Observability.
	// NOTE: The goroutine ID and state may differ from the actual one.
	// See: https://cloud.google.com/error-reporting/reference/rest/v1beta1/projects.events/report#ReportedErrorEvent
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
		def    *Definition
		msg    string
		cause  error
		stack  stack
		joined bool
	}
)

var (
	_ Error          = (*definedError)(nil)
	_ DebugStacker   = (*definedError)(nil)
	_ fmt.Formatter  = (*definedError)(nil)
	_ json.Marshaler = (*definedError)(nil)
	_ stackTracer    = (*definedError)(nil)
	_ causer         = (*definedError)(nil)
)

func newError(d *Definition, cause error, msg string, joined bool, stackSkip int) error {
	var stack stack
	if !d.noTrace {
		stack = newStack(d.stackSkip + stackSkip)
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
	if e.def.boundary {
		return nil // Break the error chain.
	}
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

func (e *definedError) Is(target error) bool {
	if e == target {
		return true
	}
	if d, ok := target.(*Definition); ok {
		return e.def.kind == d.kind
	}
	return false
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

func (e *definedError) Format(s fmt.State, verb rune) {
	if e.def.formatter != nil {
		e.def.formatter(e, s, verb)
		return
	}

	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = fmt.Fprintf(s, "%s\n\n", e.Error())

			if e.Kind() != "" {
				_, _ = io.WriteString(s, "Kind:\n")
				_, _ = fmt.Fprintf(s, "\t%v\n", e.Kind())
			}

			if e.Fields().Len() > 0 {
				_, _ = io.WriteString(s, "Fields:\n")
				for k, v := range e.Fields().SortedSeq() {
					_, _ = fmt.Fprintf(s, "\t%v: %+v\n", k, v)
				}
			}

			if e.Stack().Len() > 0 {
				_, _ = io.WriteString(s, "Stack:\n")
				for _, f := range e.Stack().Frames() {
					if f.File != "" {
						_, _ = fmt.Fprintf(s, "\t%s\n\t\t%s:%d\n", f.Func, f.File, f.Line)
					}
				}
			}

			for i, cause := range e.Unwrap() {
				if i == 0 {
					_, _ = io.WriteString(s, "Causes:\n")
				}

				causeStr := strings.Trim(fmt.Sprintf("%+v", cause), "\n")

				for line := range strings.SplitSeq(causeStr, "\n") {
					_, _ = fmt.Fprintf(s, "\t%s\n", line)
				}
			}
		case s.Flag('#'):
			// Avoid infinite recursion in case someone does %#v on definedError.
			type definedError struct {
				def    *Definition
				msg    string
				cause  error
				stack  stack
				joined bool
			}
			var tmp = definedError(*e)
			_, _ = fmt.Fprintf(s, "%#v", &tmp)
		default:
			_, _ = io.WriteString(s, e.Error())
		}
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

func (e *definedError) MarshalJSON() ([]byte, error) {
	if e.def.jsonMarshaler != nil {
		return e.def.jsonMarshaler(e)
	}

	fields, err := e.Fields().MarshalJSON()
	if err != nil {
		return nil, err
	}

	var causes []json.RawMessage
	for _, c := range e.Unwrap() {
		if marshaler, ok := c.(json.Marshaler); ok {
			b, err := marshaler.MarshalJSON()
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		} else {
			b, err := json.Marshal(c.Error())
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		}
	}

	return json.Marshal(struct {
		Message string            `json:"message"`
		Kind    string            `json:"kind,omitempty"`
		Fields  json.RawMessage   `json:"fields,omitempty"`
		Stack   []Frame           `json:"stack,omitempty"`
		Causes  []json.RawMessage `json:"causes,omitempty"`
	}{
		Message: e.Error(),
		Kind:    string(e.Kind()),
		Fields:  fields,
		Stack:   e.stack.Frames(),
		Causes:  causes,
	})
}
