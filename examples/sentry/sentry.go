package sentry

import (
	"context"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/shiwano/errdef"
)

// Level sets the Sentry severity level for the error.
var Level, levelFrom = errdef.DefineField[sentry.Level]("sentry.level")

// CaptureError reports an error to Sentry with context from errdef data.
//
// This function:
//   - Returns false if the error is nil or unreportable (errdef.IsUnreportable)
//   - Retrieves the Sentry hub from the context
//   - Configures the Sentry scope with error metadata:
//   - Level (defaults to sentry.LevelError)
//   - Kind as a tag
//   - Domain as a tag
//   - HTTPStatus as a tag
//   - TraceID as a tag
//   - All custom fields as context data
//   - Cause tree with detailed information for each cause (message, kind, fields, origin)
//   - Captures the error exception
func CaptureError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	} else if errdef.IsUnreportable(err) {
		return false
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetLevel(levelFrom.OrDefault(err, sentry.LevelError))

		if kind, ok := errdef.KindFrom(err); ok {
			scope.SetTag("error.kind", string(kind))
		}

		if domain, ok := errdef.DomainFrom(err); ok {
			scope.SetTag("error.domain", domain)
		}

		if status, ok := errdef.HTTPStatusFrom(err); ok {
			scope.SetTag("http.status", strconv.Itoa(status))
		}

		if traceID, ok := errdef.TraceIDFrom(err); ok {
			scope.SetTag("trace.id", traceID)
		}

		errorContext := make(map[string]any)
		if fields, ok := errdef.FieldsFrom(err); ok {
			fieldsData := make(map[string]any, fields.Len())
			for k, v := range fields.All() {
				// Skip sentry-specific fields as they're already in scope
				if k.String() == "sentry.level" {
					continue
				}
				fieldsData[k.String()] = v.Value()
			}
			if len(fieldsData) > 0 {
				errorContext["fields"] = fieldsData
			}
		}
		if causes := buildCauses(err); len(causes) > 0 {
			errorContext["causes"] = causes
		}
		if len(errorContext) > 0 {
			scope.SetContext("error", errorContext)
		}
	})

	hub.CaptureException(err)
	return true
}

const maxCauseDepth = 3

func buildCauses(err error) []map[string]any {
	nodes, ok := errdef.UnwrapTreeFrom(err)
	if !ok {
		return nil
	}
	return buildCausesFromNodes(nodes, 0)
}

func buildCausesFromNodes(nodes errdef.Nodes, depth int) []map[string]any {
	if depth >= maxCauseDepth {
		return nil
	}

	var causes []map[string]any
	for _, node := range nodes {
		if data := buildCauseData(node, depth); data != nil {
			causes = append(causes, data)
		}
	}
	return causes
}

func buildCauseData(node *errdef.Node, depth int) map[string]any {
	data := map[string]any{
		"message": node.Error.Error(),
	}

	e, ok := node.Error.(errdef.Error)
	if !ok {
		if len(node.Causes) > 0 {
			if nestedCauses := buildCausesFromNodes(node.Causes, depth+1); len(nestedCauses) > 0 {
				data["causes"] = nestedCauses
			}
		}
		return data
	}

	if kind := e.Kind(); kind != "" {
		data["kind"] = string(kind)
	}

	if fields := e.Fields(); fields.Len() > 0 {
		fieldsData := make(map[string]any, fields.Len())
		for k, v := range fields.All() {
			// Skip sentry-specific fields as they're already in scope
			if k.String() == "sentry.level" {
				continue
			}
			fieldsData[k.String()] = v.Value()
		}
		if len(fieldsData) > 0 {
			data["fields"] = fieldsData
		}
	}

	if stack := e.Stack(); stack.Len() > 0 {
		if frame, ok := stack.HeadFrame(); ok {
			data["origin"] = map[string]any{
				"func": frame.Func,
				"file": frame.File,
				"line": frame.Line,
			}
		}
	}

	if len(node.Causes) > 0 {
		if nestedCauses := buildCausesFromNodes(node.Causes, depth+1); len(nestedCauses) > 0 {
			data["causes"] = nestedCauses
		}
	}

	return data
}
