package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockServiceFactory is a mock for ServiceFactory
type MockServiceFactory struct {
	mock.Mock
}

// NewMockServiceFactory creates a new MockServiceFactory
func NewMockServiceFactory() *MockServiceFactory {
	return &MockServiceFactory{}
}

// NewService mocks the NewService method
func (m *MockServiceFactory) NewService(params kai.ServiceParams) kai.ServiceOperator {
	args := m.Called(params)
	return args.Get(0).(kai.ServiceOperator)
}

// MockService is a mock implementation of the ServiceOperator interface
type MockService struct {
	mock.Mock
	Params kai.ServiceParams
}

// NewMockService creates a new MockService
func NewMockService(params kai.ServiceParams) *MockService {
	return &MockService{
		Params: params,
	}
}

// Create mocks the Create method
func (m *MockService) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method
func (m *MockService) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method
func (m *MockService) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method
func (m *MockService) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
