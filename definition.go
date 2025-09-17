package errdef

import (
	"context"
	"errors"
	"fmt"
)

// Definition represents an error definition with customizable options.
type Definition struct {
	kind          Kind
	fields        fields
	noTrace       bool
	stackSkip     int
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
		return "[unnamed]"
	}
	return string(d.kind)
}

// With creates a new Definition with options from the context and additional options applied.
func (d *Definition) With(ctx context.Context, opts ...Option) *Definition {
	ctxOpts := optionsFromContext(ctx)
	return d.WithOptions(append(ctxOpts, opts...)...)
}

// WithOptions creates a new Definition with the given options applied.
func (d *Definition) WithOptions(opts ...Option) *Definition {
	if len(opts) == 0 {
		return d
	}
	def := d.clone()
	applyOptionsTo(def, opts)
	return def
}

// New creates a new error with the given message using this definition.
func (d *Definition) New(msg string) error {
	return d.newError(nil, msg, false, callersSkip)
}

// Errorf creates a new error with a formatted message using this definition.
func (d *Definition) Errorf(format string, args ...any) error {
	return d.newError(nil, fmt.Sprintf(format, args...), false, callersSkip)
}

// Wrap wraps an existing error using this definition.
// Returns nil if cause is nil.
func (d *Definition) Wrap(cause error) error {
	if cause == nil {
		return nil
	}
	return d.newError(cause, cause.Error(), false, callersSkip)
}

// Wrapf wraps an existing error with a formatted message using this definition.
// Returns nil if cause is nil.
func (d *Definition) Wrapf(cause error, format string, args ...any) error {
	if cause == nil {
		return nil
	}
	return d.newError(cause, fmt.Sprintf(format, args...), false, callersSkip)
}

// Join creates a new error by joining multiple errors using this definition.
// Returns nil if all causes are nil.
func (d *Definition) Join(causes ...error) error {
	cause := errors.Join(causes...)
	if cause == nil {
		return nil
	}
	return d.newError(cause, cause.Error(), true, callersSkip)
}

// Is reports whether this definition matches the given error.
func (d *Definition) Is(err error) bool {
	return errors.Is(err, d)
}

func (d *Definition) clone() *Definition {
	return &Definition{
		kind:          d.kind,
		fields:        d.fields.clone(),
		noTrace:       d.noTrace,
		stackSkip:     d.stackSkip,
		boundary:      d.boundary,
		formatter:     d.formatter,
		jsonMarshaler: d.jsonMarshaler,
	}
}

func (d *Definition) newError(cause error, msg string, joined bool, stackSkip int) error {
	var stack stack
	if !d.noTrace {
		stack = newStack(d.stackSkip + stackSkip)
	}

	e := &definedError{
		def:    d,
		msg:    msg,
		cause:  cause,
		stack:  stack,
		joined: joined,
	}
	return e
}
