package resolver

import "github.com/shiwano/errdef"

// FallbackResolver wraps a Resolver with fallback functionality,
// returning a fallback definition when resolution fails.
type FallbackResolver struct {
	resolver Resolver
	fallback errdef.Definition
}

var _ Resolver = (*FallbackResolver)(nil)

// ResolveKind resolves a definition by its Kind.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveKind(kind errdef.Kind) errdef.Definition {
	if def, ok := r.resolver.ResolveKindStrict(kind); ok {
		return def
	}
	return r.fallback
}

// ResolveField resolves a definition by matching a field value.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveField(key errdef.FieldKey, want any) errdef.Definition {
	if def, ok := r.resolver.ResolveFieldStrict(key, want); ok {
		return def
	}
	return r.fallback
}

// ResolveFieldFunc resolves a definition using a custom field evaluation function.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveFieldFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) errdef.Definition {
	if def, ok := r.resolver.ResolveFieldStrictFunc(key, eq); ok {
		return def
	}
	return r.fallback
}

// Fallback returns the fallback definition.
func (r *FallbackResolver) Fallback() errdef.Definition {
	return r.fallback
}

// Definitions implements Resolver.
func (r *FallbackResolver) Definitions() []errdef.Definition {
	return r.resolver.Definitions()
}

// ResolveKindStrict implements Resolver.
func (r *FallbackResolver) ResolveKindStrict(kind errdef.Kind) (errdef.Definition, bool) {
	return r.resolver.ResolveKindStrict(kind)
}

// ResolveFieldStrict implements Resolver.
func (r *FallbackResolver) ResolveFieldStrict(key errdef.FieldKey, want any) (errdef.Definition, bool) {
	return r.resolver.ResolveFieldStrict(key, want)
}

// ResolveFieldStrictFunc implements Resolver.
func (r *FallbackResolver) ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (errdef.Definition, bool) {
	return r.resolver.ResolveFieldStrictFunc(key, eq)
}
