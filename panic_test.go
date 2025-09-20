package errdef_test

import (
	"errors"
	"testing"

	"github.com/shiwano/errdef"
)

func TestPanicError_Error(t *testing.T) {
	panicValue := errors.New("panic error")
	var err error
	errdef.CapturePanic(&err, panicValue)

	var panicErr errdef.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("want error to be a PanicError")
	}

	if panicErr.Error() != "panic error" {
		t.Errorf("want error message %q, got %q", "panic error", panicErr.Error())
	}
}

func TestPanicError_PanicValue(t *testing.T) {
	panicValue := errors.New("panic error")
	var err error
	errdef.CapturePanic(&err, panicValue)

	var panicErr errdef.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("want error to be a PanicError")
	}

	if panicErr.PanicValue() != panicValue {
		t.Errorf("want panic value %v, got %v", panicValue, panicErr.PanicValue())
	}
}

func TestPanicError_Unwrap(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		panicValue := errors.New("panic error")
		var err error
		errdef.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != panicValue {
			t.Errorf("want unwrapped error %v, got %v", panicValue, panicErr.Unwrap())
		}
	})

	t.Run("with non error", func(t *testing.T) {
		panicValue := "panic string"
		var err error
		errdef.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != nil {
			t.Errorf("want unwrapped error nil, got %v", panicErr.Unwrap())
		}
	})
}
