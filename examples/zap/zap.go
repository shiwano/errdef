package zap

import (
	"github.com/shiwano/errdef"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type errorMarshaler struct {
	err errdef.Error
}

// Error wraps an errdef.Error for Zap's structured logging.
// It returns a Field that nests error information under the "error" key.
//
// The error object contains the following fields:
//   - message: The error message
//   - kind: The error kind (if present)
//   - fields: Custom fields (if present)
//   - origin: The origin stack frame (if present) with func, file, and line
//   - causes: Array of cause error messages (if present)
//
// For top-level field expansion, use ErrorInline instead.
//
// Example:
//
//	err := ErrNotFound.With(ctx, UserID("u123")).New("user not found")
//	logger.Info("operation failed", Error(err))
func Error(err error) zapcore.Field {
	if e, ok := err.(errdef.Error); ok {
		return zap.Object("error", &errorMarshaler{err: e})
	}
	return zap.Object("error", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("message", err.Error())
		return nil
	}))
}

// ErrorInline wraps an errdef.Error for Zap's structured logging.
// It returns a Field that expands all error information at the top level.
//
// The following fields are added at the top level:
//   - message: The error message
//   - kind: The error kind (if present)
//   - fields: Custom fields (if present)
//   - origin: The origin stack frame (if present) with func, file, and line
//   - causes: Array of cause error messages (if present)
//
// This is useful when you want errdef's rich error information to be directly
// accessible at the top level of the log entry.
//
// Example:
//
//	err := ErrNotFound.With(ctx, UserID("u123")).New("user not found")
//	logger.Info("operation failed", ErrorInline(err))
func ErrorInline(err error) zapcore.Field {
	if e, ok := err.(errdef.Error); ok {
		return zap.Inline(&errorMarshaler{err: e})
	}
	return zap.Inline(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("message", err.Error())
		return nil
	}))
}

func (m *errorMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("message", m.err.Error())

	if m.err.Kind() != "" {
		enc.AddString("kind", string(m.err.Kind()))
	}

	if m.err.Fields().Len() > 0 {
		_ = enc.AddObject("fields", fieldsMarshaler{fields: m.err.Fields()})
	}

	if m.err.Stack().Len() > 0 {
		if frame, ok := m.err.Stack().HeadFrame(); ok {
			_ = enc.AddObject("origin", frameMarshaler{frame: frame})
		}
	}

	causes := m.err.Unwrap()
	if len(causes) > 0 {
		_ = enc.AddArray("causes", causesMarshaler{causes: causes})
	}

	return nil
}

type fieldsMarshaler struct {
	fields errdef.Fields
}

func (m fieldsMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for k, v := range m.fields.All() {
		_ = enc.AddReflected(k.String(), v.Value())
	}
	return nil
}

type frameMarshaler struct {
	frame errdef.Frame
}

func (m frameMarshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("func", m.frame.Func)
	enc.AddString("file", m.frame.File)
	enc.AddInt("line", m.frame.Line)
	return nil
}

type causesMarshaler struct {
	causes []error
}

func (m causesMarshaler) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, cause := range m.causes {
		enc.AppendString(cause.Error())
	}
	return nil
}
