package unmarshaler

import "github.com/shiwano/errdef"

var (
	// ErrDecodeFailure is returned when the decoder fails to decode byte data.
	ErrDecodeFailure = errdef.Define("errdef/unmarshaler.decode_failure", errdef.NoTrace())
	// ErrKindNotFound is returned when the resolver cannot resolve the error kind.
	ErrKindNotFound = errdef.Define("errdef/unmarshaler.kind_not_found", errdef.NoTrace())
	// ErrUnknownField is returned when an unknown field is encountered in strict mode.
	ErrUnknownField = errdef.Define("errdef/unmarshaler.unknown_field", errdef.NoTrace())
	// ErrFieldUnmarshalFailure is returned when the unmarshaler fails to unmarshal a field value.
	ErrFieldUnmarshalFailure = errdef.Define("errdef/unmarshaler.field_unmarshal_failure", errdef.NoTrace())
	// ErrInternal is returned when an unexpected error occurs within the unmarshaler.
	ErrInternal = errdef.Define("errdef/unmarshaler.internal", errdef.NoTrace())

	// KindFromError extracts the failed kind from errors.
	kindField, KindFromError = errdef.DefineField[errdef.Kind]("kind")
	// FieldNameFromError extracts the field name from errors.
	fieldNameField, FieldNameFromError = errdef.DefineField[string]("field_name")
)
