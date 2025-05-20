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

// Test case structs for table-driven tests for the new tests
type describeDeploymentTestCase struct {
	name                     string
	args                     map[string]interface{}
	expectedParams           kai.DeploymentParams
	mockSetup                func(*testmocks.MockClusterManager, *testmocks.MockDeploymentFactory, *testmocks.MockDeployment)
	expectedOutput           string
	expectDeploymentCreation bool
}

type updateDeploymentTestCase struct {
	name                string
	args                map[string]interface{}
	expectedParams      kai.DeploymentParams
	mockSetup           func(*testmocks.MockClusterManager, *testmocks.MockDeploymentFactory, *testmocks.MockDeployment)
	expectedOutput      string
	expectDeployCreated bool
}

func TestDescribeDeploymentHandler(t *testing.T) {
	deploymentName := "test-deployment"

	testCases := []describeDeploymentTestCase{
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
					Return(fmt.Sprintf("Deployment %q in namespace %q:\nReplicas: 3/3\nSelector: app=%s", deploymentName, defaultNamespace, deploymentName), nil)
			},
			expectedOutput:           fmt.Sprintf("Deployment %q in namespace %q", deploymentName, defaultNamespace),
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
				mockDeployment.On("Get", mock.Anything, mockCM).
					Return("", fmt.Errorf("deployment %q not found in namespace %q", "nonexistent-deployment", defaultNamespace))
			},
			expectedOutput:           fmt.Sprintf("deployment %q not found in namespace %q", "nonexistent-deployment", defaultNamespace),
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
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
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
	testCases := []updateDeploymentTestCase{
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
			expectedOutput:      fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeployCreated: true,
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
			expectedOutput:      fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeployCreated: true,
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
			expectedOutput:      fmt.Sprintf("Deployment %q updated successfully", "test-deployment"),
			expectDeployCreated: true,
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
			expectedOutput:      "Required parameter 'name' is missing",
			expectDeployCreated: false,
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
			expectedOutput:      "Parameter 'name' must be a non-empty string",
			expectDeployCreated: false,
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
			expectedOutput:      "At least one field to update must be specified",
			expectDeployCreated: false,
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
			expectedOutput:      "Port must be a number",
			expectDeployCreated: false, // the handler will return before creating deployment
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
			expectedOutput:      "failed to update deployment: deployment not found",
			expectDeployCreated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()

			var mockDeployment *testmocks.MockDeployment
			if tc.expectDeployCreated {
				mockDeployment = testmocks.NewMockDeployment(tc.expectedParams)
				mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			}

			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := updateDeploymentHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
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
