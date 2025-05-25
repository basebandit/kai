package testmocks

import (
	"github.com/basebandit/kai"
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

func (m *MockClusterManager) DeleteContext(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockClusterManager) GetContextInfo(name string) (*kai.ContextInfo, error) {
	args := m.Called(name)
	if contextInfo, ok := args.Get(0).(*kai.ContextInfo); ok {
		return contextInfo, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockClusterManager) RenameContext(oldName, newName string) error {
	args := m.Called(oldName, newName)
	return args.Error(0)
}

func (m *MockClusterManager) ListContexts() []*kai.ContextInfo {
	args := m.Called()
	return args.Get(0).([]*kai.ContextInfo)
}
