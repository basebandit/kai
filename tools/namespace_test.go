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

func TestNamespaceTools(t *testing.T) {
	t.Run("CreateNamespace", testCreateNamespaceHandler)
	t.Run("GetNamespace", testGetNamespaceHandler)
	t.Run("ListNamespaces", testListNamespacesHandler)
	t.Run("DeleteNamespace", testDeleteNamespaceHandler)
}

func testCreateNamespaceHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockNamespace)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockNS *testmocks.MockNamespace) {},
			expectedOutput: missingNameError,
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockNS *testmocks.MockNamespace) {},
			expectedOutput: emptyNameError,
		},
		{
			name: "SuccessfulCreate",
			args: map[string]interface{}{"name": testNamespace},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Create", mock.Anything, mock.Anything).Return("Namespace \"test-namespace\" created successfully", nil)
			},
			expectedOutput: "Namespace \"test-namespace\" created successfully",
		},
		{
			name: "CreateWithLabels",
			args: map[string]interface{}{
				"name": testNamespace,
				"labels": map[string]interface{}{
					"env": "test",
				},
			},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Create", mock.Anything, mock.Anything).Return("Namespace \"test-namespace\" created successfully", nil)
			},
			expectedOutput: "Namespace \"test-namespace\" created successfully",
		},
		{
			name: "CreateError",
			args: map[string]interface{}{"name": testNamespace},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Create", mock.Anything, mock.Anything).Return("", errors.New("namespace already exists"))
			},
			expectedOutput: "Failed to create namespace: namespace already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockNamespaceFactory()
			mockNS := testmocks.NewMockNamespace()

			tt.setupMock(mockNS)

			if tt.name != "MissingName" && tt.name != "EmptyName" {
				mockFactory.On("NewNamespace", mock.Anything).Return(mockNS)
			}

			handler := createNamespaceHandlerWithFactory(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockNS.AssertExpectations(t)
		})
	}
}

func testGetNamespaceHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockNamespace)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockNS *testmocks.MockNamespace) {},
			expectedOutput: missingNameError,
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockNS *testmocks.MockNamespace) {},
			expectedOutput: emptyNameError,
		},
		{
			name: "SuccessfulGet",
			args: map[string]interface{}{"name": testNamespace},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Get", mock.Anything, mock.Anything).Return("Namespace: test-namespace\nStatus: Active", nil)
			},
			expectedOutput: "Namespace: test-namespace\nStatus: Active",
		},
		{
			name: "GetError",
			args: map[string]interface{}{"name": "nonexistent"},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Get", mock.Anything, mock.Anything).Return("", errors.New("namespace 'nonexistent' not found"))
			},
			expectedOutput: "Failed to get namespace: namespace 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockNamespaceFactory()
			mockNS := testmocks.NewMockNamespace()

			tt.setupMock(mockNS)

			if tt.name != "MissingName" && tt.name != "EmptyName" {
				mockFactory.On("NewNamespace", mock.Anything).Return(mockNS)
			}

			handler := getNamespaceHandlerWithFactory(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockNS.AssertExpectations(t)
		})
	}
}

func testListNamespacesHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockNamespace)
		expectedOutput string
	}{
		{
			name: "ListAllNamespaces",
			args: map[string]interface{}{},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("List", mock.Anything, mock.Anything, "").Return("Namespaces:\n• default\n• kube-system\n\nTotal: 2 namespace(s)", nil)
			},
			expectedOutput: "Namespaces:\n• default\n• kube-system\n\nTotal: 2 namespace(s)",
		},
		{
			name: "ListWithLabelSelector",
			args: map[string]interface{}{"label_selector": "env=prod"},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("List", mock.Anything, mock.Anything, "env=prod").Return("Namespaces matching label selector 'env=prod':\n• prod-namespace\n\nTotal: 1 namespace(s)", nil)
			},
			expectedOutput: "Namespaces matching label selector 'env=prod':\n• prod-namespace\n\nTotal: 1 namespace(s)",
		},
		{
			name: "ListError",
			args: map[string]interface{}{},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("List", mock.Anything, mock.Anything, "").Return("", errors.New("connection failed"))
			},
			expectedOutput: "Failed to list namespaces: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockNamespaceFactory()
			mockNS := testmocks.NewMockNamespace()

			tt.setupMock(mockNS)
			mockFactory.On("NewNamespace", mock.Anything).Return(mockNS)

			handler := listNamespacesHandlerWithFactory(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockNS.AssertExpectations(t)
		})
	}
}

func testDeleteNamespaceHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockNamespace)
		expectedOutput string
	}{
		{
			name: "DeleteByName",
			args: map[string]interface{}{"name": testNamespace},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Delete", mock.Anything, mock.Anything).Return("Namespace \"test-namespace\" deleted successfully", nil)
			},
			expectedOutput: "Namespace \"test-namespace\" deleted successfully",
		},
		{
			name: "DeleteByLabels",
			args: map[string]interface{}{
				"labels": map[string]interface{}{
					"env": "test",
				},
			},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Delete", mock.Anything, mock.Anything).Return("Deleted 2 namespaces with label selector \"env=test\"", nil)
			},
			expectedOutput: "Deleted 2 namespaces with label selector \"env=test\"",
		},
		{
			name:           "MissingNameAndLabels",
			args:           map[string]interface{}{},
			setupMock:      func(mockNS *testmocks.MockNamespace) {},
			expectedOutput: "Either namespace name or label selector must be provided",
		},
		{
			name: "DeleteError",
			args: map[string]interface{}{"name": "nonexistent"},
			setupMock: func(mockNS *testmocks.MockNamespace) {
				mockNS.On("Delete", mock.Anything, mock.Anything).Return("", errors.New("namespace not found"))
			},
			expectedOutput: "Failed to delete namespace: namespace not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockNamespaceFactory()
			mockNS := testmocks.NewMockNamespace()

			tt.setupMock(mockNS)

			if tt.name != "MissingNameAndLabels" {
				mockFactory.On("NewNamespace", mock.Anything).Return(mockNS)
			}

			handler := deleteNamespaceHandlerWithFactory(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockNS.AssertExpectations(t)
		})
	}
}

func TestRegisterNamespaceTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(4)

	RegisterNamespaceTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

// Helper functions for testing with factory pattern
func createNamespaceHandlerWithFactory(cm kai.ClusterManager, factory testmocks.NamespaceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(missingNameError), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(emptyNameError), nil
		}

		params := kai.NamespaceParams{
			Name: name,
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		namespace := factory.NewNamespace(params)

		result, err := namespace.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getNamespaceHandlerWithFactory(cm kai.ClusterManager, factory testmocks.NamespaceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(missingNameError), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(emptyNameError), nil
		}

		params := kai.NamespaceParams{
			Name: name,
		}

		namespace := factory.NewNamespace(params)

		result, err := namespace.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listNamespacesHandlerWithFactory(cm kai.ClusterManager, factory testmocks.NamespaceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		labelSelector := ""
		if selectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = selectorArg
		}

		params := kai.NamespaceParams{}
		namespace := factory.NewNamespace(params)

		result, err := namespace.List(ctx, cm, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list namespaces: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteNamespaceHandlerWithFactory(cm kai.ClusterManager, factory testmocks.NamespaceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := kai.NamespaceParams{}

		if nameArg, ok := request.Params.Arguments["name"].(string); ok && nameArg != "" {
			params.Name = nameArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if params.Name == "" && len(params.Labels) == 0 {
			return mcp.NewToolResultText("Either namespace name or label selector must be provided"), nil
		}

		namespace := factory.NewNamespace(params)

		result, err := namespace.Delete(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete namespace: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
