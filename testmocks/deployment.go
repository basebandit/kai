package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockDeploymentFactory implements the tools.DeploymentFactory interface
type MockDeploymentFactory struct {
	mock.Mock
}

// NewMockDeploymentFactory creates a new MockDeploymentFactory
func NewMockDeploymentFactory() *MockDeploymentFactory {
	return &MockDeploymentFactory{}
}

// NewDeployment returns a mocked DeploymentOperator
func (m *MockDeploymentFactory) NewDeployment(params kai.DeploymentParams) kai.DeploymentOperator {
	args := m.Called(params)
	return args.Get(0).(kai.DeploymentOperator)
}

// MockDeployment implements the kai.DeploymentOperator interface
type MockDeployment struct {
	mock.Mock
	Name             string
	Namespace        string
	Image            string
	Replicas         float64
	Labels           map[string]interface{}
	ContainerPort    string
	Env              map[string]interface{}
	ImagePullPolicy  string
	ImagePullSecrets []interface{}
}

// NewMockDeployment creates a new MockDeployment
func NewMockDeployment(params kai.DeploymentParams) *MockDeployment {
	return &MockDeployment{
		Name:             params.Name,
		Image:            params.Image,
		Namespace:        params.Namespace,
		Replicas:         params.Replicas,
		Labels:           params.Labels,
		ContainerPort:    params.ContainerPort,
		Env:              params.Env,
		ImagePullPolicy:  params.ImagePullPolicy,
		ImagePullSecrets: params.ImagePullSecrets,
	}
}

// Create mocks the Create method
func (m *MockDeployment) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method
func (m *MockDeployment) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}
