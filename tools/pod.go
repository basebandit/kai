package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

type PodFactory interface {
	NewPod(params kai.PodParams) kai.PodOperator
}

type DefaultPodFactory struct{}

func (f *DefaultPodFactory) NewPod(params kai.PodParams) kai.PodOperator {
	return &cluster.Pod{
		Name:             params.Name,
		Image:            params.Image,
		Namespace:        params.Namespace,
		ContainerName:    params.ContainerName,
		ContainerPort:    params.ContainerPort,
		ImagePullPolicy:  params.ImagePullPolicy,
		ImagePullSecrets: params.ImagePullSecrets,
		RestartPolicy:    params.RestartPolicy,
		ServiceAccount:   params.ServiceAccountName,
		Command:          params.Command,
		Args:             params.Args,
		NodeSelector:     params.NodeSelector,
		Labels:           params.Labels,
		Env:              params.Env,
	}
}

func RegisterPodTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := &DefaultPodFactory{}
	RegisterPodToolsWithFactory(s, cm, factory)
}

func RegisterPodToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory PodFactory) {
	createPodTool := mcp.NewTool("create_pod",
		mcp.WithDescription("Create a new pod in the current namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the pod"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the pod (defaults to current namespace)"),
		),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Container image to use for the pod"),
		),
		mcp.WithArray("command",
			mcp.Description("Command to run in the container"),
		),
		mcp.WithArray("args",
			mcp.Description("Arguments to the command"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the pod"),
		),
		mcp.WithString("container_name",
			mcp.Description("Name of the container (defaults to pod name)"),
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
		mcp.WithString("restart_policy",
			mcp.Description("Restart policy for the pod (Always, OnFailure, Never)"),
		),
		mcp.WithObject("node_selector",
			mcp.Description("Node selector as key-value pairs"),
		),
		mcp.WithString("service_account",
			mcp.Description("Service account to use for the pod"),
		),
	)

	s.AddTool(createPodTool, createPodHandler(cm, factory))

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

	s.AddTool(listPodTools, listPodsHandler(cm, factory))

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

	s.AddTool(getPodTool, getPodHandler(cm, factory))

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

	s.AddTool(deletePodTool, deletePodHandler(cm, factory))

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

	s.AddTool(streamLogsTool, streamLogsHandler(cm, factory))
}

// createPodHandler handles the create_pod tool
func createPodHandler(cm kai.ClusterManager, factory PodFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := kai.PodParams{
			RestartPolicy: "Always", // Default restart policy
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

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params.Name = name
		params.Image = image
		params.Namespace = namespace

		if commandArg, ok := request.Params.Arguments["command"].([]interface{}); ok && len(commandArg) > 0 {
			params.Command = make([]interface{}, len(commandArg))
			for i, cmd := range commandArg {
				if cmdStr, ok := cmd.(string); ok {
					params.Command[i] = cmdStr
				}
			}
		}

		if argsArg, ok := request.Params.Arguments["args"].([]interface{}); ok && len(argsArg) > 0 {
			params.Args = make([]interface{}, len(argsArg))
			for i, arg := range argsArg {
				if argStr, ok := arg.(string); ok {
					params.Args[i] = argStr
				}
			}
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if containerNameArg, ok := request.Params.Arguments["container_name"].(string); ok && containerNameArg != "" {
			params.ContainerName = containerNameArg
		} else {
			params.ContainerName = name
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

		if imagePullPolicyArg, ok := request.Params.Arguments["image_pull_policy"].(string); ok && imagePullPolicyArg != "" {
			validPolicies := map[string]bool{"Always": true, "IfNotPresent": true, "Never": true}
			if validPolicies[imagePullPolicyArg] {
				params.ImagePullPolicy = imagePullPolicyArg
			} else {
				return mcp.NewToolResultText("Invalid image_pull_policy. Must be one of: Always, IfNotPresent, Never"), nil
			}
		}

		if imagePullSecretsArg, ok := request.Params.Arguments["image_pull_secrets"].([]interface{}); ok {
			params.ImagePullSecrets = imagePullSecretsArg
		}

		if restartPolicyArg, ok := request.Params.Arguments["restart_policy"].(string); ok && restartPolicyArg != "" {
			validPolicies := map[string]bool{"Always": true, "OnFailure": true, "Never": true}
			if validPolicies[restartPolicyArg] {
				params.RestartPolicy = restartPolicyArg
			} else {
				return mcp.NewToolResultText("Invalid restart_policy. Must be one of: Always, OnFailure, Never"), nil
			}
		}

		if nodeSelectorArg, ok := request.Params.Arguments["node_selector"].(map[string]interface{}); ok {
			params.NodeSelector = nodeSelectorArg
		}

		if serviceAccountArg, ok := request.Params.Arguments["service_account"].(string); ok && serviceAccountArg != "" {
			params.ServiceAccountName = serviceAccountArg
		}

		pod := factory.NewPod(params)

		resultText, err := pod.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func listPodsHandler(cm kai.ClusterManager, factory PodFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.PodParams{
			Namespace: namespace,
		}
		pod := factory.NewPod(params)

		resultText, err := pod.List(ctx, cm, limit, labelSelector, fieldSelector)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func getPodHandler(cm kai.ClusterManager, factory PodFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.PodParams{
			Name:      name,
			Namespace: namespace,
		}

		pod := factory.NewPod(params)

		resultText, err := pod.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func deletePodHandler(cm kai.ClusterManager, factory PodFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		params := kai.PodParams{
			Name:      name,
			Namespace: namespace,
		}

		pod := factory.NewPod(params)

		resultText, err := pod.Delete(ctx, cm, force)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func streamLogsHandler(cm kai.ClusterManager, factory PodFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		var sinceDuration *time.Duration
		if sinceArg, ok := request.Params.Arguments["since"].(string); ok && sinceArg != "" {
			duration, err := time.ParseDuration(sinceArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Failed to parse 'since' parameter: %v", err)), nil
			}
			sinceDuration = &duration
		}

		params := kai.PodParams{
			Name:          podName,
			Namespace:     namespace,
			ContainerName: containerName,
		}

		pod := factory.NewPod(params)

		resultText, err := pod.StreamLogs(ctx, cm, tailLines, previous, sinceDuration)

		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}
		return mcp.NewToolResultText(resultText), nil
	}
}
