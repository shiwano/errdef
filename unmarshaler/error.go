package unmarshaler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/shiwano/errdef"
)

type (
	unmarshaledError struct {
		definedError  errdef.Error
		fields        map[errdef.FieldKey]errdef.FieldValue
		unknownFields map[string]any
		stack         stack
		causes        []error
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
	_ errdef.Error        = (*unmarshaledError)(nil)
	_ errdef.DebugStacker = (*unmarshaledError)(nil)
	_ fmt.Formatter       = (*unmarshaledError)(nil)
	_ json.Marshaler      = (*unmarshaledError)(nil)
	_ slog.LogValuer      = (*unmarshaledError)(nil)
	_ causer              = (*unmarshaledError)(nil)
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

func (e *unmarshaledError) Is(target error) bool {
	if is, ok := e.definedError.(interface{ Is(error) bool }); ok {
		if is.Is(target) {
			return true
		}
	}
	return false
}

func (e *unmarshaledError) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('#') {
		// Avoid infinite recursion in case someone does %#v on unmarshaledError.
		type (
			unmarshaledError_ unmarshaledError
			unmarshaledError  unmarshaledError_
		)
		_, _ = fmt.Fprintf(s, "%#v", (*unmarshaledError)(e))
		return
	}
	e.definedError.(errorEncoder).ErrorFormatter(e, s, verb)
}

func (e *unmarshaledError) MarshalJSON() ([]byte, error) {
	return e.definedError.(errorEncoder).ErrorJSONMarshaler(e)
}

func (e *unmarshaledError) LogValue() slog.Value {
	return e.definedError.(errorEncoder).ErrorLogValuer(e)
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
	return e.causes[0]
}
