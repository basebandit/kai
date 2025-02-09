package tools

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterPodTools(s *kai.Server, cm *kai.ClusterManager) {

	listPodTools := mcp.NewTool("list_pods",
		mcp.WithDescription("List pods in the current namespace or across all namespaces"),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list pods from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter pods"),
		),
		mcp.WithString("field_selector",
			mcp.Description("Field selector to filter pods"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pods to list"),
		),
	)

	s.AddTool(listPodTools, listPodsHandler(cm))

	getPodTool := mcp.NewTool("get_pod",
		mcp.WithDescription("Get detailed information about a specific pod"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the pod"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the pod (defaults to current namespace)"),
		),
	)

	s.AddTool(getPodTool, getPodHandler(cm))
}

func listPodsHandler(cm *kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		var labelSelector string
		if LabelSelectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = LabelSelectorArg
		}

		var fieldSelector string
		if fieldSelectorArg, ok := request.Params.Arguments["field_selector"].(string); ok {
			fieldSelector = fieldSelectorArg
		}

		limit := int64(0) // default to unlimited
		if limitArg, ok := request.Params.Arguments["limit"].(float64); ok && limitArg > 0 {
			limit = int64(limitArg)
		}

		resultText, err := cm.ListPods(ctx, limit, namespace, labelSelector, fieldSelector)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func getPodHandler(cm *kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		resultText, err := cm.GetPod(ctx, name, namespace)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}
