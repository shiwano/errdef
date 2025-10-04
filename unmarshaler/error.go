package unmarshaler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shiwano/errdef"
)

type unmarshaledError struct {
	definedError  errdef.Error
	fields        map[errdef.FieldKey]errdef.FieldValue
	unknownFields map[string]any
	stack         stack
	causes        []error
}

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
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, e.Error())

			causes := e.Unwrap()

			if e.Kind() != "" || e.Fields().Len() > 0 || e.Stack().Len() > 0 || len(causes) > 0 {
				_, _ = io.WriteString(s, "\n")
			}

			if e.Kind() != "" {
				_, _ = io.WriteString(s, "\nKind:\n")
				_, _ = io.WriteString(s, "\t")
				_, _ = io.WriteString(s, string(e.Kind()))
			}

			if e.Fields().Len() > 0 {
				_, _ = io.WriteString(s, "\nFields:\n")
				i := 0
				for k, v := range e.Fields().SortedSeq() {
					if i > 0 {
						_, _ = io.WriteString(s, "\n")
					}
					_, _ = io.WriteString(s, "\t")
					_, _ = io.WriteString(s, k.String())
					_, _ = io.WriteString(s, ": ")
					_, _ = fmt.Fprintf(s, "%+v", v.Value())
					i++
				}
			}

			if e.Stack().Len() > 0 {
				_, _ = io.WriteString(s, "\nStack:\n")
				i := 0
				for _, f := range e.Stack().Frames() {
					if f.File != "" {
						if i > 0 {
							_, _ = io.WriteString(s, "\n")
						}
						_, _ = io.WriteString(s, "\t")
						_, _ = io.WriteString(s, f.Func)
						_, _ = io.WriteString(s, "\n\t\t")
						_, _ = io.WriteString(s, f.File)
						_, _ = io.WriteString(s, ":")
						_, _ = io.WriteString(s, strconv.Itoa(f.Line))
						i++
					}
				}
			}

			for i, cause := range causes {
				if i == 0 {
					_, _ = io.WriteString(s, "\nCauses:\n")
				} else {
					_, _ = io.WriteString(s, "\n\t---\n")
				}

				var buf bytes.Buffer
				_, _ = fmt.Fprintf(&buf, "%+v", cause)
				causeStr := strings.Trim(buf.String(), "\n")

				j := 0
				for line := range strings.SplitSeq(causeStr, "\n") {
					if j > 0 {
						_, _ = io.WriteString(s, "\n")
					}
					_, _ = io.WriteString(s, "\t")
					_, _ = io.WriteString(s, line)
					j++
				}
			}
		case s.Flag('#'):
			type (
				unmarshaledError_ unmarshaledError
				unmarshaledError  unmarshaledError_
			)
			_, _ = fmt.Fprintf(s, "%#v", (*unmarshaledError)(e))
		default:
			_, _ = io.WriteString(s, e.Error())
		}
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

func (e *unmarshaledError) MarshalJSON() ([]byte, error) {
	var causes []json.RawMessage
	for _, c := range e.causes {
		switch t := c.(type) {
		case *unmarshaledError:
			b, err := t.MarshalJSON()
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		case json.Marshaler:
			data, err := t.MarshalJSON()
			if err != nil {
				return nil, err
			}

			b, err := json.Marshal(struct {
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}{
				Message: c.Error(),
				Data:    data,
			})
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		default:
			b, err := json.Marshal(struct {
				Message string `json:"message"`
			}{
				Message: c.Error(),
			})
			if err != nil {
				return nil, err
			}
			causes = append(causes, b)
		}
	}

	var fields errdef.Fields
	if e.Fields().Len() > 0 {
		fields = e.Fields()
	}

	var stackFrames []errdef.Frame
	if e.Stack().Len() > 0 {
		stackFrames = e.stack.Frames()
	}

	return json.Marshal(struct {
		Message string            `json:"message"`
		Kind    string            `json:"kind,omitempty"`
		Fields  errdef.Fields     `json:"fields,omitempty"`
		Stack   []errdef.Frame    `json:"stack,omitempty"`
		Causes  []json.RawMessage `json:"causes,omitempty"`
	}{
		Message: e.Error(),
		Kind:    string(e.Kind()),
		Fields:  fields,
		Stack:   stackFrames,
		Causes:  causes,
	})
}

func (e *unmarshaledError) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 5)

	attrs = append(attrs, slog.String("message", e.definedError.Error()))

	if e.Kind() != "" {
		attrs = append(attrs, slog.String("kind", string(e.Kind())))
	}

	if e.Fields().Len() > 0 {
		attrs = append(attrs, slog.Any("fields", e.Fields()))
	}

	if e.Stack().Len() > 0 {
		if frame, ok := e.Stack().HeadFrame(); ok {
			attrs = append(attrs, slog.Any("origin", frame))
		}
	}

	if len(e.causes) > 0 {
		causeMessages := make([]string, len(e.causes))
		for i, c := range e.causes {
			causeMessages[i] = c.Error()
		}
		attrs = append(attrs, slog.Any("causes", causeMessages))
	}

	return slog.GroupValue(attrs...)
}
