package kai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

// ClusterManager maintains connections to Kubernetes clusters
type ClusterManager struct {
	kubeconfigs      map[string]string
	clients          map[string]kubernetes.Interface
	dynamicClients   map[string]dynamic.Interface
	currentContext   string
	currentNamespace string
}

// NewClusterManager creates a new ClusterManager
func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		kubeconfigs:      make(map[string]string),
		clients:          make(map[string]kubernetes.Interface),
		dynamicClients:   make(map[string]dynamic.Interface),
		currentNamespace: "default",
	}
}

// LoadKubeConfig loads a kubeconfig file into the manager
func (cm *ClusterManager) LoadKubeConfig(name, path string) error {
	if err := validateInputs(name, path); err != nil {
		return err
	}

	resolvedPath, err := resolvePath(path)
	if err != nil {
		return err
	}

	if err := validateFile(resolvedPath); err != nil {
		return err
	}

	contextName, err := extractContextName(resolvedPath)
	if err != nil {
		return err
	}
	cm.currentContext = contextName

	clientset, dynamicClient, err := createClients(resolvedPath)
	if err != nil {
		return err
	}

	if err := testConnection(clientset); err != nil {
		return err
	}

	// Store the configuration
	// If a context with the same name already exists, remove it first
	if _, exists := cm.clients[name]; exists {
		delete(cm.clients, name)
		delete(cm.dynamicClients, name)
		delete(cm.kubeconfigs, name)
	}

	cm.kubeconfigs[name] = path
	cm.clients[name] = clientset
	cm.dynamicClients[name] = dynamicClient

	return nil
}

// GetClient returns the Kubernetes client for a specific cluster
func (cm *ClusterManager) GetClient(clusterName string) (kubernetes.Interface, error) {
	client, exists := cm.clients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetDynamicClient returns the dynamic client for a specific cluster
func (cm *ClusterManager) GetDynamicClient(clusterName string) (dynamic.Interface, error) {
	client, exists := cm.dynamicClients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetCurrentClient returns the client for the current context
func (cm *ClusterManager) GetCurrentClient() (kubernetes.Interface, error) {
	if len(cm.clients) == 0 {
		return nil, errors.New("no clusters configured - use the load_kubeconfig tool first")
	}

	// First try to get the client for the current context
	if client, exists := cm.clients[cm.currentContext]; exists {
		return client, nil
	}

	// Fall back to the first client if the current context isn't found
	for _, client := range cm.clients {
		return client, nil
	}

	return nil, errors.New("no clients available")
}

// GetCurrentDynamicClient returns the dynamic client for the current context
func (cm *ClusterManager) GetCurrentDynamicClient() (dynamic.Interface, error) {
	if len(cm.dynamicClients) == 0 {
		return nil, errors.New("no clusters configured - use the load_kubeconfig tool first")
	}

	// First try to get the client for the current context
	if client, exists := cm.dynamicClients[cm.currentContext]; exists {
		return client, nil
	}

	// Fall back to the first client if the current context isn't found
	for _, client := range cm.dynamicClients {
		return client, nil
	}

	return nil, errors.New("no dynamic clients available")
}

// SetCurrentNamespace sets the current namespace
func (cm *ClusterManager) SetCurrentNamespace(namespace string) {
	if namespace == "" {
		namespace = "default"
	}
	cm.currentNamespace = namespace
}

// GetCurrentNamespace returns the current namespace
func (cm *ClusterManager) GetCurrentNamespace() string {
	return cm.currentNamespace
}

// ListClusters returns a list of all configured clusters
func (cm *ClusterManager) ListClusters() []string {
	clusters := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		clusters = append(clusters, name)
	}
	return clusters
}

// SetCurrentContext sets the current context
func (cm *ClusterManager) SetCurrentContext(contextName string) error {
	if _, exists := cm.clients[contextName]; !exists {
		return fmt.Errorf("cluster %s not found", contextName)
	}
	cm.currentContext = contextName
	return nil
}

// GetCurrentContext returns the current context name
func (cm *ClusterManager) GetCurrentContext() string {
	return cm.currentContext
}

func (cm *ClusterManager) GetPod(ctx context.Context, name, namespace string) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, err
	}

	// Verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace '%s' not found: %v", namespace, err)
	}

	// Use retry for potential transient issues
	var pod *corev1.Pod
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		// Only retry on transient errors
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		pod, getErr = client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("pod '%s' not found in namespace '%s'", name, namespace)
		}
		return result, fmt.Errorf("failed to get pod '%s' in namespace '%s': %v", name, namespace, err)
	}

	return formatPod(pod), nil
}

