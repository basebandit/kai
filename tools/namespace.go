package tools

import (
	"context"
	"fmt"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterNamespaceTools(s kai.ServerInterface, cm kai.ClusterManager) {
	createNamespaceTool := mcp.NewTool("create_namespace",
		mcp.WithDescription("Create a new Kubernetes namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the namespace to create"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the namespace"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to apply to the namespace"),
		),
	)
	s.AddTool(createNamespaceTool, createNamespaceHandler(cm))

	getNamespaceTool := mcp.NewTool("get_namespace",
		mcp.WithDescription("Get detailed information about a specific namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the namespace to get"),
		),
	)
	s.AddTool(getNamespaceTool, getNamespaceHandler(cm))

	listNamespacesTool := mcp.NewTool("list_namespaces",
		mcp.WithDescription("List all namespaces in the cluster"),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter namespaces (e.g., 'env=prod,tier=backend')"),
		),
	)
	s.AddTool(listNamespacesTool, listNamespacesHandler(cm))

	deleteNamespaceTool := mcp.NewTool("delete_namespace",
		mcp.WithDescription("Delete a namespace or namespaces matching label selector"),
		mcp.WithString("name",
			mcp.Description("Name of the namespace to delete"),
		),
		mcp.WithObject("labels",
			mcp.Description("Label selector to delete multiple namespaces"),
		),
	)
	s.AddTool(deleteNamespaceTool, deleteNamespaceHandler(cm))
}

func createNamespaceHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cluster.Namespace{
			Name: name,
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			namespace.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			namespace.Annotations = annotationsArg
		}

		result, err := namespace.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getNamespaceHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cluster.Namespace{
			Name: name,
		}

		result, err := namespace.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listNamespacesHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		labelSelector := ""
		if selectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = selectorArg
		}

		namespace := cluster.Namespace{}

		result, err := namespace.List(ctx, cm, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list namespaces: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteNamespaceHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace := cluster.Namespace{}

		if nameArg, ok := request.Params.Arguments["name"].(string); ok && nameArg != "" {
			namespace.Name = nameArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			namespace.Labels = labelsArg
		}

		if namespace.Name == "" && len(namespace.Labels) == 0 {
			return mcp.NewToolResultText("Either namespace name or label selector must be provided"), nil
		}

		result, err := namespace.Delete(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
