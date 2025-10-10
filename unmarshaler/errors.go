package unmarshaler

import "github.com/shiwano/errdef"

var (
	// ErrDecodeFailure is returned when the decoder fails to decode byte data.
	ErrDecodeFailure = errdef.Define("errdef/unmarshaler.decode_failure", errdef.NoTrace())
	// ErrUnknownKind is returned when the resolver cannot resolve the error kind.
	ErrUnknownKind = errdef.Define("errdef/unmarshaler.unknown_kind", errdef.NoTrace())
	// ErrUnknownField is returned when an unknown field is encountered in strict mode.
	ErrUnknownField = errdef.Define("errdef/unmarshaler.unknown_field", errdef.NoTrace())
	// ErrInternal is returned when an unexpected error occurs within the unmarshaler.
	ErrInternal = errdef.Define("errdef/unmarshaler.internal", errdef.NoTrace())

	// KindFromError extracts the failed kind from errors.
	kindField, KindFromError = errdef.DefineField[errdef.Kind]("kind")
	// FieldNameFromError extracts the field name from errors.
	fieldNameField, FieldNameFromError = errdef.DefineField[string]("field_name")
)
