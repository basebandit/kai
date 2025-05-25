package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterContextTools registers all context-related tools with the server
func RegisterContextTools(s kai.ServerInterface, cm kai.ClusterManager) {
	listContextsTool := mcp.NewTool("list_contexts",
		mcp.WithDescription("List all available Kubernetes contexts"),
	)
	s.AddTool(listContextsTool, listContextsHandler(cm))

	getCurrentContextTool := mcp.NewTool("get_current_context",
		mcp.WithDescription("Get the currently active Kubernetes context"),
	)
	s.AddTool(getCurrentContextTool, getCurrentContextHandler(cm))

	switchContextTool := mcp.NewTool("switch_context",
		mcp.WithDescription("Switch to a different Kubernetes context"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the context to switch to"),
		),
	)
	s.AddTool(switchContextTool, switchContextHandler(cm))

	loadKubeconfigTool := mcp.NewTool("load_kubeconfig",
		mcp.WithDescription("Load a kubeconfig file and register it as a new context"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name to assign to this context"),
		),
		mcp.WithString("path",
			mcp.Description("Path to the kubeconfig file (defaults to ~/.kube/config)"),
		),
	)
	s.AddTool(loadKubeconfigTool, loadKubeconfigHandler(cm))

	deleteContextTool := mcp.NewTool("delete_context",
		mcp.WithDescription("Remove a context from the manager"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the context to delete"),
		),
	)
	s.AddTool(deleteContextTool, deleteContextHandler(cm))

	renameContextTool := mcp.NewTool("rename_context",
		mcp.WithDescription("Rename an existing context"),
		mcp.WithString("old_name",
			mcp.Required(),
			mcp.Description("Current name of the context"),
		),
		mcp.WithString("new_name",
			mcp.Required(),
			mcp.Description("New name for the context"),
		),
	)
	s.AddTool(renameContextTool, renameContextHandler(cm))

	describeContextTool := mcp.NewTool("describe_context",
		mcp.WithDescription("Get detailed information about a specific context"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the context to describe"),
		),
	)
	s.AddTool(describeContextTool, describeContextHandler(cm))
}

func listContextsHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contexts := cm.ListContexts()

		if len(contexts) == 0 {
			return mcp.NewToolResultText("No contexts available"), nil
		}

		var result strings.Builder
		result.WriteString("Available contexts:\n")

		for _, contextInfo := range contexts {
			marker := " "
			if contextInfo.IsActive {
				marker = "*"
			}

			result.WriteString(fmt.Sprintf("%s %s\n", marker, contextInfo.Name))
			result.WriteString(fmt.Sprintf("  Cluster: %s\n", contextInfo.Cluster))
			result.WriteString(fmt.Sprintf("  User: %s\n", contextInfo.User))
			result.WriteString(fmt.Sprintf("  Namespace: %s\n", contextInfo.Namespace))
			result.WriteString("\n")
		}

		result.WriteString(fmt.Sprintf("Total: %d context(s)", len(contexts)))

		return mcp.NewToolResultText(result.String()), nil
	}
}

func getCurrentContextHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentContext := cm.GetCurrentContext()

		if currentContext == "" {
			return mcp.NewToolResultText("No active context"), nil
		}

		contextInfo, err := cm.GetContextInfo(currentContext)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error getting context info: %s", err.Error())), nil
		}

		result := fmt.Sprintf("Current context: %s\n", contextInfo.Name)
		result += fmt.Sprintf("Cluster: %s\n", contextInfo.Cluster)
		result += fmt.Sprintf("User: %s\n", contextInfo.User)
		result += fmt.Sprintf("Namespace: %s\n", contextInfo.Namespace)
		result += fmt.Sprintf("Server: %s", contextInfo.ServerURL)

		return mcp.NewToolResultText(result), nil
	}
}

func switchContextHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		if err := cm.SetCurrentContext(name); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to switch context: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Switched to context '%s'", name)), nil
	}
}

func loadKubeconfigHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		path := ""
		if pathArg, ok := request.Params.Arguments["path"].(string); ok {
			path = pathArg
		}

		if err := cm.LoadKubeConfig(name, path); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to load kubeconfig: %s", err.Error())), nil
		}

		configPath := path
		if configPath == "" {
			configPath = "~/.kube/config"
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully loaded kubeconfig from '%s' as context '%s'", configPath, name)), nil
	}
}

func deleteContextHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		if err := cm.DeleteContext(name); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete context: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted context '%s'", name)), nil
	}
}

func renameContextHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		oldNameArg, ok := request.Params.Arguments["old_name"]
		if !ok || oldNameArg == nil {
			return mcp.NewToolResultText("Required parameter 'old_name' is missing"), nil
		}

		oldName, ok := oldNameArg.(string)
		if !ok || oldName == "" {
			return mcp.NewToolResultText("Parameter 'old_name' must be a non-empty string"), nil
		}

		newNameArg, ok := request.Params.Arguments["new_name"]
		if !ok || newNameArg == nil {
			return mcp.NewToolResultText("Required parameter 'new_name' is missing"), nil
		}

		newName, ok := newNameArg.(string)
		if !ok || newName == "" {
			return mcp.NewToolResultText("Parameter 'new_name' must be a non-empty string"), nil
		}

		if err := cm.RenameContext(oldName, newName); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to rename context: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully renamed context '%s' to '%s'", oldName, newName)), nil
	}
}

func describeContextHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		contextInfo, err := cm.GetContextInfo(name)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get context info: %s", err.Error())), nil
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("Context: %s\n", contextInfo.Name))
		result.WriteString(fmt.Sprintf("Cluster: %s\n", contextInfo.Cluster))
		result.WriteString(fmt.Sprintf("User: %s\n", contextInfo.User))
		result.WriteString(fmt.Sprintf("Namespace: %s\n", contextInfo.Namespace))
		result.WriteString(fmt.Sprintf("Server: %s\n", contextInfo.ServerURL))
		result.WriteString(fmt.Sprintf("Config Path: %s\n", contextInfo.ConfigPath))

		status := "inactive"
		if contextInfo.IsActive {
			status = "active"
		}
		result.WriteString(fmt.Sprintf("Status: %s", status))

		return mcp.NewToolResultText(result.String()), nil
	}
}
