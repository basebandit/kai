package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConfigMapFactory is an interface for creating ConfigMap operators.
type ConfigMapFactory interface {
	NewConfigMap(params kai.ConfigMapParams) kai.ConfigMapOperator
}

// DefaultConfigMapFactory implements the ConfigMapFactory interface.
type DefaultConfigMapFactory struct{}

// NewDefaultConfigMapFactory creates a new DefaultConfigMapFactory.
func NewDefaultConfigMapFactory() *DefaultConfigMapFactory {
	return &DefaultConfigMapFactory{}
}

// NewConfigMap creates a new ConfigMap operator.
func (f *DefaultConfigMapFactory) NewConfigMap(params kai.ConfigMapParams) kai.ConfigMapOperator {
	return &cluster.ConfigMap{
		Name:        params.Name,
		Namespace:   params.Namespace,
		Data:        params.Data,
		BinaryData:  params.BinaryData,
		Labels:      params.Labels,
		Annotations: params.Annotations,
	}
}

// RegisterConfigMapTools registers all ConfigMap-related tools with the server.
func RegisterConfigMapTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultConfigMapFactory()
	RegisterConfigMapToolsWithFactory(s, cm, factory)
}

// RegisterConfigMapToolsWithFactory registers all ConfigMap-related tools using the provided factory.
func RegisterConfigMapToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory ConfigMapFactory) {
	createConfigMapTool := mcp.NewTool("create_configmap",
		mcp.WithDescription("Create a new ConfigMap in the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the ConfigMap"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the ConfigMap (defaults to current namespace)"),
		),
		mcp.WithObject("data",
			mcp.Description("Key-value pairs of configuration data"),
		),
		mcp.WithObject("binary_data",
			mcp.Description("Key-value pairs of binary data (base64 encoded)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the ConfigMap"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to apply to the ConfigMap"),
		),
	)
	s.AddTool(createConfigMapTool, createConfigMapHandler(cm, factory))

	getConfigMapTool := mcp.NewTool("get_configmap",
		mcp.WithDescription("Get detailed information about a specific ConfigMap"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the ConfigMap"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the ConfigMap (defaults to current namespace)"),
		),
	)
	s.AddTool(getConfigMapTool, getConfigMapHandler(cm, factory))

	listConfigMapsTool := mcp.NewTool("list_configmaps",
		mcp.WithDescription("List ConfigMaps in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list ConfigMaps across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list ConfigMaps from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter ConfigMaps (e.g., 'app=nginx,env=prod')"),
		),
	)
	s.AddTool(listConfigMapsTool, listConfigMapsHandler(cm, factory))

	deleteConfigMapTool := mcp.NewTool("delete_configmap",
		mcp.WithDescription("Delete a ConfigMap from the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the ConfigMap to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the ConfigMap (defaults to current namespace)"),
		),
	)
	s.AddTool(deleteConfigMapTool, deleteConfigMapHandler(cm, factory))

	updateConfigMapTool := mcp.NewTool("update_configmap",
		mcp.WithDescription("Update an existing ConfigMap"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the ConfigMap to update"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the ConfigMap (defaults to current namespace)"),
		),
		mcp.WithObject("data",
			mcp.Description("New key-value pairs of configuration data (replaces existing data)"),
		),
		mcp.WithObject("binary_data",
			mcp.Description("New key-value pairs of binary data (replaces existing binary data)"),
		),
		mcp.WithObject("labels",
			mcp.Description("New labels to apply to the ConfigMap (replaces existing labels)"),
		),
		mcp.WithObject("annotations",
			mcp.Description("New annotations to apply to the ConfigMap (replaces existing annotations)"),
		),
	)
	s.AddTool(updateConfigMapTool, updateConfigMapHandler(cm, factory))
}

func createConfigMapHandler(cm kai.ClusterManager, factory ConfigMapFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_configmap"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ConfigMapParams{
			Name:      name,
			Namespace: namespace,
		}

		if dataArg, ok := request.Params.Arguments["data"].(map[string]interface{}); ok {
			params.Data = dataArg
		}

		if binaryDataArg, ok := request.Params.Arguments["binary_data"].(map[string]interface{}); ok {
			params.BinaryData = binaryDataArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		configMap := factory.NewConfigMap(params)
		result, err := configMap.Create(ctx, cm)
		if err != nil {
			slog.Warn("failed to create ConfigMap",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create ConfigMap: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getConfigMapHandler(cm kai.ClusterManager, factory ConfigMapFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_configmap"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ConfigMapParams{
			Name:      name,
			Namespace: namespace,
		}

		configMap := factory.NewConfigMap(params)
		result, err := configMap.Get(ctx, cm)
		if err != nil {
			slog.Warn("failed to get ConfigMap",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get ConfigMap: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listConfigMapsHandler(cm kai.ClusterManager, factory ConfigMapFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_configmaps"))

		var allNamespaces bool
		if allNamespacesArg, ok := request.Params.Arguments["all_namespaces"].(bool); ok {
			allNamespaces = allNamespacesArg
		}

		var namespace string
		if !allNamespaces {
			if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
				namespace = namespaceArg
			} else {
				namespace = cm.GetCurrentNamespace()
			}
		}

		var labelSelector string
		if labelSelectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = labelSelectorArg
		}

		params := kai.ConfigMapParams{
			Namespace: namespace,
		}

		configMap := factory.NewConfigMap(params)
		result, err := configMap.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			slog.Warn("failed to list ConfigMaps",
				slog.Bool("all_namespaces", allNamespaces),
				slog.String("namespace", namespace),
				slog.String("label_selector", labelSelector),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list ConfigMaps: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteConfigMapHandler(cm kai.ClusterManager, factory ConfigMapFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_configmap"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ConfigMapParams{
			Name:      name,
			Namespace: namespace,
		}

		configMap := factory.NewConfigMap(params)
		result, err := configMap.Delete(ctx, cm)
		if err != nil {
			slog.Warn("failed to delete ConfigMap",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete ConfigMap: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func updateConfigMapHandler(cm kai.ClusterManager, factory ConfigMapFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "update_configmap"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ConfigMapParams{
			Name:      name,
			Namespace: namespace,
		}

		if dataArg, ok := request.Params.Arguments["data"].(map[string]interface{}); ok {
			params.Data = dataArg
		}

		if binaryDataArg, ok := request.Params.Arguments["binary_data"].(map[string]interface{}); ok {
			params.BinaryData = binaryDataArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		configMap := factory.NewConfigMap(params)
		result, err := configMap.Update(ctx, cm)
		if err != nil {
			slog.Warn("failed to update ConfigMap",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to update ConfigMap: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
