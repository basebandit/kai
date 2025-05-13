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

// Test case structs for table-driven tests
type listDeploymentsTestCase struct {
	name           string
	args           map[string]interface{}
	expectedParams kai.DeploymentParams
	mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockDeploymentFactory, *testmocks.MockDeployment)
	expectedOutput string
}

type createDeploymentTestCase struct {
	name                string
	args                map[string]interface{}
	expectedParams      kai.DeploymentParams
	mockSetup           func(*testmocks.MockClusterManager, *testmocks.MockDeploymentFactory, *testmocks.MockDeployment)
	expectedOutput      string
	expectDeployCreated bool
}

func TestRegisterDeploymentTools(t *testing.T) {
	// Setup mock server and cluster manager
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	// Expect AddTool to be called once for each tool we register
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(2)

	// Call the function
	RegisterDeploymentTools(mockServer, mockClusterMgr)

	// Verify expectations were met
	mockServer.AssertExpectations(t)
}

func TestRegisterDeploymentToolsWithFactory(t *testing.T) {
	// Setup mock server, cluster manager, and factory
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()
	mockFactory := testmocks.NewMockDeploymentFactory()

	// Expect AddTool to be called once for each tool we register
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(2)

	// Call the function
	RegisterDeploymentToolsWithFactory(mockServer, mockClusterMgr, mockFactory)

	// Verify expectations were met
	mockServer.AssertExpectations(t)
}

func TestListDeploymentsHandler(t *testing.T) {
	testCases := []listDeploymentsTestCase{
		{
			name: "DefaultNamespace",
			args: map[string]interface{}{},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Deployments in namespace %q:\n- deployment1\n- deployment2", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Deployments in namespace %q:", defaultNamespace),
		},
		{
			name: "AllNamespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockDeployment.On("List", mock.Anything, mockCM, true, "").
					Return("Deployments across all namespaces:\n- ns1/deployment1\n- ns2/deployment2", nil)
			},
			expectedOutput: "Deployments across all namespaces:",
		},
		{
			name: "SpecificNamespace",
			args: map[string]interface{}{
				"namespace": testNamespace,
			},
			expectedParams: kai.DeploymentParams{
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Deployments in namespace %q:\n- deployment1", testNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Deployments in namespace %q:", testNamespace),
		},
		{
			name: "WithLabelSelector",
			args: map[string]interface{}{
				"label_selector": "app=backend",
			},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "app=backend").
					Return(fmt.Sprintf("Deployments in namespace %q with label 'app=backend':\n- backend-deployment", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Deployments in namespace %q with label 'app=backend':", defaultNamespace),
		},
		{
			name: "Error",
			args: map[string]interface{}{},
			expectedParams: kai.DeploymentParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("List", mock.Anything, mockCM, false, "").
					Return("", errors.New(connectionFailedError))
			},
			expectedOutput: connectionFailedError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockDeploymentFactory()
			mockDeployment := testmocks.NewMockDeployment(tc.expectedParams)

			mockFactory.On("NewDeployment", tc.expectedParams).Return(mockDeployment)
			tc.mockSetup(mockCM, mockFactory, mockDeployment)

			handler := listDeploymentsHandler(mockCM, mockFactory)

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
			mockDeployment.AssertExpectations(t)
		})
	}
}

func TestCreateDeploymentHandler(t *testing.T) {
	// Define common test data
	testDeployName := "test-deployment"
	fullDeployName := "full-deployment"
	errorDeployName := "error-deployment"
	defaultReplicas := 1.0

	testCases := []createDeploymentTestCase{
		{
			name: "RequiredParams",
			args: map[string]interface{}{
				"name":  testDeployName,
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{
				Name:      testDeployName,
				Image:     nginxImage,
				Namespace: defaultNamespace,
				Replicas:  defaultReplicas,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf(deploymentCreatedFmt, testDeployName, defaultNamespace, int(defaultReplicas)), nil)
			},
			expectedOutput:      fmt.Sprintf("Deployment %q created successfully", testDeployName),
			expectDeployCreated: true,
		},
		{
			name: "AllParams",
			args: map[string]interface{}{
				"name":               fullDeployName,
				"image":              myAppImage,
				"namespace":          testNamespace,
				"replicas":           float64(3),
				"labels":             map[string]interface{}{"app": "myapp", "env": "test"},
				"container_port":     defaultContainerPort,
				"env":                map[string]interface{}{"DEBUG": "true"},
				"image_pull_policy":  alwaysImagePullPolicy,
				"image_pull_secrets": []interface{}{registrySecretName},
			},
			expectedParams: kai.DeploymentParams{
				Name:             fullDeployName,
				Image:            myAppImage,
				Namespace:        testNamespace,
				Replicas:         3.0,
				Labels:           map[string]interface{}{"app": "myapp", "env": "test"},
				ContainerPort:    defaultContainerPort,
				Env:              map[string]interface{}{"DEBUG": "true"},
				ImagePullPolicy:  alwaysImagePullPolicy,
				ImagePullSecrets: []interface{}{registrySecretName},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf(deploymentCreatedFmt, fullDeployName, testNamespace, 3), nil)
			},
			expectedOutput:      fmt.Sprintf("Deployment %q created successfully", fullDeployName),
			expectDeployCreated: true,
		},
		{
			name: "MissingName",
			args: map[string]interface{}{
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:      missingNameError,
			expectDeployCreated: false,
		},
		{
			name: "MissingImage",
			args: map[string]interface{}{
				"name": testDeployName,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:      missingImageError,
			expectDeployCreated: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name":  "",
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:      emptyNameError,
			expectDeployCreated: false,
		},
		{
			name: "EmptyImage",
			args: map[string]interface{}{
				"name":  testDeployName,
				"image": "",
			},
			expectedParams: kai.DeploymentParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				// No setup needed
			},
			expectedOutput:      emptyImageError,
			expectDeployCreated: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name":  errorDeployName,
				"image": nginxImage,
			},
			expectedParams: kai.DeploymentParams{
				Name:      errorDeployName,
				Image:     nginxImage,
				Namespace: defaultNamespace,
				Replicas:  defaultReplicas,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockDeploymentFactory, mockDeployment *testmocks.MockDeployment) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockDeployment.On("Create", mock.Anything, mockCM).
					Return("", errors.New(quotaExceededError))
			},
			expectedOutput:      quotaExceededError,
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

			handler := createDeploymentHandler(mockCM, mockFactory)

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
