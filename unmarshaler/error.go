package unmarshaler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"

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

	unmarshaledError struct {
		definedError  errdef.Error
		fields        map[errdef.FieldKey]errdef.FieldValue
		unknownFields map[string]any
		stack         stack
		causes        []error
	}

	unknownCauseError struct {
		message  string
		typeName string
		causes   []error
	}

	errorEncoder interface {
		ErrorFormatter(err errdef.Error, s fmt.State, verb rune)
		ErrorJSONMarshaler(err errdef.Error) ([]byte, error)
		ErrorLogValuer(err errdef.Error) slog.Value
	}

	causer interface {
		Cause() error
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

	_ errdef.ErrorTypeNamer = (*unknownCauseError)(nil)
)

func (e *unmarshaledError) Error() string {
	return e.definedError.Error()
}

func (e *unmarshaledError) Kind() errdef.Kind {
	return e.definedError.Kind()
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
	return e.causes
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
	if is, ok := e.definedError.(interface{ Is(error) bool }); ok {
		if is.Is(target) {
			return true
		}
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
	e.definedError.(errorEncoder).ErrorFormatter(e, s, verb)
}

func (e *unmarshaledError) MarshalJSON() ([]byte, error) {
	return e.definedError.(errorEncoder).ErrorJSONMarshaler(e)
}

func (e *unmarshaledError) LogValue() slog.Value {
	return e.definedError.(errorEncoder).ErrorLogValuer(e)
}

func (e *unmarshaledError) UnwrapTree() ([]errdef.ErrorNode, bool) {
	// Unmarshaled errors don't need cycle detection since they come from serialized data
	// which shouldn't contain cycles
	return buildCauseNodes(e.causes), true
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

func (e *unknownCauseError) Error() string {
	return e.message
}

func (e *unknownCauseError) TypeName() string {
	return e.typeName
}

func (e *unknownCauseError) Unwrap() []error {
	return e.causes
}

func buildCauseNodes(causes []error) []errdef.ErrorNode {
	if len(causes) == 0 {
		return nil
	}

	nodes := make([]errdef.ErrorNode, 0, len(causes))
	for _, c := range causes {
		if c == nil {
			continue
		}
		nodes = append(nodes, buildCauseNode(c))
	}
	return nodes
}

func buildCauseNode(err error) errdef.ErrorNode {
	var causes []error
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if nested := unwrapper.Unwrap(); nested != nil {
			causes = []error{nested}
		}
	} else if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		causes = unwrapper.Unwrap()
	}

	return errdef.ErrorNode{
		Error:  err,
		Causes: buildCauseNodes(causes),
	}
}
