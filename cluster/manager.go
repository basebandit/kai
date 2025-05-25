package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/basebandit/kai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Manager maintains connections to Kubernetes clusters
type Manager struct {
	kubeconfigs      map[string]string
	clients          map[string]kubernetes.Interface
	dynamicClients   map[string]dynamic.Interface
	contexts         map[string]*kai.ContextInfo
	currentContext   string
	currentNamespace string
}

// New creates a new cluster Manager
func New() *Manager {
	return &Manager{
		kubeconfigs:      make(map[string]string),
		clients:          make(map[string]kubernetes.Interface),
		dynamicClients:   make(map[string]dynamic.Interface),
		contexts:         make(map[string]*kai.ContextInfo),
		currentNamespace: "default",
	}
}

// LoadKubeConfig loads a kubeconfig file into the manager
func (cm *Manager) LoadKubeConfig(name, path string) error {
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

	if _, exists := cm.contexts[name]; exists {
		return fmt.Errorf("context %s already exists", name)
	}

	// Load all contexts from the kubeconfig file
	allContexts, currentContext, err := extractAllContextsInfo(resolvedPath, name)
	if err != nil {
		return err
	}

	clientset, dynamicClient, err := createClients(resolvedPath)
	if err != nil {
		return err
	}

	if err := testConnection(clientset); err != nil {
		return err
	}

	// Store all contexts from this kubeconfig
	for contextName, contextInfo := range allContexts {
		uniqueName := contextName
		if name != "" {
			uniqueName = fmt.Sprintf("%s-%s", name, contextName)
		}

		if _, exists := cm.contexts[uniqueName]; !exists {
			cm.kubeconfigs[uniqueName] = resolvedPath
			cm.clients[uniqueName] = clientset
			cm.dynamicClients[uniqueName] = dynamicClient
			cm.contexts[uniqueName] = contextInfo
			contextInfo.Name = uniqueName
		}
	}

	// Set the current context from the kubeconfig as active
	if currentContext != "" {
		currentUniqueName := currentContext
		if name != "" {
			currentUniqueName = fmt.Sprintf("%s-%s", name, currentContext)
		}

		if cm.currentContext == "" && cm.contexts[currentUniqueName] != nil {
			cm.currentContext = currentUniqueName
		}
	}

	return nil
}

// DeleteContext removes a context from the manager
func (cm *Manager) DeleteContext(name string) error {
	if _, exists := cm.contexts[name]; !exists {
		return fmt.Errorf("context %s not found", name)
	}

	if cm.currentContext == name {
		delete(cm.contexts, name)
		delete(cm.clients, name)
		delete(cm.dynamicClients, name)
		delete(cm.kubeconfigs, name)

		cm.currentContext = ""
		for contextName := range cm.contexts {
			cm.currentContext = contextName
			cm.contexts[contextName].IsActive = true
			break
		}
		return nil
	}

	delete(cm.contexts, name)
	delete(cm.clients, name)
	delete(cm.dynamicClients, name)
	delete(cm.kubeconfigs, name)

	return nil
}

// GetContextInfo returns detailed information about a specific context
func (cm *Manager) GetContextInfo(name string) (*kai.ContextInfo, error) {
	contextInfo, exists := cm.contexts[name]
	if !exists {
		return nil, fmt.Errorf("context %s not found", name)
	}

	contextCopy := *contextInfo
	return &contextCopy, nil
}

// RenameContext renames an existing context
func (cm *Manager) RenameContext(oldName, newName string) error {
	if oldName == newName {
		return errors.New("old and new context names cannot be the same")
	}

	contextInfo, exists := cm.contexts[oldName]
	if !exists {
		return fmt.Errorf("context %s not found", oldName)
	}

	if _, exists := cm.contexts[newName]; exists {
		return fmt.Errorf("context %s already exists", newName)
	}

	contextInfo.Name = newName
	cm.contexts[newName] = contextInfo
	cm.clients[newName] = cm.clients[oldName]
	cm.dynamicClients[newName] = cm.dynamicClients[oldName]
	cm.kubeconfigs[newName] = cm.kubeconfigs[oldName]

	delete(cm.contexts, oldName)
	delete(cm.clients, oldName)
	delete(cm.dynamicClients, oldName)
	delete(cm.kubeconfigs, oldName)

	if cm.currentContext == oldName {
		cm.currentContext = newName
	}

	return nil
}

