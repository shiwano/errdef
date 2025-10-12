package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNotFound        = errdef.Define("not_found")
	UserID, UserIDFrom = errdef.DefineField[string]("user_id")
)

func main() {
	orig := ErrNotFound.WithOptions(
		UserID("u123"),
		errdef.Details{"info": "additional info"},
	).Wrapf(io.EOF, "user not found")

	fmt.Printf("original error: %+v\n", orig)
	fmt.Printf("original error user_id: %s\n", UserIDFrom.OrZero(orig))
	fmt.Printf("original error details: %+v\n", errdef.DetailsFrom.OrZero(orig))
	fmt.Printf("original error is io.EOF: %v\n", errors.Is(orig, io.EOF))

	protoBytes, err := marshalProto(orig.(errdef.Error))
	if err != nil {
		panic(err)
	}

	var protoMsg ErrorProto
	if err := proto.Unmarshal(protoBytes, &protoMsg); err != nil {
		panic(err)
	}

	r := resolver.New(ErrNotFound)
	u := unmarshaler.New(r, protoDecoder,
		unmarshaler.WithBuiltinFields(),
		unmarshaler.WithStandardSentinelErrors(),
	)

	restored, err := u.Unmarshal(&protoMsg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("restored error: %+v\n", restored)
	fmt.Printf("restored error user_id: %s\n", UserIDFrom.OrZero(restored))
	fmt.Printf("restored error details: %+v\n", errdef.DetailsFrom.OrZero(restored))
	fmt.Printf("restored error is io.EOF: %v\n", errors.Is(restored, io.EOF))
}

func marshalProto(e errdef.Error) ([]byte, error) {
	msg := &ErrorProto{
		Message: e.Error(),
		Kind:    string(e.Kind()),
	}

	if e.Fields().Len() > 0 {
		msg.Fields = make(map[string]*FieldValue)
		for k, v := range e.Fields().SortedSeq() {
			fv, err := anyToFieldValue(v.Value())
			if err != nil {
				return nil, err
			}
			msg.Fields[k.String()] = fv
		}
	}

	if e.Stack().Len() > 0 {
		for _, frame := range e.Stack().Frames() {
			msg.Stack = append(msg.Stack, &StackFrame{
				Func: frame.Func,
				File: frame.File,
				Line: int32(frame.Line),
			})
		}
	}

	causes, _ := e.UnwrapTree()
	if len(causes) > 0 {
		msg.Causes = make([]*CauseProto, len(causes))
		for i, node := range causes {
			causeProto, err := errorNodeToCauseProto(node)
			if err != nil {
				return nil, err
			}
			msg.Causes[i] = causeProto
		}
	}
	return proto.Marshal(msg)
}

func errorNodeToCauseProto(node errdef.ErrorNode) (*CauseProto, error) {
	cp := &CauseProto{
		Message: node.Error.Error(),
	}

	switch e := node.Error.(type) {
	case errdef.Error:
		cp.Kind = string(e.Kind())

		if e.Fields().Len() > 0 {
			cp.Fields = make(map[string]*FieldValue)
			for k, v := range e.Fields().SortedSeq() {
				fv, err := anyToFieldValue(v.Value())
				if err != nil {
					return nil, err
				}
				cp.Fields[k.String()] = fv
			}
		}

		if e.Stack().Len() > 0 {
			for _, frame := range e.Stack().Frames() {
				cp.Stack = append(cp.Stack, &StackFrame{
					Func: frame.Func,
					File: frame.File,
					Line: int32(frame.Line),
				})
			}
		}
	case errdef.ErrorTypeNamer:
		cp.Type = e.TypeName()
	default:
		cp.Type = fmt.Sprintf("%T", node.Error)
	}

	if len(node.Causes) > 0 {
		cp.Causes = make([]*CauseProto, len(node.Causes))
		for i, cause := range node.Causes {
			causeProto, err := errorNodeToCauseProto(cause)
			if err != nil {
				return nil, err
			}
			cp.Causes[i] = causeProto
		}
	}
	return cp, nil
}

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

func causeProtoToDecodedData(cp *CauseProto) (*unmarshaler.DecodedData, error) {
	d := &unmarshaler.DecodedData{
		Message: cp.Message,
		Kind:    errdef.Kind(cp.Kind),
		Type:    cp.Type,
	}

	if len(cp.Fields) > 0 {
		d.Fields = make(map[string]any)
		for k, v := range cp.Fields {
			fv, err := fieldValueToAny(v)
			if err != nil {
				return nil, err
			}
			d.Fields[k] = fv
		}
	}

	if len(cp.Stack) > 0 {
		d.Stack = make([]errdef.Frame, len(cp.Stack))
		for i, frame := range cp.Stack {
			d.Stack[i] = errdef.Frame{
				Func: frame.Func,
				File: frame.File,
				Line: int(frame.Line),
			}
		}
	}

	if len(cp.Causes) > 0 {
		d.Causes = make([]*unmarshaler.DecodedData, len(cp.Causes))
		for i, cause := range cp.Causes {
			causeData, err := causeProtoToDecodedData(cause)
			if err != nil {
				return nil, err
			}
			d.Causes[i] = causeData
		}
	}
	return d, nil
}

func anyToFieldValue(v any) (*FieldValue, error) {
	switch val := v.(type) {
	case string:
		return &FieldValue{Value: &FieldValue_StringValue{StringValue: val}}, nil
	case int:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case int8:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case int16:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case int32:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case int64:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: val}}, nil
	case uint:
		if val > math.MaxInt64 {
			return nil, fmt.Errorf("uint value %d exceeds maximum int64 value", val)
		}
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case uint8:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case uint16:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case uint32:
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case uint64:
		if val > math.MaxInt64 {
			return nil, fmt.Errorf("uint64 value %d exceeds maximum int64 value", val)
		}
		return &FieldValue{Value: &FieldValue_IntValue{IntValue: int64(val)}}, nil
	case float32:
		return &FieldValue{Value: &FieldValue_DoubleValue{DoubleValue: float64(val)}}, nil
	case float64:
		return &FieldValue{Value: &FieldValue_DoubleValue{DoubleValue: val}}, nil
	case bool:
		return &FieldValue{Value: &FieldValue_BoolValue{BoolValue: val}}, nil
	case complex64, complex128:
		return nil, fmt.Errorf("unsupported type: %T (complex numbers are not supported)", v)
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value of type %T: %w", v, err)
		}
		return &FieldValue{Value: &FieldValue_BytesValue{BytesValue: jsonBytes}}, nil
	}
}

func fieldValueToAny(fv *FieldValue) (any, error) {
	switch v := fv.Value.(type) {
	case *FieldValue_StringValue:
		return v.StringValue, nil
	case *FieldValue_IntValue:
		return v.IntValue, nil
	case *FieldValue_DoubleValue:
		return v.DoubleValue, nil
	case *FieldValue_BoolValue:
		return v.BoolValue, nil
	case *FieldValue_BytesValue:
		var data map[string]any
		if err := json.Unmarshal(v.BytesValue, &data); err != nil {
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unknown field value type: %T", fv.Value)
	}
}
