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
// when printed, marshaled, or logged, while still allowing access
// to the original value via Value().
type Redacted[T any] struct {
	value T
}

var (
	_ fmt.Stringer           = Redacted[any]{}
	_ fmt.Formatter          = Redacted[any]{}
	_ json.Marshaler         = Redacted[any]{}
	_ encoding.TextMarshaler = Redacted[any]{}
	_ slog.LogValuer         = Redacted[any]{}
)

// Redact wraps value in a Redacted[T], which always renders as "[REDACTED]"
// when formatted, marshaled, or logged (fmt, json, slog), while preserving
// the original value for in-process use via the Value() method.
// Use this to prevent accidental exposure of sensitive data in logs/output.
func Redact[T any](value T) Redacted[T] {
	return Redacted[T]{value: value}
}

func (r Redacted[T]) Value() T {
	return r.value
}

func (r Redacted[T]) String() string {
	return redactedStr
}

func (r Redacted[T]) Format(s fmt.State, verb rune) {
	_, _ = io.WriteString(s, redactedStr)
}

func (r Redacted[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(redactedStr)
}

func (r Redacted[T]) MarshalText() ([]byte, error) {
	return []byte(redactedStr), nil
}

func (r Redacted[T]) LogValue() slog.Value {
	return slog.StringValue(redactedStr)
}
