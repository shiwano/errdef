package errdef

import (
	"runtime"
)

type (
	// Stack represents a stack trace captured when an error was created.
	Stack interface {
		// StackTrace returns the raw stack trace as program counters.
		StackTrace() []uintptr
		// Frames returns the stack trace as structured frame information.
		Frames() []Frame
	}

	// Frame represents a single frame in a stack trace.
	Frame struct {
		Func string `json:"func"`
		File string `json:"file"`
		Line int    `json:"line"`
	}

	stack []uintptr
)

var _ Stack = (*stack)(nil)

const (
	maxStackDepth = 32

	// callersSkip is the number of skip frames when using the Definition methods.
	// 4 frames: runtime.Callers, newStack, newErr, and the Definition methods.
	callersSkip = 4
)

func newStack(skip int) stack {
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[:n]
}

func (s stack) StackTrace() []uintptr {
	if len(s) == 0 {
		return nil
	}
	return s[:]
}

func (s stack) Frames() []Frame {
	if len(s) == 0 {
		return nil
	}
	fs := runtime.CallersFrames(s.StackTrace())
	frames := make([]Frame, 0, maxStackDepth)
	for {
		f, more := fs.Next()
		frames = append(frames, Frame{
			Func: f.Function,
			File: f.File,
			Line: f.Line,
		})
		if !more {
			break
		}
	}
	return frames
}
