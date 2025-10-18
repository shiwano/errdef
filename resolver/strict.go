package resolver

import (
	"slices"

	"github.com/shiwano/errdef"
)

// StrictResolver manages multiple error definitions and provides resolution
// functionality based on Kind or Field criteria.
type StrictResolver struct {
	defs   []errdef.Definition
	byKind map[errdef.Kind]errdef.Definition
}

var _ Resolver = (*StrictResolver)(nil)

// WithFallback creates a new FallbackResolver that uses the given definition
// as a fallback when resolution fails.
func (r *StrictResolver) WithFallback(fallback errdef.Definition) *FallbackResolver {
	allDefs := append(r.Definitions(), fallback)
	return &FallbackResolver{
		resolver: New(allDefs...),
		fallback: fallback,
	}
}

// Definitions implements Resolver.
func (r *StrictResolver) Definitions() []errdef.Definition {
	return slices.Clone(r.defs)
}

// ResolveKindStrict implements Resolver.
func (r *StrictResolver) ResolveKindStrict(kind errdef.Kind) (errdef.Definition, bool) {
	def, ok := r.byKind[kind]
	return def, ok
}

// ResolveFieldStrict implements Resolver.
func (r *StrictResolver) ResolveFieldStrict(key errdef.FieldKey, want any) (errdef.Definition, bool) {
	return r.ResolveFieldStrictFunc(key, func(v errdef.FieldValue) bool {
		if fv, ok := want.(errdef.FieldValue); ok {
			return v.Equal(fv.Value())
		}
		return v.Equal(want)
	})
}

// ResolveFieldStrictFunc implements Resolver.
func (r *StrictResolver) ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (errdef.Definition, bool) {
	for _, def := range r.defs {
		v, ok := def.Fields().Get(key)
		if !ok || !eq(v) {
			continue
		}

		return def, true // First definition wins
	}
	return nil, false
}
