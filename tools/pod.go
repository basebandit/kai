package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/clustermanager"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterPodTools(s kai.ServerInterface, cm kai.ClusterManagerInterface) {

	listPodTools := mcp.NewTool("list_pods",
		mcp.WithDescription("List pods in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list pods across all namespaces"),
		),
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

	deletePodTool := mcp.NewTool("delete_pod",
		mcp.WithDescription("Delete a pod by name"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the pod to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the pod (defaults to current namespace)"),
		),
		mcp.WithBoolean("force", mcp.Description("Force deletes the pod if set to true")),
	)

	s.AddTool(deletePodTool, deletePodHandler(cm))

	streamLogsTool := mcp.NewTool("stream_logs",
		mcp.WithDescription("Stream logs from a container in a pod"),
		mcp.WithString("pod",
			mcp.Required(),
			mcp.Description("Name of the pod"),
		),
		mcp.WithString("container",
			mcp.Description("Name of the container (defaults to the first container)"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the pod (defaults to current namespace)"),
		),
		mcp.WithNumber("tail",
			mcp.Description("Number of lines to show from the end of the logs (defaults to all)"),
		),
		mcp.WithBoolean("previous",
			mcp.Description("Whether to get logs from a previous container instance"),
		),
		mcp.WithString("since",
			mcp.Description("Only return logs newer than a relative duration like 5s, 2m, or 3h"),
		),
	)

	s.AddTool(streamLogsTool, streamLogsHandler(cm))
}

func listPodsHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var allNamespaces bool

		if allNamespacesArg, ok := request.Params.Arguments["all_namespaces"].(bool); ok {
			allNamespaces = allNamespacesArg
		}

		var namespace string
		if !allNamespaces {
			if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok {
				namespace = namespaceArg
			} else {
				namespace = cm.GetCurrentNamespace()
			}
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

		resultText, err := clustermanager.ListPods(ctx, cm, clustermanager.PodParams{limit, namespace, labelSelector, fieldSelector})
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func getPodHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func deletePodHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be anon-empty string"), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		var force bool
		if forceArg, ok := request.Params.Arguments["force"].(bool); ok {
			force = forceArg
		}

		resultText, err := cm.DeletePod(ctx, name, namespace, force)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func streamLogsHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		podArg, ok := request.Params.Arguments["pod"]
		if !ok || podArg == nil {
			return mcp.NewToolResultText("Required parameter 'pod' is missing"), nil
		}

		podName, ok := podArg.(string)
		if !ok || podName == "" {
			return mcp.NewToolResultText("Parameter 'pod' must be a non-empty string"), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		var containerName string
		if containerArg, ok := request.Params.Arguments["container"].(string); ok {
			containerName = containerArg
		}

		var tailLines int64 // Default to all lines
		if tailArg, ok := request.Params.Arguments["tail"].(float64); ok {
			tailLines = int64(tailArg)
		}

		var previous bool
		if previousArg, ok := request.Params.Arguments["previous"].(bool); ok {
			previous = previousArg
		}

		// var sinceTime *metav1.Time
		var sinceDuration *time.Duration
		if sinceArg, ok := request.Params.Arguments["since"].(string); ok && sinceArg != "" {
			duration, err := time.ParseDuration(sinceArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Failed to parse 'since' parameter: %v", err)), nil
			}
			sinceDuration = &duration
		}

		resultText, err := cm.StreamPodLogs(ctx, tailLines, previous, sinceDuration, podName, containerName, namespace)

		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}
		return mcp.NewToolResultText(resultText), nil
	}
}
