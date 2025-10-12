# Protocol Buffers Example

This example demonstrates how to implement a custom decoder for Protocol Buffers format with `errdef/unmarshaler`.

## Custom Decoder Implementation

The unmarshaler accepts a custom decoder function to convert your serialized format into `DecodedData`:

```go
func protoDecoder(msg *ErrorProto) (*unmarshaler.DecodedData, error) {
  d := &unmarshaler.DecodedData{
    Message: msg.Message,
    Kind:    errdef.Kind(msg.Kind),
  }

  if len(msg.Fields) > 0 {
    d.Fields = make(map[string]any)
    for k, v := range msg.Fields {
      fv, err := fieldValueToAny(v)
      if err != nil {
        return nil, err
      }
      d.Fields[k] = fv
    }
  }

  if len(msg.Stack) > 0 {
    d.Stack = make([]errdef.Frame, len(msg.Stack))
    for i, frame := range msg.Stack {
      d.Stack[i] = errdef.Frame{
        Func: frame.Func,
        File: frame.File,
        Line: int(frame.Line),
      }
    }
  }

  if len(msg.Causes) > 0 {
    d.Causes = make([]*unmarshaler.DecodedData, len(msg.Causes))
    for i, cause := range msg.Causes {
      causeData, err := causeProtoToDecodedData(cause)
      if err != nil {
        return nil, err
      }
      d.Causes[i] = causeData
    }
  }
  return d, nil
}
```

Register the custom decoder with `unmarshaler.New`:

```go
r := resolver.New(ErrNotFound)
u := unmarshaler.New(r, protoDecoder,
  unmarshaler.WithBuiltinFields(),
  unmarshaler.WithStandardSentinelErrors(),
)

restored, err := u.Unmarshal(&protoMsg)
```

> **Note:** See [main.go](./main.go) for the complete implementation including `fieldValueToAny` and `causeProtoToMap` helper functions.

## Running the Example

```bash
go run .
```

## Regenerating Protocol Buffers Code

If you modify `error.proto`, regenerate the Go code with:

```bash
protoc --go_out=. --go_opt=paths=source_relative error.proto
```

**Requirements:**
- Install `protoc` (Protocol Buffers compiler)
- Install `protoc-gen-go`:

  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  ```
