package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockCronJobFactory is a mock for CronJobFactory.
type MockCronJobFactory struct {
	mock.Mock
}

// NewMockCronJobFactory creates a new MockCronJobFactory.
func NewMockCronJobFactory() *MockCronJobFactory {
	return &MockCronJobFactory{}
}

// NewCronJob mocks the NewCronJob method.
func (m *MockCronJobFactory) NewCronJob(params kai.CronJobParams) kai.CronJobOperator {
	args := m.Called(params)
	return args.Get(0).(kai.CronJobOperator)
}

// MockCronJob is a mock implementation of the CronJobOperator interface.
type MockCronJob struct {
	mock.Mock
	Params kai.CronJobParams
}

// NewMockCronJob creates a new MockCronJob.
func NewMockCronJob(params kai.CronJobParams) *MockCronJob {
	return &MockCronJob{
		Params: params,
	}
}

// Create mocks the Create method.
func (m *MockCronJob) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method.
func (m *MockCronJob) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method.
func (m *MockCronJob) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockCronJob) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
