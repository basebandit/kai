package testmocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// MockClusterManager implements the ClusterManager interface for testing
type MockClusterManager struct {
	mock.Mock
	currentNamespace string
}

// NewMockClusterManager initializes with defaults similar to the real implementation
func NewMockClusterManager() *MockClusterManager {
	m := &MockClusterManager{
		currentNamespace: "default",
	}
	return m
}

func (m *MockClusterManager) LoadKubeConfig(name, path string) error {
	args := m.Called(name, path)
	return args.Error(0)
}

func (m *MockClusterManager) GetClient(clusterName string) (kubernetes.Interface, error) {
	args := m.Called(clusterName)
	if client, ok := args.Get(0).(kubernetes.Interface); ok {
		return client, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClusterManager) GetDynamicClient(clusterName string) (dynamic.Interface, error) {
	args := m.Called(clusterName)
	if client, ok := args.Get(0).(dynamic.Interface); ok {
		return client, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClusterManager) GetCurrentClient() (kubernetes.Interface, error) {
	args := m.Called()
	if client, ok := args.Get(0).(kubernetes.Interface); ok {
		return client, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClusterManager) GetCurrentDynamicClient() (dynamic.Interface, error) {
	args := m.Called()
	if client, ok := args.Get(0).(dynamic.Interface); ok {
		return client, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClusterManager) SetCurrentNamespace(namespace string) {
	m.Called(namespace)
	// Additionally track the current namespace value in the mock
	// to mimic the real implementation's behavior
	if namespace == "" {
		m.currentNamespace = "default"
	} else {
		m.currentNamespace = namespace
	}
}

func (m *MockClusterManager) GetCurrentNamespace() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockClusterManager) ListClusters() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockClusterManager) SetCurrentContext(contextName string) error {
	args := m.Called(contextName)
	return args.Error(0)
}

func (m *MockClusterManager) GetCurrentContext() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockClusterManager) GetPod(ctx context.Context, name, namespace string) (string, error) {
	args := m.Called(ctx, name, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockClusterManager) ListPods(ctx context.Context, limit int64, namespace, labelSelector, fieldSelector string) (string, error) {
	args := m.Called(ctx, limit, namespace, labelSelector, fieldSelector)
	return args.String(0), args.Error(1)
}

func (m *MockClusterManager) DeletePod(ctx context.Context, name, namespace string, force bool) (string, error) {
	args := m.Called(ctx, name, namespace, force)
	return args.String(0), args.Error(1)
}

func (m *MockClusterManager) StreamPodLogs(ctx context.Context, tailLines int64, previous bool, since *time.Duration, podName, containerName, namespace string) (string, error) {
	args := m.Called(ctx, tailLines, previous, since, podName, containerName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockClusterManager) ListDeployments(ctx context.Context, allNamespaces bool, labelSelector, namespace string) (string, error) {
	args := m.Called(ctx, allNamespaces, labelSelector, namespace)
	return args.String(0), args.Error(1)
}
