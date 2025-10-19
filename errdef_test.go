package errdef_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shiwano/errdef"
)

func TestDefine(t *testing.T) {
	t.Run("basic kind", func(t *testing.T) {
		want := errdef.Kind("test_error")
		def := errdef.Define(want)

		if got := def.Kind(); got != want {
			t.Errorf("want kind %v, got %v", want, got)
		}
	})

	t.Run("empty kind", func(t *testing.T) {
		def := errdef.Define("")

		if got := def.Kind(); got != "" {
			t.Errorf("want empty kind, got %v", got)
		}
	})
}

func TestDefineField(t *testing.T) {
	t.Run("constructor and extractor", func(t *testing.T) {
		ctor, extr := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", ctor("test_value"))
		err := def.New("test error")

		got, ok := extr(err)
		if !ok {
			t.Error("want field to be found")
		}
		if want := "test_value"; got != want {
			t.Errorf("want value %q, got %q", want, got)
		}
	})

	t.Run("extractor with wrong value type", func(t *testing.T) {
		type valueType string

		ctor, _ := errdef.DefineField[string]("test_field")
		_, extr := errdef.DefineField[valueType]("test_field")

		def := errdef.Define("test_error", ctor("test_value"))
		err := def.New("test error")

		if _, ok := extr(err); ok {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor with wrong key type", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("test_field")
		_, extr := errdef.DefineField[string]("test_field")

		def := errdef.Define("test_error", ctor("test_value"))
		err := def.New("test error")

		if _, ok := extr(err); ok {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor on non-errdef error", func(t *testing.T) {
		_, extr := errdef.DefineField[string]("test_field")
		err := errors.New("regular error")

		if _, ok := extr(err); ok {
			t.Error("want field not to be found on non-errdef error")
		}
	})

	t.Run("extractor on definition", func(t *testing.T) {
		ctor, extr := errdef.DefineField[int]("code")
		def := errdef.Define("test_error", ctor(404))

		code, ok := extr(def)
		if !ok {
			t.Error("want field to be found in definition")
		}
		if code != 404 {
			t.Errorf("want code to be 404, got %d", code)
		}
	})

	t.Run("extractor on wrapped definition", func(t *testing.T) {
		ctor, extr := errdef.DefineField[int]("code")
		def := errdef.Define("test_error", ctor(404))

		wrapped := fmt.Errorf("wrapped: %w", def)
		code, ok := extr(wrapped)
		if !ok {
			t.Error("want field to be found in wrapped definition")
		}
		if code != 404 {
			t.Errorf("want code to be 404, got %d", code)
		}
	})
}

func TestKindFrom(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		kind, ok := errdef.KindFrom(nil)
		if ok {
			t.Error("want kind not to be found from nil error")
		}
		if kind != "" {
			t.Errorf("want empty kind, got %v", kind)
		}
	})

	t.Run("non-errdef error", func(t *testing.T) {
		err := errors.New("regular error")
		kind, ok := errdef.KindFrom(err)
		if ok {
			t.Error("want kind not to be found from non-errdef error")
		}
		if kind != "" {
			t.Errorf("want empty kind, got %v", kind)
		}
	})

	t.Run("errdef error", func(t *testing.T) {
		want := errdef.Kind("test_error")
		def := errdef.Define(want)
		err := def.New("test message")

		got, ok := errdef.KindFrom(err)
		if !ok {
			t.Error("want kind to be found from errdef error")
		}
		if got != want {
			t.Errorf("want kind %v, got %v", want, got)
		}
	})

	t.Run("wrapped errdef error", func(t *testing.T) {
		want := errdef.Kind("test_error")
		def := errdef.Define(want)
		err := def.New("test message")
		wrapped := fmt.Errorf("wrapped: %w", err)

		got, ok := errdef.KindFrom(wrapped)
		if !ok {
			t.Error("want kind to be found from wrapped errdef error")
		}
		if got != want {
			t.Errorf("want kind %v, got %v", want, got)
		}
	})

	t.Run("definition with kind", func(t *testing.T) {
		want := errdef.Kind("test_error")
		def := errdef.Define(want)

		got, ok := errdef.KindFrom(def)
		if !ok {
			t.Error("want kind to be found from definition")
		}
		if got != want {
			t.Errorf("want kind %v, got %v", want, got)
		}
	})
}

