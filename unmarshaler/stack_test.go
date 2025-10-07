package unmarshaler_test

import (
	"encoding/json"
	"log/slog"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
)

func TestStack(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	if stack == nil {
		t.Fatal("want stack to exist")
	}

	frames := stack.Frames()
	if len(frames) == 0 {
		t.Error("want stack frames to exist")
	}

	if frame, ok := stack.HeadFrame(); ok {
		if frame.Func == "" {
			t.Error("want function name to be set")
		}
		if frame.File == "" {
			t.Error("want file name to be set")
		}
		if frame.Line == 0 {
			t.Error("want line number to be set")
		}
	} else {
		t.Error("want head frame to exist")
	}
}

func TestStack_Len(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	if stack.Len() == 0 {
		t.Error("want non-zero stack length")
	}
}

func TestStack_Frames(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	frames := stack.Frames()
	if len(frames) != stack.Len() {
		t.Errorf("want frames length %d, got %d", stack.Len(), len(frames))
	}
}

func TestStack_HeadFrame(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	frame, ok := stack.HeadFrame()
	if !ok {
		t.Fatal("want head frame to exist")
	}

	frames := stack.Frames()
	if frame != frames[0] {
		t.Error("want head frame to be first frame")
	}
}

func TestStack_LogValue(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	logValue := stack.(slog.LogValuer).LogValue()
	if logValue.Kind() != slog.KindAny {
		t.Errorf("want log value kind %v, got %v", slog.KindAny, logValue.Kind())
	}
}

func TestStack_MarshalJSON(t *testing.T) {
	def := errdef.Define("test_error")
	r := resolver.New(def)
	u := unmarshaler.NewJSON(r)

	original := def.New("test message")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	unmarshaled, err := u.Unmarshal(data)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stack := unmarshaled.Stack()
	marshaled, err := json.Marshal(stack)
	if err != nil {
		t.Fatalf("failed to marshal stack: %v", err)
	}

	var result []any
	if err := json.Unmarshal(marshaled, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(result) != stack.Len() {
		t.Errorf("length mismatch: want %d, got %d", stack.Len(), len(result))
	} else if len(result) == 0 {
		t.Fatal("want non-empty stack")
	}

	headFrame, ok := stack.HeadFrame()
	if !ok {
		t.Fatal("want head frame to exist")
	}

	firstFrame, ok := result[0].(map[string]any)
	if !ok {
		t.Fatalf("want first frame to be map[string]any, got %T", result[0])
	}

	want := map[string]any{
		"func": headFrame.Func,
		"file": headFrame.File,
		"line": float64(headFrame.Line),
	}

	if !reflect.DeepEqual(firstFrame, want) {
		t.Errorf("data mismatch:\nwant: %+v\ngot:  %+v", want, firstFrame)
	}
}
