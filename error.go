package errdef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
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

	// ErrorLogValuer is a function type for custom slog.Value conversion of errors.
	ErrorLogValuer func(err Error) slog.Value

	// DebugStacker returns a string that resembles the output of debug.Stack().
	// This is useful for integrating with Google Cloud Error Reporting.
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

	// errorEncoder defines methods for exporting error information in various formats.
	// This interface is used internally to implement fmt.Formatter, json.Marshaler,
	// and slog.LogValuer in the unmarshaler package.
	errorEncoder interface {
		ErrorFormatter(err Error, s fmt.State, verb rune)
		ErrorJSONMarshaler(err Error) ([]byte, error)
		ErrorLogValuer(err Error) slog.Value
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
	_ slog.LogValuer = (*definedError)(nil)
	_ stackTracer    = (*definedError)(nil)
	_ causer         = (*definedError)(nil)
	_ errorEncoder   = (*definedError)(nil)
)

func newError(d *Definition, cause error, msg string, joined bool, stackSkip int) error {
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
		return e.def.root == d.root
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
	if e.def.boundary {
		return nil // Break the error chain.
	}
	return e.cause
}

func (e *definedError) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('#') {
		// Avoid infinite recursion in case someone does %#v on definedError.
		type (
			definedError_ definedError
			definedError  definedError_
		)
		_, _ = fmt.Fprintf(s, "%#v", (*definedError)(e))
		return
	}
	e.ErrorFormatter(e, s, verb)
}

func (e *definedError) MarshalJSON() ([]byte, error) {
	return e.ErrorJSONMarshaler(e)
}

func (e *definedError) LogValue() slog.Value {
	return e.ErrorLogValuer(e)
}

func (e *definedError) ErrorFormatter(err Error, s fmt.State, verb rune) {
	if e.def.formatter != nil {
		e.def.formatter(err, s, verb)
		return
	}

	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, err.Error())

			causes := err.Unwrap()

			if err.Kind() != "" || err.Fields().Len() > 0 || err.Stack().Len() > 0 || len(causes) > 0 {
				_, _ = io.WriteString(s, "\n")
			}

			if err.Kind() != "" {
				_, _ = io.WriteString(s, "\nKind:\n")
				_, _ = io.WriteString(s, "\t")
				_, _ = io.WriteString(s, string(err.Kind()))
			}

			if err.Fields().Len() > 0 {
				_, _ = io.WriteString(s, "\nFields:\n")
				i := 0
				for k, v := range err.Fields().SortedSeq() {
					if i > 0 {
						_, _ = io.WriteString(s, "\n")
					}
					_, _ = io.WriteString(s, "\t")
					_, _ = io.WriteString(s, k.String())
					_, _ = io.WriteString(s, ": ")
					_, _ = fmt.Fprintf(s, "%+v", v.Value())
					i++
				}
			}

			if err.Stack().Len() > 0 {
				_, _ = io.WriteString(s, "\nStack:\n")
				i := 0
				for _, f := range err.Stack().Frames() {
					if f.File != "" {
						if i > 0 {
							_, _ = io.WriteString(s, "\n")
						}
						_, _ = io.WriteString(s, "\t")
						_, _ = io.WriteString(s, f.Func)
						_, _ = io.WriteString(s, "\n\t\t")
						_, _ = io.WriteString(s, f.File)
						_, _ = io.WriteString(s, strconv.Itoa(f.Line))
						i++
					}
				}
			}

			for i, cause := range causes {
				if i == 0 {
					_, _ = io.WriteString(s, "\nCauses:\n")
				} else {
					_, _ = io.WriteString(s, "\n\t---\n")
				}

				causeStr := strings.Trim(fmt.Sprintf("%+v", cause), "\n")

				j := 0
				for line := range strings.SplitSeq(causeStr, "\n") {
					if j > 0 {
						_, _ = io.WriteString(s, "\n")
					}
					_, _ = io.WriteString(s, "\t")
					_, _ = io.WriteString(s, line)
					j++
				}
			}
		case s.Flag('#'):
			// Don't support %#v to avoid infinite recursion in this method.
			_, _ = io.WriteString(s, err.Error())
		default:
			_, _ = io.WriteString(s, err.Error())
		}
	case 's':
		_, _ = io.WriteString(s, err.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", err.Error())
	}
}

func (e *definedError) ErrorJSONMarshaler(err Error) ([]byte, error) {
	if e.def.jsonMarshaler != nil {
		return e.def.jsonMarshaler(e)
	}

	var fields Fields
	if err.Fields().Len() > 0 {
		fields = err.Fields()
	}

	var stacks []Frame
	if err.Stack().Len() > 0 {
		stacks = err.Stack().Frames()
	}

	var causes []json.RawMessage
	for _, c := range err.Unwrap() {
		switch t := c.(type) {
		case Error:
			if m, ok := t.(json.Marshaler); ok {
				b, err := m.MarshalJSON()
				if err != nil {
					return nil, err
				}
				causes = append(causes, b)
			} else {
				b, err := json.Marshal(struct {
					Message string `json:"message"`
				}{
					Message: c.Error(),
				})
				if err != nil {
					return nil, err
				}
				causes = append(causes, b)
			}
		case json.Marshaler:
			data, err := t.MarshalJSON()
			if err != nil {
				return nil, err
			}

			b, err := json.Marshal(struct {
				Message string          `json:"message"`
				Type    string          `json:"type"`
				Data    json.RawMessage `json:"data"`
			}{
				Message: c.Error(),
				Type:    fmt.Sprintf("%T", c),
				Data:    data,
			})
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		default:
			b, err := json.Marshal(struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: c.Error(),
				Type:    fmt.Sprintf("%T", c),
			})
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		}
	}
	return json.Marshal(struct {
		Message string            `json:"message"`
		Kind    string            `json:"kind,omitempty"`
		Fields  Fields            `json:"fields,omitempty"`
		Stack   []Frame           `json:"stack,omitempty"`
		Causes  []json.RawMessage `json:"causes,omitempty"`
	}{
		Message: err.Error(),
		Kind:    string(err.Kind()),
		Fields:  fields,
		Stack:   stacks,
		Causes:  causes,
	})
}

func (e *definedError) ErrorLogValuer(err Error) slog.Value {
	if e.def.logValuer != nil {
		return e.def.logValuer(e)
	}

	attrs := make([]slog.Attr, 0, 5)

	attrs = append(attrs, slog.String("message", err.Error()))

	if err.Kind() != "" {
		attrs = append(attrs, slog.String("kind", string(err.Kind())))
	}

	if err.Fields().Len() > 0 {
		attrs = append(attrs, slog.Any("fields", err.Fields()))
	}

	if err.Stack().Len() > 0 {
		if frame, ok := err.Stack().HeadFrame(); ok {
			attrs = append(attrs, slog.Any("origin", frame))
		}
	}

	causes := err.Unwrap()
	if len(causes) > 0 {
		causeMessages := make([]string, len(causes))
		for i, c := range causes {
			causeMessages[i] = c.Error()
		}
		attrs = append(attrs, slog.Any("causes", causeMessages))
	}
	return slog.GroupValue(attrs...)
}
