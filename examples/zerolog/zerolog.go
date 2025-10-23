package zerolog

import (
	"github.com/rs/zerolog"
	"github.com/shiwano/errdef"
)

type errorMarshaler struct {
	err errdef.Error
}

type stdErrorMarshaler struct {
	err error
}

// Error wraps an errdef.Error for zerolog's structured logging.
// It returns a LogObjectMarshaler that can be used with Object() or EmbedObject().
//
// The error object contains the following fields:
//   - message: The error message
//   - kind: The error kind (if present)
//   - fields: Custom fields (if present)
//   - origin: The origin stack frame (if present) with func, file, and line
//   - causes: Array of cause error messages (if present)
//
// Example with Object() (nested under "error" key):
//
//	err := ErrNotFound.With(ctx, UserID("u123")).New("user not found")
//	logger.Info().Object("error", Error(err)).Msg("operation failed")
//
// Example with EmbedObject() (fields at top level):
//
//	err := ErrNotFound.With(ctx, UserID("u123")).New("user not found")
//	logger.Info().EmbedObject(Error(err)).Msg("operation failed")
func Error(err error) zerolog.LogObjectMarshaler {
	if e, ok := err.(errdef.Error); ok {
		return &errorMarshaler{err: e}
	}
	return &stdErrorMarshaler{err: err}
}

func (m *errorMarshaler) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message", m.err.Error())

	if m.err.Kind() != "" {
		e.Str("kind", string(m.err.Kind()))
	}

	if m.err.Fields().Len() > 0 {
		e.Object("fields", fieldsMarshaler{fields: m.err.Fields()})
	}

	if m.err.Stack().Len() > 0 {
		if frame, ok := m.err.Stack().HeadFrame(); ok {
			e.Object("origin", frameMarshaler{frame: frame})
		}
	}

	causes := m.err.Unwrap()
	if len(causes) > 0 {
		e.Array("causes", causesMarshaler{causes: causes})
	}
}

type fieldsMarshaler struct {
	fields errdef.Fields
}

func (m fieldsMarshaler) MarshalZerologObject(e *zerolog.Event) {
	for k, v := range m.fields.All() {
		e.Interface(k.String(), v.Value())
	}
}

type frameMarshaler struct {
	frame errdef.Frame
}

func (m frameMarshaler) MarshalZerologObject(e *zerolog.Event) {
	e.Str("func", m.frame.Func)
	e.Str("file", m.frame.File)
	e.Int("line", m.frame.Line)
}

type causesMarshaler struct {
	causes []error
}

func (m causesMarshaler) MarshalZerologArray(a *zerolog.Array) {
	for _, cause := range m.causes {
		a.Str(cause.Error())
	}
}

func (m *stdErrorMarshaler) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message", m.err.Error())
}
