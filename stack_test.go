package errdef_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/shiwano/errdef"
)

func TestStack_StackTrace(t *testing.T) {
	err := errdef.New("test error")
	result := err.(errdef.Error).Stack().StackTrace()

	if len(result) == 0 {
		t.Error("want non-empty stack trace")
	}

	hasValidPC := false
	for _, pc := range result {
		if pc != 0 {
			hasValidPC = true
			break
		}
	}
	if !hasValidPC {
		t.Error("want at least one valid program counter")
	}
}

func TestStack_Frames(t *testing.T) {
	err := errdef.New("test error")
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

func TestStack_Len(t *testing.T) {
	err := errdef.New("test error")
	stack := err.(errdef.Error).Stack()

	length := stack.Len()
	if length == 0 {
		t.Error("want non-zero length")
	}

	stackTrace := stack.StackTrace()
	if length != len(stackTrace) {
		t.Errorf("want Len() == len(StackTrace()), got Len()=%d, len(StackTrace())=%d", length, len(stackTrace))
	}

	frames := stack.Frames()
	if length != len(frames) {
		t.Errorf("want Len() == len(Frames()), got Len()=%d, len(Frames())=%d", length, len(frames))
	}
}

func TestStack_LogValue(t *testing.T) {
	t.Run("with stack", func(t *testing.T) {
		err := errdef.New("test error")
		stack := err.(errdef.Error).Stack()

		logValuer, ok := stack.(slog.LogValuer)
		if !ok {
			t.Fatal("want stack to implement slog.LogValuer")
		}

		value := logValuer.LogValue()
		frames := value.Any().([]errdef.Frame)

		if len(frames) == 0 {
			t.Error("want non-empty frames in log value")
		}

		firstFrame := frames[0]
		if !strings.Contains(firstFrame.Func, "TestStack_LogValue") {
			t.Errorf("want function name to contain TestStack_LogValue, got %q", firstFrame.Func)
		}
		if !strings.Contains(firstFrame.File, "stack_test.go") {
			t.Errorf("want file to contain stack_test.go, got %q", firstFrame.File)
		}
		if firstFrame.Line <= 0 {
			t.Errorf("want positive line number, got %d", firstFrame.Line)
		}
	})

	t.Run("empty stack", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test error")
		stack := err.(errdef.Error).Stack()

		if stack == nil {
			t.Skip("stack is nil when trace is disabled")
		}

		logValuer := stack.(slog.LogValuer)
		value := logValuer.LogValue()
		frames := value.Any().([]errdef.Frame)

		if len(frames) != 0 {
			t.Errorf("want empty frames when trace is disabled, got %d frames", len(frames))
		}
	})
}
