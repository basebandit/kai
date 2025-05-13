package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
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

// GetResource gets any resource using dynamic client
func (cm *Cluster) GetResource(ctx context.Context, resourceType, name, namespace, group, version string) (*unstructured.Unstructured, error) {
	if name == "" {
		return nil, fmt.Errorf("resource name must be specified")
	}

	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	client, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return nil, err
	}

	gvr, err := cm.findGroupVersionResource(ctx, resourceType, group, version)
	if err != nil {
		return nil, err
	}

	namespaced, err := cm.isResourceNamespaced(ctx, gvr)
	if err != nil {
		return nil, err
	}

	var resource *unstructured.Unstructured
	if namespaced {
		resource, err = client.Resource(*gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		resource, err = client.Resource(*gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get %s %q:%v", resourceType, name, err)
	}

	return resource, nil
}

// ListResources lists resources based on the provided parameters
func (cm *Cluster) ListResources(ctx context.Context, limit int64, resourceType, namespace, labelSelector, fieldSelector, group, version string) (*unstructured.UnstructuredList, error) {
	client, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return nil, err
	}

	gvr, err := cm.findGroupVersionResource(ctx, resourceType, group, version)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	if limit > 0 {
		listOptions.Limit = limit
	}

	namespaced, err := cm.isResourceNamespaced(ctx, gvr)
	if err != nil {
		return nil, err
	}

	var list *unstructured.UnstructuredList
	if namespaced && namespace != "" {
		list, err = client.Resource(*gvr).Namespace(namespace).List(ctx, listOptions)
	} else {
		list, err = client.Resource(*gvr).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %v", resourceType, err)
	}

	return list, nil
}

// DeleteResource deletes any resource using dynamic client
func (cm *Cluster) DeleteResource(ctx context.Context, force bool, resourceType, name, namespace, group, version string) error {
	if name == "" {
		return fmt.Errorf("resource name must be specified")
	}

	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	client, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return err
	}

	gvr, err := cm.findGroupVersionResource(ctx, resourceType, group, version)
	if err != nil {
		return err
	}

	namespaced, err := cm.isResourceNamespaced(ctx, gvr)
	if err != nil {
		return err
	}

	deleteOptions := metav1.DeleteOptions{}

	if force {
		deleteOptions.GracePeriodSeconds = ptr(int64(0))
	}

	if namespaced {
		err = client.Resource(*gvr).Namespace(namespace).Delete(ctx, name, deleteOptions)
	} else {
		err = client.Resource(*gvr).Delete(ctx, name, deleteOptions)
	}

	return err
}

// findGroupVersionResource finds the GroupVersionResource for a resource
func (cm *Cluster) findGroupVersionResource(ctx context.Context, resource, group, version string) (*schema.GroupVersionResource, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return nil, err
	}

	discoveryClient := discovery.NewDiscoveryClient(client.CoreV1().RESTClient())

	// if group and version are specified, use them directly
	if group != "" && version != "" {
		return &schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resource,
		}, nil
	}

	// otherwise, discover the resource
	_, apiResourceLists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		// handle partial errors
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, fmt.Errorf("failed to get server resources: %v", err)
		}
	}

	for _, apiResourceList := range apiResourceLists {
		// extract group and version from GroupVersion string
		gv := strings.Split(apiResourceList.GroupVersion, "/")
		currentGroup := ""
		currentVersion := gv[0]

		if len(gv) > 1 {
			currentGroup = gv[0]
			currentVersion = gv[1]
		}

		// check if this group/version has our resource
		for _, apiResource := range apiResourceList.APIResources {
			// check if this is the resource we are looking for
			if apiResource.Name == resource ||
				apiResource.SingularName == resource ||
				contains(apiResource.ShortNames, resource) {
				return &schema.GroupVersionResource{
					Group:    currentGroup,
					Version:  currentVersion,
					Resource: apiResource.Name,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("resource %q not found", resource)
}

// isResourceNamespaced checks if a resource is namespaced
func (cm *Cluster) isResourceNamespaced(ctx context.Context, gvr *schema.GroupVersionResource) (bool, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return false, err
	}

	discoveryClient := discovery.NewDiscoveryClient(client.CoreV1().RESTClient())

	// Get API resources for this group/version
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(
		schema.GroupVersion{Group: gvr.Group, Version: gvr.Version}.String(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to get resources for %s/%s: %v:", gvr.Group, gvr.Version, err)
	}

	// Look for our resource
	for _, resource := range resourceList.APIResources {
		if resource.Name == gvr.Resource {
			return resource.Namespaced, nil
		}
	}

	return false, fmt.Errorf("resource %q not found in group %s/%s", gvr.Resource, gvr.Group, gvr.Version)
}

// Helper functions
// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
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
