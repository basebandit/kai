package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Cluster maintains connections to Kubernetes clusters
type Cluster struct {
	kubeconfigs      map[string]string
	clients          map[string]kubernetes.Interface
	dynamicClients   map[string]dynamic.Interface
	currentContext   string
	currentNamespace string
}

// New creates a new ClusterManager
func New() *Cluster {
	return &Cluster{
		kubeconfigs:      make(map[string]string),
		clients:          make(map[string]kubernetes.Interface),
		dynamicClients:   make(map[string]dynamic.Interface),
		currentNamespace: "default",
	}
}

// LoadKubeConfig loads a kubeconfig file into the manager
func (cm *Cluster) LoadKubeConfig(name, path string) error {
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
func (cm *Cluster) GetClient(clusterName string) (kubernetes.Interface, error) {
	client, exists := cm.clients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetDynamicClient returns the dynamic client for a specific cluster
func (cm *Cluster) GetDynamicClient(clusterName string) (dynamic.Interface, error) {
	client, exists := cm.dynamicClients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetCurrentClient returns the client for the current context
func (cm *Cluster) GetCurrentClient() (kubernetes.Interface, error) {
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
func (cm *Cluster) GetCurrentDynamicClient() (dynamic.Interface, error) {
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
func (cm *Cluster) SetCurrentNamespace(namespace string) {
	if namespace == "" {
		namespace = "default"
	}
	cm.currentNamespace = namespace
}

// GetCurrentNamespace returns the current namespace
func (cm *Cluster) GetCurrentNamespace() string {
	return cm.currentNamespace
}

// ListClusters returns a list of all configured clusters
func (cm *Cluster) ListClusters() []string {
	clusters := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		clusters = append(clusters, name)
	}
	return clusters
}

// SetCurrentContext sets the current context
func (cm *Cluster) SetCurrentContext(contextName string) error {
	if _, exists := cm.clients[contextName]; !exists {
		return fmt.Errorf("cluster %s not found", contextName)
	}
	cm.currentContext = contextName
	return nil
}

// GetCurrentContext returns the current context name
func (cm *Cluster) GetCurrentContext() string {
	return cm.currentContext
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
