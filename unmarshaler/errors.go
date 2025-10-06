package unmarshaler

import "github.com/shiwano/errdef"

var (
	ErrDecodeFailure = errdef.Define("errdef/unmarshaler.decode_failure", errdef.NoTrace())
	ErrKindNotFound  = errdef.Define("errdef/unmarshaler.kind_not_found", errdef.NoTrace())
	ErrInternal      = errdef.Define("errdef/unmarshaler.internal", errdef.NoTrace())

	kindField, _ = errdef.DefineField[errdef.Kind]("kind")
)
