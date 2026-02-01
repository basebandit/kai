package cluster

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/basebandit/kai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
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

// LoadInClusterConfig loads the in-cluster Kubernetes configuration
// This is used when kai is running inside a Kubernetes pod
func (cm *Manager) LoadInClusterConfig(name string) error {
	if name == "" {
		name = "in-cluster"
	}

	if _, exists := cm.contexts[name]; exists {
		return fmt.Errorf("context %s already exists", name)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating dynamic client: %w", err)
	}

	if err := testConnection(clientset); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Detect the namespace from the service account namespace file
	namespace := detectInClusterNamespace("")

	contextInfo := &kai.ContextInfo{
		Name:       name,
		Cluster:    "in-cluster",
		User:       "service-account",
		Namespace:  namespace,
		ServerURL:  config.Host,
		ConfigPath: "",
		IsActive:   true,
	}

	cm.kubeconfigs[name] = ""
	cm.clients[name] = clientset
	cm.dynamicClients[name] = dynamicClient
	cm.contexts[name] = contextInfo
	cm.currentContext = name

	slog.Info("in-cluster config loaded",
		slog.String("context", name),
		slog.String("server", config.Host),
		slog.String("namespace", namespace),
	)

	return nil
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
			cm.contexts[currentUniqueName].IsActive = true
		}
	}

	return nil
}

// DeleteContext removes a context from the manager
func (cm *Manager) DeleteContext(name string) error {
	if _, exists := cm.contexts[name]; !exists {
		slog.Debug("context not found for deletion", slog.String("context", name))
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
		slog.Info("context deleted", slog.String("context", name), slog.String("new_current", cm.currentContext))
		return nil
	}

	delete(cm.contexts, name)
	delete(cm.clients, name)
	delete(cm.dynamicClients, name)
	delete(cm.kubeconfigs, name)

	slog.Info("context deleted", slog.String("context", name))
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
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Name < contexts[j].Name
	})
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

// SetCurrentContext sets the current context and updates the kubeconfig file
func (cm *Manager) SetCurrentContext(contextName string) error {
	if _, exists := cm.clients[contextName]; !exists {
		slog.Debug("context not found", slog.String("context", contextName))
		return fmt.Errorf("cluster %s not found", contextName)
	}

	previousContext := cm.currentContext
	if cm.currentContext != "" {
		if currentInfo, exists := cm.contexts[cm.currentContext]; exists {
			currentInfo.IsActive = false
		}
	}

	cm.currentContext = contextName
	if contextInfo, exists := cm.contexts[contextName]; exists {
		contextInfo.IsActive = true

		// Update the kubeconfig file to reflect the context switch
		if err := cm.updateKubeconfigCurrentContext(contextName, contextInfo.ConfigPath); err != nil {
			return fmt.Errorf("failed to update kubeconfig file: %w", err)
		}
	}

	slog.Info("context switched",
		slog.String("from", previousContext),
		slog.String("to", contextName),
	)
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

		contexts[contextName] = &kai.ContextInfo{
			Name:       contextName,
			Cluster:    context.Cluster,
			User:       context.AuthInfo,
			Namespace:  context.Namespace,
			ServerURL:  cluster.Server,
			ConfigPath: cleanPath,
			IsActive:   false,
		}
	}

	return contexts, rawConfig.CurrentContext, nil
}

// updateKubeconfigCurrentContext updates the current-context in the kubeconfig file
func (cm *Manager) updateKubeconfigCurrentContext(contextName, configPath string) error {
	// #nosec G304
	kubeconfigBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading kubeconfig file: %w", err)
	}

	config, err := clientcmd.Load([]byte(kubeconfigBytes))
	if err != nil {
		return fmt.Errorf("error parsing kubeconfig: %w", err)
	}

	// Find the original context name in the kubeconfig
	// (since we might have prefixed it in our manager)
	originalContextName := ""
	for name := range config.Contexts {
		if strings.HasSuffix(contextName, name) || contextName == name {
			originalContextName = name
			break
		}
	}

	if originalContextName == "" {
		return fmt.Errorf("context %s not found in kubeconfig", contextName)
	}

	config.CurrentContext = originalContextName

	if err := clientcmd.WriteToFile(*config, configPath); err != nil {
		return fmt.Errorf("error writing kubeconfig file: %w", err)
	}

	return nil
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

