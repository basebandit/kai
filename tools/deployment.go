package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// DeploymentFactory is an interface for creating deployment operators
type DeploymentFactory interface {
	NewDeployment(params kai.DeploymentParams) kai.DeploymentOperator
}

// DefaultDeploymentFactory implements the DeploymentFactory interface
type DefaultDeploymentFactory struct{}

// NewDefaultDeploymentFactory creates a new DefaultDeploymentFactory
func NewDefaultDeploymentFactory() *DefaultDeploymentFactory {
	return &DefaultDeploymentFactory{}
}

// NewDeployment creates a new deployment operator
func (f *DefaultDeploymentFactory) NewDeployment(params kai.DeploymentParams) kai.DeploymentOperator {
	return &cluster.Deployment{
		Name:             params.Name,
		Image:            params.Image,
		Namespace:        params.Namespace,
		Replicas:         params.Replicas,
		Labels:           params.Labels,
		ContainerPort:    params.ContainerPort,
		Env:              params.Env,
		ImagePullPolicy:  params.ImagePullPolicy,
		ImagePullSecrets: params.ImagePullSecrets,
	}
}

// RegisterDeploymentTools registers all deployment-related tools with the server
func RegisterDeploymentTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultDeploymentFactory()
	RegisterDeploymentToolsWithFactory(s, cm, factory)
}

// RegisterDeploymentToolsWithFactory registers all deployment-related tools with the server using the provided factory
func RegisterDeploymentToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory DeploymentFactory) {
	listDeploymentTool := mcp.NewTool("list_deployments",
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

	s.AddTool(listDeploymentTool, listDeploymentsHandler(cm, factory))

	createDeploymentTool := mcp.NewTool("create_deployment",
		mcp.WithDescription("Create a new deployment in the current namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the deployment"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the deployment (defaults to current namespace)"),
		),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Container image to use for the deployment"),
		),
		mcp.WithNumber("replicas",
			mcp.Description("Number of replicas (defaults to 1)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the deployment and pods"),
		),
		mcp.WithString("container_port",
			mcp.Description("Container port to expose (format: 'port' or 'port/protocol')"),
		),
		mcp.WithObject("env",
			mcp.Description("Environment variables as key-value pairs"),
		),
		mcp.WithArray("image_pull_secrets",
			mcp.Description("Names of image pull secrets"),
		),
		mcp.WithString("image_pull_policy",
			mcp.Description("Image pull policy (Always, IfNotPresent, Never)"),
		),
	)

	s.AddTool(createDeploymentTool, createDeploymentHandler(cm, factory))
}

// listDeploymentsHandler handles the list_deployments tool
func listDeploymentsHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.DeploymentParams{
			Namespace: namespace, // will be used if allNamespaces is false
		}
		deployment := factory.NewDeployment(params)

		// List deployments
		resultText, err := deployment.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// createDeploymentHandler handles the create_deployment tool
func createDeploymentHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Initialize params with default values
		params := kai.DeploymentParams{
			Replicas: 1, // Set default replica count to 1
		}

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText("Required parameter 'name' is missing"), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		imageArg, ok := request.Params.Arguments["image"]
		if !ok || imageArg == nil {
			return mcp.NewToolResultText("Required parameter 'image' is missing"), nil
		}

		image, ok := imageArg.(string)
		if !ok || image == "" {
			return mcp.NewToolResultText("Parameter 'image' must be a non-empty string"), nil
		}

		// Process optional parameters
		if replicasArg, ok := request.Params.Arguments["replicas"].(float64); ok {
			params.Replicas = replicasArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if containerPortArg, ok := request.Params.Arguments["container_port"].(string); ok && containerPortArg != "" {
			if valid, errMsg := validateContainerPort(containerPortArg); valid {
				params.ContainerPort = containerPortArg
			} else {
				return mcp.NewToolResultText(errMsg), nil
			}
		}

		if envArg, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			params.Env = envArg
		}

		if imagePullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			params.ImagePullSecrets = imagePullSecretsArg
		}

		if imagePullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok {
			params.ImagePullPolicy = imagePullPolicyArg
		}

		// Get namespace (optional with default)
		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params.Namespace = namespace
		params.Image = image
		params.Name = name

		deployment := factory.NewDeployment(params)

		resultText, err := deployment.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// validateContainerPort checks if the containerPort string has the correct format
// Returns true if valid, false if invalid
func validateContainerPort(portStr string) (bool, string) {
	parts := strings.Split(portStr, "/")

	// Check basic format: should be "port" or "port/protocol"
	if len(parts) != 1 && len(parts) != 2 {
		return false, "Container port should be in format 'port' or 'port/protocol'"
	}

	// Validate port part is a number
	var port int
	if _, err := fmt.Sscanf(parts[0], "%d", &port); err != nil {
		return false, "Port must be a number"
	}

	// Validate protocol if provided
	if len(parts) == 2 {
		protocol := strings.ToUpper(parts[1])
		validProtocols := map[string]bool{"TCP": true, "UDP": true, "SCTP": true}
		if !validProtocols[protocol] {
			return false, fmt.Sprintf("Protocol %s is not valid. Must be one of: TCP, UDP, SCTP", parts[1])
		}
	}

	return true, ""
}
