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
func (m *MockPodFactory) NewPod(params kai.PodParams) kai.PodOperator {
	args := m.Called(params)
	return args.Get(0).(kai.PodOperator)
}

// MockPod implements the PodOperator interface for testing
type MockPod struct {
	mock.Mock
	Name               string
	Namespace          string
	Image              string
	Command            []interface{}
	Args               []interface{}
	Labels             map[string]interface{}
	ContainerName      string
	ContainerPort      string
	Env                map[string]interface{}
	ImagePullPolicy    string
	ImagePullSecrets   []interface{}
	RestartPolicy      string
	NodeSelector       map[string]interface{}
	ServiceAccountName string
	Volumes            []interface{}
	VolumeMounts       []interface{}
}

// NewMockPod creates a new MockPod instance
func NewMockPod(params kai.PodParams) *MockPod {
	return &MockPod{
		Name:               params.Name,
		Namespace:          params.Namespace,
		Image:              params.Image,
		Command:            params.Command,
		Args:               params.Args,
		Labels:             params.Labels,
		ContainerName:      params.ContainerName,
		ContainerPort:      params.ContainerPort,
		Env:                params.Env,
		ImagePullPolicy:    params.ImagePullPolicy,
		ImagePullSecrets:   params.ImagePullSecrets,
		RestartPolicy:      params.RestartPolicy,
		NodeSelector:       params.NodeSelector,
		ServiceAccountName: params.ServiceAccountName,
		Volumes:            params.Volumes,
		VolumeMounts:       params.VolumeMounts,
	}
}

// Create mocks the Create method
func (m *MockPod) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method
func (m *MockPod) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method
func (m *MockPod) List(ctx context.Context, cm kai.ClusterManager, limit int64, labelSelector, fieldSelector string) (string, error) {
	args := m.Called(ctx, cm, limit, labelSelector, fieldSelector)
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
