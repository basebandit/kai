package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterPodTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	// Set expectations
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return()

	// Test the function call
	RegisterPodTools(mockServer, mockClusterMgr)

	mockServer.AssertExpectations(t)
}

func TestListPodsHandler(t *testing.T) {
	var (
		namespace,
		labelSelector,
		fieldSelector string
	)
	var limit int64

	ctx := context.Background()

	// List pods when namespace is not specified; namespace=""
	t.Run("List pods with empty namespace", func(t *testing.T) {

		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return("Pods across all namespaces:\npod1\npod2", nil)

		handler := listPodsHandler(mockClusterMgr)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					"namespace": namespace,
				},
			},
		}

		result, err := handler(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok, "Expected first content item to be a mcp.TextContent")
		assert.Equal(t, "Pods across all namespaces:\npod1\npod2", textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with specific namespace", func(t *testing.T) {
		namespace = "test-namespace"

		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf("Pods in namespace %q:\npod1", namespace), nil)

		handler := listPodsHandler(mockClusterMgr)

		// Create request with namespace=test-namespace
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					"namespace": namespace,
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
		assert.Equal(t, fmt.Sprintf("Pods in namespace %q:\npod1", namespace), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with default namespace", func(t *testing.T) {
		namespace = "default"
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return(namespace)

		// Set up ListPods to expect the default namespace
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf("Pods in namespace %q:\npod1\npod2", namespace), nil)

		// Create the handler
		handler := listPodsHandler(mockClusterMgr)

		// Create request without specifying namespace
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					// No namespace parameter
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
		assert.Equal(t, fmt.Sprintf("Pods in namespace %q:\npod1\npod2", namespace), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with label selector", func(t *testing.T) {
		namespace = "default"
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return(namespace)

		labelSelector = "app=frontend"
		// Set up ListPods to expect the label selector
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf(
			"Pods in namespace %q with label %q:\nfrontend-pod-1\nfrontend-pod-2", namespace, labelSelector), nil)

		// Create the handler
		handler := listPodsHandler(mockClusterMgr)

		// Create request with label selector
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					"label_selector": labelSelector,
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
		assert.Equal(t, fmt.Sprintf("Pods in namespace %q with label %q:\nfrontend-pod-1\nfrontend-pod-2", namespace, labelSelector), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with field selector", func(t *testing.T) {
		namespace = "default"

		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return(namespace)

		fieldSelector = "status.phase=Running"
		labelSelector = ""
		// Set up ListPods to expect the field selector
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf(
			"Pods in namespace %q with field %q:\nrunning-pod-1\nrunning-pod-2", namespace, fieldSelector), nil)

		// Create the handler
		handler := listPodsHandler(mockClusterMgr)

		// Create request with field selector
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					"field_selector": fieldSelector,
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
		assert.Equal(t, fmt.Sprintf("Pods in namespace %q with field %q:\nrunning-pod-1\nrunning-pod-2", namespace, fieldSelector), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with both label and field selectors", func(t *testing.T) {
		namespace = "default"

		mockClusterMgr := testmocks.NewMockClusterManager()
		mockClusterMgr.On("GetCurrentNamespace").Return(namespace)

		labelSelector = "app=frontend"
		fieldSelector = "status.phase=Running"

		// Set up ListPods to expect both selectors
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf(
			"Pods in namespace %q with label %q and field %q:\nfrontend-running-pod", namespace, labelSelector, fieldSelector), nil)

		// Create the handler
		handler := listPodsHandler(mockClusterMgr)

		// Create request with both selectors
		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Arguments: map[string]interface{}{
					"label_selector": labelSelector,
					"field_selector": fieldSelector,
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
		assert.Equal(t, fmt.Sprintf("Pods in namespace %q with label %q and field %q:\nfrontend-running-pod", namespace, labelSelector, fieldSelector), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})

	t.Run("List pods with selectors and all_namespaces flag", func(t *testing.T) {
		namespace = ""
		// Setup mock
		mockClusterMgr := testmocks.NewMockClusterManager()

		labelSelector = "app=backend"
		fieldSelector = "status.phase=Pending"

		// Set up ListPods to expect both selectors and empty namespace
		mockClusterMgr.On("ListPods", ctx, limit, namespace, labelSelector, fieldSelector).Return(fmt.Sprintf(
			"Pods across all namespaces with label %q and field %q:\nbackend-pending-pod-1\nbackend-pending-pod-2", labelSelector, fieldSelector), nil)

		// Create the handler
		handler := listPodsHandler(mockClusterMgr)

		// Create request with both selectors and all_namespaces
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
					"label_selector": labelSelector,
					"field_selector": fieldSelector,
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
		assert.Equal(t, fmt.Sprintf("Pods across all namespaces with label %q and field %q:\nbackend-pending-pod-1\nbackend-pending-pod-2",
			labelSelector, fieldSelector), textContent.Text)

		mockClusterMgr.AssertExpectations(t)
	})
}
