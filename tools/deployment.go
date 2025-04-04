package tools

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterDeploymentTools(s kai.ServerInterface, cm kai.ClusterManagerInterface) {
	listDeploymentTools := mcp.NewTool("list_deployments",
		mcp.WithDescription("List deployments in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list deployments across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list deployments from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter deployments"),
		),
	)

	s.AddTool(listDeploymentTools, listDeploymentsHandler(cm))
}

func listDeploymentsHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var allNamespaces bool

		if allNamespacesArg, ok := request.Params.Arguments["all_namespaces"].(bool); ok {
			allNamespaces = allNamespacesArg
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		var labelSelector string
		if labelSelectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = labelSelectorArg
		}

		resultText, err := cm.ListDeployments(ctx, allNamespaces, labelSelector, namespace)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}
