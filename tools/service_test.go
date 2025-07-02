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

type listServicesTestCase struct {
	name           string
	args           map[string]interface{}
	expectedParams kai.ServiceParams
	mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockServiceFactory, *testmocks.MockService)
	expectedOutput string
}

type getServiceTestCase struct {
	name                  string
	args                  map[string]interface{}
	expectedParams        kai.ServiceParams
	mockSetup             func(*testmocks.MockClusterManager, *testmocks.MockServiceFactory, *testmocks.MockService)
	expectedOutput        string
	expectServiceCreation bool
}

type createServiceTestCase struct {
	name                  string
	args                  map[string]interface{}
	expectedParams        kai.ServiceParams
	mockSetup             func(*testmocks.MockClusterManager, *testmocks.MockServiceFactory, *testmocks.MockService)
	expectedOutput        string
	expectServiceCreation bool
}

type deleteServiceTestCase struct {
	name                  string
	args                  map[string]interface{}
	expectedParams        kai.ServiceParams
	mockSetup             func(*testmocks.MockClusterManager, *testmocks.MockServiceFactory, *testmocks.MockService)
	expectedOutput        string
	expectServiceCreation bool
}

func TestRegisterServiceTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	// Expect AddTool to be called once for each tool we register
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(4)
	RegisterServiceTools(mockServer, mockClusterMgr)
	mockServer.AssertExpectations(t)
}

func TestRegisterServiceToolsWithFactory(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()
	mockFactory := testmocks.NewMockServiceFactory()

	// Expect AddTool to be called once for each tool we register
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(4)
	RegisterServiceToolsWithFactory(mockServer, mockClusterMgr, mockFactory)
	mockServer.AssertExpectations(t)
}

func TestListServicesHandler(t *testing.T) {
	testCases := []listServicesTestCase{
		{
			name: "DefaultNamespace",
			args: map[string]interface{}{},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Services in namespace %q:\n- service1\n- service2", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Services in namespace %q:", defaultNamespace),
		},
		{
			name: "AllNamespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockService.On("List", mock.Anything, mockCM, true, "").
					Return("Services across all namespaces:\n- ns1/service1\n- ns2/service2", nil)
			},
			expectedOutput: "Services across all namespaces:",
		},
		{
			name: "SpecificNamespace",
			args: map[string]interface{}{
				"namespace": testNamespace,
			},
			expectedParams: kai.ServiceParams{
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockService.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Services in namespace %q:\n- service1", testNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Services in namespace %q:", testNamespace),
		},
		{
			name: "WithLabelSelector",
			args: map[string]interface{}{
				"label_selector": "app=backend",
			},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("List", mock.Anything, mockCM, false, "app=backend").
					Return(fmt.Sprintf("Services in namespace %q with label 'app=backend':\n- backend-service", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Services in namespace %q with label 'app=backend':", defaultNamespace),
		},
		{
			name: "Error",
			args: map[string]interface{}{},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("List", mock.Anything, mockCM, false, "").
					Return("", errors.New(errConnectionFailed))
			},
			expectedOutput: errConnectionFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockServiceFactory()
			mockService := testmocks.NewMockService(tc.expectedParams)

			mockFactory.On("NewService", tc.expectedParams).Return(mockService)
			tc.mockSetup(mockCM, mockFactory, mockService)

			handler := listServicesHandler(mockCM, mockFactory)

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
			mockService.AssertExpectations(t)
		})
	}
}

