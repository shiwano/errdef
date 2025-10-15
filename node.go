package errdef

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"reflect"
)

type (
	// Nodes is a slice of error nodes representing an error tree structure.
	Nodes []*Node

	// Node represents a node in the cause tree with cycle detection already performed.
	Node struct {
		// Error is the error at this node.
		Error error
		// Causes are the nested causes of this error.
		Causes Nodes

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
		Message string  `json:"message"`
		Type    string  `json:"type"`
		Causes  []*Node `json:"causes,omitempty"`
	}
)

var (
	_ json.Marshaler = (*Node)(nil)
	_ slog.LogValuer = (*Node)(nil)
)

// HasCycle returns true if any node in the error tree contains a cycle.
func (ns Nodes) HasCycle() bool {
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
// The iterator yields pairs of depth (int) and node (*Node) for each error in the tree.
func (ns Nodes) Walk() iter.Seq2[int, *Node] {
	return func(yield func(int, *Node) bool) {
		for _, n := range ns {
			if !n.walk(0, yield) {
				return
			}
		}
	}
}

func (n *Node) walk(depth int, yield func(int, *Node) bool) bool {
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
func (n *Node) IsCyclic() bool {
	if n.ptr == 0 {
		return false
	}
	ptr, hasCycle := n.visited[0]
	if !hasCycle {
		return false
	}
	return n.ptr == ptr
}

// MarshalJSON implements json.Marshaler for Node.
func (n *Node) MarshalJSON() ([]byte, error) {
	switch err := n.Error.(type) {
	case Error:
		return json.Marshal(jsonErrorData{
			Message: err.Error(),
			Kind:    string(err.Kind()),
			Fields:  err.Fields(),
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

// LogValue implements slog.LogValuer for Node.
func (e *Node) LogValue() slog.Value {
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

func buildNodes(causes []error, visited map[uintptr]uintptr) []*Node {
	if len(causes) == 0 {
		return nil
	}

	nodes := make([]*Node, 0, len(causes))
	for _, c := range causes {
		if c == nil {
			continue
		}
		if node, ok := buildNode(c, visited); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func buildNode(err error, visited map[uintptr]uintptr) (*Node, bool) {
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

	return &Node{
		Error:   err,
		Causes:  buildNodes(causes, visited),
		ptr:     ptr,
		visited: visited,
	}, true
}
