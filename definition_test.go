package errdef_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/shiwano/errdef"
)

func TestDefinition_Kind(t *testing.T) {
	want := errdef.Kind("test_error")
	def := errdef.Define(want)

	if got := def.Kind(); got != want {
		t.Errorf("want kind %v, got %v", want, got)
	}
}

func TestDefinition_Error(t *testing.T) {
	kind := errdef.Kind("test_error")
	def := errdef.Define(kind)

	if def.Error() != "test_error" {
		t.Errorf("want error string %q, got %q", "test_error", def.Error())
	}
}

func TestDefinition_With(t *testing.T) {
	t.Run("with context options", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("context_field")
		ctx := errdef.ContextWithOptions(context.Background(), ctor("ctx_value"))

		def := errdef.Define("test_error")
		newDef := def.With(ctx)

		err := newDef.New("test message")
		got, ok := extr(err)
		if !ok {
			t.Error("want context field to be found")
		}
		if want := "ctx_value"; got != want {
			t.Errorf("want context field value %q, got %q", want, got)
		}
	})

	t.Run("with additional options", func(t *testing.T) {
		ctor1, extr1 := errdef.DefineField[string]("context_field")
		ctor2, extr2 := errdef.DefineField[string]("additional_field")

		ctx := errdef.ContextWithOptions(context.Background(), ctor1("ctx_value"))
		def := errdef.Define("test_error")
		newDef := def.With(ctx, ctor2("additional_value"))

		err := newDef.New("test message")

		got1, ok1 := extr1(err)
		if !ok1 {
			t.Error("want context field to be found")
		}
		if want1 := "ctx_value"; got1 != want1 {
			t.Errorf("want context field value %q, got %q", want1, got1)
		}

		got2, ok2 := extr2(err)
		if !ok2 {
			t.Error("want additional field to be found")
		}
		if want2 := "additional_value"; got2 != want2 {
			t.Errorf("want additional field value %q, got %q", want2, got2)
		}
	})

	t.Run("empty context", func(t *testing.T) {
		def := errdef.Define("test_error")
		newDef := def.With(context.Background())

		if newDef != def {
			t.Errorf("want same, got %#v vs %#v", def, newDef)
		}
	})
}

func TestDefinition_WithOptions(t *testing.T) {
	t.Run("with field option", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("test_field")

		def := errdef.Define("test_error")
		newDef := def.WithOptions(ctor("test_value"))

		err := newDef.New("test message")
		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "test_value"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		ctor1, extr1 := errdef.DefineField[string]("field1")
		ctor2, extr2 := errdef.DefineField[int]("field2")

		def := errdef.Define("test_error")
		newDef := def.WithOptions(ctor1("value1"), ctor2(42))

		err := newDef.New("test message")

		got1, ok1 := extr1(err)
		if !ok1 {
			t.Error("want field1 to be found")
		}
		if want1 := "value1"; got1 != want1 {
			t.Errorf("want field1 value %q, got %q", want1, got1)
		}

		got2, ok2 := extr2(err)
		if !ok2 {
			t.Error("want field2 to be found")
		}
		if want2 := 42; got2 != want2 {
			t.Errorf("want field2 value %d, got %d", want2, got2)
		}
	})

	t.Run("no options", func(t *testing.T) {
		def := errdef.Define("test_error")
		newDef := def.WithOptions()

		if newDef != def {
			t.Errorf("want same, got %#v vs %#v", def, newDef)
		}
	})
}

func TestDefinition_New(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		if err.Error() != "test message" {
			t.Errorf("want message %q, got %q", "test message", err.Error())
		}
	})

	t.Run("with definition field", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("user_id")
		def := errdef.Define("test_error").WithOptions(ctor("user123"))

		err := def.New("test message")

		if want, got := "test message", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "user123"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})
}

func TestDefinition_Errorf(t *testing.T) {
	t.Run("formatted error creation", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.Errorf("test message with %s and %d", "string", 42)

		want := "test message with string and 42"
		if err.Error() != want {
			t.Errorf("want message %q, got %q", want, err.Error())
		}
	})

	t.Run("no format args", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.Errorf("simple message")

		if err.Error() != "simple message" {
			t.Errorf("want message %q, got %q", "simple message", err.Error())
		}
	})

	t.Run("with definition field", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("operation")
		def := errdef.Define("test_error").WithOptions(ctor("update"))

		err := def.Errorf("failed to %s user %d", "update", 123)

		if want, got := "failed to update user 123", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "update"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})
}

