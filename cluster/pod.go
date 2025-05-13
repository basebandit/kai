package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type Pod struct {
	Name             string
	Image            string
	Namespace        string
	ContainerName    string
	ContainerPort    string
	ImagePullPolicy  string
	RestartPolicy    string
	ServiceAccount   string
	Command          []interface{}
	Args             []interface{}
	ImagePullSecrets []interface{}
	NodeSelector     map[string]interface{}
	Labels           map[string]interface{}
	Env              map[string]interface{}
}

// Create creates a new pod in the cluster
func (p *Pod) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	// Validate required fields
	if p.Image == "" {
		return result, fmt.Errorf("failed to create pod: image cannot be empty")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, p.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", p.Namespace, err)
	}

	// Create the pod object
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: corev1.PodSpec{},
	}

	// Set labels if provided
	if p.Labels != nil {
		labels := make(map[string]string)
		for k, v := range p.Labels {
			if strVal, ok := v.(string); ok {
				labels[k] = strVal
			}
		}
		if len(labels) > 0 {
			pod.ObjectMeta.Labels = labels
		}
	}

	// Create container
	container := corev1.Container{
		Name:  p.ContainerName,
		Image: p.Image,
	}

	// If container name is not provided, use the pod name
	if container.Name == "" {
		container.Name = p.Name
	}

	// Set container port if specified
	if p.ContainerPort != "" {
		parts := strings.Split(p.ContainerPort, "/")
		var portVal int32
		if _, err := fmt.Sscanf(parts[0], "%d", &portVal); err == nil {
			portDefinition := corev1.ContainerPort{
				ContainerPort: portVal,
			}

			// Add protocol if specified
			if len(parts) > 1 {
				protocol := corev1.Protocol(strings.ToUpper(parts[1]))
				if protocol == corev1.ProtocolTCP || protocol == corev1.ProtocolUDP || protocol == corev1.ProtocolSCTP {
					portDefinition.Protocol = protocol
				}
			}

			container.Ports = []corev1.ContainerPort{portDefinition}
		}
	}

	// Set image pull policy if specified
	if p.ImagePullPolicy != "" {
		policyMap := map[string]corev1.PullPolicy{
			"Always":       corev1.PullAlways,
			"IfNotPresent": corev1.PullIfNotPresent,
			"Never":        corev1.PullNever,
		}
		if policy, ok := policyMap[p.ImagePullPolicy]; ok {
			container.ImagePullPolicy = policy
		}
	}

	// Set command if specified
	if p.Command != nil {
		command := make([]string, 0, len(p.Command))
		for _, cmd := range p.Command {
			if cmdStr, ok := cmd.(string); ok {
				command = append(command, cmdStr)
			}
		}
		if len(command) > 0 {
			container.Command = command
		}
	}

	// Set args if specified
	if p.Args != nil {
		args := make([]string, 0, len(p.Args))
		for _, arg := range p.Args {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
		if len(args) > 0 {
			container.Args = args
		}
	}

	// Set environment variables if specified
	if p.Env != nil {
		envVars := make([]corev1.EnvVar, 0, len(p.Env))
		for k, v := range p.Env {
			if strVal, ok := v.(string); ok {
				envVars = append(envVars, corev1.EnvVar{
					Name:  k,
					Value: strVal,
				})
			}
		}
		if len(envVars) > 0 {
			container.Env = envVars
		}
	}

	// Add the container to the pod
	pod.Spec.Containers = []corev1.Container{container}

	// Set restart policy if specified
	if p.RestartPolicy != "" {
		policyMap := map[string]corev1.RestartPolicy{
			"Always":    corev1.RestartPolicyAlways,
			"OnFailure": corev1.RestartPolicyOnFailure,
			"Never":     corev1.RestartPolicyNever,
		}
		if policy, ok := policyMap[p.RestartPolicy]; ok {
			pod.Spec.RestartPolicy = policy
		}
	}

	// Set service account if specified
	if p.ServiceAccount != "" {
		pod.Spec.ServiceAccountName = p.ServiceAccount
	}

	// Set node selector if specified
	if p.NodeSelector != nil {
		nodeSelector := make(map[string]string)
		for k, v := range p.NodeSelector {
			if strVal, ok := v.(string); ok {
				nodeSelector[k] = strVal
			}
		}
		if len(nodeSelector) > 0 {
			pod.Spec.NodeSelector = nodeSelector
		}
	}

	// Set image pull secrets if specified
	if p.ImagePullSecrets != nil {
		pullSecrets := make([]corev1.LocalObjectReference, 0, len(p.ImagePullSecrets))
		for _, v := range p.ImagePullSecrets {
			if strVal, ok := v.(string); ok && strVal != "" {
				pullSecrets = append(pullSecrets, corev1.LocalObjectReference{
					Name: strVal,
				})
			}
		}
		if len(pullSecrets) > 0 {
			pod.Spec.ImagePullSecrets = pullSecrets
		}
	}

	// Create the pod
	createdPod, err := client.CoreV1().Pods(p.Namespace).Create(timeoutCtx, pod, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create pod: %w", err)
	}

	result = fmt.Sprintf("Pod %q created successfully in namespace %q", createdPod.Name, createdPod.Namespace)
	return result, nil
}

func (p *Pod) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, err
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

func (p *Pod) List(ctx context.Context, cm kai.ClusterManager, limit int64, labelSelector, fieldSelector string) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, nil
	}

	// Create list options
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	if limit > 0 {
		listOptions.Limit = int64(limit)
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
		if labelSelector != "" || fieldSelector != "" {
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
		return result, fmt.Errorf("error: %v", err)
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
		return result, fmt.Errorf("error: %v", err)
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

	// Configure log options
	logOptions := &corev1.PodLogOptions{
		Container: p.ContainerName,
		Previous:  previous,
		Follow:    false, // We don't want to follow logs in this context
	}

	if tailLines > 0 {
		logOptions.TailLines = &tailLines
	}

	if since != nil {
		logOptions.SinceSeconds = ptr(int64(since.Seconds()))
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
		return result, fmt.Errorf("failed to stream logs: %v", err)
	}
	defer logsStream.Close()

	// Read the logs with a max size limit to prevent excessive output
	maxSize := 100 * 1024 // Limit to ~100KB of logs
	logs, err := io.ReadAll(io.LimitReader(logsStream, int64(maxSize)))
	if err != nil {
		return result, fmt.Errorf("failed to read logs: %v", err)
	}

	if len(logs) == 0 {
		if previous {
			return result, fmt.Errorf("no previous logs found for container '%s' in pod '%s'", p.ContainerName, p.Name)
		}
		return result, fmt.Errorf("no logs found for container '%s' in pod '%s'", p.ContainerName, p.Name)
	}

	// Build the result
	options := []string{}
	if previous {
		options = append(options, "previous=true")
	}
	if tailLines > 0 {
		options = append(options, fmt.Sprintf("tail=%d", tailLines))
	}
	if since != nil {
		options = append(options, fmt.Sprintf("since=%s", since.String()))
	}

	result = fmt.Sprintf("Logs from container '%s' in pod '%s/%s'", p.ContainerName, p.Namespace, p.Name)
	if len(options) > 0 {
		result += fmt.Sprintf(" (%s)", strings.Join(options, ", "))
	}
	result += ":\n\n"
	result += string(logs)

	// Check if we reached the size limit
	if len(logs) == maxSize {
		result += "\n\n[Output truncated due to size limits. Use the 'tail' or 'since' parameters to view specific sections of logs.]"
	}

	return result, nil
}
