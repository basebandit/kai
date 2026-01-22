package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockConfigMapFactory is a mock for ConfigMapFactory.
type MockConfigMapFactory struct {
	mock.Mock
}

// NewMockConfigMapFactory creates a new MockConfigMapFactory.
func NewMockConfigMapFactory() *MockConfigMapFactory {
	return &MockConfigMapFactory{}
}

// NewConfigMap mocks the NewConfigMap method.
func (m *MockConfigMapFactory) NewConfigMap(params kai.ConfigMapParams) kai.ConfigMapOperator {
	args := m.Called(params)
	return args.Get(0).(kai.ConfigMapOperator)
}

// MockConfigMap is a mock implementation of the ConfigMapOperator interface.
type MockConfigMap struct {
	mock.Mock
	Params kai.ConfigMapParams
}

// NewMockConfigMap creates a new MockConfigMap.
func NewMockConfigMap(params kai.ConfigMapParams) *MockConfigMap {
	return &MockConfigMap{
		Params: params,
	}
}

// Create mocks the Create method.
func (m *MockConfigMap) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method.
func (m *MockConfigMap) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method.
func (m *MockConfigMap) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockConfigMap) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method.
func (m *MockConfigMap) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
