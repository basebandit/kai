package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockDeployment is a mock implementation of the DeploymentOperator interface
type MockDeployment struct {
	mock.Mock
	Params kai.DeploymentParams
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

// Get mocks the Get method
func (m *MockDeployment) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method
func (m *MockDeployment) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// NewMockDeployment creates a new MockDeployment
func NewMockDeployment(params kai.DeploymentParams) *MockDeployment {
	return &MockDeployment{
		Params: params,
	}
}

// MockDeploymentFactory is a mock for DeploymentFactory
type MockDeploymentFactory struct {
	mock.Mock
}

// NewMockDeploymentFactory creates a new MockDeploymentFactory
func NewMockDeploymentFactory() *MockDeploymentFactory {
	return &MockDeploymentFactory{}
}

// NewDeployment mocks the NewDeployment method
func (m *MockDeploymentFactory) NewDeployment(params kai.DeploymentParams) kai.DeploymentOperator {
	args := m.Called(params)
	return args.Get(0).(kai.DeploymentOperator)
}
