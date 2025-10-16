package unmarshaler

import (
	"github.com/shiwano/errdef"
)

type UnknownCauseError struct {
	msg      string
	typeName string
	causes   []error
}

func (e *UnknownCauseError) Error() string              { return e.msg }
func (e *UnknownCauseError) TypeName() string           { return e.typeName }
func (e *UnknownCauseError) Unwrap() []error            { return e.causes[:] }
func (e *UnknownCauseError) UnwrapTree() []*errdef.Node { return buildNodes(e.causes) }

func buildNodes(causes []error) []*errdef.Node {
	if len(causes) == 0 {
		return nil
	}

	nodes := make([]*errdef.Node, 0, len(causes))
	for _, c := range causes {
		if c == nil {
			continue
		}
		nodes = append(nodes, buildNode(c))
	}
	return nodes
}

func buildNode(err error) *errdef.Node {
	var causes []error
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if nested := unwrapper.Unwrap(); nested != nil {
			causes = []error{nested}
		}
	} else if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		causes = unwrapper.Unwrap()
	}

	return &errdef.Node{
		Error:  err,
		Causes: buildNodes(causes),
	}
}
