package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockIngressFactory is a mock for IngressFactory.
type MockIngressFactory struct {
	mock.Mock
}

// NewMockIngressFactory creates a new MockIngressFactory.
func NewMockIngressFactory() *MockIngressFactory {
	return &MockIngressFactory{}
}

// NewIngress mocks the NewIngress method.
func (m *MockIngressFactory) NewIngress(params kai.IngressParams) kai.IngressOperator {
	args := m.Called(params)
	return args.Get(0).(kai.IngressOperator)
}

// MockIngress is a mock implementation of the IngressOperator interface.
type MockIngress struct {
	mock.Mock
	Params kai.IngressParams
}

// NewMockIngress creates a new MockIngress.
func NewMockIngress(params kai.IngressParams) *MockIngress {
	return &MockIngress{
		Params: params,
	}
}

// Create mocks the Create method.
func (m *MockIngress) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method.
func (m *MockIngress) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method.
func (m *MockIngress) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method.
func (m *MockIngress) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockIngress) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
