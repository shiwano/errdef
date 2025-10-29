package errdef

import (
	"encoding/json"
	"log/slog"
	"runtime"
	"slices"
)

type (
	// Stack represents a stack trace captured when an error was created.
	Stack interface {
		// Frames returns the stack trace as structured frame information.
		Frames() []Frame
		// HeadFrame returns the top frame of the stack trace.
		HeadFrame() (Frame, bool)
		// Len returns the number of frames in the stack trace.
		Len() int
		// IsZero returns true if the stack trace is empty.
		IsZero() bool
	}

	// Frame represents a single frame in a stack trace.
	Frame struct {
		Func string `json:"func"`
		File string `json:"file"`
		Line int    `json:"line"`
	}

	stackGetter interface {
		Stack() Stack
	}

	stack []uintptr
)

const (
	callersDepth = 32

	// callersSkip is the number of skip frames when using the Definition methods.
	// 4 frames: runtime.Callers, newStack, newError, and the Definition methods.
	callersSkip = 4
)

var (
	_ Stack          = stack{}
	_ StackTracer    = stack{}
	_ json.Marshaler = stack{}
	_ slog.LogValuer = stack{}

	_ slog.LogValuer = Frame{}
)

func newStack(depth int, skip int) stack {
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip, pcs)
	return pcs[:n]
}

func (s stack) Frames() []Frame {
	if len(s) == 0 {
		return nil
	}
	fs := runtime.CallersFrames(s)
	frames := make([]Frame, 0, len(s))
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

func (s stack) HeadFrame() (Frame, bool) {
	if len(s) == 0 {
		return Frame{}, false
	}
	fs := runtime.CallersFrames(s)
	f, _ := fs.Next()
	frame := Frame{
		Func: f.Function,
		File: f.File,
		Line: f.Line,
	}
	return frame, true
}

func (s stack) Len() int {
	return len(s)
}

func (s stack) IsZero() bool {
	return len(s) == 0
}

func (s stack) StackTrace() []uintptr {
	return slices.Clone(s)
}

func (s stack) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Frames())
}

func (s stack) LogValue() slog.Value {
	return slog.AnyValue(s.Frames())
}

func (f Frame) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("func", f.Func),
		slog.String("file", f.File),
		slog.Int("line", f.Line),
	)
}
