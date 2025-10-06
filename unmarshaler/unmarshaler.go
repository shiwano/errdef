package unmarshaler

import (
	"errors"

	"github.com/shiwano/errdef"
)

type (
	Unmarshaler struct {
		resolver            Resolver
		decoder             Decoder
		sentinelErrors      map[sentinelKey]error
		additionalFieldKeys []errdef.FieldKey
	}

	Resolver interface {
		Definitions() []*errdef.Definition
	}

	Option func(*Unmarshaler)

	sentinelKey struct {
		typeName string
		message  string
	}
)

const redactedStr = "[REDACTED]"

func New(resolver Resolver, decoder Decoder, opts ...Option) *Unmarshaler {
	u := &Unmarshaler{
		resolver: resolver,
		decoder:  decoder,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

func NewJSON(resolver Resolver, opts ...Option) *Unmarshaler {
	u := &Unmarshaler{
		resolver: resolver,
		decoder:  jsonDecoder,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

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
			if v, ok := tryConvertFieldValue(key, fieldValue); ok {
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
					if v, ok := tryConvertFieldValue(additionalKey, fieldValue); ok {
						fields[additionalKey] = v
						matched = true
						break
					}
				}
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
	causeDecoded, err := mapToDecodedData(causeData)
	if err != nil {
		return nil, ErrInternal.Wrapf(err, "failed to convert cause data")
	}

	cause, err := d.unmarshal(causeDecoded)
	if err != nil {
		if errors.Is(err, ErrInternal) {
			return nil, ErrInternal.Wrapf(err, "failed to unmarshal cause data")
		}

		msg, hasMessage := causeData["message"].(string)
		if !hasMessage {
			msg = "<unknown>"
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
			if sentinelErr, ok := d.sentinelErrors[sentinelKey{typeName: typeName, message: msg}]; ok {
				return sentinelErr, nil
			}
		}

		var unknownErr error
		if len(nestedCauses) == 0 {
			unknownErr = UnknownError.
				WithOptions(typeField(typeName)).
				New(msg)
		} else if len(nestedCauses) == 1 {
			unknownErr = UnknownError.
				WithOptions(typeField(typeName)).
				Wrapf(nestedCauses[0], "%s", msg)
		} else {
			unknownErr = UnknownError.
				WithOptions(typeField(typeName)).
				Join(nestedCauses...)
		}
		return unknownErr, nil
	}

	return cause, nil
}

func (d *Unmarshaler) resolveKind(kind errdef.Kind) (*errdef.Definition, error) {
	if strict, ok := d.resolver.(*errdef.Resolver); ok {
		def, ok := strict.ResolveKind(kind)
		if !ok {
			return nil, ErrKindNotFound.WithOptions(kindField(kind)).New("kind not found")
		}
		return def, nil
	} else if fallback, ok := d.resolver.(*errdef.FallbackResolver); ok {
		return fallback.ResolveKind(kind), nil
	}
	return nil, ErrInternal.New("resolver does not support kind resolution")
}