func (cm *ClusterManager) ListPods(ctx context.Context, limit int64, namespace, labelSelector, fieldSelector string) (string, error) {
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

	if namespace == "" {
		allNamespaces = true
	}

	if allNamespaces {
		pods, listErr = client.CoreV1().Pods("").List(timeoutCtx, listOptions)
		resultText = "Pods across all namespaces:\n"
	} else {
		// First verify the namespace exists
		_, err = client.CoreV1().Namespaces().Get(timeoutCtx, namespace, metav1.GetOptions{})
		if err != nil {
			return result, fmt.Errorf("namespace %q not found: %v", namespace, err)
		}

		pods, listErr = client.CoreV1().Pods(namespace).List(timeoutCtx, listOptions)
		resultText = fmt.Sprintf("Pods in namespace '%s':\n", namespace)
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

func (cm *ClusterManager) DeletePod(ctx context.Context, name, namespace string, force bool) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error: %v", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// verify namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %v", namespace, err)
	}

	// verify the pod exists
	_, err = client.CoreV1().Pods(namespace).Get(timeoutCtx, name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("pod %q not found in namespace %q", name, namespace)
	}

	deleteOptions := metav1.DeleteOptions{}
	if force {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	err = client.CoreV1().Pods(namespace).Delete(timeoutCtx, name, deleteOptions)
	if err != nil {
		return result, fmt.Errorf("failed to delete pod %q in namespace %q: %v", name, namespace, err)
	}

	return fmt.Sprintf("Successfully delete pod %q in namespace %q", name, namespace), nil
}

func (cm *ClusterManager) StreamPodLogs(ctx context.Context, tailLines int64, previous bool, since *time.Duration, podName, containerName, namespace string) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error: %v", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	//verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %v", namespace, err)
	}

	// Get the pod to find the container name if not specified and verify pod exists
	pod, err := client.CoreV1().Pods(namespace).Get(timeoutCtx, podName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("pod '%s' not found in namespace '%s'", podName, namespace)
		}
		return result, fmt.Errorf("failed to get pod '%s' in namespace '%s': %v", podName, namespace, err)
	}

	// Check if pod is running or has run before
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded && !previous {
		return result, fmt.Errorf("pod '%s' is in '%s' state. Logs may not be available. Use previous=true for crashed containers",
			podName, pod.Status.Phase)
	}

	if len(pod.Spec.Containers) == 0 {
		return result, fmt.Errorf("no containers found in pod '%s'", podName)
	}

	// Set default container if not specified
	if containerName == "" {
		containerName = pod.Spec.Containers[0].Name
	}

	// Verify the container exists in the pod
	containerExists := false
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
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
			containerName, podName, strings.Join(availableContainers, ", "))
	}

	// Configure log options
	logOptions := &corev1.PodLogOptions{
		Container: containerName,
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
		logsReq := client.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
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
			return result, fmt.Errorf("no previous logs found for container '%s' in pod '%s'", containerName, podName)
		}
		return result, fmt.Errorf("no logs found for container '%s' in pod '%s'", containerName, podName)
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

	result = fmt.Sprintf("Logs from container '%s' in pod '%s/%s'", containerName, namespace, podName)
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

func (cm *ClusterManager) ListDeployments(ctx context.Context, allNamespaces bool, labelSelector, namespace string) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error: %v", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if allNamespaces {
		deployments, err := client.AppsV1().Deployments("").List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list deployments: %v", err)
		}

		if len(deployments.Items) == 0 {
			result = "No deployments found across all namespaces"
			return result, nil
		}
		result = "Deployments across all namespaces:\n"
		result += formatDeploymentList(deployments)
	} else {
		deployments, err := client.AppsV1().Deployments(namespace).List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list deployments: %v", err)
		}

		if len(deployments.Items) == 0 {
			result = fmt.Sprintf("No deployments found in namespace %q.", namespace)
			return result, nil
		}

		result = fmt.Sprintf("Deployments in namespace %q:\n", namespace)
		result += formatDeploymentList(deployments)
	}

	return result, nil
}

