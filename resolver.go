package errdef

type (
	// Resolver manages multiple error definitions and provides resolution
	// functionality based on Kind or Field criteria.
	Resolver struct {
		defs   []*Definition
		byKind map[Kind]*Definition
	}

	// FallbackResolver wraps a Resolver with fallback functionality,
	// returning a fallback definition when resolution fails.
	FallbackResolver struct {
		resolver *Resolver
		fallback *Definition
	}

	resolver interface {
		Definitions() []*Definition
	}
)

var (
	_ resolver = (*Resolver)(nil)
	_ resolver = (*FallbackResolver)(nil)
)

// NewResolver creates a new Resolver with the given definitions.
// If multiple definitions have the same Kind, the first one wins.
func NewResolver(defs ...*Definition) *Resolver {
	byKind := make(map[Kind]*Definition, len(defs))
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
	return &Resolver{
		defs:   defs,
		byKind: byKind,
	}
}

// Definitions returns all definitions managed by the resolver.
func (r *Resolver) Definitions() []*Definition {
	return r.defs[:]
}

// WithFallback creates a new FallbackResolver that uses the given definition
// as a fallback when resolution fails.
func (r *Resolver) WithFallback(fallback *Definition) *FallbackResolver {
	return &FallbackResolver{
		resolver: r,
		fallback: fallback,
	}
}

// ResolveKind resolves a definition by its Kind.
// Returns the definition and true if found, nil and false otherwise.
func (r *Resolver) ResolveKind(kind Kind) (*Definition, bool) {
	def, ok := r.byKind[kind]
	return def, ok
}

// ResolveField resolves a definition by matching a field value.
// Returns the first definition that has the specified field key with the exact value.
func (r *Resolver) ResolveField(key FieldKey, want any) (*Definition, bool) {
	return r.ResolveFieldFunc(key, func(v FieldValue) bool {
		if fv, ok := want.(FieldValue); ok {
			return v.Equals(fv.Value())
		}
		return v.Equals(want)
	})
}

// ResolveFieldFunc resolves a definition using a custom field evaluation function.
// Returns the first definition where the eq function returns true for the field value.
func (r *Resolver) ResolveFieldFunc(key FieldKey, eq func(v FieldValue) bool) (*Definition, bool) {
	for _, def := range r.defs {
		v, ok := def.fields.Get(key)
		if !ok || !eq(v) {
			continue
		}

		return def, true // First definition wins
	}
	return nil, false
}

// Definitions returns all definitions managed by the resolver.
func (r *FallbackResolver) Definitions() []*Definition {
	defs := r.resolver.Definitions()
	defs = append(defs, r.fallback)
	return defs
}

// ResolveKind resolves a definition by its Kind.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveKind(kind Kind) *Definition {
	if def, ok := r.resolver.ResolveKind(kind); ok {
		return def
	}
	return r.fallback
}

// ResolveField resolves a definition by matching a field value.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveField(key FieldKey, want any) *Definition {
	if def, ok := r.resolver.ResolveField(key, want); ok {
		return def
	}
	return r.fallback
}

// ResolveFieldFunc resolves a definition using a custom field evaluation function.
// Returns the fallback definition if resolution fails.
func (r *FallbackResolver) ResolveFieldFunc(key FieldKey, eq func(v FieldValue) bool) *Definition {
	if def, ok := r.resolver.ResolveFieldFunc(key, eq); ok {
		return def
	}
	return r.fallback
}

// Fallback returns the fallback definition.
func (r *FallbackResolver) Fallback() *Definition {
	return r.fallback
}
