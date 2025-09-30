package errdef

import (
	"context"
	"net/http"
)

type (
	// Option represents a configuration option that can be applied to error definitions.
	Option interface {
		// ApplyOption applies this option to the given applier.
		ApplyOption(o OptionApplier)
	}

	// OptionApplier provides methods for applying options to error definitions.
	OptionApplier interface {
		// SetField sets a field value.
		SetField(key FieldKey, value FieldValue)
		// DisableTrace disables stack trace collection.
		DisableTrace()
		// AddStackSkip adds frames to skip during stack trace collection.
		AddStackSkip(skip int)
		// SetStackDepth sets the absolute depth for stack trace collection.
		SetStackDepth(depth int)
		// SetBoundary marks this error as a boundary in the error chain.
		SetBoundary()
		// SetFormatter sets a custom error formatter.
		SetFormatter(formatter ErrorFormatter)
		// SetJSONMarshaler sets a custom JSON marshaler.
		SetJSONMarshaler(marshaler ErrorJSONMarshaler)
		// SetLogValuer sets a custom slog.Value converter.
		SetLogValuer(valuer ErrorLogValuer)
	}

	// FieldOptionConstructor creates an Option that sets a field value.
	FieldOptionConstructor[T any] func(value T) Option

	// FieldOptionExtractor extracts a field value from an error.
	FieldOptionExtractor[T any] func(err error) (T, bool)

	// FieldOptionConstructorNoArgs creates an Option with a default value when called with no arguments.
	FieldOptionConstructorNoArgs[T any] func() Option

	// FieldOptionExtractorSingleReturn extracts a field value from an error, returning only the value.
	FieldOptionExtractorSingleReturn[T any] func(err error) T

	optionApplier struct {
		def *Definition
	}

	field[T any] struct {
		key   FieldKey
		value T
	}

	noTrace struct{}

	stackSkip struct {
		skip int
	}

	stackDepth struct {
		depth int
	}

	boundary struct{}

	formatter struct {
		formatter ErrorFormatter
	}

	jsonMarshaler struct {
		marshaler ErrorJSONMarshaler
	}

	logValuer struct {
		valuer ErrorLogValuer
	}
)

// FieldKey returns the key associated with this constructor.
func (f FieldOptionConstructor[T]) FieldKey() FieldKey {
	var zero T
	return fieldKeyFromOption(f(zero))
}

// FieldKey returns the key associated with this constructor.
func (f FieldOptionConstructorNoArgs[T]) FieldKey() FieldKey {
	return fieldKeyFromOption(f())
}

// WithValue creates a field option constructor that sets a specific value.
func (f FieldOptionConstructor[T]) WithValue(value T) FieldOptionConstructorNoArgs[T] {
	return func() Option {
		return f(value)
	}
}

// WithValueFunc creates a field option constructor that sets a value using a function.
func (f FieldOptionConstructor[T]) WithValueFunc(fn func() T) FieldOptionConstructorNoArgs[T] {
	return func() Option {
		return f(fn())
	}
}

// WithContextFunc creates a field option constructor that sets a value using a function that takes a context.
func (f FieldOptionConstructor[T]) WithContextFunc(fn func(ctx context.Context) T) FieldOptionConstructor[context.Context] {
	return func(ctx context.Context) Option {
		val := fn(ctx)
		return f(val)
	}
}

// WithHTTPRequestFunc creates a field option constructor that sets a value using a function that takes an HTTP request.
func (f FieldOptionConstructor[T]) WithHTTPRequestFunc(fn func(r *http.Request) T) FieldOptionConstructor[*http.Request] {
	return func(r *http.Request) Option {
		val := fn(r)
		return f(val)
	}
}

// WithZero creates a field extractor that returns only the value, ignoring the boolean.
func (f FieldOptionExtractor[T]) WithZero() FieldOptionExtractorSingleReturn[T] {
	return func(err error) T {
		val, _ := f(err)
		return val
	}
}

// WithDefault creates a field extractor that returns a default value if the field is not found.
func (f FieldOptionExtractor[T]) WithDefault(value T) FieldOptionExtractorSingleReturn[T] {
	return func(err error) T {
		if val, ok := f(err); ok {
			return val
		}
		return value
	}
}

// WithFallback creates a field extractor that calls a function to obtain a value if the field is not found.
func (f FieldOptionExtractor[T]) WithFallback(fn func(err error) T) FieldOptionExtractorSingleReturn[T] {
	return func(err error) T {
		if val, ok := f(err); ok {
			return val
		}
		return fn(err)
	}
}

// OrZero extracts the field value from the error, returning the zero value if not found.
func (f FieldOptionExtractor[T]) OrZero(err error) T {
	return f.WithZero()(err)
}

// OrDefault extracts the field value from the error, returning a default value if not found.
func (f FieldOptionExtractor[T]) OrDefault(err error, value T) T {
	return f.WithDefault(value)(err)
}

// OrFallback extracts the field value from the error, calling a function to obtain a value if not found.
func (f FieldOptionExtractor[T]) OrFallback(err error, fn func(err error) T) T {
	return f.WithFallback(fn)(err)
}

func (a *optionApplier) SetField(key FieldKey, value FieldValue) {
	a.def.fields.set(key, value)
}

func (a *optionApplier) DisableTrace() {
	a.def.noTrace = true
}

func (a *optionApplier) AddStackSkip(skip int) {
	a.def.stackSkip += skip
}

func (a *optionApplier) SetStackDepth(depth int) {
	a.def.stackDepth = depth
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

func (a *optionApplier) SetLogValuer(valuer ErrorLogValuer) {
	a.def.logValuer = valuer
}

func (o *field[T]) ApplyOption(a OptionApplier) {
	a.SetField(o.key, &fieldValue[T]{value: o.value})
}

func (o *noTrace) ApplyOption(a OptionApplier) {
	a.DisableTrace()
}

func (o *stackSkip) ApplyOption(a OptionApplier) {
	a.AddStackSkip(o.skip)
}

func (o *stackDepth) ApplyOption(a OptionApplier) {
	a.SetStackDepth(o.depth)
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

func (o *logValuer) ApplyOption(a OptionApplier) {
	a.SetLogValuer(o.valuer)
}

func fieldKeyFromOption(opt Option) FieldKey {
	applier := &optionApplier{def: &Definition{fields: fields{}}}
	opt.ApplyOption(applier)
	for k := range applier.def.fields {
		return k
	}
	panic("no field key")
}
