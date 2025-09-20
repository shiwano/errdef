package errdef

import (
	"context"
	"errors"
	"fmt"
)

// Definition represents an error definition with customizable options.
type Definition struct {
	root          *Definition
	kind          Kind
	fields        fields
	noTrace       bool
	stackSkip     int
	stackDepth    int
	boundary      bool
	formatter     ErrorFormatter
	jsonMarshaler ErrorJSONMarshaler
}

// Kind returns the kind of this error definition.
func (d *Definition) Kind() Kind {
	return d.kind
}

// Error returns the string representation of this error definition.
// This makes Definition implement the error interface.
func (d *Definition) Error() string {
	if d.kind == "" {
		return "errdef: <unnamed>"
	}
	return fmt.Sprintf("errdef: %s", d.kind)
}

// With creates a new Definition with options from the context and additional options applied.
func (d *Definition) With(ctx context.Context, opts ...Option) *Definition {
	ctxOpts := optionsFromContext(ctx)
	if len(ctxOpts) == 0 && len(opts) == 0 {
		return d
	}
	def := d.clone()
	def.applyOptions(ctxOpts)
	def.applyOptions(opts)
	return def
}

// WithOptions creates a new Definition with the given options applied.
func (d *Definition) WithOptions(opts ...Option) *Definition {
	if len(opts) == 0 {
		return d
	}
	def := d.clone()
	def.applyOptions(opts)
	return def
}

// New creates a new error with the given message using this definition.
func (d *Definition) New(msg string) error {
	return newError(d, nil, msg, false, callersSkip)
}

// Errorf creates a new error with a formatted message using this definition.
func (d *Definition) Errorf(format string, args ...any) error {
	return newError(d, nil, fmt.Sprintf(format, args...), false, callersSkip)
}

// Wrap wraps an existing error using this definition.
// Returns nil if cause is nil.
func (d *Definition) Wrap(cause error) error {
	if cause == nil {
		return nil
	}
	return newError(d, cause, cause.Error(), false, callersSkip)
}

// Wrapf wraps an existing error with a formatted message using this definition.
// Returns nil if cause is nil.
func (d *Definition) Wrapf(cause error, format string, args ...any) error {
	if cause == nil {
		return nil
	}
	return newError(d, cause, fmt.Sprintf(format, args...), false, callersSkip)
}

// Join creates a new error by joining multiple errors using this definition.
// Returns nil if all causes are nil.
func (d *Definition) Join(causes ...error) error {
	cause := errors.Join(causes...)
	if cause == nil {
		return nil
	}
	return newError(d, cause, cause.Error(), true, callersSkip)
}

// CapturePanic captures a panic value and converts it to an error wrapped by this definition.
// If errPtr is nil or panicValue is nil, this function does nothing.
// The resulting error implements PanicError interface to preserve the original panic value.
func (d *Definition) CapturePanic(errPtr *error, panicValue any) {
	if panicValue == nil || errPtr == nil {
		return
	}
	cause := newPanicError(panicValue)
	*errPtr = d.Wrapf(cause, "panic: %s", cause.Error())
}

// Is reports whether this definition matches the given error.
func (d *Definition) Is(err error) bool {
	return errors.Is(err, d)
}

func (d *Definition) clone() *Definition {
	return &Definition{
		root:          d.root,
		kind:          d.kind,
		fields:        d.fields.clone(),
		noTrace:       d.noTrace,
		stackSkip:     d.stackSkip,
		stackDepth:    d.stackDepth,
		boundary:      d.boundary,
		formatter:     d.formatter,
		jsonMarshaler: d.jsonMarshaler,
	}
}

func (d *Definition) applyOptions(opts []Option) {
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
