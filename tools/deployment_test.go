package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestListDeploymentsHandler_DefaultNamespace(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Define the expected parameters that the handler will create
	expectedParams := kai.DeploymentParams{
		Namespace: "default",
	}

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(expectedParams)
	mockDeployment.On("List", mock.Anything, mockCM, false, "").Return("Deployments in namespace 'default':\n- deployment1\n- deployment2", nil)

	// Setup mock factory with the exact same expected params
	mockFactory := testmocks.NewMockDeploymentFactory()
	mockFactory.On("NewDeployment", expectedParams).Return(mockDeployment)

	// Create the handler
	handler := listDeploymentsHandler(mockCM, mockFactory)

	// Create a request
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployments in namespace 'default':")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestListDeploymentsHandler_AllNamespaces(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	// Don't expect GetCurrentNamespace since it's not called when all_namespaces is true

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(kai.DeploymentParams{})
	mockDeployment.On("List", mock.Anything, mockCM, true, "").Return("Deployments across all namespaces:\n- ns1/deployment1\n- ns2/deployment2", nil)

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	params := kai.DeploymentParams{} // Empty params for all namespaces
	mockFactory.On("NewDeployment", params).Return(mockDeployment)

	// Create the handler
	handler := listDeploymentsHandler(mockCM, mockFactory)

	// Create a request with all_namespaces=true
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"all_namespaces": true,
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployments across all namespaces:")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestListDeploymentsHandler_SpecificNamespace(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	// Don't expect GetCurrentNamespace since a specific namespace is provided

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(kai.DeploymentParams{
		Namespace: "test-namespace",
	})
	mockDeployment.On("List", mock.Anything, mockCM, false, "").Return("Deployments in namespace 'test-namespace':\n- deployment1", nil)

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	params := kai.DeploymentParams{
		Namespace: "test-namespace",
	}
	mockFactory.On("NewDeployment", params).Return(mockDeployment)

	// Create the handler
	handler := listDeploymentsHandler(mockCM, mockFactory)

	// Create a request with specific namespace
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"namespace": "test-namespace",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployments in namespace 'test-namespace':")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestListDeploymentsHandler_WithLabelSelector(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(kai.DeploymentParams{
		Namespace: "default",
	})
	mockDeployment.On("List", mock.Anything, mockCM, false, "app=backend").Return("Deployments in namespace 'default' with label 'app=backend':\n- backend-deployment", nil)

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	params := kai.DeploymentParams{
		Namespace: "default",
	}
	mockFactory.On("NewDeployment", params).Return(mockDeployment)

	// Create the handler
	handler := listDeploymentsHandler(mockCM, mockFactory)

	// Create a request with label selector
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"label_selector": "app=backend",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployments in namespace 'default' with label 'app=backend':")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestListDeploymentsHandler_Error(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(kai.DeploymentParams{
		Namespace: "default",
	})
	mockDeployment.On("List", mock.Anything, mockCM, false, "").Return("", errors.New("connection failed"))

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	params := kai.DeploymentParams{
		Namespace: "default",
	}
	mockFactory.On("NewDeployment", params).Return(mockDeployment)

	// Create the handler
	handler := listDeploymentsHandler(mockCM, mockFactory)

	// Create a request
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err) // Handler doesn't return error, it returns a result with error text
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "connection failed")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestCreateDeploymentHandler_RequiredParams(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(kai.DeploymentParams{
		Name:      "test-deployment",
		Image:     "nginx:latest",
		Namespace: "default",
		Replicas:  1,
	})
	mockDeployment.On("Create", mock.Anything, mockCM).Return("Deployment \"test-deployment\" created successfully in namespace \"default\" with 1 replica(s)", nil)

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	params := kai.DeploymentParams{
		Name:      "test-deployment",
		Image:     "nginx:latest",
		Namespace: "default",
		Replicas:  1, // Default value
	}
	mockFactory.On("NewDeployment", params).Return(mockDeployment)

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with required parameters
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":  "test-deployment",
				"image": "nginx:latest",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployment \"test-deployment\" created successfully")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestCreateDeploymentHandler_AllParams(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Define expected parameters
	expectedParams := kai.DeploymentParams{
		Name:             "full-deployment",
		Image:            "myapp:v1.2.3",
		Namespace:        "test-namespace",
		Replicas:         3.0,
		Labels:           map[string]interface{}{"app": "myapp", "env": "test"},
		ContainerPort:    "8080/TCP",
		Env:              map[string]interface{}{"DEBUG": "true"},
		ImagePullPolicy:  "Always",
		ImagePullSecrets: []interface{}{"registry-secret"},
	}

	// Setup mock deployment
	mockDeployment := testmocks.NewMockDeployment(expectedParams)
	mockDeployment.On("Create", mock.Anything, mockCM).Return("Deployment \"full-deployment\" created successfully in namespace \"test-namespace\" with 3 replica(s)", nil)

	// Setup mock factory with full DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	mockFactory.On("NewDeployment", expectedParams).Return(mockDeployment)

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with all parameters
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":               "full-deployment",
				"image":              "myapp:v1.2.3",
				"namespace":          "test-namespace",
				"replicas":           float64(3),
				"labels":             map[string]interface{}{"app": "myapp", "env": "test"},
				"container_port":     "8080/TCP",
				"env":                map[string]interface{}{"DEBUG": "true"},
				"image_pull_policy":  "Always",
				"image_pull_secrets": []interface{}{"registry-secret"},
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Deployment \"full-deployment\" created successfully")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}

func TestCreateDeploymentHandler_MissingName(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := testmocks.NewMockDeploymentFactory()

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with missing name
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"image": "nginx:latest",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Required parameter 'name' is missing")

	// Verify that no deployment was created
	mockFactory.AssertNotCalled(t, "NewDeployment")
}

func TestCreateDeploymentHandler_MissingImage(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := testmocks.NewMockDeploymentFactory()

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with missing image
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name": "test-deployment",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Required parameter 'image' is missing")

	// Verify that no deployment was created
	mockFactory.AssertNotCalled(t, "NewDeployment")
}

func TestCreateDeploymentHandler_EmptyName(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := testmocks.NewMockDeploymentFactory()

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with empty name
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":  "",
				"image": "nginx:latest",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Parameter 'name' must be a non-empty string")

	// Verify that no deployment was created
	mockFactory.AssertNotCalled(t, "NewDeployment")
}

func TestCreateDeploymentHandler_EmptyImage(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := testmocks.NewMockDeploymentFactory()

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request with empty image
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":  "test-deployment",
				"image": "",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Parameter 'image' must be a non-empty string")

	// Verify that no deployment was created
	mockFactory.AssertNotCalled(t, "NewDeployment")
}

func TestCreateDeploymentHandler_Error(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Define expected parameters
	expectedParams := kai.DeploymentParams{
		Name:      "error-deployment",
		Image:     "nginx:latest",
		Namespace: "default",
		Replicas:  1,
	}

	// Setup mock deployment with an error on Create
	mockDeployment := testmocks.NewMockDeployment(expectedParams)
	mockDeployment.On("Create", mock.Anything, mockCM).Return("", errors.New("failed to create deployment: resource quota exceeded"))

	// Setup mock factory with DeploymentParams
	mockFactory := testmocks.NewMockDeploymentFactory()
	mockFactory.On("NewDeployment", expectedParams).Return(mockDeployment)

	// Create the handler
	handler := createDeploymentHandler(mockCM, mockFactory)

	// Create a request
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":  "error-deployment",
				"image": "nginx:latest",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "failed to create deployment: resource quota exceeded")

	// Verify that the expected methods were called
	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockDeployment.AssertExpectations(t)
}
