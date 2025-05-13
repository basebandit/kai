package cluster

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
)

type Pod struct {
	Name          string
	Namespace     string
	ContainerName string
	LabelSelector string
	FieldSelector string
}

func (p *Pod) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", err
	}

	// Verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(ctx, p.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace '%s' not found: %v", p.Namespace, err)
	}

	// Use retry for potential transient issues
	var pod *corev1.Pod
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		// Only retry on transient errors
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		pod, getErr = client.CoreV1().Pods(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("pod '%s' not found in namespace '%s'", p.Name, p.Namespace)
		}
		return result, fmt.Errorf("failed to get pod '%s' in namespace '%s': %v", p.Name, p.Namespace, err)
	}

	return formatPod(pod), nil
}

func (p *Pod) List(ctx context.Context, cm kai.ClusterManager, limit int64) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", err
	}

	// Create list options
	listOptions := metav1.ListOptions{
		LabelSelector: p.LabelSelector,
		FieldSelector: p.FieldSelector,
	}

	showNamespace := (params.Namespace == "")
	resultText := "Pods across all namespaces"
	if !showNamespace {
		resultText = fmt.Sprintf("Pods in namespace %q", params.Namespace)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var pods *corev1.PodList
	var resultText string
	var listErr error
	var allNamespaces bool

	if p.Namespace == "" {
		allNamespaces = true
	}

	if allNamespaces {
		pods, listErr = client.CoreV1().Pods("").List(timeoutCtx, listOptions)
		resultText = "Pods across all namespaces:\n"
	} else {
		// First verify the namespace exists
		_, err = client.CoreV1().Namespaces().Get(timeoutCtx, p.Namespace, metav1.GetOptions{})
		if err != nil {
			return result, fmt.Errorf("namespace %q not found: %v", p.Namespace, err)
		}

		pods, listErr = client.CoreV1().Pods(p.Namespace).List(timeoutCtx, listOptions)
		resultText = fmt.Sprintf("Pods in namespace '%s':\n", p.Namespace)
	}

	if listErr != nil {
		return result, fmt.Errorf("failed to list pods: %v", listErr)
	}

	if len(pods.Items) == 0 {
		if p.LabelSelector != "" || p.FieldSelector != "" {
			return result, errors.New("no pods found matching the specified selectors")
		}
		return result, errors.New("no pods found")
	}

	return formatPodList(pods, allNamespaces, limit, resultText), nil
}

func (p *Pod) Delete(ctx context.Context, cm kai.ClusterManager, force bool) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("failed to delete pod %q in namespace %q: %v", params.Name, params.Namespace, err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// verify namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, p.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %v", p.Namespace, err)
	}

	// verify the pod exists
	_, err = client.CoreV1().Pods(p.Namespace).Get(timeoutCtx, p.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("pod %q not found in namespace %q", p.Name, p.Namespace)
	}

	deleteOptions := metav1.DeleteOptions{}
	if force {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	err = client.CoreV1().Pods(p.Namespace).Delete(timeoutCtx, p.Name, deleteOptions)
	if err != nil {
		return result, fmt.Errorf("failed to delete pod %q in namespace %q: %v", p.Name, p.Namespace, err)
	}

	return fmt.Sprintf("Successfully delete pod %q in namespace %q", p.Name, p.Namespace), nil
}

func (p *Pod) StreamLogs(ctx context.Context, cm kai.ClusterManager, tailLines int64, previous bool, since *time.Duration) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	//verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, p.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %v", p.Namespace, err)
	}

	// Get the pod to find the container name if not specified and verify pod exists
	pod, err := client.CoreV1().Pods(p.Namespace).Get(timeoutCtx, p.Name, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("pod '%s' not found in namespace '%s'", p.Name, p.Namespace)
		}
		return result, fmt.Errorf("failed to get pod '%s' in namespace '%s': %v", p.Name, p.Namespace, err)
	}

	// Check if pod is running or has run before
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded && !previous {
		return result, fmt.Errorf("pod '%s' is in '%s' state. Logs may not be available. Use previous=true for crashed containers",
			p.Name, pod.Status.Phase)
	}

	if len(pod.Spec.Containers) == 0 {
		return result, fmt.Errorf("no containers found in pod '%s'", p.Name)
	}

	// Set default container if not specified
	if p.ContainerName == "" {
		p.ContainerName = pod.Spec.Containers[0].Name
	}

	// Verify the container exists in the pod
	containerExists := false
	for _, container := range pod.Spec.Containers {
		if container.Name == p.ContainerName {
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

		return result, fmt.Errorf("container '%s' not found in pod '%s'. Available containers: %s",
			p.ContainerName, p.Name, strings.Join(availableContainers, ", "))
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
		Container: p.ContainerName,
		Previous:  previous,
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
		logsReq := client.CoreV1().Pods(p.Namespace).GetLogs(p.Name, logOptions)
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
		if previous {
			return result, fmt.Errorf("no previous logs found for container '%s' in pod '%s'", p.ContainerName, p.Name)
		}
		return result, fmt.Errorf("no logs found for container '%s' in pod '%s'", p.ContainerName, p.Name)
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

	result = fmt.Sprintf("Logs from container '%s' in pod '%s/%s'", p.ContainerName, p.Namespace, p.Name)
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
