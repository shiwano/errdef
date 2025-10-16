package errdef

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type (
	// Definition represents an error definition with customizable options.
	// It serves as a reusable template for creating structured errors with a specific kind,
	// fields, and behavior (e.g., stack traces, formatting, serialization).
	//
	// Definition can be used as a sentinel error for identity checks with errors.Is,
	// similar to standard errors like io.EOF. It can also be configured with additional
	// options using With or WithOptions to create an ErrorFactory for generating errors
	// with context-specific or request-scoped data.
	//
	// Definition implements both the error interface and ErrorFactory interface,
	// making it suitable for both direct error creation and identity comparison.
	Definition struct {
		rootDef       *Definition
		kind          Kind
		fields        *fields
		noTrace       bool
		stackSkip     int
		stackDepth    int
		formatter     func(err Error, s fmt.State, verb rune)
		jsonMarshaler func(err Error) ([]byte, error)
		logValuer     func(err Error) slog.Value
	}

	// Factory is an interface for creating errors from a configured Definition.
	// It provides only error creation methods, preventing misuse such as identity
	// comparison (errors.Is) or further configuration (With/WithOptions).
	//
	// Factory instances are typically created by Definition.With or
	// Definition.WithOptions methods, and are intended to be used immediately
	// for error creation rather than stored as sentinel values.
	Factory interface {
		// New creates a new error with the given message using this definition.
		New(msg string) error
		// Errorf creates a new error with a formatted message using this definition.
		Errorf(format string, args ...any) error
		// Wrap wraps an existing error using this definition.
		// Returns nil if cause is nil.
		Wrap(cause error) error
		// Wrapf wraps an existing error with a formatted message using this definition.
		// Returns nil if cause is nil.
		Wrapf(cause error, format string, args ...any) error
		// Join creates a new error by joining multiple errors using this definition.
		// Returns nil if all causes are nil.
		Join(causes ...error) error
		// Recover executes the given function and recovers from any panic that occurs within it.
		// If a panic occurs, it wraps the panic as an error using this definition and returns it.
		// If no panic occurs, it returns the function's return value as is.
		// The resulting error implements PanicError interface to preserve the original panic value.
		Recover(fn func() error) error
	}
)

var (
	_ error        = (*Definition)(nil)
	_ Factory      = (*Definition)(nil)
	_ fieldsGetter = (*Definition)(nil)
)

// Kind returns the kind of this error definition.
func (d *Definition) Kind() Kind {
	return d.kind
}

// Error returns the string representation of this error definition.
// This makes Definition implement the error interface.
func (d *Definition) Error() string {
	return string(d.kind)
}

// With creates a new ErrorFactory and applies options from context first (if any),
// then the given opts. Later options override earlier ones.
func (d *Definition) With(ctx context.Context, opts ...Option) Factory {
	ctxOpts := optionsFromContext(ctx)
	if len(ctxOpts) == 0 && len(opts) == 0 {
		return d
	}
	def := d.clone()
	def.applyOptions(ctxOpts)
	def.applyOptions(opts)
	return def
}

// WithOptions creates a new ErrorFactory with the given options applied.
// Later options override earlier ones.
func (d *Definition) WithOptions(opts ...Option) Factory {
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

// Recover executes the given function and recovers from any panic that occurs within it.
// If a panic occurs, it wraps the panic as an error using this definition and returns it.
// If no panic occurs, it returns the function's return value as is.
// The resulting error implements PanicError interface to preserve the original panic value.
func (d *Definition) Recover(fn func() error) error {
	var err error
	func() {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				cause := newPanicError(panicValue)
				err = newError(d, cause, fmt.Sprintf("panic: %s", cause.Error()), false, callersSkip+2)
			}
		}()
		err = fn()
	}()
	return err
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
