package tools

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type deploymentTestCase struct {
	name                     string
	args                     map[string]interface{}
	expectedParams           kai.DeploymentParams
	mockSetup                func(*testmocks.MockClusterManager, *testmocks.MockDeploymentFactory, *testmocks.MockDeployment)
	expectedOutput           string
	expectDeploymentCreation bool
}

// TestCreateDeploymentHandler tests the createDeploymentHandler function
func TestCreateDeploymentHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Create basic deployment",
			args: map[string]interface{}{
				"name":  "nginx-deployment",
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{
				Name:      "nginx-deployment",
				Namespace: defaultNamespace,
				Image:     nginxImage,
				Replicas:  1, // Default to 1 replica
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q created successfully in namespace %q with %g replica(s)", "nginx-deployment", defaultNamespace, float64(1)), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q created successfully", "nginx-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "Create deployment with custom replicas",
			args: map[string]interface{}{
				"name":     "app-deployment",
				"image":    myAppImage,
				"replicas": float64(3),
			},
			expectedParams: kai.DeploymentParams{
				Name:      "app-deployment",
				Namespace: defaultNamespace,
				Image:     myAppImage,
				Replicas:  3,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q created successfully in namespace %q with %g replica(s)", "app-deployment", defaultNamespace, float64(3)), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q created successfully", "app-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "Create deployment with all parameters",
			args: map[string]interface{}{
				"name":              "custom-deployment",
				"image":             nginxImage,
				"namespace":         testNamespace,
				"replicas":          float64(2),
				"container_port":    defaultContainerPort,
				"image_pull_policy": alwaysImagePullPolicy,
				"image_pull_secrets": []interface{}{
					registrySecretName,
				},
				"labels": map[string]interface{}{
					"app":     "custom",
					"version": "v1",
				},
				"env": map[string]interface{}{
					"DEBUG": "true",
					"ENV":   "dev",
				},
			},
			expectedParams: kai.DeploymentParams{
				Name:            "custom-deployment",
				Image:           nginxImage,
				Namespace:       testNamespace,
				Replicas:        2,
				ContainerPort:   defaultContainerPort,
				ImagePullPolicy: alwaysImagePullPolicy,
				ImagePullSecrets: []interface{}{
					registrySecretName,
				},
				Labels: map[string]interface{}{
					"app":     "custom",
					"version": "v1",
				},
				Env: map[string]interface{}{
					"DEBUG": "true",
					"ENV":   "dev",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q created successfully in namespace %q with %g replica(s)", "custom-deployment", testNamespace, float64(2)), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q created successfully", "custom-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "Missing name",
			args: map[string]interface{}{
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "Empty name",
			args: map[string]interface{}{
				"name":  "",
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Missing image",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           errMissingImage,
			expectDeploymentCreation: false,
		},
		{
			name: "Empty image",
			args: map[string]interface{}{
				"name":  "test-deployment",
				"image": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           errEmptyImage,
			expectDeploymentCreation: false,
		},
		{
			name: "Invalid container port",
			args: map[string]interface{}{
				"name":           "test-deployment",
				"image":          nginxImage,
				"container_port": "invalid-port",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed. It will fail early due to invalid port error
			},
			expectedOutput:           "Port must be a number",
			expectDeploymentCreation: false,
		},
		{
			name: "Creation error",
			args: map[string]interface{}{
				"name":  "error-deployment",
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{
				Name:      "error-deployment",
				Namespace: defaultNamespace,
				Image:     nginxImage,
				Replicas:  1,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return("", errors.New(errQuotaExceeded))
			},
			expectedOutput:           errQuotaExceeded,
			expectDeploymentCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := createDeploymentHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}

func TestDescribeDeploymentHandler(t *testing.T) {
	deploymentName := "test-deployment"

	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": deploymentName,
			},
			expectedParams: kai.DeploymentParams{
				Name:      deploymentName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Describe", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment: %s\nNamespace: %s\nReplicas: 3/3 (available/total)\nStrategy: RollingUpdate\nContainers:\n1. %s (Image: nginx:latest)",
						deploymentName, defaultNamespace, deploymentName), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment: %s", deploymentName),
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Required parameter 'name' is missing",
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Parameter 'name' must be a non-empty string",
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "nonexistent-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "nonexistent-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Describe", mock.Anything, mockCM).
					Return("", fmt.Errorf("deployment %q not found in namespace %q", "nonexistent-deployment", defaultNamespace))
			},
			expectedOutput:           fmt.Sprintf("deployment %q not found in namespace %q", "nonexistent-deployment", defaultNamespace),
			expectDeploymentCreation: true,
		},
		{
			name: "SpecifyNamespace",
			args: map[string]interface{}{
				"name":      deploymentName,
				"namespace": testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Name:      deploymentName,
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Describe", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment: %s\nNamespace: %s\nReplicas: 2/2 (available/total)", deploymentName, testNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment: %s\nNamespace: %s", deploymentName, testNamespace),
			expectDeploymentCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := describeDeploymentHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateDeploymentHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "UpdateImage",
			args: map[string]interface{}{
				"name":  "test-deployment",
				"image": "nginx:1.20",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
				Image:     "nginx:1.20",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q updated successfully in namespace %q", "test-deployment", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "UpdateReplicas",
			args: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": float64(5),
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
				Replicas:  float64(5),
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q updated successfully in namespace %q with 5 replica(s)", "test-deployment", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "UpdateMultipleFields",
			args: map[string]interface{}{
				"name":              "test-deployment",
				"image":             "nginx:1.20",
				"replicas":          float64(3),
				"container_port":    "8080/TCP",
				"env":               map[string]interface{}{"DEBUG": "true"},
				"image_pull_policy": "Always",
			},
			expectedParams: kai.DeploymentParams{
				Name:            "test-deployment",
				Namespace:       defaultNamespace,
				Image:           "nginx:1.20",
				Replicas:        float64(3),
				ContainerPort:   "8080/TCP",
				Env:             map[string]interface{}{"DEBUG": "true"},
				ImagePullPolicy: "Always",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q updated successfully in namespace %q with 3 replica(s)", "test-deployment", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "MissingName",
			args: map[string]interface{}{
				"image": "nginx:1.20",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Required parameter 'name' is missing",
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name":  "",
				"image": "nginx:1.20",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Parameter 'name' must be a non-empty string",
			expectDeploymentCreation: false,
		},
		{
			name: "NoFieldsToUpdate",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:           "At least one field to update must be specified",
			expectDeploymentCreation: false,
		},
		{
			name: "InvalidContainerPort",
			args: map[string]interface{}{
				"name":           "test-deployment",
				"container_port": "invalid",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:           "Port must be a number",
			expectDeploymentCreation: false, // the handler will return before creating deployment
		},
		{
			name: "UpdateError",
			args: map[string]interface{}{
				"name":  "error-deployment",
				"image": "nginx:1.20",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "error-deployment",
				Namespace: defaultNamespace,
				Image:     "nginx:1.20",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Update", mock.Anything, mockCM).
					Return("", errors.New("failed to update deployment: deployment not found"))
			},
			expectedOutput:           "failed to update deployment: deployment not found",
			expectDeploymentCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := updateDeploymentHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}

// TestListDeploymentsHandler tests the listDeploymentsHandler function
func TestListDeploymentsHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "List in current namespace",
			args: map[string]interface{}{
				"all_namespaces": false,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Deployments in namespace %q:\n• test-deployment-1: 1/1 replicas ready\n• test-deployment-2: 2/2 replicas ready", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployments in namespace %q", defaultNamespace),
			expectDeploymentCreation: true,
		},
		{
			name: "List in specific namespace",
			args: map[string]interface{}{
				"all_namespaces": false,
				"namespace":      testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Deployments in namespace %q:\n• test-deployment-1: 1/1 replicas ready", testNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployments in namespace %q", testNamespace),
			expectDeploymentCreation: true,
		},
		{
			name: "List across all namespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: "", // This should be ignored because all_namespaces is true
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockDeployment.On("List", mock.Anything, mockCM, true, "").
					Return("Deployments across all namespaces:\n• default/test-deployment-1: 1/1 replicas ready\n• test-namespace/test-deployment-2: 2/2 replicas ready", nil)
			},
			expectedOutput:           "Deployments across all namespaces",
			expectDeploymentCreation: true,
		},
		{
			name: "List with label selector",
			args: map[string]interface{}{
				"all_namespaces": false,
				"label_selector": "app=nginx",
			},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "app=nginx").
					Return(fmt.Sprintf("Deployments in namespace %q with label selector 'app=nginx':\n• nginx-deployment: 3/3 replicas ready", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployments in namespace %q with label selector", defaultNamespace),
			expectDeploymentCreation: true,
		},
		{
			name: "No deployments found",
			args: map[string]interface{}{
				"all_namespaces": false,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("No deployments found in namespace %q", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("No deployments found in namespace %q", defaultNamespace),
			expectDeploymentCreation: true,
		},
		{
			name: "Error listing deployments",
			args: map[string]interface{}{
				"all_namespaces": false,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return("", errors.New("failed to list deployments: unauthorized"))
			},
			expectedOutput:           "failed to list deployments: unauthorized",
			expectDeploymentCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := listDeploymentsHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}

// TestGetDeploymentHandler tests the getDeploymentHandler function
func TestGetDeploymentHandler(t *testing.T) {
	deploymentName := "test-deployment"

	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": deploymentName,
			},
			expectedParams: kai.DeploymentParams{
				Name:      deploymentName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment: %s\nNamespace: %s\nReplicas: 3/3\nSelector: app=%s", deploymentName, defaultNamespace, deploymentName), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment: %s", deploymentName),
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Required parameter 'name' is missing",
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:           "Parameter 'name' must be a non-empty string",
			expectDeploymentCreation: false,
		},
		{
			name: "Deployment not found",
			args: map[string]interface{}{
				"name": "nonexistent-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "nonexistent-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Get", mock.Anything, mockCM).
					Return("", fmt.Errorf("failed to get deployment: %s not found", "nonexistent-deployment"))
			},
			expectedOutput:           "failed to get deployment: nonexistent-deployment not found",
			expectDeploymentCreation: true,
		},
		{
			name: "With specific namespace",
			args: map[string]interface{}{
				"name":      deploymentName,
				"namespace": testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Name:      deploymentName,
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment: %s\nNamespace: %s\nReplicas: 2/2", deploymentName, testNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment: %s\nNamespace: %s", deploymentName, testNamespace),
			expectDeploymentCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			// Use getDeploymentHandler here instead of describeDeploymentHandler
			handler := getDeploymentHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteDeploymentHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q deleted successfully from namespace %q", "test-deployment", defaultNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q deleted successfully", "test-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "WithNamespace",
			args: map[string]interface{}{
				"name":      "test-deployment",
				"namespace": testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deployment %q deleted successfully from namespace %q", "test-deployment", testNamespace), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q deleted successfully", "test-deployment"),
			expectDeploymentCreation: true,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Delete", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, deleteDeploymentHandler)
}

func TestScaleDeploymentHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": float64(3),
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
				Replicas:  float64(3),
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Scale", mock.Anything, mockCM).
					Return("Deployment \"test-deployment\" scaled to 3 replicas", nil)
			},
			expectedOutput:           "scaled to 3 replicas",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{"replicas": float64(3)},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name":     "",
				"replicas": float64(3),
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "MissingReplicas",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           "missing required parameter: replicas",
			expectDeploymentCreation: false,
		},
		{
			name: "InvalidReplicas",
			args: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": "invalid",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           "invalid replicas parameter: must be a number",
			expectDeploymentCreation: false,
		},
		{
			name: "WithNamespace",
			args: map[string]interface{}{
				"name":      "test-deployment",
				"replicas":  float64(5),
				"namespace": testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: testNamespace,
				Replicas:  float64(5),
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Scale", mock.Anything, mockCM).
					Return("Deployment \"test-deployment\" scaled to 5 replicas", nil)
			},
			expectedOutput:           "scaled to 5 replicas",
			expectDeploymentCreation: true,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": float64(3),
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
				Replicas:  float64(3),
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Scale", mock.Anything, mockCM).
					Return("", errors.New("failed to scale deployment"))
			},
			expectedOutput:           "failed to scale deployment",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, scaleDeploymentHandler)
}

func TestRolloutStatusHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutStatus", mock.Anything, mockCM).
					Return("deployment \"test-deployment\" successfully rolled out", nil)
			},
			expectedOutput:           "successfully rolled out",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutStatus", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutStatusHandler)
}

func TestRolloutHistoryHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutHistory", mock.Anything, mockCM).
					Return("deployment.apps/test-deployment\nREVISION  CHANGE-CAUSE\n1         <none>\n2         <none>", nil)
			},
			expectedOutput:           "REVISION",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutHistory", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutHistoryHandler)
}

func TestRolloutUndoHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "UndoToPreviousRevision",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutUndo", mock.Anything, mockCM, int64(0)).
					Return("deployment.apps/test-deployment rolled back", nil)
			},
			expectedOutput:           "rolled back",
			expectDeploymentCreation: true,
		},
		{
			name: "UndoToSpecificRevision",
			args: map[string]interface{}{
				"name":     "test-deployment",
				"revision": float64(2),
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutUndo", mock.Anything, mockCM, int64(2)).
					Return("deployment.apps/test-deployment rolled back to revision 2", nil)
			},
			expectedOutput:           "rolled back",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutUndo", mock.Anything, mockCM, int64(0)).
					Return("", errors.New("no rollout history found"))
			},
			expectedOutput:           "no rollout history found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutUndoHandler)
}

func TestRolloutRestartHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutRestart", mock.Anything, mockCM).
					Return("deployment.apps/test-deployment restarted", nil)
			},
			expectedOutput:           "restarted",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutRestart", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutRestartHandler)
}

func TestRolloutPauseHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutPause", mock.Anything, mockCM).
					Return("deployment.apps/test-deployment paused", nil)
			},
			expectedOutput:           "paused",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutPause", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutPauseHandler)
}

func TestRolloutResumeHandler(t *testing.T) {
	testCases := []deploymentTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutResume", mock.Anything, mockCM).
					Return("deployment.apps/test-deployment resumed", nil)
			},
			expectedOutput:           "resumed",
			expectDeploymentCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errMissingName,
			expectDeploymentCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
			},
			expectedOutput:           errEmptyName,
			expectDeploymentCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "test-deployment",
			},
			expectedParams: kai.DeploymentParams{
				Name:      "test-deployment",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("RolloutResume", mock.Anything, mockCM).
					Return("", errors.New("deployment not found"))
			},
			expectedOutput:           "deployment not found",
			expectDeploymentCreation: true,
		},
	}

	runDeploymentTests(t, testCases, rolloutResumeHandler)
}

// runDeploymentTests is a helper that runs a set of deployment test cases with a given handler constructor.
func runDeploymentTests(t *testing.T, testCases []deploymentTestCase, handlerFn func(kai.ClusterManager, DeploymentFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeploymentCreation {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := handlerFn(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockDeployment != nil {
				mockDeployment.AssertExpectations(t)
			}
		})
	}
}
