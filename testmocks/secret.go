package testmocks

import (
	"context"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/mock"
)

// MockSecretFactory is a mock for SecretFactory.
type MockSecretFactory struct {
	mock.Mock
}

// NewMockSecretFactory creates a new MockSecretFactory.
func NewMockSecretFactory() *MockSecretFactory {
	return &MockSecretFactory{}
}

// NewSecret mocks the NewSecret method.
func (m *MockSecretFactory) NewSecret(params kai.SecretParams) kai.SecretOperator {
	args := m.Called(params)
	return args.Get(0).(kai.SecretOperator)
}

// MockSecret is a mock implementation of the SecretOperator interface.
type MockSecret struct {
	mock.Mock
	Params kai.SecretParams
}

// NewMockSecret creates a new MockSecret.
func NewMockSecret(params kai.SecretParams) *MockSecret {
	return &MockSecret{
		Params: params,
	}
}

// Create mocks the Create method.
func (m *MockSecret) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Get mocks the Get method.
func (m *MockSecret) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// List mocks the List method.
func (m *MockSecret) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	args := m.Called(ctx, cm, allNamespaces, labelSelector)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockSecret) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}

// Update mocks the Update method.
func (m *MockSecret) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	args := m.Called(ctx, cm)
	return args.String(0), args.Error(1)
}
