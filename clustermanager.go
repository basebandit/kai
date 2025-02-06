package kai

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ClusterManager maintains connections to Kubernetes Clusters
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
		currentNamespace: "default",
	}
}

// LoadKubeConfig loads a kubeconfig file into the manager. It will use the default kubeconfig path
// if none is given.
func (cm *ClusterManager) LoadKubeConfig(name, path string) error {

	if path == "" {
		if home := homedir.HomeDir(); home != "" {
			path = filepath.Join(home, ".kube", "config")
		} else {
			return errors.New("kubeconfig path not provided and home directory not found")
		}
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing kubeconfig file: %w", err)
	}
	if fileInfo.IsDir() {
		return errors.New("the provided path is a directory, not a file")
	}

	kubeconfig, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading kubeconfig file: %w", err)
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return fmt.Errorf("error creating client config: %w", err)
	}

	// get current context
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("error getting raw config: %w", err)
	}

	contextName := rawConfig.CurrentContext
	cm.currentContext = contextName

	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return fmt.Errorf("error building config from flags: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating dynamic client: %w", err)
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
	for _, client := range cm.clients {
		return client, nil
	}
	return nil, errors.New("no clusters configured")
}

// GetCurrentDynamicClient returns the dynamic client for the current context
func (cm *ClusterManager) GetCurrentDynamicClient() (dynamic.Interface, error) {
	for _, client := range cm.dynamicClients {
		return client, nil
	}
	return nil, errors.New("no clusters configured")
}

// SetCurrentNamespace sets the current namespace
func (cm *ClusterManager) SetCurrentNamespace(namespace string) {
	cm.currentNamespace = namespace
}

// GetCurrentNamespace returns the current namespace
func (cm *ClusterManager) GetCurrentNamespace() string {
	return cm.currentNamespace
}
