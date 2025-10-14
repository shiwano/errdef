package errdef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type (
	// Error extends the built-in error interface with additional functionality
	// for structured error handling including kinds, fields, and stack traces.
	//
	// Error instances are created from a Definition and remain immutable after creation.
	// They provide rich context through Kind (error classification), Fields (structured data),
	// and Stack (call stack information), while maintaining compatibility with standard
	// Go error handling via errors.Is and errors.As.
	//
	// Error implements several standard interfaces for formatting and serialization:
	// fmt.Formatter for detailed output, json.Marshaler for JSON serialization,
	// and slog.LogValuer for structured logging. It also integrates with external
	// error tracking services like Sentry (via stackTracer) and Google Cloud Error
	// Reporting (via DebugStacker), as well as legacy pkg/errors (via causer).
	//
	// Error chains are supported through Unwrap() for standard error unwrapping,
	// and UnwrapTree() for accessing the full error tree with cycle detection.
	Error interface {
		error
		// Kind returns the type of this error.
		Kind() Kind
		// Fields returns the structured fields associated with this error.
		Fields() Fields
		// Stack returns the stack trace where this error was created.
		Stack() Stack
		// Unwrap returns the errors that this error wraps.
		Unwrap() []error
		// UnwrapTree returns all causes as a tree structure.
		// When a circular reference is detected, the node that would create the cycle
		// is excluded, ensuring the result remains acyclic.
		UnwrapTree() ErrorNodes
	}

	// ErrorNodes is a slice of error nodes representing an error tree structure.
	ErrorNodes []*ErrorNode

	// ErrorNode represents a node in the cause tree with cycle detection already performed.
	ErrorNode struct {
		// Error is the error at this node.
		Error error
		// Causes are the nested causes of this error.
		Causes ErrorNodes

		// ptr is the pointer value of the error, used internally for cycle detection.
		ptr uintptr
		// visited is used internally to track visited errors during tree construction.
		visited map[uintptr]uintptr
	}

	// ErrorTypeNamer is a simple error implementation that wraps a message and a type name.
	ErrorTypeNamer interface {
		error
		TypeName() string
	}

	// DebugStacker returns a string that resembles the output of debug.Stack().
	// This is useful for integrating with Google Cloud Error Reporting.
	// See: https://cloud.google.com/error-reporting/reference/rest/v1beta1/projects.events/report#ReportedErrorEvent
	//
	// NOTE: The goroutine ID and state may differ from the actual one.
	DebugStacker interface {
		DebugStack() string
	}

	// stackTracer is used by Sentry SDK to extract stack traces from errors.
	// See: https://github.com/getsentry/sentry-go/blob/54a69e05ea609d3fc32fb1393770258dde6796c1/stacktrace.go#L84-L87
	stackTracer interface {
		StackTrace() []uintptr
	}

	// causer is used by pkg/errors to extract the cause of an error.
	// See: https://github.com/golang/go/issues/31778
	causer interface {
		Cause() error
	}

	// errorExporter defines methods for exporting error information in various formats.
	// This interface is used internally in the unmarshaler package.
	errorExporter interface {
		ErrorFormatter(err Error, s fmt.State, verb rune)
		ErrorJSONMarshaler(err Error) ([]byte, error)
		ErrorLogValuer(err Error) slog.Value
		ErrorTreeBuilder(errs []error) ErrorNodes
	}

	definedError struct {
		def    *Definition
		msg    string
		cause  error
		stack  stack
		joined bool
	}

	jsonErrorData struct {
		Message string       `json:"message"`
		Kind    string       `json:"kind,omitempty"`
		Fields  Fields       `json:"fields,omitempty"`
		Stack   []Frame      `json:"stack,omitempty"`
		Causes  []*ErrorNode `json:"causes,omitempty"`
	}

	jsonCauseData struct {
		Message string       `json:"message"`
		Type    string       `json:"type"`
		Causes  []*ErrorNode `json:"causes,omitempty"`
	}
)

var (
	_ Error          = (*definedError)(nil)
	_ DebugStacker   = (*definedError)(nil)
	_ fmt.GoStringer = (*definedError)(nil)
	_ fmt.Formatter  = (*definedError)(nil)
	_ json.Marshaler = (*definedError)(nil)
	_ slog.LogValuer = (*definedError)(nil)
	_ stackTracer    = (*definedError)(nil)
	_ causer         = (*definedError)(nil)
	_ errorExporter  = (*definedError)(nil)
	_ fieldsGetter   = (*definedError)(nil)

	_ json.Marshaler = (*ErrorNode)(nil)
	_ slog.LogValuer = (*ErrorNode)(nil)
)

