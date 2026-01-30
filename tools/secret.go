package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// SecretFactory is an interface for creating Secret operators.
type SecretFactory interface {
	NewSecret(params kai.SecretParams) kai.SecretOperator
}

// DefaultSecretFactory implements the SecretFactory interface.
type DefaultSecretFactory struct{}

// NewDefaultSecretFactory creates a new DefaultSecretFactory.
func NewDefaultSecretFactory() *DefaultSecretFactory {
	return &DefaultSecretFactory{}
}

// NewSecret creates a new Secret operator.
func (f *DefaultSecretFactory) NewSecret(params kai.SecretParams) kai.SecretOperator {
	return &cluster.Secret{
		Name:        params.Name,
		Namespace:   params.Namespace,
		Type:        params.Type,
		Data:        params.Data,
		StringData:  params.StringData,
		Labels:      params.Labels,
		Annotations: params.Annotations,
	}
}

// RegisterSecretTools registers all Secret-related tools with the server.
func RegisterSecretTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultSecretFactory()
	RegisterSecretToolsWithFactory(s, cm, factory)
}

// RegisterSecretToolsWithFactory registers all Secret-related tools using the provided factory.
func RegisterSecretToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory SecretFactory) {
	createSecretTool := mcp.NewTool("create_secret",
		mcp.WithDescription("Create a new Secret in the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Secret"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the Secret (defaults to current namespace)"),
		),
		mcp.WithString("type",
			mcp.Description("Secret type (Opaque, kubernetes.io/tls, kubernetes.io/dockerconfigjson, etc.)"),
		),
		mcp.WithObject("data",
			mcp.Description("Key-value pairs of secret data (values should be base64 encoded or plain text)"),
		),
		mcp.WithObject("string_data",
			mcp.Description("Key-value pairs of secret data in plain text (auto-encoded by Kubernetes)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the Secret"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to apply to the Secret"),
		),
	)
	s.AddTool(createSecretTool, createSecretHandler(cm, factory))

	getSecretTool := mcp.NewTool("get_secret",
		mcp.WithDescription("Get information about a specific Secret (values are masked for security)"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Secret"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Secret (defaults to current namespace)"),
		),
	)
	s.AddTool(getSecretTool, getSecretHandler(cm, factory))

	listSecretsTool := mcp.NewTool("list_secrets",
		mcp.WithDescription("List Secrets in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list Secrets across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list Secrets from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter Secrets (e.g., 'app=nginx,env=prod')"),
		),
	)
	s.AddTool(listSecretsTool, listSecretsHandler(cm, factory))

	deleteSecretTool := mcp.NewTool("delete_secret",
		mcp.WithDescription("Delete a Secret from the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Secret to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Secret (defaults to current namespace)"),
		),
	)
	s.AddTool(deleteSecretTool, deleteSecretHandler(cm, factory))

	updateSecretTool := mcp.NewTool("update_secret",
		mcp.WithDescription("Update an existing Secret"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Secret to update"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Secret (defaults to current namespace)"),
		),
		mcp.WithString("type",
			mcp.Description("Secret type (Opaque, kubernetes.io/tls, kubernetes.io/dockerconfigjson, etc.)"),
		),
		mcp.WithObject("data",
			mcp.Description("New key-value pairs of secret data (replaces existing data)"),
		),
		mcp.WithObject("string_data",
			mcp.Description("New key-value pairs of secret data in plain text (replaces existing string data)"),
		),
		mcp.WithObject("labels",
			mcp.Description("New labels to apply to the Secret (replaces existing labels)"),
		),
		mcp.WithObject("annotations",
			mcp.Description("New annotations to apply to the Secret (replaces existing annotations)"),
		),
	)
	s.AddTool(updateSecretTool, updateSecretHandler(cm, factory))
}

func createSecretHandler(cm kai.ClusterManager, factory SecretFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_secret"))

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

		params := kai.SecretParams{
			Name:      name,
			Namespace: namespace,
		}

		if typeArg, ok := request.Params.Arguments["type"].(string); ok && typeArg != "" {
			if err := validateSecretType(typeArg); err != nil {
				return mcp.NewToolResultText(err.Error()), nil
			}
			params.Type = typeArg
		}

		if dataArg, ok := request.Params.Arguments["data"].(map[string]interface{}); ok {
			params.Data = dataArg
		}

		if stringDataArg, ok := request.Params.Arguments["string_data"].(map[string]interface{}); ok {
			params.StringData = stringDataArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		secret := factory.NewSecret(params)
		result, err := secret.Create(ctx, cm)
		if err != nil {
			slog.Warn("failed to create Secret",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create Secret: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getSecretHandler(cm kai.ClusterManager, factory SecretFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_secret"))

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

		params := kai.SecretParams{
			Name:      name,
			Namespace: namespace,
		}

		secret := factory.NewSecret(params)
		result, err := secret.Get(ctx, cm)
		if err != nil {
			slog.Warn("failed to get Secret",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get Secret: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listSecretsHandler(cm kai.ClusterManager, factory SecretFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_secrets"))

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

		params := kai.SecretParams{
			Namespace: namespace,
		}

		secret := factory.NewSecret(params)
		result, err := secret.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			slog.Warn("failed to list Secrets",
				slog.Bool("all_namespaces", allNamespaces),
				slog.String("namespace", namespace),
				slog.String("label_selector", labelSelector),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list Secrets: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteSecretHandler(cm kai.ClusterManager, factory SecretFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_secret"))

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

		params := kai.SecretParams{
			Name:      name,
			Namespace: namespace,
		}

		secret := factory.NewSecret(params)
		result, err := secret.Delete(ctx, cm)
		if err != nil {
			slog.Warn("failed to delete Secret",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete Secret: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func updateSecretHandler(cm kai.ClusterManager, factory SecretFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "update_secret"))

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

		params := kai.SecretParams{
			Name:      name,
			Namespace: namespace,
		}

		if typeArg, ok := request.Params.Arguments["type"].(string); ok && typeArg != "" {
			if err := validateSecretType(typeArg); err != nil {
				return mcp.NewToolResultText(err.Error()), nil
			}
			params.Type = typeArg
		}

		if dataArg, ok := request.Params.Arguments["data"].(map[string]interface{}); ok {
			params.Data = dataArg
		}

		if stringDataArg, ok := request.Params.Arguments["string_data"].(map[string]interface{}); ok {
			params.StringData = stringDataArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		secret := factory.NewSecret(params)
		result, err := secret.Update(ctx, cm)
		if err != nil {
			slog.Warn("failed to update Secret",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to update Secret: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