func TestDefinition_Wrap(t *testing.T) {
	t.Run("wrap error", func(t *testing.T) {
		def := errdef.Define("test_error")
		cause := errors.New("original error")

		err := def.Wrap(cause)

		if want, got := "original error", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}

		if !errors.Is(err, cause) {
			t.Error("want wrapped error to be the cause")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.Wrap(nil)

		if err != nil {
			t.Error("want nil when wrapping nil error")
		}
	})

	t.Run("with definition field", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("context")
		def := errdef.Define("test_error").WithOptions(ctor("database"))
		cause := errors.New("connection failed")

		err := def.Wrap(cause)

		if want, got := "connection failed", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}

		if !errors.Is(err, cause) {
			t.Error("want wrapped error to be the cause")
		}

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "database"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})
}

func TestDefinition_Wrapf(t *testing.T) {
	t.Run("wrap with format", func(t *testing.T) {
		def := errdef.Define("test_error")
		cause := errors.New("connection failed")

		err := def.Wrapf(cause, "failed to connect to %s:%d", "localhost", 5432)

		want := "failed to connect to localhost:5432: connection failed"
		if err.Error() != want {
			t.Errorf("want message %q, got %q", want, err.Error())
		}

		if !errors.Is(err, cause) {
			t.Error("want wrapped error to be the cause")
		}
	})

	t.Run("wrap nil error with format", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.Wrapf(nil, "this should not create error")

		if err != nil {
			t.Error("want nil when wrapping nil error")
		}
	})

	t.Run("with definition field", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("service")
		def := errdef.Define("test_error").WithOptions(ctor("auth"))
		cause := fmt.Errorf("invalid token")

		err := def.Wrapf(cause, "authentication failed for service")

		if want, got := "authentication failed for service: invalid token", err.Error(); got != want {
			t.Errorf("want message %q, got %q", want, got)
		}

		if !errors.Is(err, cause) {
			t.Error("want wrapped error to be the cause")
		}

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "auth"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})
}

func TestDefinition_Join(t *testing.T) {
	t.Run("join errors", func(t *testing.T) {
		def := errdef.Define("test_error")
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		joined := def.Join(err1, err2)

		if !errors.Is(joined, err1) {
			t.Error("want joined error to contain err1")
		}
		if !errors.Is(joined, err2) {
			t.Error("want joined error to contain err2")
		}
	})

	t.Run("join with nil errors", func(t *testing.T) {
		def := errdef.Define("test_error")
		err1 := errors.New("error 1")

		joined := def.Join(err1, nil)

		if !errors.Is(joined, err1) {
			t.Error("want joined error to contain err1")
		}
	})

	t.Run("join all nil errors", func(t *testing.T) {
		def := errdef.Define("test_error")
		joined := def.Join(nil, nil)

		if joined != nil {
			t.Error("want nil when joining only nil errors")
		}
	})

	t.Run("join no errors", func(t *testing.T) {
		def := errdef.Define("test_error")
		joined := def.Join()

		if joined != nil {
			t.Error("want nil when joining no errors")
		}
	})

	t.Run("with definition field", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("batch_id")
		def := errdef.Define("batch_error").WithOptions(ctor("batch_123"))
		err1 := errors.New("validation failed")
		err2 := errors.New("save failed")

		joined := def.Join(err1, err2)

		if !errors.Is(joined, err1) {
			t.Error("want joined error to contain err1")
		}
		if !errors.Is(joined, err2) {
			t.Error("want joined error to contain err2")
		}

		got, ok := extr(joined)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "batch_123"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}
	})
}