func newError(d *Definition, cause error, msg string, joined bool, stackSkip int) error {
	var stack stack
	if !d.noTrace {
		depth := callersDepth
		if d.stackDepth > 0 {
			depth = d.stackDepth
		}
		stack = newStack(depth, d.stackSkip+stackSkip)
	}
	return &definedError{
		def:    d,
		msg:    msg,
		cause:  cause,
		stack:  stack,
		joined: joined,
	}
}

func (e *definedError) Error() string {
	return e.msg
}

func (e *definedError) Kind() Kind {
	return e.def.kind
}

func (e *definedError) Fields() Fields {
	return e.def.fields
}

func (e *definedError) Stack() Stack {
	return e.stack
}

func (e *definedError) Unwrap() []error {
	if e.cause == nil {
		return nil
	}
	if e.joined {
		if u, ok := e.cause.(interface{ Unwrap() []error }); ok {
			return u.Unwrap()
		}
	}
	return []error{e.cause}
}

func (e *definedError) UnwrapTree() ErrorNodes {
	return e.ErrorTreeBuilder(e.Unwrap())
}

func (e *definedError) Is(target error) bool {
	if e == target {
		return true
	}
	if d, ok := target.(*Definition); ok {
		return e.def.root() == d.root()
	}
	return false
}

func (e *definedError) DebugStack() string {
	buf := bytes.NewBufferString(e.Error())

	// hard-coded cause we can't get it in pure Go.
	buf.WriteString("\n\ngoroutine 1 [running]:")

	for _, pc := range e.stack.StackTrace() {
		if fn := runtime.FuncForPC(pc); fn != nil {
			buf.WriteByte('\n')
			file, line := fn.FileLine(pc)
			fmt.Fprintf(buf, "%s()\n\t%s:%d +%#x", fn.Name(), file, line, fn.Entry())
		}
	}
	return buf.String()
}

func (e *definedError) StackTrace() []uintptr {
	return e.stack.StackTrace()
}

func (e *definedError) Cause() error {
	return e.cause
}

func (e *definedError) GoString() string {
	type (
		definedError_ definedError
		definedError  definedError_
	)
	return fmt.Sprintf("%#v", (*definedError)(e))
}

func (e *definedError) Format(s fmt.State, verb rune) {
	e.ErrorFormatter(e, s, verb)
}

func (e *definedError) MarshalJSON() ([]byte, error) {
	return e.ErrorJSONMarshaler(e)
}

func (e *definedError) LogValue() slog.Value {
	return e.ErrorLogValuer(e)
}

func (e *definedError) ErrorFormatter(err Error, s fmt.State, verb rune) {
	if e.def.formatter != nil {
		e.def.formatter(err, s, verb)
		return
	}

	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			causes := err.UnwrapTree()
			formatErrorDetails(err, s, "", len(causes) > 0)

			if len(causes) > 0 {
				formatCausesHeader(s, "", len(causes))
				formatErrorNodes(causes, s, "  ")
			}
		case s.Flag('#'):
			if gs, ok := err.(fmt.GoStringer); ok {
				_, _ = io.WriteString(s, gs.GoString())
			} else {
				_, _ = io.WriteString(s, err.Error())
			}
		default:
			_, _ = io.WriteString(s, err.Error())
		}
	case 's':
		_, _ = io.WriteString(s, err.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", err.Error())
	}
}

func (e *definedError) ErrorJSONMarshaler(err Error) ([]byte, error) {
	if e.def.jsonMarshaler != nil {
		return e.def.jsonMarshaler(e)
	}

	var fields Fields
	if err.Fields().Len() > 0 {
		fields = err.Fields()
	}
	return json.Marshal(jsonErrorData{
		Message: err.Error(),
		Kind:    string(err.Kind()),
		Fields:  fields,
		Stack:   err.Stack().Frames(),
		Causes:  err.UnwrapTree(),
	})
}

