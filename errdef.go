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
		kind: kind,
	}
	def.root = def
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
	opts = append(opts, StackSkip(1))
	return Define("", opts...).New(msg)
}

// Wrap wraps an existing error with additional options.
// Returns nil if cause is nil.
func Wrap(cause error, opts ...Option) error {
	if cause == nil {
		return nil
	}
	opts = append(opts, StackSkip(1))
	return Define("", opts...).Wrap(cause)
}

// CapturePanic captures a panic value and converts it to an error with the given options.
// If errPtr is nil or panicValue is nil, this function does nothing.
// The resulting error implements PanicError interface to preserve the original panic value.
func CapturePanic(errPtr *error, panicValue any, opts ...Option) {
	if panicValue == nil || errPtr == nil {
		return
	}
	opts = append(opts, StackSkip(1))
	Define("", opts...).CapturePanic(errPtr, panicValue)
}
