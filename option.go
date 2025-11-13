package errdef

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type (
	// Option represents a configuration option that can be applied to error definitions.
	Option interface {
		applyOption(d *definition)
	}

	// FieldConstructor creates an Option that sets a field value.
	FieldConstructor[T any] func(value T) Option

	// FieldExtractor extracts a field value from an error.
	FieldExtractor[T any] func(err error) (T, bool)

	// FieldConstructorNoArgs creates an Option with a default value when called with no arguments.
	FieldConstructorNoArgs[T any] func() Option

	// FieldExtractorSingleReturn extracts a field value from an error, returning only the value.
	FieldExtractorSingleReturn[T any] func(err error) T

	field[T any] struct {
		key   FieldKey
		value T
	}

	noopOption struct{}

	noTrace struct{}

	stackSkip struct {
		skip int
	}

	stackDepth struct {
		depth int
	}

	stackSource struct {
		around int
		depth  int
	}

	formatter struct {
		formatter func(err Error, s fmt.State, verb rune)
	}

	jsonMarshaler struct {
		marshaler func(err Error) ([]byte, error)
	}

	logValuer struct {
		valuer func(err Error) slog.Value
	}
)

// Key returns the key associated with this constructor.
func (f FieldConstructor[T]) Key() FieldKey {
	var zero T
	return fieldKeyFromOption(f(zero))
}

// Key returns the key associated with this constructor.
func (f FieldConstructorNoArgs[T]) Key() FieldKey {
	return fieldKeyFromOption(f())
}

// WithValue creates a field option constructor that sets a specific value.
func (f FieldConstructor[T]) WithValue(value T) FieldConstructorNoArgs[T] {
	return func() Option {
		return f(value)
	}
}

// WithValueFunc creates a field option constructor that sets a value using a function.
func (f FieldConstructor[T]) WithValueFunc(fn func() T) FieldConstructorNoArgs[T] {
	return func() Option {
		return f(fn())
	}
}

// WithErrorFunc creates a field option constructor that sets a value using a function that takes an error.
func (f FieldConstructor[T]) WithErrorFunc(fn func(err error) T) FieldConstructor[error] {
	return func(err error) Option {
		val := fn(err)
		return f(val)
	}
}

// WithContextFunc creates a field option constructor that sets a value using a function that takes a context.
func (f FieldConstructor[T]) WithContextFunc(fn func(ctx context.Context) T) FieldConstructor[context.Context] {
	return func(ctx context.Context) Option {
		val := fn(ctx)
		return f(val)
	}
}

// WithHTTPRequestFunc creates a field option constructor that sets a value using a function that takes an HTTP request.
func (f FieldConstructor[T]) WithHTTPRequestFunc(fn func(r *http.Request) T) FieldConstructor[*http.Request] {
	return func(r *http.Request) Option {
		val := fn(r)
		return f(val)
	}
}

// WithZero creates a field extractor that returns only the value, ignoring the boolean.
func (f FieldExtractor[T]) WithZero() FieldExtractorSingleReturn[T] {
	return func(err error) T {
		val, _ := f(err)
		return val
	}
}

// WithDefault creates a field extractor that returns a default value if the field is not found.
func (f FieldExtractor[T]) WithDefault(value T) FieldExtractorSingleReturn[T] {
	return func(err error) T {
		if val, ok := f(err); ok {
			return val
		}
		return value
	}
}

// WithFallback creates a field extractor that calls a function to obtain a value if the field is not found.
func (f FieldExtractor[T]) WithFallback(fn func(err error) T) FieldExtractorSingleReturn[T] {
	return func(err error) T {
		if val, ok := f(err); ok {
			return val
		}
		return fn(err)
	}
}

// OrZero extracts the field value from the error, returning the zero value if not found.
func (f FieldExtractor[T]) OrZero(err error) T {
	return f.WithZero()(err)
}

// OrDefault extracts the field value from the error, returning a default value if not found.
func (f FieldExtractor[T]) OrDefault(err error, value T) T {
	return f.WithDefault(value)(err)
}

// OrFallback extracts the field value from the error, calling a function to obtain a value if not found.
func (f FieldExtractor[T]) OrFallback(err error, fn func(err error) T) T {
	return f.WithFallback(fn)(err)
}

func (o *field[T]) applyOption(d *definition) {
	d.fields.set(o.key, &fieldValue[T]{value: o.value})
}

func (o *noopOption) applyOption(d *definition) {}

func (o *noTrace) applyOption(d *definition) {
	d.noTrace = true
}

func (o *stackSkip) applyOption(d *definition) {
	d.stackSkip += o.skip
}

func (o *stackDepth) applyOption(d *definition) {
	d.stackDepth = o.depth
}

func (o *stackSource) applyOption(d *definition) {
	d.stackSourceLines = o.around
	d.stackSourceDepth = o.depth
}

func (o *formatter) applyOption(d *definition) {
	d.formatter = o.formatter
}

func (o *jsonMarshaler) applyOption(d *definition) {
	d.jsonMarshaler = o.marshaler
}

func (o *logValuer) applyOption(d *definition) {
	d.logValuer = o.valuer
}

func fieldKeyFromOption(opt Option) FieldKey {
	def := &definition{fields: newFields()}
	opt.applyOption(def)
	for k := range def.fields.data {
		return k
	}
	panic("no field key")
}
