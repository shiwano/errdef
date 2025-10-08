package errdef

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Definition represents an error definition with customizable options.
type Definition struct {
	rootDef       *Definition
	kind          Kind
	fields        *fields
	noTrace       bool
	stackSkip     int
	stackDepth    int
	boundary      bool
	formatter     func(err Error, s fmt.State, verb rune)
	jsonMarshaler func(err Error) ([]byte, error)
	logValuer     func(err Error) slog.Value
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

// With creates a new Definition and applies options from context first (if any),
// then the given opts. Later options override earlier ones.
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
// Later options override earlier ones.
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

// CapturePanic captures a panic value and converts it to an error wrapped by this definition,
// and returns the original panic value and true.
// If errPtr is nil or panicValue is nil, this function does nothing and returns nil and false.
// The resulting error implements PanicError interface to preserve the original panic value.
func (d *Definition) CapturePanic(errPtr *error, panicValue any) (any, bool) {
	if panicValue == nil || errPtr == nil {
		return nil, false
	}
	cause := newPanicError(panicValue)
	*errPtr = d.Wrapf(cause, "panic: %s", cause.Error())
	return panicValue, true
}

// Is reports whether this definition matches the given error.
func (d *Definition) Is(err error) bool {
	var e *Definition
	if errors.As(err, &e) {
		return d.root() == e.root()
	}
	return errors.Is(err, d)
}

// Fields returns the fields associated with this definition.
func (d *Definition) Fields() Fields {
	return d.fields
}

func (d *Definition) isRoot() bool {
	return d.rootDef == nil
}

func (d *Definition) root() *Definition {
	if d.isRoot() {
		return d
	}
	return d.rootDef
}

func (d *Definition) clone() *Definition {
	clone := *d
	clone.fields = d.fields.clone()
	if d.isRoot() {
		clone.rootDef = d
	}
	return &clone
}

func (d *Definition) applyOptions(opts []Option) {
	for _, opt := range opts {
		opt.applyOption(d)
	}
}
