package resolver

import "github.com/shiwano/errdef"

// DefaultResolver wraps a Resolver with default functionality,
// returning a default definition when resolution fails.
type DefaultResolver struct {
	resolver   Resolver
	defaultDef errdef.Definition
}

var _ Resolver = (*DefaultResolver)(nil)

// ResolveKindOrDefault resolves a definition by its Kind.
// Returns the default definition if resolution fails.
func (r *DefaultResolver) ResolveKindOrDefault(kind errdef.Kind) errdef.Definition {
	if def, ok := r.resolver.ResolveKind(kind); ok {
		return def
	}
	return r.defaultDef
}

// ResolveFieldOrDefault resolves a definition by matching a field value.
// Returns the default definition if resolution fails.
func (r *DefaultResolver) ResolveFieldOrDefault(key errdef.FieldKey, want any) errdef.Definition {
	if def, ok := r.resolver.ResolveField(key, want); ok {
		return def
	}
	return r.defaultDef
}

// ResolveFieldFuncOrDefault resolves a definition using a custom field evaluation function.
// Returns the default definition if resolution fails.
func (r *DefaultResolver) ResolveFieldFuncOrDefault(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) errdef.Definition {
	if def, ok := r.resolver.ResolveFieldFunc(key, eq); ok {
		return def
	}
	return r.defaultDef
}

// Default returns the default definition.
func (r *DefaultResolver) Default() errdef.Definition {
	return r.defaultDef
}

// ResolveKind implements Resolver.
func (r *DefaultResolver) ResolveKind(kind errdef.Kind) (errdef.Definition, bool) {
	return r.resolver.ResolveKind(kind)
}

// ResolveField implements Resolver.
func (r *DefaultResolver) ResolveField(key errdef.FieldKey, want any) (errdef.Definition, bool) {
	return r.resolver.ResolveField(key, want)
}

// ResolveFieldFunc implements Resolver.
func (r *DefaultResolver) ResolveFieldFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (errdef.Definition, bool) {
	return r.resolver.ResolveFieldFunc(key, eq)
}
