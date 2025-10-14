package errdef

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"reflect"
)

type (
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

	// ErrorTypeNamer is an interface for errors that have a type name.
	// This interface is used internally for causes created by the unmarshaler package.
	ErrorTypeNamer interface {
		error
		TypeName() string
	}

	jsonCauseData struct {
		Message string       `json:"message"`
		Type    string       `json:"type"`
		Causes  []*ErrorNode `json:"causes,omitempty"`
	}
)

var (
	_ json.Marshaler = (*ErrorNode)(nil)
	_ slog.LogValuer = (*ErrorNode)(nil)
)

// HasCycle returns true if any node in the error tree contains a cycle.
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
