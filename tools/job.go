package tools

import (
	"context"
	"fmt"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// JobFactory is an interface for creating Job operators.
type JobFactory interface {
	NewJob(params kai.JobParams) kai.JobOperator
}

// DefaultJobFactory implements the JobFactory interface.
type DefaultJobFactory struct{}

// NewDefaultJobFactory creates a new DefaultJobFactory.
func NewDefaultJobFactory() *DefaultJobFactory {
	return &DefaultJobFactory{}
}

// NewJob creates a new Job operator.
func (f *DefaultJobFactory) NewJob(params kai.JobParams) kai.JobOperator {
	return &cluster.Job{
		Name:             params.Name,
		Namespace:        params.Namespace,
		Image:            params.Image,
		Command:          params.Command,
		Args:             params.Args,
		RestartPolicy:    params.RestartPolicy,
		BackoffLimit:     params.BackoffLimit,
		Completions:      params.Completions,
		Parallelism:      params.Parallelism,
		Labels:           params.Labels,
		Env:              params.Env,
		ImagePullPolicy:  params.ImagePullPolicy,
		ImagePullSecrets: params.ImagePullSecrets,
	}
}

// RegisterJobTools registers all Job-related tools with the server.
func RegisterJobTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultJobFactory()
	RegisterJobToolsWithFactory(s, cm, factory)
}

// RegisterJobToolsWithFactory registers all Job-related tools using the provided factory.
func RegisterJobToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory JobFactory) {
	createJobTool := mcp.NewTool("create_job",
		mcp.WithDescription("Create a new Job in the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Job"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the Job (defaults to current namespace)"),
		),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Container image to run"),
		),
		mcp.WithArray("command",
			mcp.Description("Command to run in the container (overrides entrypoint)"),
		),
		mcp.WithArray("args",
			mcp.Description("Arguments to pass to the command"),
		),
		mcp.WithString("restart_policy",
			mcp.Description("Restart policy for the pod (OnFailure, Never)"),
		),
		mcp.WithNumber("backoff_limit",
			mcp.Description("Number of retries before marking the Job as failed"),
		),
		mcp.WithNumber("completions",
			mcp.Description("Number of successful pod completions needed"),
		),
		mcp.WithNumber("parallelism",
			mcp.Description("Maximum number of pods running in parallel"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the Job"),
		),
		mcp.WithObject("env",
			mcp.Description("Environment variables as key-value pairs"),
		),
		mcp.WithString("image_pull_policy",
			mcp.Description(descImagePullPolicy),
		),
		mcp.WithArray("image_pull_secrets",
			mcp.Description("Image pull secrets for private registries"),
		),
	)
	s.AddTool(createJobTool, createJobHandler(cm, factory))

	getJobTool := mcp.NewTool("get_job",
		mcp.WithDescription("Get information about a specific Job"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Job"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Job (defaults to current namespace)"),
		),
	)
	s.AddTool(getJobTool, getJobHandler(cm, factory))

	listJobsTool := mcp.NewTool("list_jobs",
		mcp.WithDescription("List Jobs in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list Jobs across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list Jobs from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter Jobs (e.g., 'app=nginx,env=prod')"),
		),
	)
	s.AddTool(listJobsTool, listJobsHandler(cm, factory))

	deleteJobTool := mcp.NewTool("delete_job",
		mcp.WithDescription("Delete a Job from the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Job to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Job (defaults to current namespace)"),
		),
	)
	s.AddTool(deleteJobTool, deleteJobHandler(cm, factory))
}

func createJobHandler(cm kai.ClusterManager, factory JobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.JobParams{
			Name:      name,
			Namespace: namespace,
			Image:     image,
		}

		if commandArg, ok := request.Params.Arguments["command"].([]interface{}); ok {
			params.Command = commandArg
		}

		if argsArg, ok := request.Params.Arguments["args"].([]interface{}); ok {
			params.Args = argsArg
		}

		if restartPolicyArg, ok := request.Params.Arguments["restart_policy"].(string); ok && restartPolicyArg != "" {
			params.RestartPolicy = restartPolicyArg
		}

		if backoffLimitArg, ok := request.Params.Arguments["backoff_limit"].(float64); ok {
			backoffLimit := int32(backoffLimitArg)
			params.BackoffLimit = &backoffLimit
		}

		if completionsArg, ok := request.Params.Arguments["completions"].(float64); ok {
			completions := int32(completionsArg)
			params.Completions = &completions
		}

		if parallelismArg, ok := request.Params.Arguments["parallelism"].(float64); ok {
			parallelism := int32(parallelismArg)
			params.Parallelism = &parallelism
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if envArg, ok := request.Params.Arguments["env"].(map[string]interface{}); ok {
			params.Env = envArg
		}

		if imagePullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok && imagePullPolicyArg != "" {
			params.ImagePullPolicy = imagePullPolicyArg
		}

		if imagePullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			params.ImagePullSecrets = imagePullSecretsArg
		}

		job := factory.NewJob(params)
		result, err := job.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create Job: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getJobHandler(cm kai.ClusterManager, factory JobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.JobParams{
			Name:      name,
			Namespace: namespace,
		}

		job := factory.NewJob(params)
		result, err := job.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get Job: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listJobsHandler(cm kai.ClusterManager, factory JobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.JobParams{
			Namespace: namespace,
		}

		job := factory.NewJob(params)
		result, err := job.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list Jobs: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteJobHandler(cm kai.ClusterManager, factory JobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.JobParams{
			Name:      name,
			Namespace: namespace,
		}

		job := factory.NewJob(params)
		result, err := job.Delete(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete Job: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
