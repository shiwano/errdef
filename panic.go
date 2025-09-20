package errdef

import "fmt"

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

var _ PanicError = (*panicError)(nil)

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
