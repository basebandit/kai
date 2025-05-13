package testmocks

import (
	"context"
	"time"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockPodFactory implements the PodFactory interface for testing
type MockPodFactory struct {
	mock.Mock
}

// NewPod returns a mocked PodOperator
func (m *MockPodFactory) NewPod(name, namespace, containerName, labelSelector, fieldSelector string) kai.PodOperator {
	args := m.Called(name, namespace, containerName, labelSelector, fieldSelector)
	return args.Get(0).(kai.PodOperator)
}

// MockPod implements the PodOperator interface for testing
type MockPod struct {
	mock.Mock
}

// NewMockPod creates a new MockPod instance
func NewMockPod() *MockPod {
	return &MockPod{}
}

// Get mocks the Get method
func (m *MockPod) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method
func (m *MockPod) List(ctx context.Context, cm kai.ClusterManager, limit int64) (string, error) {
	args := m.Called(ctx, cm, limit)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method
func (m *MockPod) Delete(ctx context.Context, cm kai.ClusterManager, force bool) (string, error) {
	args := m.Called(ctx, cm, force)
	return args.String(0), args.Error(1)
}

// StreamLogs mocks the StreamLogs method
func (m *MockPod) StreamLogs(ctx context.Context, cm kai.ClusterManager, tailLines int64, previous bool, since *time.Duration) (string, error) {
	args := m.Called(ctx, cm, tailLines, previous, since)
	return args.String(0), args.Error(1)
}
