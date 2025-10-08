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