func (e *definedError) ErrorLogValuer(err Error) slog.Value {
	if e.def.logValuer != nil {
		return e.def.logValuer(e)
	}

	attrs := make([]slog.Attr, 0, 5)

	attrs = append(attrs, slog.String("message", err.Error()))

	if err.Kind() != "" {
		attrs = append(attrs, slog.String("kind", string(err.Kind())))
	}

	if err.Fields().Len() > 0 {
		attrs = append(attrs, slog.Any("fields", err.Fields()))
	}

	if err.Stack().Len() > 0 {
		if frame, ok := err.Stack().HeadFrame(); ok {
			attrs = append(attrs, slog.Any("origin", frame))
		}
	}

	causes := err.Unwrap()
	if len(causes) > 0 {
		causeMessages := make([]string, len(causes))
		for i, c := range causes {
			causeMessages[i] = c.Error()
		}
		attrs = append(attrs, slog.Any("causes", causeMessages))
	}
	return slog.GroupValue(attrs...)
}

func (e *definedError) ErrorTreeBuilder(errs []error) ErrorNodes {
	visited := make(map[uintptr]uintptr)
	nodes := buildErrorNodes(errs, visited)
	return nodes
}

func (ns ErrorNodes) HasCycle() bool {
	for _, n := range ns {
		if n.IsCyclic() {
			return true
		}
		if n.Causes.HasCycle() {
			return true
		}
	}
	return false
}

// Walk returns an iterator that traverses the error tree in depth-first order.
// The iterator yields pairs of depth (int) and node (*ErrorNode) for each error in the tree.
func (ns ErrorNodes) Walk() iter.Seq2[int, *ErrorNode] {
	return func(yield func(int, *ErrorNode) bool) {
		for _, n := range ns {
			if !n.walk(0, yield) {
				return
			}
		}
	}
}

func (n *ErrorNode) walk(depth int, yield func(int, *ErrorNode) bool) bool {
	if !yield(depth, n) {
		return false
	}
	for _, cause := range n.Causes {
		if !cause.walk(depth+1, yield) {
			return false
		}
	}
	return true
}

// IsCyclic returns true if this node is part of a cycle in the error tree.
func (n *ErrorNode) IsCyclic() bool {
	if n.ptr == 0 {
		return false
	}
	ptr, hasCycle := n.visited[0]
	if !hasCycle {
		return false
	}
	return n.ptr == ptr
}

// MarshalJSON implements json.Marshaler for ErrorNode.
func (n *ErrorNode) MarshalJSON() ([]byte, error) {
	switch err := n.Error.(type) {
	case Error:
		var fields Fields
		if err.Fields().Len() > 0 {
			fields = err.Fields()
		}
		return json.Marshal(jsonErrorData{
			Message: err.Error(),
			Kind:    string(err.Kind()),
			Fields:  fields,
			Stack:   err.Stack().Frames(),
			Causes:  n.Causes,
		})
	case ErrorTypeNamer:
		return json.Marshal(jsonCauseData{
			Message: err.Error(),
			Type:    err.TypeName(),
			Causes:  n.Causes,
		})
	default:
		return json.Marshal(jsonCauseData{
			Message: n.Error.Error(),
			Type:    fmt.Sprintf("%T", n.Error),
			Causes:  n.Causes,
		})
	}
}

// LogValue implements slog.LogValuer for ErrorNode.
func (e *ErrorNode) LogValue() slog.Value {
	switch te := e.Error.(type) {
	case Error:
		return te.(slog.LogValuer).LogValue()
	case ErrorTypeNamer:
		attrs := []slog.Attr{
			slog.String("message", te.Error()),
		}
		if len(e.Causes) > 0 {
			causes := make([]any, len(e.Causes))
			for i, cause := range e.Causes {
				causes[i] = slogValueToAny(cause.LogValue())
			}
			attrs = append(attrs, slog.Any("causes", causes))
		}
		return slog.GroupValue(attrs...)
	default:
		attrs := []slog.Attr{
			slog.String("message", e.Error.Error()),
		}
		if len(e.Causes) > 0 {
			causes := make([]any, len(e.Causes))
			for i, cause := range e.Causes {
				causes[i] = slogValueToAny(cause.LogValue())
			}
			attrs = append(attrs, slog.Any("causes", causes))
		}
		return slog.GroupValue(attrs...)
	}
}

func slogValueToAny(v slog.Value) any {
	switch v.Kind() {
	case slog.KindGroup:
		result := make(map[string]any)
		for _, attr := range v.Group() {
			result[attr.Key] = slogValueToAny(attr.Value)
		}
		return result
	default:
		return v.Any()
	}
}