func TestDefinition_Recover(t *testing.T) {
	t.Run("recover string panic", func(t *testing.T) {
		def := errdef.Define("panic_error")
		err := def.Recover(func() error {
			panic("test panic")
		})

		if err == nil {
			t.Fatal("want error to be set")
		}

		if !errors.Is(err, def) {
			t.Fatal("want error to be wrapped by the definition")
		}
		if err.Error() != "panic: test panic" {
			t.Errorf("want error message %q, got %q", "panic: test panic", err.Error())
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != "test panic" {
			t.Errorf("want panic value %q, got %v", "test panic", panicErr.PanicValue())
		}
	})

	t.Run("recover error panic", func(t *testing.T) {
		def := errdef.Define("panic_error")
		panicValue := errors.New("panic error")
		err := def.Recover(func() error {
			panic(panicValue)
		})

		if err == nil {
			t.Fatal("want error to be set")
		}

		if !errors.Is(err, def) {
			t.Fatal("want error to be wrapped by the definition")
		}
		if err.Error() != "panic: panic error" {
			t.Errorf("want error message %q, got %q", "panic: panic error", err.Error())
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != panicValue {
			t.Errorf("want panic value %v, got %v", panicValue, panicErr.PanicValue())
		}
	})

	t.Run("no panic", func(t *testing.T) {
		def := errdef.Define("panic_error")
		err := def.Recover(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("want no error when no panic occurs, got %v", err)
		}

		if errors.Is(err, def) {
			t.Error("want error not to match definition when no panic occurs")
		}
	})

	t.Run("return error without panic", func(t *testing.T) {
		def := errdef.Define("panic_error")
		returnedErr := errors.New("normal error")
		err := def.Recover(func() error {
			return returnedErr
		})

		if err != returnedErr {
			t.Errorf("want returned error %v, got %v", returnedErr, err)
		}

		if errors.Is(err, def) {
			t.Error("want error not to match definition when no panic occurs")
		}
	})

	t.Run("with definition fields", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("context")
		originalDef := errdef.Define("panic_error")
		def := originalDef.WithOptions(ctor("service_call"))
		err := def.Recover(func() error {
			panic("service panic")
		})

		if err == nil {
			t.Fatal("want error to be set")
		}

		if !errors.Is(err, originalDef) {
			t.Fatal("want error to match definition")
		}

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "service_call"; got != want {
			t.Errorf("want field value %q, got %q", want, got)
		}

		var panicErr errdef.PanicError
		if !errors.As(err, &panicErr) {
			t.Fatal("want error to be a PanicError")
		}

		if panicErr.PanicValue() != "service panic" {
			t.Errorf("want panic value %q, got %v", "service panic", panicErr.PanicValue())
		}
	})

	t.Run("nested recover", func(t *testing.T) {
		defOuter := errdef.Define("outer_panic")
		defInner := errdef.Define("inner_panic")

		outerErr := defOuter.Recover(func() error {
			innerErr := defInner.Recover(func() error {
				panic("actual panic")
			})
			if !errors.Is(innerErr, defInner) {
				t.Error("want inner error to match inner definition")
			}
			return innerErr
		})

		if errors.Is(outerErr, defOuter) {
			t.Error("want outer error not to match outer definition (no panic at outer level)")
		}

		if outerErr == nil {
			t.Fatal("want error to be returned from inner")
		}

		if !errors.Is(outerErr, defInner) {
			t.Error("want outer error to match inner definition")
		}
	})
}

func TestDefinition_Is(t *testing.T) {
	t.Run("is same definition", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		if !def.Is(err) {
			t.Error("want definition to match its own error")
		}
	})

	t.Run("is different definition", func(t *testing.T) {
		def1 := errdef.Define("error1")
		def2 := errdef.Define("error2")
		err := def1.New("test message")

		if def2.Is(err) {
			t.Error("want different definition not to match")
		}
	})

	t.Run("is not errdef error", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := errors.New("regular error")

		if def.Is(err) {
			t.Error("want definition not to match regular error")
		}
	})

	t.Run("is wrapped error", func(t *testing.T) {
		def := errdef.Define("test_error")
		originalErr := def.New("original")
		wrappedErr := fmt.Errorf("wrapped: %w", originalErr)

		if !def.Is(wrappedErr) {
			t.Error("want definition to match wrapped error containing its error")
		}
	})

	t.Run("different definitions with same kind", func(t *testing.T) {
		def1 := errdef.Define("same_kind")
		def2 := errdef.Define("same_kind")
		err := def1.New("test message")

		if def2.Is(err) {
			t.Error("want different definitions with same kind not to match")
		}

		if !def1.Is(err) {
			t.Error("want original definition to match its own error")
		}
	})

	t.Run("WithOptions creates errors matched by original definition", func(t *testing.T) {
		orig := errdef.Define("test_error")
		ctor, _ := errdef.DefineField[string]("test_field")

		factory := orig.WithOptions(ctor("test_value"))
		err := factory.New("test message")

		if !orig.Is(err) {
			t.Error("want original definition to match error from WithOptions factory")
		}

		if !errors.Is(err, orig) {
			t.Error("want errors.Is to match original definition")
		}
	})

	t.Run("With creates errors matched by original definition", func(t *testing.T) {
		orig := errdef.Define("test_error")
		ctor, _ := errdef.DefineField[string]("test_field")

		ctx := context.Background()
		factory := orig.With(ctx, ctor("test_value"))
		err := factory.New("test message")

		if !orig.Is(err) {
			t.Error("want original definition to match error from With factory")
		}

		if !errors.Is(err, orig) {
			t.Error("want errors.Is to match original definition")
		}
	})

	t.Run("definition as sentinel error", func(t *testing.T) {
		def := errdef.Define("test_error")

		if !def.Is(def) {
			t.Error("want definition to match itself with Is method")
		}

		if !errors.Is(def, def) {
			t.Error("want definition to match with errors.Is")
		}
	})

	t.Run("wrapped definition as sentinel error", func(t *testing.T) {
		def := errdef.Define("test_error")
		wrapped := fmt.Errorf("wrapped: %w", def)

		if !def.Is(wrapped) {
			t.Error("want wrapped definition to match its root with Is method")
		}

		if !errors.Is(wrapped, def) {
			t.Error("want wrapped definition to match with errors.Is")
		}
	})
}
