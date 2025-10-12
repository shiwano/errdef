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
		Kind errdef.Kind `json:"kind,omitempty"`

		// Type identifies the Go type name of external/unknown errors.
		// This field is used for errors from external libraries or unknown sources,
		// formatted as fmt.Sprintf("%T", err) (e.g., "*errors.errorString").
		// When both Type and Message match a registered sentinel error, it will be resolved.
		Type string `json:"type,omitempty"`

		// Fields contains structured data associated with the error.
		// The keys should match the field names defined in the error definition.
		//
		// Type conversion behavior:
		// Numeric values are represented differently depending on the input format:
		// - JSON (encoding/json): All numbers are decoded as float64
		// - Other formats (e.g., Protocol Buffers): Integers are often decoded as int64
		//
		// To handle numeric conversions flexibly across different formats, the unmarshaler
		// supports conversions from int64 and float64 to various target types (int, int8, int16,
		// int32, int64, uint8, uint16, uint32, uint64, float32, float64) as long as values
		// are within the target type's range and precision requirements are met.
		//
		// The unmarshaler attempts type conversion in the following priority:
		//
		// 1. Direct type match: If the value type matches the field definition, use it as-is.
		//
		// 2. float64 conversion (tryConvertFloat64): When the value is float64:
		//    - For int/int8/int16/int32/int64: Rejects fractional values, checks range, converts
		//    - For uint/uint8/uint16/uint32/uint64: Rejects fractional/negative values, checks range
		//    - For float32: Checks overflow against MaxFloat32, converts if within limits
		//    - For float64: Converts directly
		//
		// 3. int64 conversion (tryConvertInt64): When the value is int64:
		//    - For int/int8/int16/int32/int64: Checks range bounds and converts if within limits
		//    - For uint/uint8/uint16/uint32/uint64: Rejects negative values, checks range, converts
		//    - For float32: Checks precision loss (roundtrip int64→float32→int64), converts if lossless
		//    - For float64: Converts directly without precision loss
		//
		// 4. Complex types (tryConvertViaJSON): For map[string]any or []any:
		//    - Converts to struct/map/slice types via JSON marshaling/unmarshaling
		//
		// 5. Underlying type conversion (tryConvertByUnderlyingType):
		//    - For derived types (e.g., type UserID string), converts if underlying types match
		//
		// 6. Pointer conversion (tryConvertPointer):
		//    - For pointer types to primitives, creates pointer and converts the underlying value
		//
		// If conversion fails at all steps, the field is stored in UnknownFields with its original
		// type and value preserved.
		Fields map[string]any `json:"fields,omitempty"`

		// Stack contains the stack trace where the error was created.
		Stack []errdef.Frame `json:"stack,omitempty"`

		// Causes contains the wrapped errors. Each cause can be one of two types:
		//
		// 1. Defined errors (errors registered in the error definition system):
		//   - Message (required): the error message
		//   - Kind (required): the error kind
		//   - Fields (optional): structured data
		//   - Stack (optional): stack trace
		//   - Causes (optional): nested wrapped errors
		//
		// 2. External/unknown errors (errors from external libraries or unknown sources):
		//   - Message (required): the error message
		//   - Type (optional): the Go type name formatted as fmt.Sprintf("%T", err) (e.g., "*errors.errorString")
		//   - Causes (optional): nested wrapped errors
		//
		// Causes can be nested recursively, allowing deep error chains to be preserved.
		//
		// If a cause does not conform to either format, it will be treated as a single unknown error
		// with default values for missing fields.
		Causes []*DecodedData `json:"causes,omitempty"`
	}
)

func jsonToDecodedData(data []byte) (*DecodedData, error) {
	var decoded DecodedData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}