func buildErrorNodes(causes []error, visited map[uintptr]uintptr) []*ErrorNode {
	if len(causes) == 0 {
		return nil
	}

	nodes := make([]*ErrorNode, 0, len(causes))
	for _, c := range causes {
		if c == nil {
			continue
		}
		if node, ok := buildErrorNode(c, visited); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func buildErrorNode(err error, visited map[uintptr]uintptr) (*ErrorNode, bool) {
	val := reflect.ValueOf(err)
	if !val.IsValid() {
		return nil, false
	}

	var ptr uintptr
	if val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface ||
		val.Kind() == reflect.Map || val.Kind() == reflect.Slice ||
		val.Kind() == reflect.Chan || val.Kind() == reflect.Func {
		ptr = val.Pointer()
		if ptr != 0 {
			if _, ok := visited[ptr]; ok {
				visited[0] = ptr
				return nil, false
			}

			visited[ptr] = ptr
			defer delete(visited, ptr) // Remove from visited after processing this path
		}
	}

	var causes []error
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if nested := unwrapper.Unwrap(); nested != nil {
			causes = []error{nested}
		}
	} else if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		causes = unwrapper.Unwrap()
	}

	return &ErrorNode{
		Error:   err,
		Causes:  buildErrorNodes(causes, visited),
		ptr:     ptr,
		visited: visited,
	}, true
}

func formatErrorDetails(err Error, s io.Writer, indent string, hasCauses bool) {
	_, _ = io.WriteString(s, err.Error())

	hasDetails := err.Kind() != "" || err.Fields().Len() > 0 || err.Stack().Len() > 0
	if hasDetails || hasCauses {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "---")
	}

	if err.Kind() != "" {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "kind: ")
		_, _ = io.WriteString(s, string(err.Kind()))
	}

	if err.Fields().Len() > 0 {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "fields:")
		for k, v := range err.Fields().Sorted() {
			_, _ = io.WriteString(s, "\n")
			_, _ = io.WriteString(s, indent)
			_, _ = io.WriteString(s, "  ")
			_, _ = io.WriteString(s, k.String())
			_, _ = io.WriteString(s, ": ")

			valueStr := fmt.Sprintf("%+v", v.Value())
			if strings.Contains(valueStr, "\n") {
				_, _ = io.WriteString(s, "|\n")
				for line := range strings.SplitSeq(valueStr, "\n") {
					_, _ = io.WriteString(s, indent)
					_, _ = io.WriteString(s, "    ")
					_, _ = io.WriteString(s, line)
					_, _ = io.WriteString(s, "\n")
				}
			} else {
				_, _ = io.WriteString(s, valueStr)
			}
		}
	}

	if err.Stack().Len() > 0 {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "stack:")
		for _, f := range err.Stack().Frames() {
			if f.File != "" {
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "  ")
				_, _ = io.WriteString(s, f.Func)
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "    ")
				_, _ = io.WriteString(s, f.File)
				_, _ = io.WriteString(s, ":")
				_, _ = io.WriteString(s, strconv.Itoa(f.Line))
			}
		}
	}
}

func formatCausesHeader(s io.Writer, indent string, count int) {
	_, _ = io.WriteString(s, "\n")
	_, _ = io.WriteString(s, indent)
	_, _ = io.WriteString(s, "causes: (")
	if count == 1 {
		_, _ = io.WriteString(s, "1 error")
	} else {
		_, _ = io.WriteString(s, strconv.Itoa(count))
		_, _ = io.WriteString(s, " errors")
	}
	_, _ = io.WriteString(s, ")")
}

func formatErrorNodes(nodes []*ErrorNode, s io.Writer, indent string) {
	for i, node := range nodes {
		_, _ = io.WriteString(s, "\n")
		_, _ = io.WriteString(s, indent)
		_, _ = io.WriteString(s, "[")
		_, _ = io.WriteString(s, strconv.Itoa(i+1))
		_, _ = io.WriteString(s, "] ")

		if err, ok := node.Error.(Error); ok {
			formatErrorDetails(err, s, indent+"    ", len(node.Causes) > 0)
		} else {
			_, _ = io.WriteString(s, node.Error.Error())

			if len(node.Causes) > 0 {
				_, _ = io.WriteString(s, "\n")
				_, _ = io.WriteString(s, indent)
				_, _ = io.WriteString(s, "    ---")
			}
		}

		if len(node.Causes) > 0 {
			formatCausesHeader(s, indent+"    ", len(node.Causes))
			formatErrorNodes(node.Causes, s, indent+"    ")
		}
	}
}
