package errdef_test

import (
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
