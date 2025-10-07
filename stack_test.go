package errdef_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
)

func TestStack_Frames(t *testing.T) {
	def := errdef.Define("test_error")
	err := def.New("test error")
	frames := err.(errdef.Error).Stack().Frames()

	if len(frames) == 0 {
		t.Error("want non-empty frames")
	}

	f := frames[0]
	if !strings.Contains(f.Func, "TestStack_Frames") {
		t.Error("skip runtime functions")
	}
	if !strings.Contains(f.File, "stack_test.go") {
		t.Error("want non-empty file name")
	}
	if f.Line == 0 {
		t.Error("want non-zero line number")
	}
}

func TestStack_HeadFrame(t *testing.T) {
	t.Run("with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test error")
		stack := err.(errdef.Error).Stack()

		frame, ok := stack.HeadFrame()
		if !ok {
			t.Error("want ok to be true for non-empty stack")
		}

		if !strings.Contains(frame.Func, "TestStack_HeadFrame") {
			t.Errorf("want func to contain 'TestStack_HeadFrame', got %s", frame.Func)
		}
		if !strings.Contains(frame.File, "stack_test.go") {
			t.Errorf("want file to contain 'stack_test.go', got %s", frame.File)
		}
		if frame.Line == 0 {
			t.Error("want non-zero line number")
		}
	})

	t.Run("empty stack", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test error")
		stack := err.(errdef.Error).Stack()

		frame, ok := stack.HeadFrame()
		if ok {
			t.Error("want ok to be false for empty stack")
		}
		emptyFrame := errdef.Frame{}
		if frame != emptyFrame {
			t.Errorf("want zero-value frame for empty stack, got %+v", frame)
		}
	})
}

func TestStack_Len(t *testing.T) {
	def := errdef.Define("test_error")
	err := def.New("test error")
	stack := err.(errdef.Error).Stack()

	length := stack.Len()
	if length == 0 {
		t.Error("want non-zero length")
	}

	frames := stack.Frames()
	if length != len(frames) {
		t.Errorf("want Len() == len(Frames()), got Len()=%d, len(Frames())=%d", length, len(frames))
	}
}

func TestStack_LogValue(t *testing.T) {
	t.Run("with stack", func(t *testing.T) {
		def := errdef.Define("test_error")
		err := def.New("test error")
		stack := err.(errdef.Error).Stack()
		value := stack.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("stack", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		frames := stack.Frames()

		want := make([]any, len(frames))
		for i, f := range frames {
			want[i] = map[string]any{
				"func": f.Func,
				"file": f.File,
				"line": float64(f.Line),
			}
		}

		if !reflect.DeepEqual(result["stack"], want) {
			t.Errorf("want stack %+v, got %+v", want, result["stack"])
		}
	})

	t.Run("empty stack", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test error")
		stack := err.(errdef.Error).Stack()
		value := stack.(slog.LogValuer).LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("stack", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if result["stack"] != nil {
			t.Errorf("want stack to be nil for empty stack, got %+v", result["stack"])
		}
	})
}
