package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// CronJobFactory is an interface for creating CronJob operators.
type CronJobFactory interface {
	NewCronJob(params kai.CronJobParams) kai.CronJobOperator
}

// DefaultCronJobFactory implements the CronJobFactory interface.
type DefaultCronJobFactory struct{}

// NewDefaultCronJobFactory creates a new DefaultCronJobFactory.
func NewDefaultCronJobFactory() *DefaultCronJobFactory {
	return &DefaultCronJobFactory{}
}

// NewCronJob creates a new CronJob operator.
func (f *DefaultCronJobFactory) NewCronJob(params kai.CronJobParams) kai.CronJobOperator {
	return &cluster.CronJob{
		Name:                       params.Name,
		Namespace:                  params.Namespace,
		Schedule:                   params.Schedule,
		Image:                      params.Image,
		Command:                    params.Command,
		Args:                       params.Args,
		RestartPolicy:              params.RestartPolicy,
		ConcurrencyPolicy:          params.ConcurrencyPolicy,
		Suspend:                    params.Suspend,
		SuccessfulJobsHistoryLimit: params.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     params.FailedJobsHistoryLimit,
		StartingDeadlineSeconds:    params.StartingDeadlineSeconds,
		BackoffLimit:               params.BackoffLimit,
		Labels:                     params.Labels,
		Env:                        params.Env,
		ImagePullPolicy:            params.ImagePullPolicy,
		ImagePullSecrets:           params.ImagePullSecrets,
	}
}

// RegisterCronJobTools registers all CronJob-related tools with the server.
func RegisterCronJobTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultCronJobFactory()
	RegisterCronJobToolsWithFactory(s, cm, factory)
}

// RegisterCronJobToolsWithFactory registers all CronJob-related tools using the provided factory.
func RegisterCronJobToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory CronJobFactory) {
	createCronJobTool := mcp.NewTool("create_cronjob",
		mcp.WithDescription("Create a new CronJob in the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the CronJob"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the CronJob (defaults to current namespace)"),
		),
		mcp.WithString("schedule",
			mcp.Required(),
			mcp.Description("Cron schedule expression (e.g., '*/5 * * * *' for every 5 minutes)"),
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
		mcp.WithString("concurrency_policy",
			mcp.Description("How to treat concurrent executions: Allow, Forbid, or Replace"),
		),
		mcp.WithBoolean("suspend",
			mcp.Description("Whether the CronJob is suspended"),
		),
		mcp.WithNumber("successful_jobs_history_limit",
			mcp.Description("Number of successful finished jobs to retain"),
		),
		mcp.WithNumber("failed_jobs_history_limit",
			mcp.Description("Number of failed finished jobs to retain"),
		),
		mcp.WithNumber("starting_deadline_seconds",
			mcp.Description("Deadline in seconds for starting the job if it misses scheduled time"),
		),
		mcp.WithNumber("backoff_limit",
			mcp.Description("Number of retries before marking the Job as failed"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the CronJob"),
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
	s.AddTool(createCronJobTool, createCronJobHandler(cm, factory))

	getCronJobTool := mcp.NewTool("get_cronjob",
		mcp.WithDescription("Get information about a specific CronJob"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the CronJob"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the CronJob (defaults to current namespace)"),
		),
	)
	s.AddTool(getCronJobTool, getCronJobHandler(cm, factory))

	listCronJobsTool := mcp.NewTool("list_cronjobs",
		mcp.WithDescription("List CronJobs in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list CronJobs across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list CronJobs from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter CronJobs (e.g., 'app=nginx,env=prod')"),
		),
	)
	s.AddTool(listCronJobsTool, listCronJobsHandler(cm, factory))

	deleteCronJobTool := mcp.NewTool("delete_cronjob",
		mcp.WithDescription("Delete a CronJob from the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the CronJob to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the CronJob (defaults to current namespace)"),
		),
	)
	s.AddTool(deleteCronJobTool, deleteCronJobHandler(cm, factory))
}

func createCronJobHandler(cm kai.ClusterManager, factory CronJobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_cronjob"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		scheduleArg, ok := request.Params.Arguments["schedule"]
		if !ok || scheduleArg == nil {
			return mcp.NewToolResultText("schedule is required"), nil
		}

		schedule, ok := scheduleArg.(string)
		if !ok || schedule == "" {
			return mcp.NewToolResultText("schedule cannot be empty"), nil
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

		params := kai.CronJobParams{
			Name:      name,
			Namespace: namespace,
			Schedule:  schedule,
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

		if concurrencyPolicyArg, ok := request.Params.Arguments["concurrency_policy"].(string); ok && concurrencyPolicyArg != "" {
			params.ConcurrencyPolicy = concurrencyPolicyArg
		}

		if suspendArg, ok := request.Params.Arguments["suspend"].(bool); ok {
			params.Suspend = &suspendArg
		}

		if successfulJobsHistoryLimitArg, ok := request.Params.Arguments["successful_jobs_history_limit"].(float64); ok {
			limit := int32(successfulJobsHistoryLimitArg)
			params.SuccessfulJobsHistoryLimit = &limit
		}

		if failedJobsHistoryLimitArg, ok := request.Params.Arguments["failed_jobs_history_limit"].(float64); ok {
			limit := int32(failedJobsHistoryLimitArg)
			params.FailedJobsHistoryLimit = &limit
		}

		if startingDeadlineSecondsArg, ok := request.Params.Arguments["starting_deadline_seconds"].(float64); ok {
			deadline := int64(startingDeadlineSecondsArg)
			params.StartingDeadlineSeconds = &deadline
		}

		if backoffLimitArg, ok := request.Params.Arguments["backoff_limit"].(float64); ok {
			backoffLimit := int32(backoffLimitArg)
			params.BackoffLimit = &backoffLimit
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

		cronJob := factory.NewCronJob(params)
		result, err := cronJob.Create(ctx, cm)
		if err != nil {
			slog.Warn("failed to create CronJob",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create CronJob: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getCronJobHandler(cm kai.ClusterManager, factory CronJobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_cronjob"))

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

		params := kai.CronJobParams{
			Name:      name,
			Namespace: namespace,
		}

		cronJob := factory.NewCronJob(params)
		result, err := cronJob.Get(ctx, cm)
		if err != nil {
			slog.Warn("failed to get CronJob",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get CronJob: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listCronJobsHandler(cm kai.ClusterManager, factory CronJobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_cronjobs"))

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

		params := kai.CronJobParams{
			Namespace: namespace,
		}

		cronJob := factory.NewCronJob(params)
		result, err := cronJob.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			slog.Warn("failed to list CronJobs",
				slog.Bool("all_namespaces", allNamespaces),
				slog.String("namespace", namespace),
				slog.String("label_selector", labelSelector),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list CronJobs: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteCronJobHandler(cm kai.ClusterManager, factory CronJobFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_cronjob"))

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

		params := kai.CronJobParams{
			Name:      name,
			Namespace: namespace,
		}

		cronJob := factory.NewCronJob(params)
		result, err := cronJob.Delete(ctx, cm)
		if err != nil {
			slog.Warn("failed to delete CronJob",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete CronJob: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
