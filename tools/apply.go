package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterApplyTools registers the apply_yaml tool for applying raw manifests.
func RegisterApplyTools(s kai.ServerInterface, cm kai.ClusterManager) {
	s.AddTool(mcp.NewTool(
		"apply_yaml",
		mcp.WithDescription("Apply one or more Kubernetes resources from a YAML/JSON manifest (like `kubectl apply -f`) Supports multiple documents separated by `---` and any kind, including CRDs. Uses server-side apply: resources are created if absent or merged if they already exist."),
		idempotentMutationAnnotation("Apply manifest"),
		mcp.WithString("manifest", mcp.Required(),
			mcp.Description("Raw YAML/JSON manifest text.")),
		mcp.WithString("namespace", mcp.Description("Default namespace for namespaced objects that omit metadata.namespace. Ignored for cluster-scoped kinds.")),
	), applyYAMLHandler(cm))
}

func applyYAMLHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "apply_yaml"))

		manifest, ok := request.GetArguments()["manifest"].(string)
		if !ok || manifest == "" {
			return mcp.NewToolResultText("Required parameter 'manifest' is missing"), nil
		}

		apply := cluster.Apply{Manifest: manifest}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			apply.Namespace = ns
		}

		result, err := apply.Run(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("failed to apply manifest: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
