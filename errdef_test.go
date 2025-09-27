package errdef_test

import (
	"errors"
	"testing"

	"github.com/shiwano/errdef"
)

func TestDefine(t *testing.T) {
	t.Run("basic kind", func(t *testing.T) {
		kind := errdef.Kind("test_error")
		def := errdef.Define(kind)

		if def.Kind() != kind {
			t.Errorf("want kind %v, got %v", kind, def.Kind())
		}
	})

	t.Run("empty kind", func(t *testing.T) {
		def := errdef.Define("")

		if def.Kind() != "" {
			t.Errorf("want empty kind, got %v", def.Kind())
		}
	})
}

func TestDefineField(t *testing.T) {
	t.Run("constructor and extractor", func(t *testing.T) {
		constructor, extractor := errdef.DefineField[string]("test_field")

		def := errdef.Define("test_error", constructor("test_value"))
		err := def.New("test error")

		value, found := extractor(err)
		if !found {
			t.Error("want field to be found")
		}
		if value != "test_value" {
			t.Errorf("want value %q, got %q", "test_value", value)
		}
	})

	t.Run("extractor with wrong value type", func(t *testing.T) {
		type valueType string

		constructor, _ := errdef.DefineField[string]("test_field")
		_, extractor := errdef.DefineField[valueType]("test_field")

		def := errdef.Define("test_error", constructor("test_value"))
		err := def.New("test error")

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor with wrong key type", func(t *testing.T) {
		constructor, _ := errdef.DefineField[string]("test_field")
		_, extractor := errdef.DefineField[string]("test_field")

		def := errdef.Define("test_error", constructor("test_value"))
		err := def.New("test error")

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found with wrong type")
		}
	})

	t.Run("extractor on non-errdef error", func(t *testing.T) {
		_, extractor := errdef.DefineField[string]("test_field")

		err := errors.New("regular error")

		_, found := extractor(err)
		if found {
			t.Error("want field not to be found on non-errdef error")
		}
	})
}
