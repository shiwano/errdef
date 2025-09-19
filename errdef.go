package errdef

// Kind represents the type of error.
type Kind string

// Define creates a new error definition with the specified kind and options.
func Define(kind Kind, opts ...Option) *Definition {
	def := &Definition{
		kind: kind,
	}
	def.applyOptions(opts)
	return def
}

// DefineField creates a field option constructor and extractor for the given field name.
// The constructor can be used to set a field value in error options,
// and the extractor can be used to retrieve the field value from errors.
func DefineField[T any](name string) (FieldOptionConstructor[T], FieldOptionExtractor[T]) {
	k := &fieldKey{name: name}
	constructor := func(value T) Option {
		return &field{key: k, value: value}
	}
	extractor := func(err error) (T, bool) {
		return fieldValueFrom[T](err, k)
	}
	return constructor, extractor
}

// New creates a new error with the given message and options.
func New(msg string, opts ...Option) error {
	opts = append(opts, &stackSkip{skip: 1})
	return Define("", opts...).New(msg)
}

// Wrap wraps an existing error with additional options.
// Returns nil if cause is nil.
func Wrap(cause error, opts ...Option) error {
	opts = append(opts, &stackSkip{skip: 1})
	return Define("", opts...).Wrap(cause)
}
