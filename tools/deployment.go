package tools

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterDeploymentTools(s kai.ServerInterface, cm kai.ClusterManagerInterface) {
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

	s.AddTool(listDeploymentTool, listDeploymentsHandler(cm))

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

	s.AddTool(createDeploymentTool, createDeploymentHandler(cm))
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

func createDeploymentHandler(cm kai.ClusterManagerInterface) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resultText string
		var deploymentParams kai.DeploymentParams

		name, ok := request.Params.Arguments["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultText("Parameter 'name' must be a non-empty string"), nil
		}

		image, ok := request.Params.Arguments["image"].(string)
		if !ok || image == "" {
			return mcp.NewToolResultText("Parameter 'image' must be a non-empty string"), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		deploymentParams.Namespace = namespace

		if replicasArg, ok := request.Params.Arguments["replicas"].(int); ok {
			// assign if replica count is greater than 1 otherwise use the default replica count (1)
			deploymentParams.Replicas = int64(replicasArg)
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			deploymentParams.Labels = labelsArg
		}

		if portArg, ok := request.Params.Arguments["container_port"].(string); ok && portArg != "" {
			deploymentParams.ContainerPort = portArg
		}

		if envArg, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			deploymentParams.Env = envArg
		}

		if pullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok {
			deploymentParams.ImagePullPolicy = pullPolicyArg
		}

		if pullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			deploymentParams.ImagePullSecrets = pullSecretsArg
		}

		resultText, err := cm.CreateDeployment(ctx, deploymentParams)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}
		return mcp.NewToolResultText(resultText), nil
	}
}
