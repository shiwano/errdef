package resolver_test

import (
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

func TestFallbackResolver_Definitions(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	def3 := errdef.Define("error3")

	r := resolver.New(def1, def2)
	fallbackResolver := r.WithFallback(def3)
	defs := fallbackResolver.Definitions()

	if len(defs) != 3 {
		t.Fatalf("want 3 definitions, got %d", len(defs))
	}

	if defs[0] != def1 {
		t.Errorf("want first definition to be def1, got %v", defs[0])
	}
	if defs[1] != def2 {
		t.Errorf("want second definition to be def2, got %v", defs[1])
	}
	if defs[2] != def3 {
		t.Errorf("want third definition to be def3, got %v", defs[2])
	}
}

func TestFallbackResolver_ResolveKind(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	fallbackDef := errdef.Define("fallback")

	r := resolver.New(def1, def2)
	fallbackResolver := r.WithFallback(fallbackDef)

	t.Run("resolves existing kind", func(t *testing.T) {
		result := fallbackResolver.ResolveKind("error1")

		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns fallback for non-existing kind", func(t *testing.T) {
		result := fallbackResolver.ResolveKind("non_existing")

		if result != fallbackDef {
			t.Errorf("want fallback definition for non-existing kind, got %v", result)
		}
	})
}

func TestFallbackResolver_ResolveField(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("value1"))
	def2 := errdef.Define("error2", ctor("value2"))
	fallbackDef := errdef.Define("fallback", ctor("fallback_value"))

	r := resolver.New(def1, def2)
	fallbackResolver := r.WithFallback(fallbackDef)

	t.Run("resolves existing field", func(t *testing.T) {
		result := fallbackResolver.ResolveField(ctor.Key(), "value1")

		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns fallback for non-existing field value", func(t *testing.T) {
		result := fallbackResolver.ResolveField(ctor.Key(), "non_existing")

		if result != fallbackDef {
			t.Errorf("want fallback definition for non-existing field value, got %v", result)
		}
	})

	t.Run("returns fallback for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := fallbackResolver.ResolveField(ctor2.Key(), "any_value")

		if result != fallbackDef {
			t.Errorf("want fallback definition for field not in any definition, got %v", result)
		}
	})
}

func TestFallbackResolver_ResolveFieldFunc(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("hello"))
	def2 := errdef.Define("error2", ctor("world"))
	fallbackDef := errdef.Define("fallback", ctor("fallback"))

	r := resolver.New(def1, def2)
	fallbackResolver := r.WithFallback(fallbackDef)

	t.Run("resolves with custom function", func(t *testing.T) {
		result := fallbackResolver.ResolveFieldFunc(ctor.Key(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) == 5
		})

		if result != def1 && result != def2 {
			t.Errorf("want resolved definition to be def1 or def2, got %v", result)
		}
	})

	t.Run("returns fallback when no field matches", func(t *testing.T) {
		result := fallbackResolver.ResolveFieldFunc(ctor.Key(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) > 10
		})

		if result != fallbackDef {
			t.Errorf("want fallback definition when no field matches, got %v", result)
		}
	})

	t.Run("returns fallback for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := fallbackResolver.ResolveFieldFunc(ctor2.Key(), func(v errdef.FieldValue) bool {
			return true
		})

		if result != fallbackDef {
			t.Errorf("want fallback definition for field not in any definition, got %v", result)
		}
	})
}

func TestFallbackResolver_Fallback(t *testing.T) {
	def1 := errdef.Define("error1")
	fallbackDef := errdef.Define("fallback")

	r := resolver.New(def1)
	fallbackResolver := r.WithFallback(fallbackDef)

	result := fallbackResolver.Fallback()

	if result != fallbackDef {
		t.Errorf("want fallback method to return fallback definition, got %v", result)
	}
}
