package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterDeleteTools registers the delete_yaml tool for deleting resources from
// a raw manifest.
func RegisterDeleteTools(s kai.ServerInterface, cm kai.ClusterManager) {
	s.AddTool(mcp.NewTool(
		"delete_yaml",
		mcp.WithDescription("Delete one or more Kubernetes resources described by a YAML/JSON manifest (like `kubectl delete -f`). Supports multiple documents separated by `---` and any kind, including CRDs. Objects that are already gone are reported, not errored."),
		destructiveAnnotation("Delete from manifest"),
		mcp.WithString("manifest", mcp.Required(),
			mcp.Description("Raw YAML/JSON manifest text identifying the resources to delete.")),
		mcp.WithString("namespace", mcp.Description("Default namespace for namespaced objects that omit metadata.namespace. Ignored for cluster-scoped kinds.")),
	), deleteYAMLHandler(cm))
}

func deleteYAMLHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_yaml"))

		manifest, ok := request.GetArguments()["manifest"].(string)
		if !ok || manifest == "" {
			return mcp.NewToolResultText("Required parameter 'manifest' is missing"), nil
		}

		del := cluster.Delete{Manifest: manifest}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			del.Namespace = ns
		}

		result, err := del.Run(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("failed to delete manifest: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
