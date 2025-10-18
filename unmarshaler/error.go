package unmarshaler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/shiwano/errdef"
)

type (
	// UnmarshaledError is an error type returned by Unmarshaler.
	UnmarshaledError interface {
		errdef.Error

		// UnknownFields returns an iterator over unknown fields that were present
		// in the serialized data but not defined in the error definition.
		UnknownFields() iter.Seq2[string, any]
	}

	// UnknownCauseError represents an error whose cause has an unknown type
	// that cannot be unmarshaled back into a proper error definition.
	UnknownCauseError struct {
		msg      string
		typeName string
		causes   []error
	}

	causer interface {
		Cause() error
	}

	unmarshaledError struct {
		def           errdef.Definition
		msg           string
		fields        map[errdef.FieldKey]errdef.FieldValue
		unknownFields map[string]any
		stack         stack
		causes        []error
	}
)

var (
	_ UnmarshaledError    = (*unmarshaledError)(nil)
	_ errdef.DebugStacker = (*unmarshaledError)(nil)
	_ fmt.Formatter       = (*unmarshaledError)(nil)
	_ fmt.GoStringer      = (*unmarshaledError)(nil)
	_ json.Marshaler      = (*unmarshaledError)(nil)
	_ slog.LogValuer      = (*unmarshaledError)(nil)
	_ causer              = (*unmarshaledError)(nil)

	_ error = (*UnknownCauseError)(nil)
)

func (e *unmarshaledError) Error() string {
	return e.msg
}

func (e *unmarshaledError) Kind() errdef.Kind {
	return e.def.Kind()
}

func (e *unmarshaledError) Fields() errdef.Fields {
	return &fields{
		fields:        e.fields,
		unknownFields: e.unknownFields,
	}
}

func (e *unmarshaledError) Stack() errdef.Stack {
	return e.stack
}

func (e *unmarshaledError) Unwrap() []error {
	return slices.Clone(e.causes)
}

func (e *unmarshaledError) UnwrapTree() errdef.Nodes {
	return e.def.(errdef.Presenter).BuildCauseTree(e)
}

func (e *unmarshaledError) UnknownFields() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range e.unknownFields {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (e *unmarshaledError) Is(target error) bool {
	if e == target {
		return true
	}
	if d, ok := target.(errdef.Definition); ok {
		return e.def.Is(d)
	}
	return false
}

func (e *unmarshaledError) GoString() string {
	type (
		unmarshaledError_ unmarshaledError
		unmarshaledError  unmarshaledError_
	)
	return fmt.Sprintf("%#v", (*unmarshaledError)(e))
}

func (e *unmarshaledError) Format(s fmt.State, verb rune) {
	e.def.(errdef.Presenter).FormatError(e, s, verb)
}

func (e *unmarshaledError) MarshalJSON() ([]byte, error) {
	return e.def.(errdef.Presenter).MarshalErrorJSON(e)
}

func (e *unmarshaledError) LogValue() slog.Value {
	return e.def.(errdef.Presenter).MakeErrorLogValue(e)
}

func (e *unmarshaledError) DebugStack() string {
	buf := bytes.NewBufferString(e.Error())
	// hard-coded cause we can't get it in pure Go.
	buf.WriteString("\n\ngoroutine 1 [running]:")

	for _, frame := range e.stack.Frames() {
		if frame.File != "" {
			buf.WriteByte('\n')
			// Entry point address is set to 0 because it's not available in unmarshaled errors.
			fmt.Fprintf(buf, "%s()\n\t%s:%d +%#x", frame.Func, frame.File, frame.Line, 0)
		}
	}
	return buf.String()
}

func (e *unmarshaledError) Cause() error {
	if len(e.causes) == 0 {
		return nil
	}
	return e.causes[0] // return the first cause only
}

// Error implements the error interface.
func (e *UnknownCauseError) Error() string {
	return e.msg
}

// TypeName returns the name of the unknown cause type.
func (e *UnknownCauseError) TypeName() string {
	return e.typeName
}

// Unwrap returns the errors that this error wraps.
func (e *UnknownCauseError) Unwrap() []error {
	return slices.Clone(e.causes)
}