func TestFieldsFrom(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		fields, ok := errdef.FieldsFrom(nil)
		if ok {
			t.Error("want fields not to be found from nil error")
		}
		if fields != nil {
			t.Errorf("want nil fields, got %v", fields)
		}
	})

	t.Run("non-errdef error", func(t *testing.T) {
		err := errors.New("regular error")
		fields, ok := errdef.FieldsFrom(err)
		if ok {
			t.Error("want fields not to be found from non-errdef error")
		}
		if fields != nil {
			t.Errorf("want nil fields, got %v", fields)
		}
	})

	t.Run("errdef error with fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", ctor("test_value"))
		err := def.New("test message")

		fields, ok := errdef.FieldsFrom(err)
		if !ok {
			t.Error("want fields to be found from errdef error")
		}
		if fields == nil {
			t.Error("want non-nil fields")
		}
		if fields.Len() != 1 {
			t.Errorf("want 1 field, got %d", fields.Len())
		}
	})

	t.Run("wrapped errdef error with fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", ctor("test_value"))
		err := def.New("test message")
		wrapped := fmt.Errorf("wrapped: %w", err)

		fields, ok := errdef.FieldsFrom(wrapped)
		if !ok {
			t.Error("want fields to be found from wrapped errdef error")
		}
		if fields == nil {
			t.Error("want non-nil fields")
		}
		if fields.Len() != 1 {
			t.Errorf("want 1 field, got %d", fields.Len())
		}
	})

	t.Run("errdef error without fields", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		fields, ok := errdef.FieldsFrom(err)
		if ok {
			t.Error("want fields not to be found when fields are empty")
		}
		if fields != nil {
			t.Errorf("want nil fields, got %v", fields)
		}
	})

	t.Run("definition with fields", func(t *testing.T) {
		ctor, _ := errdef.DefineField[string]("test_field")
		def := errdef.Define("test_error", ctor("test_value"))

		fields, ok := errdef.FieldsFrom(def)
		if !ok {
			t.Error("want fields to be found from definition")
		}
		if fields == nil {
			t.Error("want non-nil fields")
		}
		if fields.Len() != 1 {
			t.Errorf("want 1 field, got %d", fields.Len())
		}
	})
}

func TestStackFrom(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		stack, ok := errdef.StackFrom(nil)
		if ok {
			t.Error("want stack not to be found from nil error")
		}
		if stack != nil {
			t.Errorf("want nil stack, got %v", stack)
		}
	})

	t.Run("non-errdef error", func(t *testing.T) {
		err := errors.New("regular error")
		stack, ok := errdef.StackFrom(err)
		if ok {
			t.Error("want stack not to be found from non-errdef error")
		}
		if stack != nil {
			t.Errorf("want nil stack, got %v", stack)
		}
	})

	t.Run("errdef error with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		stack, ok := errdef.StackFrom(err)
		if !ok {
			t.Error("want stack to be found from errdef error")
		}
		if stack == nil {
			t.Error("want non-nil stack")
		}
		if stack.Len() == 0 {
			t.Error("want non-empty stack")
		}
	})

	t.Run("wrapped errdef error with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")
		wrapped := fmt.Errorf("wrapped: %w", err)

		stack, ok := errdef.StackFrom(wrapped)
		if !ok {
			t.Error("want stack to be found from wrapped errdef error")
		}
		if stack == nil {
			t.Error("want non-nil stack")
		}
		if stack.Len() == 0 {
			t.Error("want non-empty stack")
		}
	})
}

func TestUnwrapTreeFrom(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		nodes, ok := errdef.UnwrapTreeFrom(nil)
		if ok {
			t.Error("want tree not to be found from nil error")
		}
		if nodes != nil {
			t.Errorf("want nil nodes, got %v", nodes)
		}
	})

	t.Run("non-errdef error", func(t *testing.T) {
		err := errors.New("regular error")
		nodes, ok := errdef.UnwrapTreeFrom(err)
		if ok {
			t.Error("want tree not to be found from non-errdef error")
		}
		if nodes != nil {
			t.Errorf("want nil nodes, got %v", nodes)
		}
	})

	t.Run("errdef error with tree", func(t *testing.T) {
		cause := errors.New("cause error")
		def := errdef.Define("test_error")
		err := def.Wrap(cause)

		nodes, ok := errdef.UnwrapTreeFrom(err)
		if !ok {
			t.Error("want tree to be found from errdef error")
		}
		if nodes == nil {
			t.Error("want non-nil nodes")
		}
		if len(nodes) == 0 {
			t.Error("want non-empty nodes")
		}
	})

	t.Run("wrapped errdef error with tree", func(t *testing.T) {
		cause := errors.New("cause error")
		def := errdef.Define("test_error")
		err := def.Wrap(cause)
		wrapped := fmt.Errorf("wrapped: %w", err)

		nodes, ok := errdef.UnwrapTreeFrom(wrapped)
		if !ok {
			t.Error("want tree to be found from wrapped errdef error")
		}
		if nodes == nil {
			t.Error("want non-nil nodes")
		}
		if len(nodes) == 0 {
			t.Error("want non-empty nodes")
		}
	})

	t.Run("errdef error without tree", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test message")

		nodes, ok := errdef.UnwrapTreeFrom(err)
		if ok {
			t.Error("want tree not to be found when tree is empty")
		}
		if nodes != nil {
			t.Errorf("want nil nodes, got %v", nodes)
		}
	})
}
