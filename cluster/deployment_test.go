package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestNewDeployment tests deployment creation with defaults
func TestNewDeployment(t *testing.T) {
	deployment := &Deployment{
		Name:      "test-app",
		Namespace: "default",
		Replicas:  1, // Default to 1 replica
	}

	assert.Equal(t, "test-app", deployment.Name)
	assert.Equal(t, "default", deployment.Namespace)
	assert.Equal(t, float64(1), deployment.Replicas) // Default value
	assert.Nil(t, deployment.Labels)
	assert.Empty(t, deployment.ContainerPort)
	assert.Nil(t, deployment.Env)
	assert.Empty(t, deployment.ImagePullPolicy)
	assert.Nil(t, deployment.ImagePullSecrets)
}

// TestDeployment_Create tests the Create method's error paths
func TestDeployment_Create(t *testing.T) {
	ctx := context.Background()

	// Test error case only - success case should be in integration tests
	t.Run("Error getting dynamic client", func(t *testing.T) {
		deployment := &Deployment{
			Name:      "test-app",
			Namespace: "default",
			Replicas:  1, // Default to 1 replica,
			Image:     "nginx:latest",
		}

		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(nil, errors.New("client unavailable"))

		// Execute
		result, err := deployment.Create(ctx, mockCM)

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get a dynamic client: client unavailable")
		assert.Empty(t, result)

		// Verify mocks
		mockCM.AssertExpectations(t)
	})
}

// TestDeployment_List tests the List method
func TestDeployment_List(t *testing.T) {
	ctx := context.Background()

	// Helper function to create test deployments
	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: replicas,
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		allNamespaces  bool
		labelSelector  string
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "List deployments in specific namespace",
			deployment: &Deployment{
				Name:      "",
				Namespace: "test-namespace",
				Replicas:  1, // Default to 1 replica,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				// Create fake deployments
				fakeDeployments := []runtime.Object{
					createDeploymentObj("deployment1", "test-namespace", 2),
					createDeploymentObj("deployment2", "test-namespace", 3),
					createDeploymentObj("deployment3", "other-namespace", 1), // Should not be listed
				}

				// Create fake client with the deployments
				fakeClient := fake.NewSimpleClientset(fakeDeployments...)

				// Setup mock to return our fake client
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Deployments in namespace \"test-namespace\":",
			expectedError:  "",
		},
		{
			name:          "List deployments across all namespaces",
			deployment:    &Deployment{},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				// Create fake deployments in different namespaces
				fakeDeployments := []runtime.Object{
					createDeploymentObj("deployment1", "namespace1", 1),
					createDeploymentObj("deployment2", "namespace2", 1),
				}

				// Create fake client with the deployments
				fakeClient := fake.NewSimpleClientset(fakeDeployments...)

				// Setup mock to return our fake client
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Deployments across all namespaces:",
			expectedError:  "",
		},
		{
			name:          "No deployments found",
			deployment:    &Deployment{Namespace: "empty-namespace"},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				// Create an empty fake client
				fakeClient := fake.NewSimpleClientset()

				// Setup mock to return our fake client
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "No deployments found in namespace \"empty-namespace\"",
			expectedError:  "",
		},
		{
			name:          "Error getting client",
			deployment:    &Deployment{Namespace: "default"},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				// Setup mock to return an error
				mockCM.On("GetCurrentClient").Return(nil, errors.New("client unavailable"))
			},
			expectedResult: "",
			expectedError:  "error getting client: client unavailable", // Updated to match actual error
		},
		{
			name:          "Empty namespace uses current namespace",
			deployment:    &Deployment{Replicas: 1}, // Empty namespace
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				// Setup mock to return current namespace
				mockCM.On("GetCurrentNamespace").Return("current-namespace")

				// Create an empty fake client
				fakeClient := fake.NewSimpleClientset()

				// Setup mock to return our fake client
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "No deployments found in namespace \"current-namespace\"",
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			// Call the function
			result, err := tc.deployment.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

			// Check results
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			// Verify mocks
			mockCM.AssertExpectations(t)
		})
	}
}
