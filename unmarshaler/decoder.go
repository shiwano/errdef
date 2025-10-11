package unmarshaler

import (
	"encoding/json"

	"github.com/shiwano/errdef"
)

type (
	// Decoder is a function that decodes error data of type T into DecodedData.
	//
	// The type parameter T specifies the input data type, enabling type-safe
	// deserialization from various formats beyond []byte. Common examples include:
	//   - []byte for JSON, XML, or other text-based formats
	//   - Protocol Buffers messages (e.g., *ErrorProto)
	//   - Custom structured data types
	Decoder[T any] func(data T) (*DecodedData, error)

	// DecodedData represents the decoded error information that will be unmarshaled
	// into an error instance. This structure is designed to be flexible enough to
	// represent both defined errors (with Kind) and unknown external errors (with Type).
	DecodedData struct {
		// Message is the error message.
		Message string `json:"message"`

		// Kind identifies the type of defined error. This field is used for errors
		// that are registered in the error definition system.
		Kind errdef.Kind `json:"kind"`

		// Fields contains structured data associated with the error.
		// The keys should match the field names defined in the error definition.
		Fields map[string]any `json:"fields"`

		// Stack contains the stack trace where the error was created.
		Stack []errdef.Frame `json:"stack"`

		// Causes contains the wrapped errors. Each cause can be one of two types:
		//
		// 1. Defined errors (errors registered in the error definition system):
		//   - "message" (string, required): the error message
		//   - "kind" (string, required): the error kind
		//   - "fields" (map[string]any, optional): structured data
		//   - "stack" ([]Frame, optional): stack trace
		//   - "causes" ([]map[string]any, optional): nested wrapped errors
		//
		// 2. External/unknown errors (errors from external libraries or unknown sources):
		//   - "message" (string, required): the error message
		//   - "type" (string, optional): the Go type name formatted as fmt.Sprintf("%T", err) (e.g., "*errors.errorString")
		//   - "causes" ([]map[string]any, optional): nested wrapped errors
		//
		// Causes can be nested recursively, allowing deep error chains to be preserved.
		//
		// If a cause does not conform to either format, it will be treated as a single unknown error
		// with default values for missing fields.
		//
		// NOTE:
		// For external errors without nested causes, the combination of "type" and "message"
		// is used to resolve registered sentinel errors (e.g., io.EOF, sql.ErrNoRows).
		Causes []map[string]any `json:"causes"`
	}
)

func jsonToDecodedData(data []byte) (*DecodedData, error) {
	var decoded DecodedData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}

func mapToDecodedData(data map[string]any) *DecodedData {
	decoded := DecodedData{}

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
	return &decoded
}