// detectInClusterNamespace reads the namespace from the service account namespace file
// when running inside a Kubernetes pod. Falls back to "default" if the file cannot be read.
// If customPath is provided and not empty, it will be used instead of the default Kubernetes path.
func detectInClusterNamespace(customPath string) string {
	namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if customPath != "" {
		namespaceFile = customPath
	}
	
	// #nosec G304 - This is a well-known Kubernetes service account file path
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		slog.Debug("failed to read namespace from service account file, using default",
			slog.String("file", namespaceFile),
			slog.String("error", err.Error()),
		)
		return "default"
	}
	
	namespace := strings.TrimSpace(string(data))
	if namespace == "" {
		slog.Debug("namespace file is empty, using default",
			slog.String("file", namespaceFile),
		)
		return "default"
	}
	
	return namespace
}

func ptr[T any](v T) *T {
	return &v
}

// PortForwardSession represents an active port forwarding session
type PortForwardSession struct {
	ID         string
	Namespace  string
	Target     string
	TargetType string
	LocalPort  int
	RemotePort int
	PodName    string
	stopChan   chan struct{}
}

// portForwardSessions tracks active port forward sessions
var (
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex             sync.RWMutex
	pfCounter           int
)

// StartPortForward initiates a port forwarding session
func (cm *Manager) StartPortForward(
	ctx context.Context,
	namespace string,
	targetType string,
	targetName string,
	localPort int,
	remotePort int,
) (*PortForwardSession, error) {
	currentContext := cm.GetCurrentContext()
	kubeconfigPath, exists := cm.kubeconfigs[currentContext]
	if !exists {
		return nil, fmt.Errorf("kubeconfig path not found for context %s", currentContext)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	podName := targetName
	if targetType == "service" {
		svc, err := client.CoreV1().Services(namespace).Get(ctx, targetName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("service %q not found: %w", targetName, err)
		}

		if len(svc.Spec.Selector) == 0 {
			return nil, fmt.Errorf("service %q has no selector", targetName)
		}

		var labelParts []string
		for k, v := range svc.Spec.Selector {
			labelParts = append(labelParts, fmt.Sprintf("%s=%s", k, v))
		}

		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: strings.Join(labelParts, ","),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods: %w", err)
		}

		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("no pods found for service %q", targetName)
		}

		found := false
		for _, pod := range pods.Items {
			if pod.Status.Phase == "Running" {
				podName = pod.Name
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no running pods found for service %q", targetName)
		}
	}

	_, err = client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("pod %q not found in namespace %q: %w", podName, namespace, err)
	}

	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward",
		config.Host, namespace, podName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)

	pfMutex.Lock()
	pfCounter++
	sessionID := fmt.Sprintf("pf-%d", pfCounter)
	pfMutex.Unlock()

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	session := &PortForwardSession{
		ID:         sessionID,
		Namespace:  namespace,
		Target:     targetName,
		TargetType: targetType,
		LocalPort:  localPort,
		RemotePort: remotePort,
		PodName:    podName,
		stopChan:   stopChan,
	}

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			pfMutex.Lock()
			delete(portForwardSessions, sessionID)
			pfMutex.Unlock()
		}
	}()

	select {
	case <-readyChan:
	case <-ctx.Done():
		close(stopChan)
		return nil, ctx.Err()
	}

	forwardedPorts, err := fw.GetPorts()
	if err == nil && len(forwardedPorts) > 0 {
		session.LocalPort = int(forwardedPorts[0].Local)
	}

	pfMutex.Lock()
	portForwardSessions[sessionID] = session
	pfMutex.Unlock()

	slog.Info("port forward started",
		slog.String("session_id", sessionID),
		slog.String("namespace", namespace),
		slog.String("target_type", targetType),
		slog.String("target", targetName),
		slog.String("pod", podName),
		slog.Int("local_port", session.LocalPort),
		slog.Int("remote_port", remotePort),
	)

	return session, nil
}

// StopPortForward stops a port forwarding session
func (cm *Manager) StopPortForward(sessionID string) error {
	pfMutex.Lock()
	defer pfMutex.Unlock()

	session, exists := portForwardSessions[sessionID]
	if !exists {
		slog.Debug("port forward session not found", slog.String("session_id", sessionID))
		return fmt.Errorf("port forward session %q not found", sessionID)
	}

	close(session.stopChan)
	delete(portForwardSessions, sessionID)

	slog.Info("port forward stopped",
		slog.String("session_id", sessionID),
		slog.String("target", session.Target),
	)

	return nil
}

// ListPortForwards returns all active port forwarding sessions
func (cm *Manager) ListPortForwards() []*PortForwardSession {
	pfMutex.RLock()
	defer pfMutex.RUnlock()

	sessions := make([]*PortForwardSession, 0, len(portForwardSessions))
	for _, session := range portForwardSessions {
		sessions = append(sessions, session)
	}
	return sessions
}