func (cm *ClusterManager) CreateDeployment(ctx context.Context, deploymentParams DeploymentParams) (string, error) {

	var result string

	labels := map[string]interface{}{
		"app": deploymentParams.Name,
	}

	if deploymentParams.Labels != nil {
		for k, v := range deploymentParams.Labels {
			labels[k] = v
		}
	}
	// Container definition
	container := map[string]interface{}{
		"name":  deploymentParams.Name,
		"image": deploymentParams.Image,
	}

	parts := strings.Split(deploymentParams.ContainerPort, "/")
	var portVal int64
	if _, err := fmt.Sscanf(parts[0], "%d", &portVal); err == nil {
		portDefinition := map[string]interface{}{
			"containerPort": portVal,
		}

		// Add protocol if specified
		if len(parts) > 1 && (parts[1] == "TCP" || parts[1] == "UDP" || parts[1] == "SCTP") {
			portDefinition["protocol"] = parts[1]
		}

		container["ports"] = []interface{}{portDefinition}
	}

	if len(deploymentParams.Env) > 0 {
		envVars := make([]interface{}, 0, len(deploymentParams.Env))
		for k, v := range deploymentParams.Env {
			if strVal, ok := v.(string); ok {
				envVars = append(envVars, map[string]interface{}{
					"name":  k,
					"value": strVal,
				})
			}
		}
		if len(envVars) > 0 {
			container["env"] = envVars
		}
	}

	if deploymentParams.ImagePullPolicy != "" {
		validPolicies := map[string]bool{"Always": true, "IfNotPresent": true, "Never": true}
		if _, ok := validPolicies[deploymentParams.ImagePullPolicy]; ok {
			container["imagePullPolicy"] = deploymentParams.ImagePullPolicy
		}
	}

	podSpec := map[string]interface{}{
		"containers": []interface{}{container},
	}

	if len(deploymentParams.ImagePullSecrets) > 0 {
		pullSecrets := make([]interface{}, 0, len(deploymentParams.ImagePullSecrets))
		for _, v := range deploymentParams.ImagePullSecrets {
			if strVal, ok := v.(string); ok && strVal != "" {
				pullSecrets = append(pullSecrets, map[string]interface{}{
					"name": strVal,
				})
			}
		}
		if len(pullSecrets) > 0 {
			podSpec["imagePullSecrets"] = pullSecrets
		}
	}

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      deploymentParams.Name,
				"namespace": deploymentParams.Namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"replicas": deploymentParams.Replicas,
				"selector": map[string]interface{}{
					"matchLabels": labels,
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": labels,
					},
					"spec": podSpec,
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return result, fmt.Errorf("failed to get a dynamic client: %v", err)
	}

	_, err = client.Resource(gvr).Namespace(deploymentParams.Namespace).Create(timeoutCtx, deployment, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create deployment: %v", err)
	}

	result = fmt.Sprintf("Deployment %q created successfully in namespace %q with %f replica(s)", deploymentParams.Name, deploymentParams.Namespace, deploymentParams.Replicas)

	return result, nil
}

// validateInputs checks if the provided inputs are valid
func validateInputs(name, path string) error {
	if name == "" {
		return errors.New("cluster name cannot be empty")
	}
	return nil
}

// resolvePath resolves the kubeconfig path
func resolvePath(path string) (string, error) {
	// If path is empty, try to use the default kubeconfig path
	if path == "" {
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config"), nil
		} else {
			return "", errors.New("kubeconfig path not provided and home directory not found")
		}
	}
	return path, nil
}

// extractContextName reads the kubeconfig file and extracts the current context name
func extractContextName(path string) (string, error) {
	kubeconfigBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading kubeconfig file: %w", err)
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return "", fmt.Errorf("error creating client config: %w", err)
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return "", fmt.Errorf("error getting raw config: %w", err)
	}

	contextName := rawConfig.CurrentContext
	if contextName == "" {
		return "", errors.New("no current context found in kubeconfig file")
	}

	return contextName, nil
}

// createClients creates Kubernetes clients from the kubeconfig file
func createClients(path string) (kubernetes.Interface, dynamic.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, nil, fmt.Errorf("error building config from flags: %w", err)
	}

	// Increase timeouts for stability
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating dynamic client: %w", err)
	}

	return clientset, dynamicClient, nil
}

// testConnection tests the connection to the Kubernetes cluster
func testConnection(client kubernetes.Interface) error {
	_, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}

// validateFile checks if the file exists and is a regular file
func validateFile(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing file: %w", err)
	}
	if fileInfo.IsDir() {
		return errors.New("the provided path is a directory, not a file")
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

// DeploymentParams for creation of dynamic deployments
type DeploymentParams struct {
	Name,
	Image,
	Namespace,
	ContainerPort,
	ImagePullPolicy string
	ImagePullSecrets []interface{}
	Labels           map[string]interface{}
	Env              map[string]interface{}
	Replicas         float64
}
