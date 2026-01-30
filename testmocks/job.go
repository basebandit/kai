package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockJobFactory is a mock for JobFactory.
type MockJobFactory struct {
	mock.Mock
}

// NewMockJobFactory creates a new MockJobFactory.
func NewMockJobFactory() *MockJobFactory {
	return &MockJobFactory{}
}

// NewJob mocks the NewJob method.
func (m *MockJobFactory) NewJob(params kai.JobParams) kai.JobOperator {
	args := m.Called(params)
	return args.Get(0).(kai.JobOperator)
}

// MockJob is a mock implementation of the JobOperator interface.
type MockJob struct {
	mock.Mock
	Params kai.JobParams
}

// NewMockJob creates a new MockJob.
func NewMockJob(params kai.JobParams) *MockJob {
	return &MockJob{
		Params: params,
	}
}

// Create mocks the Create method.
func (m *MockJob) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method.
func (m *MockJob) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method.
func (m *MockJob) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockJob) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method.
func (m *MockJob) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
