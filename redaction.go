package errdef

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

const redactedStr = "[REDACTED]"

// Redacted[T] wraps a value so it always renders as "[REDACTED]"
// when printed, marshaled, or logged (fmt, json, slog),
// while still allowing access to the original value via Value().
// Use this to prevent accidental exposure of sensitive data in logs/output.
type Redacted[T any] struct {
	value T
}

var (
	_ fmt.Stringer             = Redacted[any]{}
	_ fmt.GoStringer           = Redacted[any]{}
	_ fmt.Formatter            = Redacted[any]{}
	_ json.Marshaler           = Redacted[any]{}
	_ encoding.TextMarshaler   = Redacted[any]{}
	_ encoding.BinaryMarshaler = Redacted[any]{}
	_ slog.LogValuer           = Redacted[any]{}
)

// Value returns the original wrapped value.
func (r Redacted[T]) Value() T {
	return r.value
}

// String implements fmt.Stringer, always returning "[REDACTED]".
func (r Redacted[T]) String() string {
	return redactedStr
}

// GoString implements fmt.GoStringer, always returning "[REDACTED]" for %#v format.
func (r Redacted[T]) GoString() string {
	return redactedStr
}

// Format implements fmt.Formatter, always rendering as "[REDACTED]" regardless of the format verb.
func (r Redacted[T]) Format(s fmt.State, verb rune) {
	_, _ = io.WriteString(s, redactedStr)
}

// MarshalJSON implements json.Marshaler, always marshaling as "[REDACTED]".
func (r Redacted[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(redactedStr)
}

// MarshalText implements encoding.TextMarshaler, always returning "[REDACTED]" as text.
func (r Redacted[T]) MarshalText() ([]byte, error) {
	return []byte(redactedStr), nil
}

// MarshalBinary implements encoding.BinaryMarshaler, always returning "[REDACTED]" as bytes.
func (r Redacted[T]) MarshalBinary() ([]byte, error) {
	return []byte(redactedStr), nil
}

// LogValue implements slog.LogValuer, always logging as "[REDACTED]".
func (r Redacted[T]) LogValue() slog.Value {
	return slog.StringValue(redactedStr)
}

// Redact wraps value in a Redacted[T], which always renders as "[REDACTED]"
// when printed, marshaled, or logged (fmt, json, slog),
// while still allowing access to the original value via Value().
// Use this to prevent accidental exposure of sensitive data in logs/output.
func Redact[T any](value T) Redacted[T] {
	return Redacted[T]{value: value}
}
