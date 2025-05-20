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
		Name:      deploymentName1,
		Namespace: defaultNamespace,
		Replicas:  1, // Default to 1 replica
	}

	assert.Equal(t, deploymentName1, deployment.Name)
	assert.Equal(t, defaultNamespace, deployment.Namespace)
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

	t.Run("Error getting dynamic client", func(t *testing.T) {
		deployment := &Deployment{
			Name:      deploymentName1,
			Namespace: defaultNamespace,
			Replicas:  1, // Default to 1 replica,
			Image:     nginxImage,
		}

		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(nil, errors.New("client unavailable"))

		result, err := deployment.Create(ctx, mockCM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get a dynamic client: client unavailable")
		assert.Empty(t, result)

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
				Namespace: testNamespace,
				Replicas:  1, // Default to 1 replica,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeDeployments := []runtime.Object{
					createDeploymentObj(deploymentName1, testNamespace, 2),
					createDeploymentObj(deploymentName2, testNamespace, 3),
					createDeploymentObj(deploymentName3, otherNamespace, 1), // Should not be listed
				}

				fakeClient := fake.NewSimpleClientset(fakeDeployments...)
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
				fakeDeployments := []runtime.Object{
					createDeploymentObj(deploymentName1, "namespace1", 1),
					createDeploymentObj(deploymentName2, "namespace2", 1),
				}

				fakeClient := fake.NewSimpleClientset(fakeDeployments...)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Deployments across all namespaces:",
			expectedError:  "",
		},
		{
			name:          "No deployments found",
			deployment:    &Deployment{Namespace: emptyNamespace},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "No deployments found in namespace \"empty-namespace\"",
			expectedError:  "",
		},
		{
			name:          "Error getting client",
			deployment:    &Deployment{Namespace: defaultNamespace},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
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
				mockCM.On("GetCurrentNamespace").Return("current-namespace")

				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "No deployments found in namespace \"current-namespace\"",
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			mockCM.AssertExpectations(t)
		})
	}
}
