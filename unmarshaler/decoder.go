package unmarshaler

import (
	"encoding/json"

	"github.com/shiwano/errdef"
)

type (
	Decoder func(data []byte) (*DecodedData, error)

	DecodedData struct {
		Message string           `json:"message"`
		Kind    errdef.Kind      `json:"kind"`
		Fields  map[string]any   `json:"fields"`
		Stack   []errdef.Frame   `json:"stack"`
		Causes  []map[string]any `json:"causes"`
	}
)

func jsonDecoder(data []byte) (*DecodedData, error) {
	var decoded DecodedData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}

func mapToDecodedData(data map[string]any) (*DecodedData, error) {
	decoded := &DecodedData{}

	if msg, ok := data["message"].(string); ok {
		decoded.Message = msg
	}

	if kind, ok := data["kind"].(string); ok {
		decoded.Kind = errdef.Kind(kind)
	}

	if fields, ok := data["fields"].(map[string]any); ok {
		decoded.Fields = fields
	}

	if stackAny, ok := data["stack"].([]any); ok {
		frames := make([]errdef.Frame, 0, len(stackAny))
		for _, s := range stackAny {
			if frameMap, ok := s.(map[string]any); ok {
				frame := errdef.Frame{}
				if fn, ok := frameMap["func"].(string); ok {
					frame.Func = fn
				}
				if file, ok := frameMap["file"].(string); ok {
					frame.File = file
				}
				if line, ok := frameMap["line"].(float64); ok {
					frame.Line = int(line)
				}
				frames = append(frames, frame)
			}
		}
		decoded.Stack = frames
	}

	if causesAny, ok := data["causes"].([]any); ok {
		causes := make([]map[string]any, 0, len(causesAny))
		for _, c := range causesAny {
			if causeMap, ok := c.(map[string]any); ok {
				causes = append(causes, causeMap)
			}
		}
		decoded.Causes = causes
	}
	return decoded, nil
}
