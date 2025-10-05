package errdef_test

import (
	"testing"

	"github.com/shiwano/errdef"
)

func TestNewResolver(t *testing.T) {
	t.Run("creates resolver with definitions", func(t *testing.T) {
		def1 := errdef.Define("error1")
		def2 := errdef.Define("error2")

		resolver := errdef.NewResolver(def1, def2)

		if resolver == nil {
			t.Fatal("want resolver to be created")
		}
	})

	t.Run("skips nil definitions", func(t *testing.T) {
		def1 := errdef.Define("error1")
		var nilDef *errdef.Definition

		resolver := errdef.NewResolver(def1, nilDef)

		if resolver == nil {
			t.Fatal("want resolver to be created")
		}

		result, ok := resolver.ResolveKind("error1")
		if !ok {
			t.Fatal("want to resolve existing kind")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("first definition wins for duplicate kinds", func(t *testing.T) {
		def1 := errdef.Define("error1")
		def2 := errdef.Define("error1")

		resolver := errdef.NewResolver(def1, def2)

		result, ok := resolver.ResolveKind("error1")
		if !ok {
			t.Fatal("want to resolve existing kind")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be first def1, got %v", result)
		}
	})
}

func TestResolver_Definitions(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	def3 := errdef.Define("error3")

	resolver := errdef.NewResolver(def1, def2, def3)
	defs := resolver.Definitions()

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

func TestResolver_WithFallback(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	fallbackDef := errdef.Define("fallback")

	resolver := errdef.NewResolver(def1, def2)
	fallbackResolver := resolver.WithFallback(fallbackDef)

	if fallbackResolver == nil {
		t.Fatal("want fallback resolver to be created")
	}

	if fallbackResolver.Fallback() != fallbackDef {
		t.Errorf("want fallback definition to be set correctly, got %v", fallbackResolver.Fallback())
	}
}

func TestResolver_ResolveKind(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	resolver := errdef.NewResolver(def1, def2)

	t.Run("resolves existing kind", func(t *testing.T) {
		result, ok := resolver.ResolveKind("error1")

		if !ok {
			t.Fatal("want to resolve existing kind")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns false for non-existing kind", func(t *testing.T) {
		result, ok := resolver.ResolveKind("non_existing")

		if ok {
			t.Fatal("want not to resolve non-existing kind")
		}
		if result != nil {
			t.Errorf("want nil result for non-existing kind, got %v", result)
		}
	})
}

func TestResolver_ResolveField(t *testing.T) {
	ctor1, _ := errdef.DefineField[string]("test_field")
	ctor2, _ := errdef.DefineField[int]("number_field")

	def1 := errdef.Define("error1", ctor1("value1"))
	def2 := errdef.Define("error2", ctor1("value2"), ctor2(100))
	def3 := errdef.Define("error3", ctor2(200))

	resolver := errdef.NewResolver(def1, def2, def3)

	t.Run("resolves by string field", func(t *testing.T) {
		result, ok := resolver.ResolveField(ctor1.FieldKey(), "value1")

		if !ok {
			t.Fatal("want to resolve by string field")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("resolves by int field", func(t *testing.T) {
		result, ok := resolver.ResolveField(ctor2.FieldKey(), 100)

		if !ok {
			t.Fatal("want to resolve by int field")
		}
		if result != def2 {
			t.Errorf("want resolved definition to be def2, got %v", result)
		}
	})

	t.Run("returns false for non-existing field", func(t *testing.T) {
		result, ok := resolver.ResolveField(ctor1.FieldKey(), "non_existing")

		if ok {
			t.Fatal("want not to resolve non-existing field value")
		}
		if result != nil {
			t.Errorf("want nil result for non-existing field value, got %v", result)
		}
	})

	t.Run("returns false for field not in any definition", func(t *testing.T) {
		ctor3, _ := errdef.DefineField[string]("missing_field")

		result, ok := resolver.ResolveField(ctor3.FieldKey(), "any_value")

		if ok {
			t.Fatal("want not to resolve field not in any definition")
		}
		if result != nil {
			t.Errorf("want nil result for field not in any definition, got %v", result)
		}
	})

	t.Run("resolves by slice field", func(t *testing.T) {
		sliceCtor, _ := errdef.DefineField[[]string]("slice_field")

		defWithSlice1 := errdef.Define("error_with_slice1", sliceCtor([]string{"a", "b"}))
		defWithSlice2 := errdef.Define("error_with_slice2", sliceCtor([]string{"c", "d"}))

		sliceResolver := errdef.NewResolver(defWithSlice1, defWithSlice2)

		result, ok := sliceResolver.ResolveField(sliceCtor.FieldKey(), []string{"a", "b"})

		if !ok {
			t.Fatal("want to resolve by slice field")
		}
		if result != defWithSlice1 {
			t.Errorf("want resolved definition to be defWithSlice1, got %v", result)
		}
	})

	t.Run("resolves by map field", func(t *testing.T) {
		mapCtor, _ := errdef.DefineField[map[string]int]("map_field")

		defWithMap1 := errdef.Define("error_with_map1", mapCtor(map[string]int{"key1": 1}))
		defWithMap2 := errdef.Define("error_with_map2", mapCtor(map[string]int{"key2": 2}))

		mapResolver := errdef.NewResolver(defWithMap1, defWithMap2)

		result, ok := mapResolver.ResolveField(mapCtor.FieldKey(), map[string]int{"key1": 1})

		if !ok {
			t.Fatal("want to resolve by map field")
		}
		if result != defWithMap1 {
			t.Errorf("want resolved definition to be defWithMap1, got %v", result)
		}
	})
}

func TestResolver_ResolveFieldFunc(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("hello"))
	def2 := errdef.Define("error2", ctor("world"))
	def3 := errdef.Define("error3")

	resolver := errdef.NewResolver(def1, def2, def3)

	t.Run("resolves with custom function", func(t *testing.T) {
		result, ok := resolver.ResolveFieldFunc(ctor.FieldKey(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) == 5
		})

		if !ok {
			t.Fatal("want to resolve with custom function")
		}
		if result != def1 && result != def2 {
			t.Errorf("want resolved definition to be def1 or def2, got %v", result)
		}
	})

	t.Run("returns false when no field matches", func(t *testing.T) {
		result, ok := resolver.ResolveFieldFunc(ctor.FieldKey(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) > 10
		})

		if ok {
			t.Fatal("want not to resolve when no field matches")
		}
		if result != nil {
			t.Errorf("want nil result when no field matches, got %v", result)
		}
	})

	t.Run("returns false for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result, ok := resolver.ResolveFieldFunc(ctor2.FieldKey(), func(v errdef.FieldValue) bool {
			return true
		})

		if ok {
			t.Fatal("want not to resolve field not in any definition")
		}
		if result != nil {
			t.Errorf("want nil result for field not in any definition, got %v", result)
		}
	})
}

func TestFallbackResolver_Definitions(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	def3 := errdef.Define("error3")

	resolver := errdef.NewResolver(def1, def2)
	fallbackResolver := resolver.WithFallback(def3)
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

	resolver := errdef.NewResolver(def1, def2)
	fallbackResolver := resolver.WithFallback(fallbackDef)

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

	resolver := errdef.NewResolver(def1, def2)
	fallbackResolver := resolver.WithFallback(fallbackDef)

	t.Run("resolves existing field", func(t *testing.T) {
		result := fallbackResolver.ResolveField(ctor.FieldKey(), "value1")

		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns fallback for non-existing field value", func(t *testing.T) {
		result := fallbackResolver.ResolveField(ctor.FieldKey(), "non_existing")

		if result != fallbackDef {
			t.Errorf("want fallback definition for non-existing field value, got %v", result)
		}
	})

	t.Run("returns fallback for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := fallbackResolver.ResolveField(ctor2.FieldKey(), "any_value")

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

	resolver := errdef.NewResolver(def1, def2)
	fallbackResolver := resolver.WithFallback(fallbackDef)

	t.Run("resolves with custom function", func(t *testing.T) {
		result := fallbackResolver.ResolveFieldFunc(ctor.FieldKey(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) == 5
		})

		if result != def1 && result != def2 {
			t.Errorf("want resolved definition to be def1 or def2, got %v", result)
		}
	})

	t.Run("returns fallback when no field matches", func(t *testing.T) {
		result := fallbackResolver.ResolveFieldFunc(ctor.FieldKey(), func(v errdef.FieldValue) bool {
			str, isString := v.Value().(string)
			return isString && len(str) > 10
		})

		if result != fallbackDef {
			t.Errorf("want fallback definition when no field matches, got %v", result)
		}
	})

	t.Run("returns fallback for field not in any definition", func(t *testing.T) {
		ctor2, _ := errdef.DefineField[string]("missing_field")

		result := fallbackResolver.ResolveFieldFunc(ctor2.FieldKey(), func(v errdef.FieldValue) bool {
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

	resolver := errdef.NewResolver(def1)
	fallbackResolver := resolver.WithFallback(fallbackDef)

	result := fallbackResolver.Fallback()

	if result != fallbackDef {
		t.Errorf("want fallback method to return fallback definition, got %v", result)
	}
}
