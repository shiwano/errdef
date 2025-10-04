package unmarshaler

import (
	"encoding/json"
	"errors"

	"github.com/shiwano/errdef"
)

type (
	Unmarshaler struct {
		resolver Resolver
		decoder  Decoder
	}

	Resolver interface {
		Definitions() []*errdef.Definition
	}
)

const redactedStr = "[REDACTED]"

func New(resolver Resolver, decoder Decoder) *Unmarshaler {
	return &Unmarshaler{
		resolver: resolver,
		decoder:  decoder,
	}
}

func NewJSON(resolver Resolver) *Unmarshaler {
	return &Unmarshaler{
		resolver: resolver,
		decoder:  jsonDecoder,
	}
}

func (d *Unmarshaler) Unmarshal(data []byte) (errdef.Error, error) {
	decoded, err := d.decoder(data)
	if err != nil {
		return nil, ErrDecodeFailure.Wrap(err)
	}
	return d.unmarshal(decoded)
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

func (d *Unmarshaler) unmarshal(decoded *DecodedData) (errdef.Error, error) {
	def, err := d.resolveKind(errdef.Kind(decoded.Kind))
	if err != nil {
		return nil, err
	}

	definedError := def.
		WithOptions(errdef.NoTrace()).
		New(decoded.Message).(errdef.Error)

	fields := make(map[errdef.FieldKey]errdef.FieldValue)
	unknownFields := make(map[string]any)

	if decoded.Fields != nil {
		for fieldName, fieldValue := range decoded.Fields {
			keys := def.Fields().FindKeys(fieldName)
			matched := false

			if s, ok := fieldValue.(string); ok && s == redactedStr {
				unknownFields[fieldName] = fieldValue
				continue
			}

			for _, key := range keys {
				if newFieldValue, ok := key.NewValue(fieldValue); ok {
					fields[key] = newFieldValue
					matched = true
					break
				}

				if f64, ok := fieldValue.(float64); ok {
					if v, ok := tryConvertFloat64(key, f64); ok {
						fields[key] = v
						matched = true
						break
					}
				}

				if m, ok := fieldValue.(map[string]any); ok {
					if v, ok := tryConvertMapToStruct(key, m); ok {
						fields[key] = v
						matched = true
						break
					}
				}

				if s, ok := fieldValue.([]any); ok {
					if v, ok := tryConvertSlice(key, s); ok {
						fields[key] = v
						matched = true
						break
					}
				}
			}

			if !matched {
				if len(keys) > 0 {
					if value, ok := def.Fields().Get(keys[0]); ok {
						fields[keys[0]] = value
						continue
					}
				}
				unknownFields[fieldName] = fieldValue
			}
		}
	}

	var causes []error
	for _, causeData := range decoded.Causes {
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

			if dataRaw, ok := causeData["data"]; ok {
				dataJSON, err := json.Marshal(dataRaw)
				if err != nil {
					return nil, ErrInternal.Wrapf(err, "failed to marshal cause data field")
				}

				cause := ForeignCause.
					WithOptions(
						typeField(typeName),
						dataField(string(dataJSON)),
					).
					New(msg)
				causes = append(causes, cause)
			} else {
				causeJSON, err := json.Marshal(causeData)
				if err != nil {
					return nil, ErrInternal.Wrapf(err, "failed to marshal cause data")
				}

				cause := ForeignCause.
					WithOptions(
						typeField(typeName),
						rawDataField(string(causeJSON)),
					).
					New(msg)
				causes = append(causes, cause)
			}
		} else {
			causes = append(causes, cause)
		}
	}

	return &unmarshaledError{
		definedError:  definedError,
		fields:        fields,
		unknownFields: unknownFields,
		stack:         decoded.Stack,
		causes:        causes,
	}, nil
}
