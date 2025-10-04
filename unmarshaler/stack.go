package unmarshaler

import (
	"encoding/json"
	"log/slog"

	"github.com/shiwano/errdef"
)

type stack []errdef.Frame

var (
	_ errdef.Stack   = stack{}
	_ json.Marshaler = stack{}
	_ slog.LogValuer = stack{}
)

func (s stack) StackTrace() []uintptr {
	return nil
}

func (s stack) Frames() []errdef.Frame {
	return s[:]
}

func (s stack) HeadFrame() (errdef.Frame, bool) {
	if len(s) == 0 {
		return errdef.Frame{}, false
	}
	return s[0], true
}

func (s stack) Len() int {
	return len(s)
}

func (s stack) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Frames())
}

func (s stack) LogValue() slog.Value {
	return slog.AnyValue(s.Frames())
}
