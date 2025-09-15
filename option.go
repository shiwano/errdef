package errdef

import "errors"

type (
	// Option represents a configuration option that can be applied to error definitions.
	Option interface {
		// ApplyOption applies this option to the given applier.
		ApplyOption(o OptionApplier)
	}

	// OptionApplier provides methods for applying options to error definitions.
	OptionApplier interface {
		// SetField sets a field value.
		SetField(key FieldKey, value any)
		// DisableTrace disables stack trace collection.
		DisableTrace()
		// AddStackSkip adds frames to skip during stack trace collection.
		AddStackSkip(skip int)
		// SetBoundary marks this error as a boundary in the error chain.
		SetBoundary()
		// SetFormatter sets a custom error formatter.
		SetFormatter(formatter ErrorFormatter)
		// SetJSONMarshaler sets a custom JSON marshaler.
		SetJSONMarshaler(marshaler ErrorJSONMarshaler)
	}

	// FieldOptionConstructor creates an Option that sets a field value.
	FieldOptionConstructor[T any] func(value T) Option
	// FieldOptionExtractor extracts a field value from an error.
	FieldOptionExtractor[T any] func(err error) (T, bool)

	// FieldOptionConstructorDefault creates an Option with a default value when called with no arguments.
	FieldOptionConstructorDefault[T any] func() Option
	// FieldOptionExtractorSingleReturn extracts a field value from an error, returning only the value.
	FieldOptionExtractorSingleReturn[T any] func(err error) T

	optionApplier struct {
		def *Definition
	}

	field struct {
		key   FieldKey
		value any
	}

	noTrace struct{}

	stackSkip struct {
		skip int
	}

	boundary struct{}

	formatter struct {
		formatter ErrorFormatter
	}

	jsonMarshaler struct {
		marshaler ErrorJSONMarshaler
	}
)

// Default creates a field option constructor that sets a default value.
func (f FieldOptionConstructor[T]) Default(value T) FieldOptionConstructorDefault[T] {
	return func() Option {
		return f(value)
	}
}

// SingleReturn creates a field extractor that returns only the value, ignoring the boolean.
func (f FieldOptionExtractor[T]) SingleReturn() FieldOptionExtractorSingleReturn[T] {
	return func(err error) T {
		val, _ := f(err)
		return val
	}
}

func fieldValueFrom[T any](err error, key FieldKey) (T, bool) {
	var e *definedError
	if errors.As(err, &e) {
		if v, found := e.def.fields.Get(key); found {
			if tv, ok := v.(T); ok {
				return tv, true
			}
		}
	}
	var zero T
	return zero, false
}

func (a *optionApplier) SetField(key FieldKey, value any) {
	a.def.fields.set(key, value)
}

func (a *optionApplier) DisableTrace() {
	a.def.noTrace = true
}

func (a *optionApplier) AddStackSkip(skip int) {
	a.def.stackSkip += skip
}

func (a *optionApplier) SetBoundary() {
	a.def.boundary = true
}

func (a *optionApplier) SetFormatter(formatter ErrorFormatter) {
	a.def.formatter = formatter
}

func (a *optionApplier) SetJSONMarshaler(marshaler ErrorJSONMarshaler) {
	a.def.jsonMarshaler = marshaler
}

func applyOptionsTo(d *Definition, opts []Option) {
	if len(opts) == 0 {
		return
	}

	if d.fields == nil {
		d.fields = newFields()
	}
	a := &optionApplier{def: d}
	for _, opt := range opts {
		opt.ApplyOption(a)
	}
}

func (o *field) ApplyOption(a OptionApplier) {
	a.SetField(o.key, o.value)
}

func (o *noTrace) ApplyOption(a OptionApplier) {
	a.DisableTrace()
}

func (o *stackSkip) ApplyOption(a OptionApplier) {
	a.AddStackSkip(o.skip)
}

func (o *boundary) ApplyOption(a OptionApplier) {
	a.SetBoundary()
}

func (o *formatter) ApplyOption(a OptionApplier) {
	a.SetFormatter(o.formatter)
}

func (o *jsonMarshaler) ApplyOption(a OptionApplier) {
	a.SetJSONMarshaler(o.marshaler)
}
