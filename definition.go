package errdef

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
)

type (
	// Definition represents an error definition with customizable options.
	// It serves as a reusable template for creating structured errors with a specific kind,
	// fields, and behavior (e.g., stack traces, formatting, serialization).
	//
	// Definition can be used as a sentinel error for identity checks with errors.Is,
	// similar to standard errors like io.EOF. It can also be configured with additional
	// options using With or WithOptions to create a Factory for generating errors
	// with context-specific or request-scoped data.
	Definition interface {
		error
		Factory

		// Kind returns the kind of this error definition.
		Kind() Kind
		// Is reports whether this definition matches the given error.
		Is(error) bool
		// Fields returns the fields associated with this definition.
		Fields() Fields
		// With creates a new Factory and applies options from context first (if any),
		// then the given opts. Later options override earlier ones.
		With(context.Context, ...Option) Factory
		// WithOptions creates a new Factory with the given options applied.
		// Later options override earlier ones.
		WithOptions(...Option) Factory
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

	// Presenter provides error presentation methods for formatting, serialization, and logging.
	// These methods are primarily used internally by Error implementations and are not
	// typically called directly by application code.
	Presenter interface {
		// FormatError formats the error using this definition's custom formatter if set,
		// otherwise uses the default format implementation.
		FormatError(err Error, s fmt.State, verb rune)
		// MarshalErrorJSON marshals the error to JSON using this definition's custom marshaler if set,
		// otherwise uses the default JSON structure.
		MarshalErrorJSON(err Error) ([]byte, error)
		// MakeErrorLogValue returns a slog.Value representing the error using this definition's custom log valuer if set,
		// otherwise uses the default log structure.
		MakeErrorLogValue(err Error) slog.Value
		// BuildCauseTree returns all causes as a tree structure.
		// This method includes cycle detection: when a circular reference is detected,
		// the node that would create the cycle is excluded, ensuring the result remains acyclic.
		// While circular references are rare in practice, this check serves as a defensive
		// programming measure.
		BuildCauseTree(err Error) Nodes
	}

	definition struct {
		rootDef       *definition
		kind          Kind
		fields        *fields
		noTrace       bool
		stackSkip     int
		stackDepth    int
		formatter     func(err Error, s fmt.State, verb rune)
		jsonMarshaler func(err Error) ([]byte, error)
		logValuer     func(err Error) slog.Value
	}
)

var (
	_ Definition   = (*definition)(nil)
	_ Presenter    = (*definition)(nil)
	_ fieldsGetter = (*definition)(nil)
)

func (d *definition) Kind() Kind {
	return d.kind
}

func (d *definition) Error() string {
	return string(d.kind)
}

func (d *definition) With(ctx context.Context, opts ...Option) Factory {
	ctxOpts := optionsFromContext(ctx)
	if len(ctxOpts) == 0 && len(opts) == 0 {
		return d
	}
	def := d.clone()
	def.applyOptions(ctxOpts)
	def.applyOptions(opts)
	return def
}

func (d *definition) WithOptions(opts ...Option) Factory {
	if len(opts) == 0 {
		return d
	}
	def := d.clone()
	def.applyOptions(opts)
	return def
}

func (d *definition) New(msg string) error {
	return newError(d, nil, msg, false, callersSkip)
}

func (d *definition) Errorf(format string, args ...any) error {
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	return newError(d, nil, msg, false, callersSkip)
}

func (d *definition) Wrap(cause error) error {
	if cause == nil {
		return nil
	}
	return newError(d, cause, cause.Error(), false, callersSkip)
}

func (d *definition) Wrapf(cause error, format string, args ...any) error {
	if cause == nil {
		return nil
	}
	fullMsg := fmt.Sprintf(format+": %s", append(args, cause.Error())...)
	return newError(d, cause, fullMsg, false, callersSkip)
}

func (d *definition) Join(causes ...error) error {
	cause := errors.Join(causes...)
	if cause == nil {
		return nil
	}
	return newError(d, cause, cause.Error(), true, callersSkip)
}

func (d *definition) Recover(fn func() error) error {
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

func (d *definition) Is(target error) bool {
	if d == target {
		return true
	}
	var derr *definedError
	if errors.As(target, &derr) {
		if d.root() == derr.def.root() {
			return true
		}
	}
	var def *definition
	if errors.As(target, &def) {
		if d.root() == def.root() {
			return true
		}
	}
	return false
}

func (d *definition) Fields() Fields {
	return d.fields
}

func (d *definition) isRoot() bool {
	return d.rootDef == nil
}

func (d *definition) root() *definition {
	if d.isRoot() {
		return d
	}
	return d.rootDef
}

func (d *definition) clone() *definition {
	clone := *d
	clone.fields = d.fields.clone()
	if d.isRoot() {
		clone.rootDef = d
	}
	return &clone
}

func (d *definition) applyOptions(opts []Option) {
	for _, opt := range opts {
		opt.applyOption(d)
	}
}

func (d *definition) FormatError(err Error, s fmt.State, verb rune) {
	if d.formatter != nil {
		d.formatter(err, s, verb)
		return
	}

	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			causes := err.UnwrapTree()
			formatErrorDetails(err, s, "", len(causes) > 0)

			if len(causes) > 0 {
				formatCausesHeader(s, "", len(causes))
				formatNodes(causes, s, "  ")
			}
		case s.Flag('#'):
			if gs, ok := err.(fmt.GoStringer); ok {
				_, _ = io.WriteString(s, gs.GoString())
			} else {
				_, _ = io.WriteString(s, err.Error())
			}
		default:
			_, _ = io.WriteString(s, err.Error())
		}
	case 's':
		_, _ = io.WriteString(s, err.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", err.Error())
	}
}

func (d *definition) MarshalErrorJSON(err Error) ([]byte, error) {
	if d.jsonMarshaler != nil {
		return d.jsonMarshaler(err)
	}

	return json.Marshal(jsonErrorData{
		Message: err.Error(),
		Kind:    string(err.Kind()),
		Fields:  err.Fields(),
		Stack:   err.Stack().Frames(),
		Causes:  err.UnwrapTree(),
	})
}

func (d *definition) MakeErrorLogValue(err Error) slog.Value {
	if d.logValuer != nil {
		return d.logValuer(err)
	}

	attrs := make([]slog.Attr, 0, 5)
	attrs = append(attrs, slog.String("message", err.Error()))

	if err.Kind() != "" {
		attrs = append(attrs, slog.String("kind", string(err.Kind())))
	}
	if err.Fields().Len() > 0 {
		attrs = append(attrs, slog.Any("fields", err.Fields()))
	}
	if err.Stack().Len() > 0 {
		if frame, ok := err.Stack().HeadFrame(); ok {
			attrs = append(attrs, slog.Any("origin", frame))
		}
	}
	causes := err.Unwrap()
	if len(causes) > 0 {
		causeMessages := make([]string, len(causes))
		for i, c := range causes {
			causeMessages[i] = c.Error()
		}
		attrs = append(attrs, slog.Any("causes", causeMessages))
	}
	return slog.GroupValue(attrs...)
}

func (d *definition) BuildCauseTree(err Error) Nodes {
	visited := make(map[uintptr]uintptr)
	nodes := buildNodes(err.Unwrap(), visited)
	return nodes
}

func formatErrorDetails(err Error, s io.Writer, indent string, hasCauses bool) {
	_, _ = io.WriteString(s, err.Error())

	hasDetails := err.Kind() != "" || err.Fields().Len() > 0 || err.Stack().Len() > 0
	if hasDetails || hasCauses {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "---")
	}

	if err.Kind() != "" {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "kind: ")
		_, _ = io.WriteString(s, string(err.Kind()))
	}

	if err.Fields().Len() > 0 {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "fields:")
		for k, v := range err.Fields().Sorted() {
			_, _ = io.WriteString(s, "\n")
			_, _ = io.WriteString(s, indent)
			_, _ = io.WriteString(s, "  ")
			_, _ = io.WriteString(s, k.String())
			_, _ = io.WriteString(s, ": ")

			valueStr := fmt.Sprintf("%+v", v.Value())
			if strings.Contains(valueStr, "\n") {
				_, _ = io.WriteString(s, "|\n")
				for line := range strings.SplitSeq(valueStr, "\n") {
					_, _ = io.WriteString(s, indent)
					_, _ = io.WriteString(s, "    ")
					_, _ = io.WriteString(s, line)
					_, _ = io.WriteString(s, "\n")
				}
			} else {
				_, _ = io.WriteString(s, valueStr)
			}
		}
	}

	if err.Stack().Len() > 0 {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "stack:")
		for _, f := range err.Stack().Frames() {
			if f.File != "" {
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "  ")
				_, _ = io.WriteString(s, f.Func)
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "    ")
				_, _ = io.WriteString(s, f.File)
				_, _ = io.WriteString(s, ":")
				_, _ = io.WriteString(s, strconv.Itoa(f.Line))
			}
		}
	}
}

func formatCausesHeader(s io.Writer, indent string, count int) {
	_, _ = io.WriteString(s, "\n")
	_, _ = io.WriteString(s, indent)
	_, _ = io.WriteString(s, "causes: (")
	if count == 1 {
		_, _ = io.WriteString(s, "1 error")
	} else {
		_, _ = io.WriteString(s, strconv.Itoa(count))
		_, _ = io.WriteString(s, " errors")
	}
	_, _ = io.WriteString(s, ")")
}

func formatNodes(nodes []*Node, s io.Writer, indent string) {
	for i, node := range nodes {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "[")
		_, _ = io.WriteString(s, strconv.Itoa(i+1))
		_, _ = io.WriteString(s, "] ")

		if err, ok := node.Error.(Error); ok {
			formatErrorDetails(err, s, indent+"    ", len(node.Causes) > 0)
		} else {
			_, _ = io.WriteString(s, node.Error.Error())

			if len(node.Causes) > 0 {
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "    ---")
			}
		}

		if len(node.Causes) > 0 {
			formatCausesHeader(s, indent+"    ", len(node.Causes))
			formatNodes(node.Causes, s, indent+"    ")
		}
	}
}
