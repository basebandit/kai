package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterEventTools registers event query tools.
func RegisterEventTools(s kai.ServerInterface, cm kai.ClusterManager) {
	listEventsTool := mcp.NewTool("list_events",
		mcp.WithDescription("List Kubernetes events, optionally filtered by namespace, type or involved object"),
		readOnlyAnnotation("List events"),
		mcp.WithString("namespace",
			mcp.Description("Namespace to list events from (defaults to current namespace)"),
		),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("List events across all namespaces"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by event type: 'Warning' or 'Normal'"),
		),
		mcp.WithString("involved_object",
			mcp.Description("Filter to events about a specific object by name (e.g. a pod name)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of events to return"),
		),
	)
	s.AddTool(listEventsTool, listEventsHandler(cm))
}

func listEventsHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_events"))

		event := cluster.Event{}

		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			event.Namespace = ns
		}
		if all, ok := request.GetArguments()["all_namespaces"].(bool); ok {
			event.AllNamespaces = all
		}
		if t, ok := request.GetArguments()["type"].(string); ok {
			event.Type = t
		}
		if obj, ok := request.GetArguments()["involved_object"].(string); ok {
			event.InvolvedObject = obj
		}
		if limit, ok := request.GetArguments()["limit"].(float64); ok {
			event.Limit = int64(limit)
		}

		result, err := event.List(ctx, cm)
		if err != nil {
			slog.Warn("failed to list events", slog.String("error", err.Error()))
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list events: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
