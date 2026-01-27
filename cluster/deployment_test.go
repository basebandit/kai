package cluster

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
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

// TestDeployment_Update tests the Update method
func TestDeployment_Update(t *testing.T) {
	ctx := context.Background()

	baseDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName1,
			Namespace: testNamespace,
			Labels: map[string]string{
				"app": deploymentName1,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { i := int32(1); return &i }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName1,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentName1,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  deploymentName1,
							Image: "nginx:1.19",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateUpdate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Update replicas",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Replicas:  3, // Increase replicas
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.NotNil(t, updated.Spec.Replicas)
				assert.Equal(t, int32(3), *updated.Spec.Replicas)
			},
		},
		{
			name: "Update image",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Image:     "nginx:1.20", // Update image version
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "nginx:1.20", updated.Spec.Template.Spec.Containers[0].Image)
			},
		},
		{
			name: "Update labels",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app":     deploymentName1,
					"version": "v2",
					"tier":    "frontend",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "frontend", updated.Labels["tier"])
				assert.Equal(t, "v2", updated.Labels["version"])
				assert.Equal(t, deploymentName1, updated.Labels["app"])

				// Check template labels were updated
				assert.Equal(t, "frontend", updated.Spec.Template.Labels["tier"])
				assert.Equal(t, "v2", updated.Spec.Template.Labels["version"])
			},
		},
		{
			name: "Update environment variables",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Env: map[string]interface{}{
					"ENV1": "updated-value", // Update existing
					"ENV2": "new-value",     // Add new
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)

				// Find ENV1 and ENV2 in the environment variables
				foundENV1 := false
				foundENV2 := false
				for _, env := range updated.Spec.Template.Spec.Containers[0].Env {
					switch env.Name {
					case "ENV1":
						assert.Equal(t, "updated-value", env.Value)
						foundENV1 = true
					case "ENV2":
						assert.Equal(t, "new-value", env.Value)
						foundENV2 = true
					}
				}
				assert.True(t, foundENV1, "ENV1 should be updated")
				assert.True(t, foundENV2, "ENV2 should be added")
			},
		},
		{
			name: "Update container port",
			deployment: &Deployment{
				Name:          deploymentName1,
				Namespace:     testNamespace,
				ContainerPort: "8080/TCP", // Update port
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)

				// Should have two ports now (original 80 and new 8080)
				foundPort := false
				for _, port := range updated.Spec.Template.Spec.Containers[0].Ports {
					if port.ContainerPort == 8080 {
						assert.Equal(t, corev1.ProtocolTCP, port.Protocol)
						foundPort = true
					}
				}
				assert.True(t, foundPort, "Port 8080 should be added")
			},
		},
		{
			name: "Update image pull policy",
			deployment: &Deployment{
				Name:            deploymentName1,
				Namespace:       testNamespace,
				ImagePullPolicy: "Always", // Update pull policy
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.PullAlways, updated.Spec.Template.Spec.Containers[0].ImagePullPolicy)
			},
		},
		{
			name: "Update image pull secrets",
			deployment: &Deployment{
				Name:             deploymentName1,
				Namespace:        testNamespace,
				ImagePullSecrets: []interface{}{"registry-secret"},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Len(t, updated.Spec.Template.Spec.ImagePullSecrets, 1)
				assert.Equal(t, "registry-secret", updated.Spec.Template.Spec.ImagePullSecrets[0].Name)
			},
		},
		{
			name: "Multiple updates at once",
			deployment: &Deployment{
				Name:            deploymentName1,
				Namespace:       testNamespace,
				Replicas:        5,
				Image:           "nginx:1.21",
				ImagePullPolicy: "Always",
				Labels: map[string]interface{}{
					"environment": "production",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(baseDeployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(testNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, int32(5), *updated.Spec.Replicas)
				assert.Equal(t, "nginx:1.21", updated.Spec.Template.Spec.Containers[0].Image)
				assert.Equal(t, corev1.PullAlways, updated.Spec.Template.Spec.Containers[0].ImagePullPolicy)
				assert.Equal(t, "production", updated.Labels["environment"])
			},
		},
		{
			name: "Deployment not found",
			deployment: &Deployment{
				Name:      "nonexistent-deployment",
				Namespace: testNamespace,
				Replicas:  3,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "",
			expectedError:  "failed to get deployment",
		},
		{
			name: "Error getting client",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Replicas:  3,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, errors.New("client unavailable"))
			},
			expectedResult: "",
			expectedError:  "error getting client: client unavailable",
		},
		{
			name: "Empty namespace uses current namespace",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: "", // Empty namespace
				Replicas:  3,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

				// Create the base deployment in the default namespace
				baseDeploymentCopy := baseDeployment.DeepCopy()
				baseDeploymentCopy.Namespace = defaultNamespace

				fakeClient := fake.NewSimpleClientset(baseDeploymentCopy)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			expectedError:  "",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				updated, err := client.AppsV1().Deployments(defaultNamespace).Get(ctx, deploymentName1, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, int32(3), *updated.Spec.Replicas)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			var fakeClient kubernetes.Interface

			tc.setupMock(mockCM)

			// Capture the fakeClient for validation after the update
			if tc.validateUpdate != nil {
				for _, call := range mockCM.ExpectedCalls {
					if call.Method == "GetCurrentClient" {
						if client, ok := call.ReturnArguments[0].(kubernetes.Interface); ok {
							fakeClient = client
						}
					}
				}
			}

			result, err := tc.deployment.Update(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				// Run custom validation if provided
				if tc.validateUpdate != nil && fakeClient != nil {
					tc.validateUpdate(t, fakeClient)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

// TestDeployment_Get tests the Get method
func TestDeployment_Get(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: replicas,
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing deployment",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 2)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: deploymentName1,
			expectedError:  "",
		},
		{
			name: "Deployment not found",
			deployment: &Deployment{
				Name:      "nonexistent-deployment",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "",
			expectedError:  "failed to get deployment",
		},
		{
			name: "Error getting client",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, errors.New("client unavailable"))
			},
			expectedResult: "",
			expectedError:  "error getting client: client unavailable",
		},
		{
			name: "Empty namespace uses current namespace",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: "", // Empty namespace
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

				deployment := createDeploymentObj(deploymentName1, defaultNamespace, 1)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: deploymentName1,
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.Get(ctx, mockCM)

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

// TestDeployment_Describe tests the Describe method
func TestDeployment_Describe(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
				CreationTimestamp: metav1.Now(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								ImagePullPolicy: corev1.PullAlways,
							},
						},
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{
						MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
						MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     replicas,
				AvailableReplicas: replicas,
				UpdatedReplicas:   replicas,
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentAvailable,
						Status:             corev1.ConditionTrue,
						LastUpdateTime:     metav1.Now(),
						LastTransitionTime: metav1.Now(),
						Reason:             "MinimumReplicasAvailable",
						Message:            "Deployment has minimum availability.",
					},
					{
						Type:               appsv1.DeploymentProgressing,
						Status:             corev1.ConditionTrue,
						LastUpdateTime:     metav1.Now(),
						LastTransitionTime: metav1.Now(),
						Reason:             "NewReplicaSetAvailable",
						Message:            fmt.Sprintf("ReplicaSet \"%s-679db4f448\" has successfully progressed.", name),
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Describe existing deployment",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 2)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: deploymentName1,
			expectedError:  "",
		},
		{
			name: "Deployment not found",
			deployment: &Deployment{
				Name:      "nonexistent-deployment",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "",
			expectedError:  "failed to get deployment",
		},
		{
			name: "Error getting client",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, errors.New("client unavailable"))
			},
			expectedResult: "",
			expectedError:  "error getting client: client unavailable",
		},
		{
			name: "Empty namespace uses current namespace",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: "", // Empty namespace
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

				deployment := createDeploymentObj(deploymentName1, defaultNamespace, 1)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: deploymentName1,
			expectedError:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.Describe(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				// Additional checks for detailed output
				if tc.name == "Describe existing deployment" {
					// Check if detailed sections exist
					assert.Contains(t, result, "Deployment: "+deploymentName1)
					assert.Contains(t, result, "Namespace: "+testNamespace)
					assert.Contains(t, result, "Replicas:")
					assert.Contains(t, result, "Conditions:")
					assert.Contains(t, result, "Strategy: RollingUpdate")
					assert.Contains(t, result, "Max Unavailable: 25%")
					assert.Contains(t, result, "Max Surge: 25%")
					assert.Contains(t, result, "Containers:")
					assert.Contains(t, result, "Image: "+nginxImage)
					assert.Contains(t, result, "Status Summary:")
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_Delete(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Delete existing deployment",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q deleted successfully from namespace %q", deploymentName1, testNamespace),
		},
		{
			name: "Deployment not found",
			deployment: &Deployment{
				Name:      "nonexistent",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to delete deployment",
		},
		{
			name: "Error getting client",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, errors.New("client unavailable"))
			},
			expectedError: "error getting client: client unavailable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.Delete(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_Scale(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Scale deployment to 5 replicas",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Replicas:  5,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q scaled to 5 replica(s) in namespace %q", deploymentName1, testNamespace),
		},
		{
			name: "Scale to 0 replicas",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
				Replicas:  0,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q scaled to 0 replica(s) in namespace %q", deploymentName1, testNamespace),
		},
		{
			name: "Deployment not found",
			deployment: &Deployment{
				Name:      "nonexistent",
				Namespace: testNamespace,
				Replicas:  3,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to get deployment",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.Scale(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutStatus(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name          string
		deployment    *Deployment
		setupMock     func(*testmocks.MockClusterManager)
		expectedError string
		checkResult   func(*testing.T, string)
	}{
		{
			name: "Get rollout status - complete",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				deployment.Status.Replicas = 3
				deployment.Status.UpdatedReplicas = 3
				deployment.Status.AvailableReplicas = 3
				deployment.Status.UnavailableReplicas = 0
				deployment.Status.ObservedGeneration = deployment.Generation
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			checkResult: func(t *testing.T, result string) {
				assert.Contains(t, result, "Deployment")
				assert.Contains(t, result, "rollout status")
				assert.Contains(t, result, "Replicas:")
				assert.Contains(t, result, "Rollout complete!")
			},
		},
		{
			name: "Rollout in progress",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				deployment.Status.Replicas = 3
				deployment.Status.UpdatedReplicas = 2
				deployment.Status.AvailableReplicas = 2
				deployment.Status.UnavailableReplicas = 1
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			checkResult: func(t *testing.T, result string) {
				assert.Contains(t, result, "Rollout in progress...")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutStatus(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutHistory(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name          string
		deployment    *Deployment
		setupMock     func(*testmocks.MockClusterManager)
		expectedError string
		checkResult   func(*testing.T, string)
	}{
		{
			name: "Get rollout history",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				rs := &appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      deploymentName1 + "-abc123",
						Namespace: testNamespace,
						Annotations: map[string]string{
							"deployment.kubernetes.io/revision": "1",
							"kubernetes.io/change-cause":        "Initial deployment",
						},
					},
				}
				fakeClient := fake.NewSimpleClientset(deployment, rs)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			checkResult: func(t *testing.T, result string) {
				assert.Contains(t, result, "Rollout history")
				assert.Contains(t, result, "REVISION")
				assert.Contains(t, result, "CHANGE-CAUSE")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutHistory(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutUndo(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		revision       int64
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Rollback to previous revision",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			revision: 0,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q rolled back to previous revision in namespace %q", deploymentName1, testNamespace),
		},
		{
			name: "Rollback to specific revision",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			revision: 2,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q rolled back to revision 2 in namespace %q", deploymentName1, testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutUndo(ctx, mockCM, tc.revision)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutRestart(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Restart deployment",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q restarted in namespace %q", deploymentName1, testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutRestart(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutPause(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Pause deployment rollout",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q paused in namespace %q", deploymentName1, testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutPause(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func TestDeployment_RolloutResume(t *testing.T) {
	ctx := context.Background()

	createDeploymentObj := func(name, namespace string, replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: nginxImage,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	testCases := []struct {
		name           string
		deployment     *Deployment
		setupMock      func(*testmocks.MockClusterManager)
		expectedError  string
		expectedResult string
	}{
		{
			name: "Resume paused deployment",
			deployment: &Deployment{
				Name:      deploymentName1,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				deployment := createDeploymentObj(deploymentName1, testNamespace, 3)
				deployment.Spec.Paused = true
				fakeClient := fake.NewSimpleClientset(deployment)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: fmt.Sprintf("Deployment %q resumed in namespace %q", deploymentName1, testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.deployment.RolloutResume(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			mockCM.AssertExpectations(t)
		})
	}
}
