package errdef

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

const redactedStr = "[REDACTED]"

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

func Redact[T any](value T) Redacted[T] {
	return Redacted[T]{value: value}
}