// ListContexts returns all available contexts
func (cm *Manager) ListContexts() []*kai.ContextInfo {
	contexts := make([]*kai.ContextInfo, 0, len(cm.contexts))
	for _, contextInfo := range cm.contexts {
		contextCopy := *contextInfo
		contexts = append(contexts, &contextCopy)
	}
	return contexts
}

// GetClient returns the Kubernetes client for a specific cluster
func (cm *Manager) GetClient(clusterName string) (kubernetes.Interface, error) {
	client, exists := cm.clients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetDynamicClient returns the dynamic client for a specific cluster
func (cm *Manager) GetDynamicClient(clusterName string) (dynamic.Interface, error) {
	client, exists := cm.dynamicClients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}
	return client, nil
}

// GetCurrentClient returns the client for the current context
func (cm *Manager) GetCurrentClient() (kubernetes.Interface, error) {
	if len(cm.clients) == 0 {
		return nil, errors.New("no clusters configured - use the load_kubeconfig tool first")
	}

	if client, exists := cm.clients[cm.currentContext]; exists {
		return client, nil
	}

	for _, client := range cm.clients {
		return client, nil
	}

	return nil, errors.New("no clients available")
}

// GetCurrentDynamicClient returns the dynamic client for the current context
func (cm *Manager) GetCurrentDynamicClient() (dynamic.Interface, error) {
	if len(cm.dynamicClients) == 0 {
		return nil, errors.New("no clusters configured - use the load_kubeconfig tool first")
	}

	if client, exists := cm.dynamicClients[cm.currentContext]; exists {
		return client, nil
	}

	for _, client := range cm.dynamicClients {
		return client, nil
	}

	return nil, errors.New("no dynamic clients available")
}

// SetCurrentNamespace sets the current namespace
func (cm *Manager) SetCurrentNamespace(namespace string) {
	if namespace == "" {
		namespace = "default"
	}
	cm.currentNamespace = namespace
}

// GetCurrentNamespace returns the current namespace
func (cm *Manager) GetCurrentNamespace() string {
	return cm.currentNamespace
}

// ListClusters returns a list of all configured clusters
func (cm *Manager) ListClusters() []string {
	clusters := make([]string, 0, len(cm.clients))
	for name := range cm.clients {
		clusters = append(clusters, name)
	}
	return clusters
}

// SetCurrentContext sets the current context
func (cm *Manager) SetCurrentContext(contextName string) error {
	if _, exists := cm.clients[contextName]; !exists {
		return fmt.Errorf("cluster %s not found", contextName)
	}

	if cm.currentContext != "" {
		if currentInfo, exists := cm.contexts[cm.currentContext]; exists {
			currentInfo.IsActive = false
		}
	}

	cm.currentContext = contextName
	if contextInfo, exists := cm.contexts[contextName]; exists {
		contextInfo.IsActive = true
	}

	return nil
}

// GetCurrentContext returns the current context name
func (cm *Manager) GetCurrentContext() string {
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
	if path == "" {
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config"), nil
		} else {
			return "", errors.New("kubeconfig path not provided and home directory not found")
		}
	}
	return path, nil
}

// extractAllContextsInfo reads the kubeconfig file and extracts all context information
func extractAllContextsInfo(path, prefix string) (map[string]*kai.ContextInfo, string, error) {
	cleanPath := filepath.Clean(path)

	kubeconfigBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, "", fmt.Errorf("error reading kubeconfig file: %w", err)
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, "", fmt.Errorf("error creating client config: %w", err)
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, "", fmt.Errorf("error getting raw config: %w", err)
	}

	contexts := make(map[string]*kai.ContextInfo)

	for contextName, context := range rawConfig.Contexts {
		cluster, exists := rawConfig.Clusters[context.Cluster]
		if !exists {
			continue // Skip contexts with missing clusters
		}

		namespace := context.Namespace
		if namespace == "" {
			namespace = "default"
		}

		contexts[contextName] = &kai.ContextInfo{
			Name:       contextName,
			Cluster:    context.Cluster,
			User:       context.AuthInfo,
			Namespace:  namespace,
			ServerURL:  cluster.Server,
			ConfigPath: cleanPath,
			IsActive:   false,
		}
	}

	return contexts, rawConfig.CurrentContext, nil
}

// createClients creates Kubernetes clients from the kubeconfig file
func createClients(path string) (kubernetes.Interface, dynamic.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, nil, fmt.Errorf("error building config from flags: %w", err)
	}

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
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error resolving absolute path: %w", err)
	}

	fileInfo, err := os.Stat(absPath)
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
