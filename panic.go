package errdef

import (
	"fmt"
	"io"
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
	if err, ok := e.panicValue.(error); ok {
		return err
	}
	return nil
}

func (e *panicError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, e.Error())
			_, _ = io.WriteString(s, "\n---")
			_, _ = io.WriteString(s, "\npanic_value: ")
			_, _ = fmt.Fprintf(s, "%+v", e.panicValue)
		case s.Flag('#'):
			type (
				panicError_ panicError
				panicError  panicError_
			)
			_, _ = fmt.Fprintf(s, "%#v", (*panicError)(e))
		default:
			_, _ = io.WriteString(s, e.Error())
		}
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}
