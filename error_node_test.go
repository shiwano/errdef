package errdef_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	"github.com/shiwano/errdef"
)

func TestErrorNode_MarshalJSON(t *testing.T) {
	t.Run("simple error node", func(t *testing.T) {
		node := &errdef.ErrorNode{
			Error: errors.New("test error"),
		}

		data, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("want no error, got %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "test error",
			"type":    "*errors.errorString",
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("with nested causes", func(t *testing.T) {
		stdErr := errors.New("standard error")
		wrappedErr := fmt.Errorf("wrapped: %w", stdErr)

		node := &errdef.ErrorNode{
			Error: wrappedErr,
			Causes: []*errdef.ErrorNode{
				{Error: stdErr},
			},
		}

		data, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("want no error, got %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "wrapped: standard error",
			"type":    "*fmt.wrapError",
			"causes": []any{
				map[string]any{
					"message": "standard error",
					"type":    "*errors.errorString",
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("nested error nodes", func(t *testing.T) {
		err1 := errors.New("level 1")
		err2 := errors.New("level 2")
		err3 := errors.New("level 3")

		node := &errdef.ErrorNode{
			Error: err1,
			Causes: []*errdef.ErrorNode{
				{
					Error: err2,
					Causes: []*errdef.ErrorNode{
						{Error: err3},
					},
				},
			},
		}

		data, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("want no error, got %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		want := map[string]any{
			"message": "level 1",
			"type":    "*errors.errorString",
			"causes": []any{
				map[string]any{
					"message": "level 2",
					"type":    "*errors.errorString",
					"causes": []any{
						map[string]any{
							"message": "level 3",
							"type":    "*errors.errorString",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("JSON mismatch:\ngot:  %#v\nwant: %#v", got, want)
		}
	})

	t.Run("with circular reference", func(t *testing.T) {
		var ce1, ce2 *circularError
		ce1 = &circularError{msg: "error 1"}
		ce2 = &circularError{msg: "error 2", cause: ce1}
		ce1.cause = ce2

		def := errdef.Define("test_error", errdef.NoTrace())
		wrapped := def.Wrap(ce1).(errdef.Error)

		tree := wrapped.UnwrapTree()
		data, err := json.Marshal(tree)
		if err != nil {
			t.Fatalf("want no error, got %v", err)
		}

		var got []any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("want valid JSON, got %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("want 1 root node, got %d", len(got))
		}

		rootNode := got[0].(map[string]any)
		if rootNode["message"] != "error 1" {
			t.Errorf("want root message %q, got %q", "error 1", rootNode["message"])
		}

		causes := rootNode["causes"].([]any)
		if len(causes) != 1 {
			t.Fatalf("want 1 cause at level 1, got %d", len(causes))
		}

		level2 := causes[0].(map[string]any)
		if level2["message"] != "error 2" {
			t.Errorf("want message %q at level 2, got %q", "error 2", level2["message"])
		}

		// Should not have causes at level 2 due to cycle detection
		if _, hasCauses := level2["causes"]; hasCauses {
			t.Error("want no causes at level 2 due to cycle detection")
		}
	})
}

func TestErrorNode_LogValue(t *testing.T) {
	t.Run("simple error node", func(t *testing.T) {
		node := &errdef.ErrorNode{
			Error: errors.New("test error"),
		}

		value := node.LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("node", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		nodeData := result["node"].(map[string]any)

		want := map[string]any{
			"message": "test error",
		}

		if !reflect.DeepEqual(nodeData, want) {
			t.Errorf("want node %+v, got %+v", want, nodeData)
		}
	})

	t.Run("with nested causes", func(t *testing.T) {
		stdErr := errors.New("standard error")
		wrappedErr := fmt.Errorf("wrapped: %w", stdErr)

		node := &errdef.ErrorNode{
			Error: wrappedErr,
			Causes: []*errdef.ErrorNode{
				{Error: stdErr},
			},
		}

		value := node.LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("node", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		nodeData := result["node"].(map[string]any)

		want := map[string]any{
			"message": "wrapped: standard error",
			"causes": []any{
				map[string]any{
					"message": "standard error",
				},
			},
		}

		if !reflect.DeepEqual(nodeData, want) {
			t.Errorf("want node %+v, got %+v", want, nodeData)
		}
	})

	t.Run("with Error type", func(t *testing.T) {
		def := errdef.Define("test_error", errdef.NoTrace())
		err := def.New("test message")

		node := &errdef.ErrorNode{
			Error: err,
		}

		value := node.LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("node", value))

		var result map[string]any
		if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
			t.Fatalf("failed to unmarshal JSON: %v", jsonErr)
		}

		nodeData := result["node"].(map[string]any)

		want := map[string]any{
			"message": "test message",
			"kind":    "test_error",
		}

		if !reflect.DeepEqual(nodeData, want) {
			t.Errorf("want node %+v, got %+v", want, nodeData)
		}
	})

	t.Run("nested error nodes", func(t *testing.T) {
		err1 := errors.New("level 1")
		err2 := errors.New("level 2")
		err3 := errors.New("level 3")

		node := &errdef.ErrorNode{
			Error: err1,
			Causes: []*errdef.ErrorNode{
				{
					Error: err2,
					Causes: []*errdef.ErrorNode{
						{Error: err3},
					},
				},
			},
		}

		value := node.LogValue()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		logger.Info("test", slog.Any("node", value))

		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		nodeData := result["node"].(map[string]any)

		want := map[string]any{
			"message": "level 1",
			"causes": []any{
				map[string]any{
					"message": "level 2",
					"causes": []any{
						map[string]any{
							"message": "level 3",
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(nodeData, want) {
			t.Errorf("want node %+v, got %+v", want, nodeData)
		}
	})
}

func TestErrorNodes_Walk(t *testing.T) {
	type walkResult struct {
		depth int
		msg   string
	}

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")
	err4 := errors.New("error 4")
	err5 := errors.New("error 5")

	tests := []struct {
		name  string
		nodes errdef.ErrorNodes
		want  []walkResult
	}{
		{
			name: "single node",
			nodes: errdef.ErrorNodes{
				{Error: err1},
			},
			want: []walkResult{
				{0, "error 1"},
			},
		},
		{
			name: "nested nodes",
			nodes: errdef.ErrorNodes{
				{
					Error: err1,
					Causes: errdef.ErrorNodes{
						{Error: err2},
					},
				},
			},
			want: []walkResult{
				{0, "error 1"},
				{1, "error 2"},
			},
		},
		{
			name: "multiple sibling nodes",
			nodes: errdef.ErrorNodes{
				{Error: err1},
				{Error: err2},
				{Error: err3},
			},
			want: []walkResult{
				{0, "error 1"},
				{0, "error 2"},
				{0, "error 3"},
			},
		},
		{
			name: "complex multi-level tree",
			nodes: errdef.ErrorNodes{
				{
					Error: err1,
					Causes: errdef.ErrorNodes{
						{
							Error: err2,
							Causes: errdef.ErrorNodes{
								{Error: err3},
							},
						},
						{Error: err4},
					},
				},
				{Error: err5},
			},
			want: []walkResult{
				{0, "error 1"},
				{1, "error 2"},
				{2, "error 3"},
				{1, "error 4"},
				{0, "error 5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var collected []walkResult
			for depth, node := range tt.nodes.Walk() {
				collected = append(collected, walkResult{depth, node.Error.Error()})
			}

			if !reflect.DeepEqual(collected, tt.want) {
				t.Errorf("walk mismatch:\ngot:  %+v\nwant: %+v", collected, tt.want)
			}
		})
	}
}
