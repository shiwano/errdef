package resolver_test

import (
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

func TestDefaultResolver_ResolveKindOrDefault(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	defaultDef := errdef.Define("default")

	r := resolver.New(def1, def2)
	defaultResolver := r.WithDefault(defaultDef)

	t.Run("resolves existing kind", func(t *testing.T) {
		result := defaultResolver.ResolveKindOrDefault("error1")

		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns default for non-existing kind", func(t *testing.T) {
		result := defaultResolver.ResolveKindOrDefault("non_existing")

		if result != defaultDef {
			t.Errorf("want default definition for non-existing kind, got %v", result)
		}
	})
}

func TestDefaultResolver_ResolveFieldOrDefault(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("value1"))
	def2 := errdef.Define("error2", ctor("value2"))
	defaultDef := errdef.Define("default", ctor("default_value"))

	r := resolver.New(def1, def2)
	defaultResolver := r.WithDefault(defaultDef)

	t.Run("resolves existing field", func(t *testing.T) {
		result := defaultResolver.ResolveFieldOrDefault(ctor.Key(), "value1")

		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns default for non-existing field value", func(t *testing.T) {
		result := defaultResolver.ResolveFieldOrDefault(ctor.Key(), "non_existing")

		if result != defaultDef {
			t.Errorf("want default definition for non-existing field value, got %v", result)
		}
	})

	t.Run("returns default for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := defaultResolver.ResolveFieldOrDefault(ctor2.Key(), "any_value")

		if result != defaultDef {
			t.Errorf("want default definition for field not in any definition, got %v", result)
		}
	})
}

func TestDefaultResolver_ResolveFieldFuncOrDefault(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("hello"))
	def2 := errdef.Define("error2", ctor("world"))
	defaultDef := errdef.Define("default", ctor("default"))

	r := resolver.New(def1, def2)
	defaultResolver := r.WithDefault(defaultDef)

	t.Run("resolves with custom function", func(t *testing.T) {
		result := defaultResolver.ResolveFieldFuncOrDefault(ctor.Key(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) == 5
		})

		if result != def1 && result != def2 {
			t.Errorf("want resolved definition to be def1 or def2, got %v", result)
		}
	})

	t.Run("returns default when no field matches", func(t *testing.T) {
		result := defaultResolver.ResolveFieldFuncOrDefault(ctor.Key(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) > 10
		})

		if result != defaultDef {
			t.Errorf("want default definition when no field matches, got %v", result)
		}
	})

	t.Run("returns default for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := defaultResolver.ResolveFieldFuncOrDefault(ctor2.Key(), func(v errdef.FieldValue) bool {
			return true
		})

		if result != defaultDef {
			t.Errorf("want default definition for field not in any definition, got %v", result)
		}
	})
}

func TestDefaultResolver_Default(t *testing.T) {
	def1 := errdef.Define("error1")
	defaultDef := errdef.Define("default")

	r := resolver.New(def1)
	defaultResolver := r.WithDefault(defaultDef)

	result := defaultResolver.Default()

	if result != defaultDef {
		t.Errorf("want default method to return default definition, got %v", result)
	}
}
