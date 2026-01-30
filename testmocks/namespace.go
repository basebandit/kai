package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockNamespaceFactory is a mock for NamespaceFactory
type MockNamespaceFactory struct {
	mock.Mock
}

// NewMockNamespaceFactory creates a new MockNamespaceFactory
func NewMockNamespaceFactory() *MockNamespaceFactory {
	return &MockNamespaceFactory{}
}

// NewNamespace mocks the NewNamespace method
func (m *MockNamespaceFactory) NewNamespace(params kai.NamespaceParams) kai.NamespaceOperator {
	args := m.Called(params)
	return args.Get(0).(kai.NamespaceOperator)
}

// MockNamespace is a mock implementation of the NamespaceOperator interface
type MockNamespace struct {
	mock.Mock
	Params kai.NamespaceParams
}

// NewMockNamespace creates a new MockNamespace
func NewMockNamespace() *MockNamespace {
	return &MockNamespace{}
}

// Create mocks the Create method
func (m *MockNamespace) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method
func (m *MockNamespace) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method
func (m *MockNamespace) List(ctx context.Context, cm kai.ClusterManager, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method
func (m *MockNamespace) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method
func (m *MockNamespace) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// NamespaceFactory interface for testing
type NamespaceFactory interface {
	NewNamespace(params kai.NamespaceParams) kai.NamespaceOperator
}
