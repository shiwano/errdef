package resolver_test

import (
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

func TestNew(t *testing.T) {
	t.Run("creates resolver with definitions", func(t *testing.T) {
		def1 := errdef.Define("error1")
		def2 := errdef.Define("error2")

		r := resolver.New(def1, def2)

		if r == nil {
			t.Fatal("want resolver to be created")
		}
	})

	t.Run("first definition wins for duplicate kinds", func(t *testing.T) {
		def1 := errdef.Define("error1")
		def2 := errdef.Define("error1")

		r := resolver.New(def1, def2)

		result, ok := r.ResolveKind("error1")
		if !ok {
			t.Fatal("want to resolve existing kind")
		}
		if result != def1 {
			t.Errorf("want resolved definition to be first def1, got %v", result)
		}
	})
}
