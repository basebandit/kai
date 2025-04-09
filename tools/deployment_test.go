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
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	// Set expectations - the server should have AddTool called once with a non-nil tool
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return()

	// Test the function call
	RegisterDeploymentTools(mockServer, mockClusterMgr)

	// Assert expectations were met
	mockServer.AssertExpectations(t)
}

func TestListDeploymentsHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("List deployments with all_namespaces true", func(t *testing.T) {
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")
		mockClusterMgr.On("ListDeployments", ctx, true, "", "default").Return("Deployments across all namespaces:\ndeployment1\ndeployment2", nil)

		// Create the handler
		handler := listDeploymentsHandler(mockClusterMgr)

		// Create request
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
		result, err := handler(ctx, request)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployments across all namespaces:\ndeployment1\ndeployment2", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List deployments with specific namespace", func(t *testing.T) {
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")
		mockClusterMgr.On("ListDeployments", ctx, false, "", "test-namespace").Return("Deployments in namespace test-namespace:\ndeployment1", nil)

		// Create the handler
		handler := listDeploymentsHandler(mockClusterMgr)

		// Create request
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
		result, err := handler(ctx, request)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployments in namespace test-namespace:\ndeployment1", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List deployments with label selector", func(t *testing.T) {
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")
		mockClusterMgr.On("ListDeployments", ctx, false, "app=backend", "default").Return("Deployments in namespace default with label app=backend:\nbackend-deployment", nil)

		// Create the handler
		handler := listDeploymentsHandler(mockClusterMgr)

		// Create request
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
		result, err := handler(ctx, request)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployments in namespace default with label app=backend:\nbackend-deployment", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("Error from ListDeployments", func(t *testing.T) {
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")
		mockClusterMgr.On("ListDeployments", ctx, false, "", "default").Return("", errors.New("connection failed"))

		// Create the handler
		handler := listDeploymentsHandler(mockClusterMgr)

		// Create request
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
		result, err := handler(ctx, request)

		// Assert
		assert.NoError(t, err) // Handler should not return error, but include error in result text
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be mcp.TextContent")
		assert.Equal(t, "connection failed", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("All parameters provided", func(t *testing.T) {
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")
		mockClusterMgr.On("ListDeployments", ctx, true, "env=prod", "prod-namespace").Return("Deployments in prod-namespace with label env=prod:\nprod-app-1\nprod-app-2", nil)

		// Create the handler
		handler := listDeploymentsHandler(mockClusterMgr)

		// Create request with all parameters
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
					"namespace":      "prod-namespace",
					"label_selector": "env=prod",
				},
			},
		}

		// Call the handler
		result, err := handler(ctx, request)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployments in prod-namespace with label env=prod:\nprod-app-1\nprod-app-2", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})
}

func TestCreateDeploymentHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("Create deployment with required parameters", func(t *testing.T) {
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")

		expectedParams := kai.DeploymentParams{
			Name:      "test-deployment",
			Image:     "nginx:latest",
			Namespace: "default",
			Replicas:  1, // Default value
		}

		mockClusterMgr.On("CreateDeployment", ctx, mock.MatchedBy(func(params kai.DeploymentParams) bool {
			return params.Name == expectedParams.Name &&
				params.Image == expectedParams.Image &&
				params.Namespace == expectedParams.Namespace &&
				params.Replicas == expectedParams.Replicas
		})).Return("Deployment test-deployment created successfully", nil)

		handler := createDeploymentHandler(mockClusterMgr)

		// Create request with minimum required parameters
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

		result, err := handler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployment test-deployment created successfully", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("Create deployment with all parameters", func(t *testing.T) {
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")

		labels := map[string]interface{}{
			"app": "test",
			"env": "dev",
		}

		env := map[string]interface{}{
			"API_URL": "http://api-service",
			"DEBUG":   "true",
		}

		pullSecrets := []interface{}{"registry-secret"}

		expectedParams := kai.DeploymentParams{
			Name:             "full-deployment",
			Image:            "myapp:v1.2.3",
			Namespace:        "test-namespace",
			Replicas:         3,
			Labels:           labels,
			ContainerPort:    "8080/TCP",
			Env:              env,
			ImagePullPolicy:  "Always",
			ImagePullSecrets: pullSecrets,
		}

		mockClusterMgr.On("CreateDeployment", ctx, mock.MatchedBy(func(params kai.DeploymentParams) bool {
			return params.Name == expectedParams.Name &&
				params.Image == expectedParams.Image &&
				params.Namespace == expectedParams.Namespace &&
				params.Replicas == expectedParams.Replicas &&
				assert.Equal(t, expectedParams.Labels, params.Labels) &&
				params.ContainerPort == expectedParams.ContainerPort &&
				assert.Equal(t, expectedParams.Env, params.Env) &&
				params.ImagePullPolicy == expectedParams.ImagePullPolicy &&
				assert.Equal(t, expectedParams.ImagePullSecrets, params.ImagePullSecrets)
		})).Return("Deployment full-deployment created successfully in namespace test-namespace", nil)

		handler := createDeploymentHandler(mockClusterMgr)

		// Create request with all parameters
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
					"replicas":           float64(3), // JSON numbers come in as float64
					"labels":             labels,
					"container_port":     "8080/TCP",
					"env":                env,
					"image_pull_policy":  "Always",
					"image_pull_secrets": pullSecrets,
				},
			},
		}

		result, err := handler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Deployment full-deployment created successfully in namespace test-namespace", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("Missing name parameter", func(t *testing.T) {
		mockClusterMgr := testmocks.NewMockClusterManager()

		handler := createDeploymentHandler(mockClusterMgr)

		// Create request with missing name
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

		result, err := handler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Parameter 'name' must be a non-empty string", textContent.Text)
	})

	t.Run("Missing image parameter", func(t *testing.T) {
		mockClusterMgr := testmocks.NewMockClusterManager()

		handler := createDeploymentHandler(mockClusterMgr)

		// Create request with missing image
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

		result, err := handler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Parameter 'image' must be a non-empty string", textContent.Text)
	})

	t.Run("Error from CreateDeployment", func(t *testing.T) {
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return("default")

		expectedParams := kai.DeploymentParams{
			Name:      "error-deployment",
			Image:     "nginx:latest",
			Namespace: "default",
			Replicas:  1,
		}

		mockClusterMgr.On("CreateDeployment", ctx, mock.MatchedBy(func(params kai.DeploymentParams) bool {
			return params.Name == expectedParams.Name &&
				params.Image == expectedParams.Image &&
				params.Namespace == expectedParams.Namespace &&
				params.Replicas == expectedParams.Replicas
		})).Return("failed to create deployment: resource quota exceeded", nil)

		handler := createDeploymentHandler(mockClusterMgr)

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

		result, err := handler(ctx, request)

		assert.NoError(t, err) // Handler should not return error, but include error in result text
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be mcp.TextContent")
		assert.Equal(t, "failed to create deployment: resource quota exceeded", textContent.Text)
		mockClusterMgr.AssertExpectations(t)
	})
}
