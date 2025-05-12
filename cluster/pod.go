package clustermanager

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
)

const (
	podResourceType       = "pods"
	podResourceApiVersion = "v1"
)

// PodParams contains parameters for pod operations
type PodParams struct {
	Name,
	Namespace,
	LabelSelector,
	FieldSelector string
	Limit int64
	Force bool
	Log   PodLogParams
}

// PodLogParams contains parameters for pod log streaming
type PodLogParams struct {
	TailLines     int64
	Previous      bool
	ContainerName string
	Since         *time.Duration
}

// GetPod gets a pod with strong typing
func (cm *Cluster) GetPod(ctx context.Context, params PodParams) (string, error) {
	resource, err := cm.GetResource(ctx, podResourceType, params.Name, params.Namespace, "", podResourceApiVersion)
	if err != nil {
		return "", err
	}

	pod, err := convertToPod(resource)
	if err != nil {
		return "", err
	}

	return formatPod(pod), nil
}

// ListPods lists pods with strong typing
func (cm *Cluster) ListPods(ctx context.Context, params PodParams) (string, error) {
	resources, err := cm.ListResources(ctx, params.Limit, podResourceType, params.Namespace, params.LabelSelector, params.FieldSelector, "", podResourceApiVersion)
	if err != nil {
		return "", err
	}

	podList, err := convertToPodList(resources)
	if err != nil {
		return "", err
	}

	showNamespace := (params.Namespace == "")
	resultText := "Pods across all namespaces"
	if !showNamespace {
		resultText = fmt.Sprintf("Pods in namespace %q", params.Namespace)
	}

	return formatPodList(podList, showNamespace, params.Limit, resultText), nil
}

// DeletePod deletes a pod by name
func (cm *Cluster) DeletePod(ctx context.Context, params PodParams) (string, error) {
	err := cm.DeleteResource(ctx, params.Force, podResourceType, params.Name, params.Namespace, "", podResourceApiVersion)
	if err != nil {
		return "", fmt.Errorf("failed to delete pod %q in namespace %q: %v", params.Name, params.Namespace, err)
	}

	return fmt.Sprintf("Successfully deleted pod %q in namespace %q", params.Name, params.Namespace), nil
}

// StreamPodLogs streams logs from a pod container
func (cm *Cluster) StreamPodLogs(ctx context.Context, params PodParams) (string, error) {
	resource, err := cm.GetResource(ctx, podResourceType, params.Name, params.Namespace, "", podResourceApiVersion)
	if err != nil {
		return "", err
	}

	pod, err := convertToPod(resource)
	if err != nil {
		return "", err
	}

	// Check if pod is running or has run before
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded && !params.Log.Previous {
		return "", fmt.Errorf("pod '%s' is in '%s' state. Logs may not be available. Use previous=true for crashed containers",
			params.Name, pod.Status.Phase)
	}

	if len(pod.Spec.Containers) == 0 {
		return "", fmt.Errorf("no containers found in pod '%s'", params.Name)
	}

	// Set default container if not specified
	if params.Log.ContainerName == "" {
		params.Log.ContainerName = pod.Spec.Containers[0].Name
	}

	// Verify the container exists in the pod
	containerExists := false
	for _, container := range pod.Spec.Containers {
		if container.Name == params.Log.ContainerName {
			containerExists = true
			break
		}
	}

	if !containerExists {
		// List available containers
		availableContainers := make([]string, 0, len(pod.Spec.Containers))
		for _, container := range pod.Spec.Containers {
			availableContainers = append(availableContainers, container.Name)
		}

		return "", fmt.Errorf("container '%s' not found in pod '%s'. Available containers: %s",
			params.Log.ContainerName, params.Name, strings.Join(availableContainers, ", "))
	}

	// We need the typed client for streaming logs
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %v", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Configure log options
	logOptions := &corev1.PodLogOptions{
		Container: params.Log.ContainerName,
		Previous:  params.Log.Previous,
		Follow:    false, // We don't want to follow logs in this context
	}

	if params.Log.TailLines > 0 {
		logOptions.TailLines = &params.Log.TailLines
	}

	if params.Log.Since != nil {
		logOptions.SinceSeconds = ptr(int64(params.Log.Since.Seconds()))
	}

	// Get the logs with retry for transient errors
	var logsStream io.ReadCloser
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		// Retry on network errors
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		logsReq := client.CoreV1().Pods(params.Namespace).GetLogs(params.Name, logOptions)
		var streamErr error
		logsStream, streamErr = logsReq.Stream(timeoutCtx)
		return streamErr
	})

	if err != nil {
		return "", fmt.Errorf("failed to stream logs: %v", err)
	}
	defer logsStream.Close()

	// Read the logs with a max size limit to prevent excessive output
	maxSize := 100 * 1024 // Limit to ~100KB of logs
	logs, err := io.ReadAll(io.LimitReader(logsStream, int64(maxSize)))
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %v", err)
	}

	if len(logs) == 0 {
		if params.Log.Previous {
			return "", fmt.Errorf("no previous logs found for container '%s' in pod '%s'", params.Log.ContainerName, params.Name)
		}
		return "", fmt.Errorf("no logs found for container '%s' in pod '%s'", params.Log.ContainerName, params.Name)
	}

	var result strings.Builder

	options := []string{}
	if params.Log.Previous {
		options = append(options, "previous=true")
	}
	if params.Log.TailLines > 0 {
		options = append(options, fmt.Sprintf("tail=%d", params.Log.TailLines))
	}
	if params.Log.Since != nil {
		options = append(options, fmt.Sprintf("since=%s", params.Log.Since.String()))
	}

	result.WriteString(fmt.Sprintf("Logs from container '%s' in pod '%s/%s'", params.Log.ContainerName, params.Namespace, params.Name))
	if len(options) > 0 {
		result.WriteString(fmt.Sprintf(" (%s)", strings.Join(options, ", ")))
	}
	result.WriteString(":\n\n")
	result.WriteString(string(logs))

	// Check if we reached the size limit
	if len(logs) == maxSize {
		result.WriteString("\n\n[Output truncated due to size limits. Use the 'tail' or 'since' parameters to view specific sections of logs.]")
	}

	return result.String(), nil
}

// convertToPod converts an unstructured object to a Pod
func convertToPod(obj *unstructured.Unstructured) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, pod)
	return pod, err
}

// convertToPodList converts an unstructured list to a PodList
func convertToPodList(list *unstructured.UnstructuredList) (*corev1.PodList, error) {
	podList := &corev1.PodList{
		Items: make([]corev1.Pod, len(list.Items)),
	}

	for i, item := range list.Items {
		pod := &corev1.Pod{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, pod)
		if err != nil {
			return nil, err
		}
		podList.Items[i] = *pod
	}

	return podList, nil
}
