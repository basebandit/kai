package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListPodsHandler(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("List", mock.Anything, mockCM, int64(0)).Return("Pods in namespace 'default':\n- pod1\n- pod2", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := listPodsHandler(mockCM, mockFactory)

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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Pods in namespace 'default':")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestListPodsHandler_AllNamespaces(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("List", mock.Anything, mockCM, int64(0)).Return("Pods across all namespaces:\n- namespace1/pod1\n- namespace2/pod2", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "", "", "", "", "").Return(mockPod)

	// Create the handler
	handler := listPodsHandler(mockCM, mockFactory)

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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Pods across all namespaces:")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestListPodsHandler_WithLabelSelector(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("List", mock.Anything, mockCM, int64(0)).Return("Pods in namespace 'default' with label 'app=nginx':\n- nginx-pod-1\n- nginx-pod-2", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "", "default", "", "app=nginx", "").Return(mockPod)

	// Create the handler
	handler := listPodsHandler(mockCM, mockFactory)

	// Create a request with label_selector
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"label_selector": "app=nginx",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Pods in namespace 'default' with label 'app=nginx':")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestListPodsHandler_WithLimit(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("List", mock.Anything, mockCM, int64(5)).Return("Pods in namespace 'default' (limited to 5):\n- pod1\n- pod2\n- pod3\n- pod4\n- pod5", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := listPodsHandler(mockCM, mockFactory)

	// Create a request with limit
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"limit": float64(5),
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Pods in namespace 'default' (limited to 5):")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestListPodsHandler_Error(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("List", mock.Anything, mockCM, int64(0)).Return("", errors.New("failed to list pods: connection error"))

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := listPodsHandler(mockCM, mockFactory)

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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "failed to list pods: connection error")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestGetPodHandler(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("Get", mock.Anything, mockCM).Return("Pod 'nginx-pod' in namespace 'default':\nStatus: Running\nNode: worker-1\nIP: 192.168.1.10", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "nginx-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := getPodHandler(mockCM, mockFactory)

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
				"name": "nginx-pod",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Pod 'nginx-pod' in namespace 'default':")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestGetPodHandler_MissingName(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)

	// Create the handler
	handler := getPodHandler(mockCM, mockFactory)

	// Create a request with missing name
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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Required parameter 'name' is missing")

	// Verify that no pod was created
	mockFactory.AssertNotCalled(t, "NewPod")
}

func TestGetPodHandler_EmptyName(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)

	// Create the handler
	handler := getPodHandler(mockCM, mockFactory)

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
				"name": "",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Parameter 'name' must be a non-empty string")

	// Verify that no pod was created
	mockFactory.AssertNotCalled(t, "NewPod")
}

func TestGetPodHandler_Error(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("Get", mock.Anything, mockCM).Return("", errors.New("pod 'non-existent-pod' not found in namespace 'default'"))

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "non-existent-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := getPodHandler(mockCM, mockFactory)

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
				"name": "non-existent-pod",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "pod 'non-existent-pod' not found in namespace 'default'")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestDeletePodHandler(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("Delete", mock.Anything, mockCM, false).Return("Successfully delete pod \"nginx-pod\" in namespace \"default\"", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "nginx-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := deletePodHandler(mockCM, mockFactory)

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
				"name": "nginx-pod",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Successfully delete pod \"nginx-pod\" in namespace \"default\"")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestDeletePodHandler_WithForce(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("Delete", mock.Anything, mockCM, true).Return("Successfully delete pod \"nginx-pod\" in namespace \"default\"", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "nginx-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := deletePodHandler(mockCM, mockFactory)

	// Create a request with force=true
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"name":  "nginx-pod",
				"force": true,
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Successfully delete pod \"nginx-pod\" in namespace \"default\"")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestDeletePodHandler_MissingName(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)

	// Create the handler
	handler := deletePodHandler(mockCM, mockFactory)

	// Create a request with missing name
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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Required parameter 'name' is missing")

	// Verify that no pod was created
	mockFactory.AssertNotCalled(t, "NewPod")
}

func TestDeletePodHandler_Error(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("Delete", mock.Anything, mockCM, false).Return("", errors.New("failed to delete pod: not found"))

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "non-existent-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := deletePodHandler(mockCM, mockFactory)

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
				"name": "non-existent-pod",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "failed to delete pod: not found")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestStreamLogsHandler(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
		Return("Logs from container 'nginx' in pod 'default/nginx-pod':\n2023-05-01T12:00:00Z INFO starting nginx\n2023-05-01T12:00:01Z INFO nginx started", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "nginx-pod", "default", "", "", "").Return(mockPod)

	// Create the handler
	handler := streamLogsHandler(mockCM, mockFactory)

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
				"pod": "nginx-pod",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Logs from container 'nginx' in pod 'default/nginx-pod':")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestStreamLogsHandler_WithContainer(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock pod
	mockPod := testmocks.NewMockPod()
	mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
		Return("Logs from container 'sidecar' in pod 'default/nginx-pod':\n2023-05-01T12:00:00Z INFO starting sidecar\n2023-05-01T12:00:01Z INFO sidecar started", nil)

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)
	mockFactory.On("NewPod", "nginx-pod", "default", "sidecar", "", "").Return(mockPod)

	// Create the handler
	handler := streamLogsHandler(mockCM, mockFactory)

	// Create a request with container name
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"pod":       "nginx-pod",
				"container": "sidecar",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Logs from container 'sidecar' in pod 'default/nginx-pod':")

	// Verify that the expected methods were called
	mockFactory.AssertExpectations(t)
	mockPod.AssertExpectations(t)
}

func TestStreamLogsHandler_InvalidSince(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentNamespace").Return("default")

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)

	// Create the handler
	handler := streamLogsHandler(mockCM, mockFactory)

	// Create a request with invalid since parameter
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: map[string]interface{}{
				"pod":   "nginx-pod",
				"since": "invalid",
			},
		},
	}

	// Call the handler
	result, err := handler(context.Background(), request)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to parse 'since' parameter")

	// Verify that no pod was created
	mockFactory.AssertNotCalled(t, "NewPod")
}

func TestStreamLogsHandler_MissingPod(t *testing.T) {
	// Setup mock cluster manager
	mockCM := testmocks.NewMockClusterManager()

	// Setup mock factory
	mockFactory := new(testmocks.MockPodFactory)

	// Create the handler
	handler := streamLogsHandler(mockCM, mockFactory)

	// Create a request with missing pod
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
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Required parameter 'pod' is missing")

	// Verify that no pod was created
	mockFactory.AssertNotCalled(t, "NewPod")
}

func TestRegisterPodTools(t *testing.T) {
	// Setup mock server and cluster manager
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()

	// Expect tool registrations - use the correct type match
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(4)

	// Call the function
	RegisterPodTools(mockServer, mockCM)

	// Verify that the expected methods were called
	mockServer.AssertExpectations(t)
}

func TestRegisterPodToolsWithFactory(t *testing.T) {
	// Setup mock server and cluster manager
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()
	mockFactory := new(testmocks.MockPodFactory)

	// Expect tool registrations - use the correct type match
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(4)

	// Call the function
	RegisterPodToolsWithFactory(mockServer, mockCM, mockFactory)

	// Verify that the expected methods were called
	mockServer.AssertExpectations(t)
}
