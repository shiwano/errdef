package errdef

import (
	"fmt"
	"io"
	"log/slog"
)

type (
	PanicError interface {
		error
		// PanicValue returns the value recovered from the panic.
		PanicValue() any
		// Unwrap returns the underlying error if the panic value is an error.
		Unwrap() error
	}

	panicError struct {
		msg        string
		panicValue any
	}
)

var (
	_ PanicError    = (*panicError)(nil)
	_ fmt.Formatter = (*panicError)(nil)
	_ slog.LogValuer = (*panicError)(nil)
)

func newPanicError(panicValue any) *panicError {
	return &panicError{
		msg:        fmt.Sprintf("%v", panicValue),
		panicValue: panicValue,
	}
}

func (e *panicError) Error() string {
	return e.msg
}

func (e *panicError) PanicValue() any {
	return e.panicValue
}

func (e *panicError) Unwrap() error {
	if uw, ok := e.panicValue.(error); ok {
		return uw
	}
	return nil
}

func (e *panicError) Format(s fmt.State, verb rune) {
	if f, ok := e.panicValue.(fmt.Formatter); ok {
		f.Format(s, verb)
		return
	}

	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = fmt.Fprintf(s, "%+v", e.panicValue)
		case s.Flag('#'):
			_, _ = fmt.Fprintf(s, "%#v", e.panicValue)
		default:
			_, _ = io.WriteString(s, e.Error())
		}
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

func (e *panicError) LogValue() slog.Value {
	if v, ok := e.panicValue.(slog.LogValuer); ok {
		return v.LogValue()
	}
	return slog.AnyValue(e.panicValue)
}
