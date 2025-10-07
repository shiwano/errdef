package resolver

import "github.com/shiwano/errdef"

type (
	// StrictResolver manages multiple error definitions and provides resolution
	// functionality based on Kind or Field criteria.
	StrictResolver struct {
		defs   []*errdef.Definition
		byKind map[errdef.Kind]*errdef.Definition
	}

	// FallbackResolver wraps a Resolver with fallback functionality,
	// returning a fallback definition when resolution fails.
	FallbackResolver struct {
		resolver *StrictResolver
		fallback *errdef.Definition
	}

	// Resolver provides error definitions for resolution.
	Resolver interface {
		// Definitions returns all definitions managed by the resolver.
		Definitions() []*errdef.Definition

		// ResolveKindStrict resolves a definition by its Kind.
		// Returns the definition and true if found, nil and false otherwise.
		ResolveKindStrict(kind errdef.Kind) (*errdef.Definition, bool)

		// ResolveFieldStrict resolves a definition by matching a field value.
		// Returns the first definition that has the specified field key with the exact value.
		ResolveFieldStrict(key errdef.FieldKey, want any) (*errdef.Definition, bool)

		// ResolveFieldStrictFunc resolves a definition using a custom field evaluation function.
		// Returns the first definition where the eq function returns true for the field value.
		ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (*errdef.Definition, bool)
	}
)

var (
	_ Resolver = (*StrictResolver)(nil)
	_ Resolver = (*FallbackResolver)(nil)
)

// New creates a new Resolver with the given definitions.
// If multiple definitions have the same Kind, the first one wins.
func New(defs ...*errdef.Definition) *StrictResolver {
	byKind := make(map[errdef.Kind]*errdef.Definition, len(defs))
	for _, d := range defs {
		if d == nil {
			continue
		}
		k := d.Kind()
		if _, exists := byKind[k]; exists {
			continue // First definition wins
		}
		byKind[k] = d
	}
	return &StrictResolver{
		defs:   defs,
		byKind: byKind,
	}
}

// Definitions returns all definitions managed by the resolver.
func (r *StrictResolver) Definitions() []*errdef.Definition {
	return r.defs[:]
}

// WithFallback creates a new FallbackResolver that uses the given definition
// as a fallback when resolution fails.
func (r *StrictResolver) WithFallback(fallback *errdef.Definition) *FallbackResolver {
	return &FallbackResolver{
		resolver: r,
		fallback: fallback,
	}
}

// ResolveKindStrict implements Resolver.
func (r *StrictResolver) ResolveKindStrict(kind errdef.Kind) (*errdef.Definition, bool) {
	def, ok := r.byKind[kind]
	return def, ok
}

// ResolveFieldStrict implements Resolver.
func (r *StrictResolver) ResolveFieldStrict(key errdef.FieldKey, want any) (*errdef.Definition, bool) {
	return r.ResolveFieldStrictFunc(key, func(v errdef.FieldValue) bool {
		if fv, ok := want.(errdef.FieldValue); ok {
			return v.Equals(fv.Value())
		}
		return v.Equals(want)
	})
}

// ResolveFieldStrictFunc implements Resolver.
func (r *StrictResolver) ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (*errdef.Definition, bool) {
	for _, def := range r.defs {
		v, ok := def.Fields().Get(key)
		if !ok || !eq(v) {
			continue
		}

		return def, true // First definition wins
	}
	return nil, false
}

// Definitions returns all definitions managed by the resolver.
func (r *FallbackResolver) Definitions() []*errdef.Definition {
	defs := r.resolver.Definitions()
	defs = append(defs, r.fallback)
	return defs
}

// ResolveKind resolves a definition by its Kind.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveKind(kind errdef.Kind) *errdef.Definition {
	if def, ok := r.resolver.ResolveKindStrict(kind); ok {
		return def
	}
	return r.fallback
}

// ResolveField resolves a definition by matching a field value.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveField(key errdef.FieldKey, want any) *errdef.Definition {
	if def, ok := r.resolver.ResolveFieldStrict(key, want); ok {
		return def
	}
	return r.fallback
}

// ResolveFieldFunc resolves a definition using a custom field evaluation function.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveFieldFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) *errdef.Definition {
	if def, ok := r.resolver.ResolveFieldStrictFunc(key, eq); ok {
		return def
	}
	return r.fallback
}

// ResolveKindStrict implements Resolver.
func (r *FallbackResolver) ResolveKindStrict(kind errdef.Kind) (*errdef.Definition, bool) {
	return r.resolver.ResolveKindStrict(kind)
}

// ResolveFieldStrict implements Resolver.
func (r *FallbackResolver) ResolveFieldStrict(key errdef.FieldKey, want any) (*errdef.Definition, bool) {
	return r.resolver.ResolveFieldStrict(key, want)
}

// ResolveFieldStrictFunc implements Resolver.
func (r *FallbackResolver) ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (*errdef.Definition, bool) {
	return r.resolver.ResolveFieldStrictFunc(key, eq)
}

// Fallback returns the fallback definition.
func (r *FallbackResolver) Fallback() *errdef.Definition {
	return r.fallback
}
