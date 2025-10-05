package unmarshaler

import (
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
)

var (
	_ errdef.Error   = (*unmarshaledError)(nil)
	_ fmt.Formatter  = (*unmarshaledError)(nil)
	_ json.Marshaler = (*unmarshaledError)(nil)
	_ slog.LogValuer = (*unmarshaledError)(nil)
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
