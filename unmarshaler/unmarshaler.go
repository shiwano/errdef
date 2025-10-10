package unmarshaler

import (
	"errors"
	"fmt"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
)

type (
	// Unmarshaler unmarshals serialized error data into UnmarshaledError.
	Unmarshaler struct {
		resolver            resolver.Resolver
		decoder             Decoder
		sentinelErrors      map[sentinelKey]error
		additionalFieldKeys []errdef.FieldKey
		strictFields        bool
	}

	// Option is a function type for customizing Unmarshaler configuration.
	Option func(*Unmarshaler)

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
func New(resolver resolver.Resolver, decoder Decoder, opts ...Option) *Unmarshaler {
	u := &Unmarshaler{
		resolver: resolver,
		decoder:  decoder,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// NewJSON creates a new Unmarshaler with a JSON decoder.
func NewJSON(resolver resolver.Resolver, opts ...Option) *Unmarshaler {
	return New(resolver, jsonDecoder, opts...)
}

// Unmarshal deserializes the given byte data into an UnmarshaledError.
func (d *Unmarshaler) Unmarshal(data []byte) (UnmarshaledError, error) {
	decoded, err := d.decoder(data)
	if err != nil {
		return nil, ErrDecodeFailure.Wrap(err)
	}
	return d.unmarshal(decoded)
}

func (d *Unmarshaler) unmarshal(decoded *DecodedData) (UnmarshaledError, error) {
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

			for _, additionalKey := range d.additionalFieldKeys {
				if additionalKey.String() == fieldName {
					if v, ok, err := tryConvertFieldValue(additionalKey, fieldValue); err != nil {
						return nil, err
					} else if ok {
						fields[additionalKey] = v
						matched = true
						break
					}
				}
			}

			if !matched && d.strictFields {
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

func (d *Unmarshaler) unmarshalCause(causeData map[string]any) (error, error) {
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

func (d *Unmarshaler) resolveDefinitionFromMessage(msg string) (*errdef.Definition, bool) {
	kind := errdef.Kind(msg)
	if kind == errdefDefinitionEmptyKindMessage {
		kind = ""
	}
	return d.resolver.ResolveKindStrict(kind)
}

func (d *Unmarshaler) resolveKind(kind errdef.Kind) (*errdef.Definition, error) {
	if fallback, ok := d.resolver.(*resolver.FallbackResolver); ok {
		return fallback.ResolveKind(kind), nil
	}
	def, ok := d.resolver.ResolveKindStrict(kind)
	if !ok {
		return nil, ErrKindNotFound.WithOptions(kindField(kind)).New("kind not found")
	}
	return def, nil
}
