package resolver

import (
	"github.com/shiwano/errdef"
)

// StrictResolver manages multiple error definitions and provides resolution
// functionality based on Kind or Field criteria.
type StrictResolver struct {
	defs   []errdef.Definition
	byKind map[errdef.Kind]errdef.Definition
}

var _ Resolver = (*StrictResolver)(nil)

// WithDefault creates a new DefaultResolver that uses the given definition
// as a default when resolution fails.
func (r *StrictResolver) WithDefault(defaultDef errdef.Definition) *DefaultResolver {
	return &DefaultResolver{
		resolver:   r,
		defaultDef: defaultDef,
	}
}

// ResolveKind implements Resolver.
func (r *StrictResolver) ResolveKind(kind errdef.Kind) (errdef.Definition, bool) {
	def, ok := r.byKind[kind]
	return def, ok
}

// ResolveField implements Resolver.
func (r *StrictResolver) ResolveField(key errdef.FieldKey, want any) (errdef.Definition, bool) {
	return r.ResolveFieldFunc(key, func(v errdef.FieldValue) bool {
		if fv, ok := want.(errdef.FieldValue); ok {
			return v.Equal(fv.Value())
		}
		return v.Equal(want)
	})
}

// ResolveFieldFunc implements Resolver.
func (r *StrictResolver) ResolveFieldFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (errdef.Definition, bool) {
	for _, def := range r.defs {
		v, ok := def.Fields().Get(key)
		if !ok || !eq(v) {
			continue
		}

		return def, true // First definition wins
	}
	return nil, false
}
