package unmarshaler

import (
	"errors"
	"fmt"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

type (
	// Unmarshaler unmarshals serialized error data into UnmarshaledError.
	//
	// The type parameter T specifies the input data type that the decoder accepts.
	// This allows type-safe deserialization from various formats beyond []byte,
	// such as Protocol Buffers messages or other structured data types.
	//
	// Common usage:
	//   - For JSON: Use NewJSON which returns *Unmarshaler[[]byte]
	//   - For custom formats: Use New with a custom Decoder[T]
	Unmarshaler[T any] struct {
		*unmarshaler
		resolver resolver.Resolver
		decoder  Decoder[T]
	}

	// Option is a function type for customizing Unmarshaler configuration.
	Option func(*unmarshaler)

	unmarshaler struct {
		sentinelErrors  map[sentinelKey]error
		customFieldKeys []errdef.FieldKey
		strictMode      bool
	}

	sentinelKey struct {
		typeName string
		message  string
	}
)

const (
	errdefDefinitionTypeName         = "*errdef.Definition"
	errdefDefinitionEmptyKindMessage = "<unnamed>"
	redactedStr                      = "[REDACTED]"
)

// New creates a new Unmarshaler with the given resolver, decoder, and options.
//
// The type parameter T is inferred from the decoder's input type, ensuring
// type-safe unmarshaling. The decoder function converts input data of type T
// into a DecodedData structure.
//
// For JSON deserialization, consider using NewJSON instead of New directly.
func New[T any](resolver resolver.Resolver, decoder Decoder[T], opts ...Option) *Unmarshaler[T] {
	u := &Unmarshaler[T]{
		unmarshaler: &unmarshaler{},
		resolver:    resolver,
		decoder:     decoder,
	}
	for _, opt := range opts {
		opt(u.unmarshaler)
	}
	return u
}

// NewJSON creates a new Unmarshaler with a JSON decoder.
func NewJSON(resolver resolver.Resolver, opts ...Option) *Unmarshaler[[]byte] {
	return New(resolver, jsonToDecodedData, opts...)
}

// Unmarshal deserializes the given data into an UnmarshaledError.
//
// The input type T is determined by the decoder provided to New.
// For Unmarshaler[[]byte] (e.g., from NewJSON), this accepts byte slices.
// For custom types (e.g., Unmarshaler[*ErrorProto]), this accepts those types directly.
func (d *Unmarshaler[T]) Unmarshal(data T) (UnmarshaledError, error) {
	decoded, err := d.decoder(data)
	if err != nil {
		return nil, ErrDecodeFailure.Wrap(err)
	}
	return d.unmarshal(decoded)
}

func (d *Unmarshaler[T]) unmarshal(decoded *DecodedData) (UnmarshaledError, error) {
	def, err := d.resolveKind(errdef.Kind(decoded.Kind))
	if err != nil {
		return nil, err
	}

	definedError := def.
		WithOptions(errdef.NoTrace()).
		New(decoded.Message).(errdef.Error)

	fields := make(map[errdef.FieldKey]errdef.FieldValue)
	unknownFields := make(map[string]any)

	for fieldName, fieldValue := range decoded.Fields {
		keys := def.Fields().FindKeys(fieldName)
		matched := false

		if s, ok := fieldValue.(string); ok && s == redactedStr {
			unknownFields[fieldName] = fieldValue
			continue
		}

		for _, key := range keys {
			if v, ok, err := tryConvertFieldValue(key, fieldValue); err != nil {
				return nil, err
			} else if ok {
				fields[key] = v
				matched = true
				break
			}
		}

		if !matched {
			if len(keys) > 0 {
				if value, ok := def.Fields().Get(keys[0]); ok {
					fields[keys[0]] = value
					continue
				}
			}

			for _, customKey := range d.customFieldKeys {
				if customKey.String() == fieldName {
					if v, ok, err := tryConvertFieldValue(customKey, fieldValue); err != nil {
						return nil, err
					} else if ok {
						fields[customKey] = v
						matched = true
						break
					}
				}
			}
			if matched {
				continue
			}

			if d.strictMode {
				return nil, ErrUnknownField.WithOptions(
					fieldNameField(fieldName),
					kindField(decoded.Kind),
				).Errorf("unknown field %q in kind %q", fieldName, decoded.Kind)
			}

			unknownFields[fieldName] = fieldValue
		}
	}

	var causes []error
	for _, causeData := range decoded.Causes {
		cause, err := d.unmarshalCause(causeData)
		if err != nil {
			return nil, err
		}
		causes = append(causes, cause)
	}

	return &unmarshaledError{
		definedError:  definedError,
		fields:        fields,
		unknownFields: unknownFields,
		stack:         decoded.Stack,
		causes:        causes,
	}, nil
}

func (d *Unmarshaler[T]) unmarshalCause(causeData map[string]any) (error, error) {
	causeDecoded := mapToDecodedData(causeData)

	cause, err := d.unmarshal(causeDecoded)
	if err != nil {
		if errors.Is(err, ErrInternal) {
			return nil, ErrInternal.Wrapf(err, "failed to unmarshal cause data")
		}

		msg, hasMessage := causeData["message"].(string)
		if !hasMessage {
			msg = fmt.Sprintf("<unknown: %v>", causeData)
		}

		typeName, hasTypeName := causeData["type"].(string)
		if !hasTypeName {
			typeName = "<unknown>"
		}

		var nestedCauses []error
		if causesData, hasCauses := causeData["causes"].([]any); hasCauses {
			for _, nestedCauseData := range causesData {
				nestedCauseMap, ok := nestedCauseData.(map[string]any)
				if !ok {
					continue
				}
				nestedCause, err := d.unmarshalCause(nestedCauseMap)
				if err != nil {
					return nil, err
				}
				nestedCauses = append(nestedCauses, nestedCause)
			}
		}

		if len(nestedCauses) == 0 {
			if typeName == errdefDefinitionTypeName {
				if def, ok := d.resolveDefinitionFromMessage(msg); ok {
					return def, nil
				}
			}

			if sentinelErr, ok := d.sentinelErrors[sentinelKey{typeName: typeName, message: msg}]; ok {
				return sentinelErr, nil
			}
		}

		unknownErr := &unknownCauseError{
			message:  msg,
			typeName: typeName,
			causes:   nestedCauses,
		}
		return unknownErr, nil
	}

	return cause, nil
}

func (d *Unmarshaler[T]) resolveDefinitionFromMessage(msg string) (*errdef.Definition, bool) {
	kind := errdef.Kind(msg)
	if kind == errdefDefinitionEmptyKindMessage {
		kind = ""
	}
	return d.resolver.ResolveKindStrict(kind)
}

func (d *Unmarshaler[T]) resolveKind(kind errdef.Kind) (*errdef.Definition, error) {
	if fallback, ok := d.resolver.(*resolver.FallbackResolver); ok {
		if d.strictMode {
			def, ok := fallback.ResolveKindStrict(kind)
			if !ok {
				return nil, ErrUnknownKind.WithOptions(kindField(kind)).New("unknown kind")
			}
			return def, nil
		}
		return fallback.ResolveKind(kind), nil
	}
	def, ok := d.resolver.ResolveKindStrict(kind)
	if !ok {
		return nil, ErrUnknownKind.WithOptions(kindField(kind)).New("unknown kind")
	}
	return def, nil
}
