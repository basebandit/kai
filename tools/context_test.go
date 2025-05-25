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

func TestContextTools(t *testing.T) {
	t.Run("ListContexts", testListContextsHandler)
	t.Run("GetCurrentContext", testGetCurrentContextHandler)
	t.Run("SwitchContext", testSwitchContextHandler)
	t.Run("LoadKubeconfig", testLoadKubeconfigHandler)
	t.Run("DeleteContext", testDeleteContextHandler)
	t.Run("RenameContext", testRenameContextHandler)
	t.Run("DescribeContext", testDescribeContextHandler)
}

func testListContextsHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name: "NoContexts",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("ListContexts").Return([]*kai.ContextInfo{})
			},
			expectedOutput: "No contexts available",
		},
		{
			name: "SingleContext",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				contexts := []*kai.ContextInfo{
					{
						Name:      "test-context",
						Cluster:   "test-cluster",
						User:      "test-user",
						Namespace: "default",
						IsActive:  true,
					},
				}
				mockCM.On("ListContexts").Return(contexts)
			},
			expectedOutput: "Available contexts:\n* test-context\n  Cluster: test-cluster\n  User: test-user\n  Namespace: default\n\nTotal: 1 context(s)",
		},
		{
			name: "MultipleContexts",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				contexts := []*kai.ContextInfo{
					{
						Name:      "context1",
						Cluster:   "cluster1",
						User:      "user1",
						Namespace: "default",
						IsActive:  true,
					},
					{
						Name:      "context2",
						Cluster:   "cluster2",
						User:      "user2",
						Namespace: "kube-system",
						IsActive:  false,
					},
				}
				mockCM.On("ListContexts").Return(contexts)
			},
			expectedOutput: "Available contexts:\n* context1\n  Cluster: cluster1\n  User: user1\n  Namespace: default\n\n  context2\n  Cluster: cluster2\n  User: user2\n  Namespace: kube-system\n\nTotal: 2 context(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := listContextsHandler(mockCM)
			request := mcp.CallToolRequest{}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockCM.AssertExpectations(t)
		})
	}
}

func testGetCurrentContextHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name: "NoActiveContext",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentContext").Return("")
			},
			expectedOutput: "No active context",
		},
		{
			name: "ActiveContextWithInfo",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentContext").Return("test-context")
				contextInfo := &kai.ContextInfo{
					Name:      "test-context",
					Cluster:   "test-cluster",
					User:      "test-user",
					Namespace: "default",
					ServerURL: "https://example.com:6443",
				}
				mockCM.On("GetContextInfo", "test-context").Return(contextInfo, nil)
			},
			expectedOutput: "Current context: test-context\nCluster: test-cluster\nUser: test-user\nNamespace: default\nServer: https://example.com:6443",
		},
		{
			name: "ActiveContextInfoError",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentContext").Return("test-context")
				mockCM.On("GetContextInfo", "test-context").Return(nil, errors.New("context not found"))
			},
			expectedOutput: "Error getting context info: context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := getCurrentContextHandler(mockCM)
			request := mcp.CallToolRequest{}

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, result.Content[0].(mcp.TextContent).Text)
			mockCM.AssertExpectations(t)
		})
	}
}

func testSwitchContextHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'name' is missing",
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'name' must be a non-empty string",
		},
		{
			name: "SuccessfulSwitch",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("SetCurrentContext", "test-context").Return(nil)
			},
			expectedOutput: "Switched to context 'test-context'",
		},
		{
			name: "SwitchError",
			args: map[string]interface{}{"name": "nonexistent-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("SetCurrentContext", "nonexistent-context").Return(errors.New("cluster nonexistent-context not found"))
			},
			expectedOutput: "Failed to switch context: cluster nonexistent-context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := switchContextHandler(mockCM)
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
			mockCM.AssertExpectations(t)
		})
	}
}

func testLoadKubeconfigHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'name' is missing",
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'name' must be a non-empty string",
		},
		{
			name: "SuccessfulLoadDefaultPath",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("LoadKubeConfig", "test-context", "").Return(nil)
			},
			expectedOutput: "Successfully loaded kubeconfig from '~/.kube/config' as context 'test-context'",
		},
		{
			name: "SuccessfulLoadCustomPath",
			args: map[string]interface{}{"name": "test-context", "path": "/custom/path/config"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("LoadKubeConfig", "test-context", "/custom/path/config").Return(nil)
			},
			expectedOutput: "Successfully loaded kubeconfig from '/custom/path/config' as context 'test-context'",
		},
		{
			name: "LoadError",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("LoadKubeConfig", "test-context", "").Return(errors.New("file not found"))
			},
			expectedOutput: "Failed to load kubeconfig: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := loadKubeconfigHandler(mockCM)
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
			mockCM.AssertExpectations(t)
		})
	}
}

func testDeleteContextHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'name' is missing",
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'name' must be a non-empty string",
		},
		{
			name: "SuccessfulDelete",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("DeleteContext", "test-context").Return(nil)
			},
			expectedOutput: "Successfully deleted context 'test-context'",
		},
		{
			name: "DeleteError",
			args: map[string]interface{}{"name": "nonexistent-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("DeleteContext", "nonexistent-context").Return(errors.New("context nonexistent-context not found"))
			},
			expectedOutput: "Failed to delete context: context nonexistent-context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := deleteContextHandler(mockCM)
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
			mockCM.AssertExpectations(t)
		})
	}
}

func testRenameContextHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name:           "MissingOldName",
			args:           map[string]interface{}{"new_name": "new-context"},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'old_name' is missing",
		},
		{
			name:           "MissingNewName",
			args:           map[string]interface{}{"old_name": "old-context"},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'new_name' is missing",
		},
		{
			name:           "EmptyOldName",
			args:           map[string]interface{}{"old_name": "", "new_name": "new-context"},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'old_name' must be a non-empty string",
		},
		{
			name:           "EmptyNewName",
			args:           map[string]interface{}{"old_name": "old-context", "new_name": ""},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'new_name' must be a non-empty string",
		},
		{
			name: "SuccessfulRename",
			args: map[string]interface{}{"old_name": "old-context", "new_name": "new-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("RenameContext", "old-context", "new-context").Return(nil)
			},
			expectedOutput: "Successfully renamed context 'old-context' to 'new-context'",
		},
		{
			name: "RenameError",
			args: map[string]interface{}{"old_name": "nonexistent-context", "new_name": "new-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("RenameContext", "nonexistent-context", "new-context").Return(errors.New("context nonexistent-context not found"))
			},
			expectedOutput: "Failed to rename context: context nonexistent-context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := renameContextHandler(mockCM)
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
			mockCM.AssertExpectations(t)
		})
	}
}

func testDescribeContextHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		setupMock      func(*testmocks.MockClusterManager)
		expectedOutput string
	}{
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Required parameter 'name' is missing",
		},
		{
			name:           "EmptyName",
			args:           map[string]interface{}{"name": ""},
			setupMock:      func(mockCM *testmocks.MockClusterManager) {},
			expectedOutput: "Parameter 'name' must be a non-empty string",
		},
		{
			name: "SuccessfulDescribeActive",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				contextInfo := &kai.ContextInfo{
					Name:       "test-context",
					Cluster:    "test-cluster",
					User:       "test-user",
					Namespace:  "default",
					ServerURL:  "https://example.com:6443",
					ConfigPath: "/home/user/.kube/config",
					IsActive:   true,
				}
				mockCM.On("GetContextInfo", "test-context").Return(contextInfo, nil)
			},
			expectedOutput: "Context: test-context\nCluster: test-cluster\nUser: test-user\nNamespace: default\nServer: https://example.com:6443\nConfig Path: /home/user/.kube/config\nStatus: active",
		},
		{
			name: "SuccessfulDescribeInactive",
			args: map[string]interface{}{"name": "test-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				contextInfo := &kai.ContextInfo{
					Name:       "test-context",
					Cluster:    "test-cluster",
					User:       "test-user",
					Namespace:  "default",
					ServerURL:  "https://example.com:6443",
					ConfigPath: "/home/user/.kube/config",
					IsActive:   false,
				}
				mockCM.On("GetContextInfo", "test-context").Return(contextInfo, nil)
			},
			expectedOutput: "Context: test-context\nCluster: test-cluster\nUser: test-user\nNamespace: default\nServer: https://example.com:6443\nConfig Path: /home/user/.kube/config\nStatus: inactive",
		},
		{
			name: "DescribeError",
			args: map[string]interface{}{"name": "nonexistent-context"},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetContextInfo", "nonexistent-context").Return(nil, errors.New("context nonexistent-context not found"))
			},
			expectedOutput: "Failed to get context info: context nonexistent-context not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tt.setupMock(mockCM)

			handler := describeContextHandler(mockCM)
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
			mockCM.AssertExpectations(t)
		})
	}
}

func TestRegisterContextTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(7)

	RegisterContextTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}
