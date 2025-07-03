package tools

import (
	"context"

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
	factory := &DefaultDeploymentFactory{}
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

	describeDeploymentTool := mcp.NewTool("describe_deployment",
		mcp.WithDescription("Get detailed information about a specific deployment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the deployment"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the deployment (defaults to current namespace)"),
		),
	)

	s.AddTool(describeDeploymentTool, describeDeploymentHandler(cm, factory))

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

	getDeploymentTool := mcp.NewTool("get_deployment",
		mcp.WithDescription("Get basic information about a specific deployment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the deployment"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the deployment (defaults to current namespace)"),
		),
	)

	s.AddTool(getDeploymentTool, getDeploymentHandler(cm, factory))

	updateDeploymentTool := mcp.NewTool("update_deployment",
		mcp.WithDescription("Update an existing deployment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the deployment to update"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the deployment (defaults to current namespace)"),
		),
		mcp.WithString("image",
			mcp.Description("New container image to use for the deployment"),
		),
		mcp.WithNumber("replicas",
			mcp.Description("New number of replicas"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to add or update on the deployment and pods"),
		),
		mcp.WithString("container_port",
			mcp.Description("Container port to expose (format: 'port' or 'port/protocol')"),
		),
		mcp.WithObject("env",
			mcp.Description("Environment variables to add or update as key-value pairs"),
		),
		mcp.WithArray("image_pull_secrets",
			mcp.Description("Names of image pull secrets"),
		),
		mcp.WithString("image_pull_policy",
			mcp.Description("Image pull policy (Always, IfNotPresent, Never)"),
		),
	)

	s.AddTool(updateDeploymentTool, updateDeploymentHandler(cm, factory))
}

// getDeploymentHandler handles the get_deployment tool
func getDeploymentHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.DeploymentParams{
			Name:      name,
			Namespace: namespace,
		}

		deployment := factory.NewDeployment(params)

		resultText, err := deployment.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
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
		resultText, err := deployment.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// describeDeploymentHandler handles the describe_deployment tool
func describeDeploymentHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.DeploymentParams{
			Name:      name,
			Namespace: namespace,
		}

		deployment := factory.NewDeployment(params)

		// Use the Describe method instead of Get
		resultText, err := deployment.Describe(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// createDeploymentHandler handles the create_deployment tool
func createDeploymentHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := kai.DeploymentParams{
			Replicas: 1, // Set default replica count to 1
		}

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		imageArg, ok := request.Params.Arguments["image"]
		if !ok || imageArg == nil {
			return mcp.NewToolResultText(errMissingImage), nil
		}

		image, ok := imageArg.(string)
		if !ok || image == "" {
			return mcp.NewToolResultText(errEmptyImage), nil
		}

		if replicasArg, ok := request.Params.Arguments["replicas"].(float64); ok {
			params.Replicas = replicasArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if containerPortArg, ok := request.Params.Arguments["container_port"].(string); ok && containerPortArg != "" {
			errMsg := validateContainerPort(containerPortArg)
			if errMsg != nil {
				return mcp.NewToolResultText(errMsg.Error()), nil
			}
			params.ContainerPort = containerPortArg
		}

		if envArg, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			params.Env = envArg
		}

		if imagePullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			params.ImagePullSecrets = imagePullSecretsArg
		}

		if imagePullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok {
			errMsg := validateImagePullPolicy(imagePullPolicyArg)
			if errMsg != nil {
				return mcp.NewToolResultText(errMsg.Error()), nil
			}
			params.ImagePullPolicy = imagePullPolicyArg
		}

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

// updateDeploymentHandler handles the update_deployment tool
func updateDeploymentHandler(cm kai.ClusterManager, factory DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := kai.DeploymentParams{}

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		params.Name = name

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}
		params.Namespace = namespace

		var hasUpdateParams bool // false

		if imageArg, ok := request.Params.Arguments["image"].(string); ok && imageArg != "" {
			params.Image = imageArg
			hasUpdateParams = true
		}

		if replicasArg, ok := request.Params.Arguments["replicas"].(float64); ok {
			params.Replicas = replicasArg
			hasUpdateParams = true
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
			hasUpdateParams = true
		}

		if containerPortArg, ok := request.Params.Arguments["container_port"].(string); ok && containerPortArg != "" {
			errMsg := validateContainerPort(containerPortArg)
			if errMsg != nil {
				return mcp.NewToolResultText(errMsg.Error()), nil
			}
			params.ContainerPort = containerPortArg
			hasUpdateParams = true
		}

		if envArg, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			params.Env = envArg
			hasUpdateParams = true
		}

		if imagePullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			params.ImagePullSecrets = imagePullSecretsArg
			hasUpdateParams = true
		}

		if imagePullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok {
			errMsg := validateImagePullPolicy(imagePullPolicyArg)
			if errMsg != nil {
				return mcp.NewToolResultText(errMsg.Error()), nil
			}
			params.ImagePullPolicy = imagePullPolicyArg
			hasUpdateParams = true
		}

		if !hasUpdateParams {
			return mcp.NewToolResultText(errNoUpdateParams), nil
		}

		deployment := factory.NewDeployment(params)
		resultText, err := deployment.Update(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}
