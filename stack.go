package errdef

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type (
	// Stack represents a stack trace captured when an error was created.
	Stack interface {
		// Frames returns the stack trace as structured frame information.
		Frames() []Frame
		// HeadFrame returns the top frame of the stack trace.
		HeadFrame() (Frame, bool)
		// FramesAndSource returns an iterator that yields frames and their source code snippets.
		// Source code will be empty string if not available or if the frame exceeds the configured depth.
		FramesAndSource() iter.Seq2[Frame, string]
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

	stack struct {
		pcs         []uintptr
		sourceLines int
		sourceDepth int
	}
)

const (
	callersDepth = 32

	// callersSkip is the number of skip frames when using the Definition methods.
	// 4 frames: runtime.Callers, newStack, newError, and the Definition methods.
	callersSkip = 4
)

var (
	_ Stack          = (*stack)(nil)
	_ StackTracer    = (*stack)(nil)
	_ json.Marshaler = (*stack)(nil)
	_ slog.LogValuer = (*stack)(nil)

	_ slog.LogValuer = Frame{}
)

var (
	sourceAvailable   *bool
	sourceAvailableMu sync.Mutex

	sourceFileCache   = make(map[string][]string)
	sourceFileCacheMu sync.RWMutex
)

func newStack(depth int, skip int, sourceLines int, sourceDepth int) *stack {
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip, pcs)
	return &stack{
		pcs:         pcs[:n],
		sourceLines: sourceLines,
		sourceDepth: sourceDepth,
	}
}

func (s *stack) Frames() []Frame {
	if s == nil || len(s.pcs) == 0 {
		return nil
	}
	fs := runtime.CallersFrames(s.pcs)
	frames := make([]Frame, 0, len(s.pcs))
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

func (s *stack) HeadFrame() (Frame, bool) {
	if s == nil || len(s.pcs) == 0 {
		return Frame{}, false
	}
	fs := runtime.CallersFrames(s.pcs)
	f, _ := fs.Next()
	frame := Frame{
		Func: f.Function,
		File: f.File,
		Line: f.Line,
	}
	return frame, true
}

func (s *stack) FramesAndSource() iter.Seq2[Frame, string] {
	return func(yield func(Frame, string) bool) {
		if s == nil || len(s.pcs) == 0 {
			return
		}

		frames := s.Frames()
		for i, frame := range frames {
			var source string

			if s.sourceLines > 0 && frame.File != "" {
				if s.sourceDepth == -1 || (s.sourceDepth > 0 && i < s.sourceDepth) {
					source = s.frameSource(frame.File, frame.Line)
				}
			}

			if !yield(frame, source) {
				return
			}
		}
	}
}

func (s *stack) Len() int {
	if s == nil {
		return 0
	}
	return len(s.pcs)
}

func (s *stack) IsZero() bool {
	return s == nil || len(s.pcs) == 0
}

func (s *stack) StackTrace() []uintptr {
	if s == nil {
		return nil
	}
	return slices.Clone(s.pcs)
}

func (s *stack) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Frames())
}

func (s *stack) LogValue() slog.Value {
	return slog.AnyValue(s.Frames())
}

func (s *stack) frameSource(file string, line int) string {
	lines := getSourceLines(file, line, s.sourceLines)
	if len(lines) == 0 {
		return ""
	}

	start := max(1, line-s.sourceLines)
	end := start + len(lines) - 1
	width := len(strconv.Itoa(end))

	var buf strings.Builder
	for i, l := range lines {
		lineNum := start + i
		prefix := "  "
		if lineNum == line {
			prefix = "> "
		}
		fmt.Fprintf(&buf, "%s%*d: %s", prefix, width, lineNum, l)
		if i < len(lines)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func (f Frame) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("func", f.Func),
		slog.String("file", f.File),
		slog.Int("line", f.Line),
	)
}

func getSourceLines(file string, line, around int) []string {
	lines, err := readSourceFile(file)
	if err != nil {
		return nil
	}

	if line < 1 || line > len(lines) {
		return nil
	}

	start := max(0, line-around-1)
	end := min(len(lines), line+around)

	return lines[start:end]
}

func readSourceFile(path string) ([]string, error) {
	if !checkSourceAvailable() {
		return nil, os.ErrNotExist
	}

	if lines, ok := getCachedSourceFile(path); ok {
		return lines, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if isSourcePermanentError(err) {
			markSourceAvailable(false)
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	markSourceAvailable(true)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	cacheSourceFile(path, lines)

	return lines, nil
}

func checkSourceAvailable() bool {
	sourceAvailableMu.Lock()
	defer sourceAvailableMu.Unlock()
	return sourceAvailable == nil || *sourceAvailable
}

func getCachedSourceFile(path string) ([]string, bool) {
	sourceFileCacheMu.RLock()
	defer sourceFileCacheMu.RUnlock()
	lines, ok := sourceFileCache[path]
	return lines, ok
}

func markSourceAvailable(available bool) {
	sourceAvailableMu.Lock()
	defer sourceAvailableMu.Unlock()
	if sourceAvailable == nil {
		sourceAvailable = &available
	}
}

func cacheSourceFile(path string, lines []string) {
	sourceFileCacheMu.Lock()
	defer sourceFileCacheMu.Unlock()
	sourceFileCache[path] = lines
}

func isSourcePermanentError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission)
}
