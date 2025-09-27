package errdef_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shiwano/errdef"
)

func TestPanicError_Error(t *testing.T) {
	def := errdef.Define("test_error")
	panicValue := errors.New("panic error")
	var err error
	def.CapturePanic(&err, panicValue)

	var panicErr errdef.PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("want error to be a PanicError")
	}

	if panicErr.Error() != "panic error" {
		t.Errorf("want error message %q, got %q", "panic error", panicErr.Error())
	}
}

func TestPanicError_PanicValue(t *testing.T) {
	def := errdef.Define("test_error")
	panicValue := errors.New("panic error")
	var err error
	def.CapturePanic(&err, panicValue)

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
		def := errdef.Define("test_error")
		panicValue := errors.New("panic error")
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != panicValue {
			t.Errorf("want unwrapped error %v, got %v", panicValue, panicErr.Unwrap())
		}
	})

	t.Run("with non error", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := "panic string"
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.Unwrap() != nil {
			t.Errorf("want unwrapped error nil, got %v", panicErr.Unwrap())
		}
	})
}

func TestPanicError_Format(t *testing.T) {
	t.Run("with standard error", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := errors.New("panic error")
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if got := fmt.Sprintf("%v", panicErr); got != "panic error" {
			t.Errorf("%%v: want %q, got %q", "panic error", got)
		}

		if got := fmt.Sprintf("%s", panicErr); got != "panic error" {
			t.Errorf("%%s: want %q, got %q", "panic error", got)
		}

		if got := fmt.Sprintf("%+v", panicErr); got != "panic error" {
			t.Errorf("%%+v: want %q, got %q", "panic error", got)
		}
	})

	t.Run("with string value", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := "panic string"
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if got := fmt.Sprintf("%v", panicErr); got != "panic string" {
			t.Errorf("%%v: want %q, got %q", "panic string", got)
		}

		if got := fmt.Sprintf("%q", panicErr); got != `"panic string"` {
			t.Errorf("%%q: want %q, got %q", `"panic string"`, got)
		}
	})

	t.Run("with errdef.Error", func(t *testing.T) {
		def := errdef.Define("test_error")
		panicValue := def.WithOptions(errdef.NoTrace(), errdef.HTTPStatus(400)).New("panic error")
		var err error
		def.CapturePanic(&err, panicValue)

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if got := fmt.Sprintf("%v", panicErr); got != "panic error" {
			t.Errorf("%%v: want %q, got %q", "panic error", got)
		}

		if got := fmt.Sprintf("%s", panicErr); got != "panic error" {
			t.Errorf("%%s: want %q, got %q", "panic error", got)
		}

		if got := fmt.Sprintf("%+v", panicErr); got != "panic error\n\nKind:\n\ttest_error\nFields:\n\thttp_status: 400" {
			t.Errorf("%%+v: want %q, got %q", "panic error", got)
		}
	})
}
