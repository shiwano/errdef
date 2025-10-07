package errdef

// Kind is a human-readable string that represents the type of an error.
// It is primarily used for classification and identification in structured logs,
// metrics, and API responses.
type Kind string

// Define creates a new error definition with the specified kind and options.
//
// NOTE:
// The error identity check performed by errors.Is relies on an internal identifier
// associated with each Definition instance, not on the string value of Kind.
//
// While this means that using the same Kind string for different definitions
// will not cause incorrect identity checks, it is strongly recommended to use
// a unique Kind value across your application to prevent confusion in logs and
// monitoring tools.
func Define(kind Kind, opts ...Option) *Definition {
	def := &Definition{
		kind:   kind,
		fields: newFields(),
	}
	def.rootDef = nil
	def.applyOptions(opts)
	return def
}

// DefineField creates a field option constructor and extractor for the given field name.
// The constructor can be used to set a field value in error options,
// and the extractor can be used to retrieve the field value from errors.
//
// NOTE:
// The identity of a field is determined by the returned constructor and extractor
// instances, not by the provided name string. This ensures that fields created
// by different calls to DefineField, even with the same name, are distinct
// and do not collide.
//
// The name string is used as the key when an error's fields are serialized
// (e.g., to JSON). To avoid ambiguity in logs and other serialized representations,
// it is strongly recommended to use a unique name for each defined field.
func DefineField[T any](name string) (FieldConstructor[T], FieldExtractor[T]) {
	k := &fieldKey[T]{name: name}
	constructor := func(value T) Option {
		return &field[T]{key: k, value: value}
	}
	extractor := func(err error) (T, bool) {
		return fieldValueFrom[T](err, k)
	}
	return constructor, extractor
}
