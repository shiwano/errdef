package resolver

import (
	"slices"

	"github.com/shiwano/errdef"
)

// Resolver provides error definitions for resolution.
type Resolver interface {
	// ResolveKindStrict resolves a definition by its Kind.
	// Returns the definition and true if found, nil and false otherwise.
	ResolveKindStrict(kind errdef.Kind) (errdef.Definition, bool)
	// ResolveFieldStrict resolves a definition by matching a field value.
	// Returns the first definition that has the specified field key with the exact value.
	ResolveFieldStrict(key errdef.FieldKey, want any) (errdef.Definition, bool)
	// ResolveFieldStrictFunc resolves a definition using a custom field evaluation function.
	// Returns the first definition where the eq function returns true for the field value.
	ResolveFieldStrictFunc(key errdef.FieldKey, eq func(v errdef.FieldValue) bool) (errdef.Definition, bool)
}

// New creates a new Resolver with the given definitions.
// If multiple definitions have the same Kind, the first one wins.
func New(defs ...errdef.Definition) *StrictResolver {
	defs = slices.CompactFunc(defs, func(a, b errdef.Definition) bool {
		return a == b
	})

	byKind := make(map[errdef.Kind]errdef.Definition, len(defs))
	for _, d := range defs {
		k := d.Kind()
		if _, exists := byKind[k]; !exists {
			byKind[k] = d
		}
	}

	return &StrictResolver{
		defs:   defs,
		byKind: byKind,
	}
}
