package kai

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

// LoadKubeConfig loads a kubeconfig file into the manager. It will
// try to use the default kubeconfig path, if the path argument is empty,
func (cm *ClusterManager) LoadKubeConfig(name, path string) error {
	if name == "" {
		return errors.New("cluster name cannot be empty")
	}

	if path == "" {
		if home := homedir.HomeDir(); home != "" {
			path = filepath.Join(home, ".kube", "config")
		} else {
			return errors.New("kubeconfig path not provided and home directory not found")
		}
	}

	// Check if the file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing kubeconfig file: %w", err)
	}
	if fileInfo.IsDir() {
		return errors.New("the provided path is a directory, not a file")
	}

	// Load the kubeconfig file content
	kubeconfigBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading kubeconfig file: %w", err)
	}

	// Create the clientconfig to get the current context
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return fmt.Errorf("error creating client config: %w", err)
	}

	// Get the current context name
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("error getting raw config: %w", err)
	}

	contextName := rawConfig.CurrentContext
	if contextName == "" {
		return errors.New("no current context found in kubeconfig file")
	}
	cm.currentContext = contextName

	// Create the rest of the config
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return fmt.Errorf("error building config from flags: %w", err)
	}

	// Increase timeouts for stability
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating dynamic client: %w", err)
	}

	// Test the connection first
	_, err = clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// If a context with the same name already exists, remove it first
	if _, exists := cm.clients[name]; exists {
		delete(cm.clients, name)
		delete(cm.dynamicClients, name)
		delete(cm.kubeconfigs, name)
	}

	// Store the kubeconfig and client
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
