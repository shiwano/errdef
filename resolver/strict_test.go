package resolver_test

import (
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

func TestStrictResolver_Definitions(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	def3 := errdef.Define("error3")

	r := resolver.New(def1, def2, def3)
	defs := r.Definitions()

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

func TestStrictResolver_WithFallback(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	fallbackDef := errdef.Define("fallback")

	r := resolver.New(def1, def2)
	fallbackResolver := r.WithFallback(fallbackDef)

	if fallbackResolver == nil {
		t.Fatal("want fallback resolver to be created")
	}

	if fallbackResolver.Fallback() != fallbackDef {
		t.Errorf("want fallback definition to be set correctly, got %v", fallbackResolver.Fallback())
	}
}

func TestStrictResolver_ResolveKind(t *testing.T) {
	def1 := errdef.Define("error1")
	def2 := errdef.Define("error2")
	r := resolver.New(def1, def2)

	t.Run("resolves existing kind", func(t *testing.T) {
		result, ok := r.ResolveKindStrict("error1")

		if !ok {
			t.Fatal("want to resolve existing kind")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("returns false for non-existing kind", func(t *testing.T) {
		result, ok := r.ResolveKindStrict("non_existing")

		if ok {
			t.Fatal("want not to resolve non-existing kind")
		}
		if result != nil {
			t.Errorf("want nil result for non-existing kind, got %v", result)
		}
	})
}

func TestStrictResolver_ResolveField(t *testing.T) {
	ctor1, _ := errdef.DefineField[string]("test_field")
	ctor2, _ := errdef.DefineField[int]("number_field")

	def1 := errdef.Define("error1", ctor1("value1"))
	def2 := errdef.Define("error2", ctor1("value2"), ctor2(100))
	def3 := errdef.Define("error3", ctor2(200))

	r := resolver.New(def1, def2, def3)

	t.Run("resolves by string field", func(t *testing.T) {
		result, ok := r.ResolveFieldStrict(ctor1.Key(), "value1")

		if !ok {
			t.Fatal("want to resolve by string field")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be def1, got %v", result)
		}
	})

	t.Run("resolves by int field", func(t *testing.T) {
		result, ok := r.ResolveFieldStrict(ctor2.Key(), 100)

		if !ok {
			t.Fatal("want to resolve by int field")
		}
		if result != def2 {
			t.Errorf("want resolved definition to be def2, got %v", result)
		}
	})

	t.Run("returns false for non-existing field", func(t *testing.T) {
		result, ok := r.ResolveFieldStrict(ctor1.Key(), "non_existing")

		if ok {
			t.Fatal("want not to resolve non-existing field value")
		}
		if result != nil {
			t.Errorf("want nil result for non-existing field value, got %v", result)
		}
	})

	t.Run("returns false for field not in any definition", func(t *testing.T) {
		ctor3, _ := errdef.DefineField[string]("missing_field")

		result, ok := r.ResolveFieldStrict(ctor3.Key(), "any_value")

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

		sliceResolver := resolver.New(defWithSlice1, defWithSlice2)

		result, ok := sliceResolver.ResolveFieldStrict(sliceCtor.Key(), []string{"a", "b"})

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

		mapResolver := resolver.New(defWithMap1, defWithMap2)

		result, ok := mapResolver.ResolveFieldStrict(mapCtor.Key(), map[string]int{"key1": 1})

		if !ok {
			t.Fatal("want to resolve by map field")
		}
		if result != defWithMap1 {
			t.Errorf("want resolved definition to be defWithMap1, got %v", result)
		}
	})
}

func TestStrictResolver_ResolveFieldFunc(t *testing.T) {
	ctor, _ := errdef.DefineField[string]("test_field")

	def1 := errdef.Define("error1", ctor("hello"))
	def2 := errdef.Define("error2", ctor("world"))
	def3 := errdef.Define("error3")

	r := resolver.New(def1, def2, def3)

	t.Run("resolves with custom function", func(t *testing.T) {
		result, ok := r.ResolveFieldStrictFunc(ctor.Key(), func(v errdef.FieldValue) bool {
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
		result, ok := r.ResolveFieldStrictFunc(ctor.Key(), func(v errdef.FieldValue) bool {
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

		result, ok := r.ResolveFieldStrictFunc(ctor2.Key(), func(v errdef.FieldValue) bool {
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