func TestGetServiceHandler(t *testing.T) {
	serviceName := "test-service"

	testCases := []getServiceTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": serviceName,
			},
			expectedParams: kai.ServiceParams{
				Name:      serviceName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q in namespace %q:\nType: ClusterIP\nClusterIP: 10.0.0.1\nPorts: 80/TCP", serviceName, defaultNamespace), nil)
			},
			expectedOutput:        fmt.Sprintf("Service %q in namespace %q:", serviceName, defaultNamespace),
			expectServiceCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Required parameter 'name' is missing",
			expectServiceCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Parameter 'name' must be a non-empty string",
			expectServiceCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "nonexistent-service",
			},
			expectedParams: kai.ServiceParams{
				Name:      "nonexistent-service",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Get", mock.Anything, mockCM).
					Return("", fmt.Errorf("service %q not found in namespace %q", "nonexistent-service", defaultNamespace))
			},
			expectedOutput:        fmt.Sprintf("service %q not found in namespace %q", "nonexistent-service", defaultNamespace),
			expectServiceCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockServiceFactory()

			var mockService *testmocks.MockService
			if tc.expectServiceCreation {
				mockService = testmocks.NewMockService(tc.expectedParams)
				mockFactory.On("NewService", tc.expectedParams).Return(mockService)
			}

			tc.mockSetup(mockCM, mockFactory, mockService)

			handler := getServiceHandler(mockCM, mockFactory)

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
			if mockService != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestCreateServiceHandler(t *testing.T) {
	testServiceName := "test-service"
	clusterIPType := "ClusterIP"
	nodePortType := "NodePort"

	testCases := []createServiceTestCase{
		{
			name: "CreateClusterIPService",
			args: map[string]interface{}{
				"name": testServiceName,
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(80),
						"targetPort": float64(8080),
					},
				},
				"selector": map[string]interface{}{
					"app": "test",
				},
			},
			expectedParams: kai.ServiceParams{
				Name:      testServiceName,
				Namespace: defaultNamespace,
				Type:      clusterIPType,
				Ports: []kai.ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q created successfully in namespace %q (Type: ClusterIP)", testServiceName, defaultNamespace), nil)
			},
			expectedOutput:        fmt.Sprintf("Service %q created successfully", testServiceName),
			expectServiceCreation: true,
		},
		{
			name: "CreateNodePortService",
			args: map[string]interface{}{
				"name": testServiceName,
				"type": nodePortType,
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(80),
						"targetPort": float64(8080),
						"nodePort":   float64(30080),
					},
				},
				"selector": map[string]interface{}{
					"app": "test",
				},
			},
			expectedParams: kai.ServiceParams{
				Name:      testServiceName,
				Namespace: defaultNamespace,
				Type:      nodePortType,
				Ports: []kai.ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
						NodePort:   30080,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q created successfully in namespace %q (Type: NodePort)", testServiceName, defaultNamespace), nil)
			},
			expectedOutput:        fmt.Sprintf("Service %q created successfully", testServiceName),
			expectServiceCreation: true,
		},
		{
			name: "MultiplePortsService",
			args: map[string]interface{}{
				"name": testServiceName,
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "http",
						"port":       float64(80),
						"targetPort": float64(8080),
						"protocol":   "TCP",
					},
					map[string]interface{}{
						"name":       "https",
						"port":       float64(443),
						"targetPort": float64(8443),
						"protocol":   "TCP",
					},
				},
				"selector": map[string]interface{}{
					"app": "test",
				},
			},
			expectedParams: kai.ServiceParams{
				Name:      testServiceName,
				Namespace: defaultNamespace,
				Type:      clusterIPType,
				Ports: []kai.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: int32(8080),
						Protocol:   "TCP",
					},
					{
						Name:       "https",
						Port:       443,
						TargetPort: int32(8443),
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q created successfully in namespace %q (Type: ClusterIP)", testServiceName, defaultNamespace), nil)
			},
			expectedOutput:        fmt.Sprintf("Service %q created successfully", testServiceName),
			expectServiceCreation: true,
		},
		{
			name: "MissingName",
			args: map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(80),
						"targetPort": float64(8080),
					},
				},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Required parameter 'name' is missing",
			expectServiceCreation: false,
		},
		{
			name: "MissingPorts",
			args: map[string]interface{}{
				"name": testServiceName,
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Required parameter 'ports' is missing",
			expectServiceCreation: false,
		},
		{
			name: "EmptyPorts",
			args: map[string]interface{}{
				"name":  testServiceName,
				"ports": []interface{}{},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Parameter 'ports' must be a non-empty array",
			expectServiceCreation: false,
		},
		{
			name: "InvalidPortType",
			args: map[string]interface{}{
				"name": testServiceName,
				"ports": []interface{}{
					"invalid-port", // Not an object
				},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Invalid ports configuration: port 0: must be an object",
			expectServiceCreation: false,
		},
		{
			name: "MissingPortNumber",
			args: map[string]interface{}{
				"name": testServiceName,
				"ports": []interface{}{
					map[string]interface{}{
						// Missing 'port' field
						"targetPort": float64(8080),
					},
				},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Invalid ports configuration: port 0: required field 'port' is missing",
			expectServiceCreation: false,
		},
		{
			name: "InvalidPortRange",
			args: map[string]interface{}{
				"name": testServiceName,
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(99999), // Invalid port number
						"targetPort": float64(8080),
					},
				},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				// No setup needed
			},
			expectedOutput:        "Invalid ports configuration: port 0: port number must be between 1 and 65535",
			expectServiceCreation: false,
		},
		{
			name: "InvalidServiceType",
			args: map[string]interface{}{
				"name": testServiceName,
				"type": "InvalidType",
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(80),
						"targetPort": float64(8080),
					},
				},
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:        "Invalid service type: InvalidType",
			expectServiceCreation: false,
		},
		{
			name: "ServiceCreationError",
			args: map[string]interface{}{
				"name": "error-service",
				"ports": []interface{}{
					map[string]interface{}{
						"port":       float64(80),
						"targetPort": float64(8080),
					},
				},
			},
			expectedParams: kai.ServiceParams{
				Name:      "error-service",
				Namespace: defaultNamespace,
				Type:      clusterIPType,
				Ports: []kai.ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
						Protocol:   "TCP",
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Create", mock.Anything, mockCM).
					Return("", errors.New("failed to create service: resource quota exceeded"))
			},
			expectedOutput:        "failed to create service: resource quota exceeded",
			expectServiceCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockServiceFactory()

			var mockService *testmocks.MockService
			if tc.expectServiceCreation {
				mockService = testmocks.NewMockService(tc.expectedParams)
				mockFactory.On("NewService", mock.MatchedBy(func(params kai.ServiceParams) bool {
					// Match essential parameters
					if params.Name != tc.expectedParams.Name ||
						params.Namespace != tc.expectedParams.Namespace ||
						params.Type != tc.expectedParams.Type {
						return false
					}

					// Match ports length
					if len(params.Ports) != len(tc.expectedParams.Ports) {
						return false
					}

					// We don't need to check every detail, just the essentials
					return true
				})).Return(mockService)
			}

			tc.mockSetup(mockCM, mockFactory, mockService)

			handler := createServiceHandler(mockCM, mockFactory)

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
			if mockService != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteServiceHandler(t *testing.T) {
	serviceName := "test-service"

	testCases := []deleteServiceTestCase{
		{
			name: "Delete by Name",
			args: map[string]interface{}{
				"name": serviceName,
			},
			expectedParams: kai.ServiceParams{
				Name:      serviceName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q deleted successfully from namespace %q", serviceName, defaultNamespace), nil)
			},
			expectedOutput:        fmt.Sprintf("Service %q deleted successfully", serviceName),
			expectServiceCreation: true,
		},
		{
			name: "Delete by Labels",
			args: map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "frontend",
				},
			},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
				Labels: map[string]interface{}{
					"app": "frontend",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deleted 2 services with label selector %q from namespace %q:\n- service1\n- service3", "app=frontend", defaultNamespace), nil)
			},
			expectedOutput:        "Deleted 2 services with label selector",
			expectServiceCreation: true,
		},
		{
			name: "Delete by Multiple Labels",
			args: map[string]interface{}{
				"labels": map[string]interface{}{
					"app":     "frontend",
					"version": "v2",
				},
			},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
				Labels: map[string]interface{}{
					"app":     "frontend",
					"version": "v2",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deleted 1 services with label selector %q from namespace %q:\n- service3", "app=frontend,version=v2", defaultNamespace), nil)
			},
			expectedOutput:        "Deleted 1 services with label selector",
			expectServiceCreation: true,
		},
		{
			name:           "No Parameters Provided",
			args:           map[string]interface{}{},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:        "Either 'name' or 'labels' parameter must be provided",
			expectServiceCreation: false,
		},
		{
			name: "Invalid Labels Type",
			args: map[string]interface{}{
				"labels": "invalid-labels", // Should be an object, not a string
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:        "Parameter 'labels' must be an object",
			expectServiceCreation: false,
		},
		{
			name: "Empty Labels Object",
			args: map[string]interface{}{
				"labels": map[string]interface{}{}, // Empty map
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:        "Parameter 'labels' must be a non-empty object",
			expectServiceCreation: false,
		},
		{
			name: "Empty Name String",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.ServiceParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:        "Parameter 'name' must be a non-empty string",
			expectServiceCreation: false,
		},
		{
			name: "Label Selector with No Matches",
			args: map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "nonexistent",
				},
			},
			expectedParams: kai.ServiceParams{
				Namespace: defaultNamespace,
				Labels: map[string]interface{}{
					"app": "nonexistent",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return("", fmt.Errorf("no services found with label selector %q in namespace %q", "app=nonexistent", defaultNamespace))
			},
			expectedOutput:        "no services found with label selector",
			expectServiceCreation: true,
		},
		{
			name: "Custom Namespace with Label Selector",
			args: map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "frontend",
				},
				"namespace": testNamespace,
			},
			expectedParams: kai.ServiceParams{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "frontend",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Deleted 2 services with label selector %q from namespace %q:\n- service1\n- service3", "app=frontend", testNamespace), nil)
			},
			expectedOutput:        "Deleted 2 services with label selector",
			expectServiceCreation: true,
		},
		{
			name: "Both Name and Labels Provided",
			args: map[string]interface{}{
				"name": serviceName,
				"labels": map[string]interface{}{
					"app": "frontend",
				},
			},
			expectedParams: kai.ServiceParams{
				Name:      serviceName,
				Namespace: defaultNamespace,
				Labels: map[string]interface{}{
					"app": "frontend",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockServiceFactory, mockService *testmocks.MockService) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockService.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Service %q deleted successfully from namespace %q", serviceName, defaultNamespace), nil)
			},
			expectedOutput:        "Service \"test-service\" deleted successfully",
			expectServiceCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockServiceFactory()

			var mockService *testmocks.MockService
			if tc.expectServiceCreation {
				mockService = testmocks.NewMockService(tc.expectedParams)
				mockFactory.On("NewService", mock.MatchedBy(func(params kai.ServiceParams) bool {
					if params.Name != tc.expectedParams.Name ||
						params.Namespace != tc.expectedParams.Namespace {
						return false
					}

					if tc.expectedParams.Labels != nil {
						if params.Labels == nil || len(params.Labels) != len(tc.expectedParams.Labels) {
							return false
						}
						for key, value := range tc.expectedParams.Labels {
							paramValue, exists := params.Labels[key]
							if !exists || paramValue != value {
								return false
							}
						}
					} else if params.Labels != nil {
						return false
					}

					return true
				})).Return(mockService)
			}

			tc.mockSetup(mockCM, mockFactory, mockService)

			handler := deleteServiceHandler(mockCM, mockFactory)

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
			if mockService != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}
