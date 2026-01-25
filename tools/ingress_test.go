package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateIngressHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockIngressFactory, *testmocks.MockIngress)
		expectedOutput string
	}{
		{
			name: "Create basic Ingress",
			args: map[string]any{
				"name":      "test-ingress",
				"namespace": defaultNamespace,
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"path_type":    "Prefix",
								"service_name": "backend",
								"service_port": float64(80),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Create", mock.Anything, mockCM).Return("Ingress \"test-ingress\" created successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" created successfully",
		},
		{
			name: "Create Ingress with TLS",
			args: map[string]any{
				"name":          "tls-ingress",
				"namespace":     defaultNamespace,
				"ingress_class": "nginx",
				"rules": []any{
					map[string]any{
						"host": "secure.example.com",
						"paths": []any{
							map[string]any{
								"path":         "/api",
								"path_type":    "Exact",
								"service_name": "api-service",
								"service_port": "http",
							},
						},
					},
				},
				"tls": []any{
					map[string]any{
						"hosts":       []any{"secure.example.com"},
						"secret_name": "tls-secret",
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Create", mock.Anything, mockCM).Return("Ingress \"tls-ingress\" created successfully", nil)
			},
			expectedOutput: "Ingress \"tls-ingress\" created successfully",
		},
		{
			name: "Create Ingress with labels and annotations",
			args: map[string]any{
				"name":      "annotated-ingress",
				"namespace": defaultNamespace,
				"labels":    map[string]any{"app": "web"},
				"annotations": map[string]any{
					"nginx.ingress.kubernetes.io/rewrite-target": "/",
				},
				"rules": []any{
					map[string]any{
						"host": "app.example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"service_name": "web-service",
								"service_port": 8080,
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Create", mock.Anything, mockCM).Return("Ingress \"annotated-ingress\" created successfully", nil)
			},
			expectedOutput: "Ingress \"annotated-ingress\" created successfully",
		},
		{
			name: "Missing Ingress name",
			args: map[string]any{
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"service_name": "backend",
								"service_port": float64(80),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errMissingName,
		},
		{
			name: "Empty Ingress name",
			args: map[string]any{
				"name": "",
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"service_name": "backend",
								"service_port": float64(80),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errEmptyName,
		},
		{
			name: "Missing rules",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: "Required parameter 'rules' is missing",
		},
		{
			name: "Empty rules",
			args: map[string]any{
				"name":  "test-ingress",
				"rules": []any{},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: "Parameter 'rules' must be a non-empty array",
		},
		{
			name: "Invalid rule format",
			args: map[string]any{
				"name":  "test-ingress",
				"rules": []any{"invalid"},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput: "Invalid rules",
		},
		{
			name: "Missing service name in path",
			args: map[string]any{
				"name": "test-ingress",
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"service_port": float64(80),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput: "Invalid rules",
		},
		{
			name: "Create error",
			args: map[string]any{
				"name": "test-ingress",
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"paths": []any{
							map[string]any{
								"path":         "/",
								"service_name": "backend",
								"service_port": float64(80),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Create", mock.Anything, mockCM).Return("", assert.AnError)
			},
			expectedOutput: "Failed to create Ingress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockIngressFactory{}
			mockIngress := &testmocks.MockIngress{}
			tt.mockSetup(mockCM, mockFactory, mockIngress)

			handler := createIngressHandler(mockCM, mockFactory)
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
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockIngress.AssertExpectations(t)
		})
	}
}

func TestGetIngressHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockIngressFactory, *testmocks.MockIngress)
		expectedOutput string
	}{
		{
			name: "Get existing Ingress",
			args: map[string]any{
				"name":      "test-ingress",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Get", mock.Anything, mockCM).Return("Ingress: test-ingress\nNamespace: default", nil)
			},
			expectedOutput: "Ingress: test-ingress",
		},
		{
			name: "Get Ingress with default namespace",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Get", mock.Anything, mockCM).Return("Ingress: test-ingress", nil)
			},
			expectedOutput: "Ingress: test-ingress",
		},
		{
			name: "Missing Ingress name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errMissingName,
		},
		{
			name: "Empty Ingress name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errEmptyName,
		},
		{
			name: "Get error",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Get", mock.Anything, mockCM).Return("", assert.AnError)
			},
			expectedOutput: "Failed to get Ingress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockIngressFactory{}
			mockIngress := &testmocks.MockIngress{}
			tt.mockSetup(mockCM, mockFactory, mockIngress)

			handler := getIngressHandler(mockCM, mockFactory)
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
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockIngress.AssertExpectations(t)
		})
	}
}

func TestListIngressesHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockIngressFactory, *testmocks.MockIngress)
		expectedOutput string
	}{
		{
			name: "List Ingresses in default namespace",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("List", mock.Anything, mockCM, false, "").Return("Ingresses in namespace default:\ningress1\ningress2", nil)
			},
			expectedOutput: "Ingresses in namespace default",
		},
		{
			name: "List Ingresses in specific namespace",
			args: map[string]any{
				"namespace": testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("List", mock.Anything, mockCM, false, "").Return("Ingresses in namespace test-namespace:\ningress3", nil)
			},
			expectedOutput: "Ingresses in namespace test-namespace",
		},
		{
			name: "List Ingresses across all namespaces",
			args: map[string]any{
				"all_namespaces": true,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("List", mock.Anything, mockCM, true, "").Return("Ingresses across all namespaces:\ndefault/ingress1\ntest/ingress2", nil)
			},
			expectedOutput: "Ingresses across all namespaces",
		},
		{
			name: "List Ingresses with label selector",
			args: map[string]any{
				"label_selector": "app=nginx",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("List", mock.Anything, mockCM, false, "app=nginx").Return("Ingresses matching app=nginx:\ningress1", nil)
			},
			expectedOutput: "Ingresses matching app=nginx",
		},
		{
			name: "List error",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("List", mock.Anything, mockCM, false, "").Return("", assert.AnError)
			},
			expectedOutput: "Failed to list Ingresses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockIngressFactory{}
			mockIngress := &testmocks.MockIngress{}
			tt.mockSetup(mockCM, mockFactory, mockIngress)

			handler := listIngressesHandler(mockCM, mockFactory)
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
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockIngress.AssertExpectations(t)
		})
	}
}

func TestUpdateIngressHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockIngressFactory, *testmocks.MockIngress)
		expectedOutput string
	}{
		{
			name: "Update Ingress class",
			args: map[string]any{
				"name":          "test-ingress",
				"namespace":     defaultNamespace,
				"ingress_class": "nginx",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Update", mock.Anything, mockCM).Return("Ingress \"test-ingress\" updated successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" updated successfully",
		},
		{
			name: "Update Ingress rules",
			args: map[string]any{
				"name": "test-ingress",
				"rules": []any{
					map[string]any{
						"host": "new.example.com",
						"paths": []any{
							map[string]any{
								"path":         "/api",
								"path_type":    "Prefix",
								"service_name": "api-service",
								"service_port": float64(8080),
							},
						},
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Update", mock.Anything, mockCM).Return("Ingress \"test-ingress\" updated successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" updated successfully",
		},
		{
			name: "Update Ingress with TLS",
			args: map[string]any{
				"name": "test-ingress",
				"tls": []any{
					map[string]any{
						"hosts":       []any{"example.com"},
						"secret_name": "tls-secret",
					},
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Update", mock.Anything, mockCM).Return("Ingress \"test-ingress\" updated successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" updated successfully",
		},
		{
			name: "Update Ingress labels and annotations",
			args: map[string]any{
				"name":        "test-ingress",
				"labels":      map[string]any{"env": "prod"},
				"annotations": map[string]any{"key": "value"},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Update", mock.Anything, mockCM).Return("Ingress \"test-ingress\" updated successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" updated successfully",
		},
		{
			name: "Missing Ingress name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errMissingName,
		},
		{
			name: "Empty Ingress name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errEmptyName,
		},
		{
			name: "Invalid rules format",
			args: map[string]any{
				"name":  "test-ingress",
				"rules": []any{"invalid"},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput: "Invalid rules",
		},
		{
			name: "Update error",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Update", mock.Anything, mockCM).Return("", assert.AnError)
			},
			expectedOutput: "Failed to update Ingress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockIngressFactory{}
			mockIngress := &testmocks.MockIngress{}
			tt.mockSetup(mockCM, mockFactory, mockIngress)

			handler := updateIngressHandler(mockCM, mockFactory)
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
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockIngress.AssertExpectations(t)
		})
	}
}

func TestDeleteIngressHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockIngressFactory, *testmocks.MockIngress)
		expectedOutput string
	}{
		{
			name: "Delete existing Ingress",
			args: map[string]any{
				"name":      "test-ingress",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Delete", mock.Anything, mockCM).Return("Ingress \"test-ingress\" deleted successfully from namespace \"default\"", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" deleted successfully",
		},
		{
			name: "Delete Ingress with default namespace",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Delete", mock.Anything, mockCM).Return("Ingress \"test-ingress\" deleted successfully", nil)
			},
			expectedOutput: "Ingress \"test-ingress\" deleted successfully",
		},
		{
			name: "Missing Ingress name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errMissingName,
		},
		{
			name: "Empty Ingress name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
			},
			expectedOutput: errEmptyName,
		},
		{
			name: "Delete error",
			args: map[string]any{
				"name": "test-ingress",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockIngressFactory, mockIngress *testmocks.MockIngress) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewIngress", mock.Anything).Return(mockIngress)
				mockIngress.On("Delete", mock.Anything, mockCM).Return("", assert.AnError)
			},
			expectedOutput: "Failed to delete Ingress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockIngressFactory{}
			mockIngress := &testmocks.MockIngress{}
			tt.mockSetup(mockCM, mockFactory, mockIngress)

			handler := deleteIngressHandler(mockCM, mockFactory)
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
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockIngress.AssertExpectations(t)
		})
	}
}

func TestNewDefaultIngressFactory(t *testing.T) {
	factory := NewDefaultIngressFactory()
	assert.NotNil(t, factory)
}

func TestDefaultIngressFactoryNewIngress(t *testing.T) {
	factory := NewDefaultIngressFactory()

	params := kai.IngressParams{
		Name:             "test-ingress",
		Namespace:        "default",
		IngressClassName: "nginx",
		Labels:           map[string]interface{}{"app": "web"},
		Annotations:      map[string]interface{}{"key": "value"},
		Rules: []kai.IngressRule{
			{
				Host: "example.com",
				Paths: []kai.IngressPath{
					{
						Path:        "/",
						PathType:    "Prefix",
						ServiceName: "backend",
						ServicePort: 80,
					},
				},
			},
		},
		TLS: []kai.IngressTLS{
			{
				Hosts:      []string{"example.com"},
				SecretName: "tls-secret",
			},
		},
	}

	ingress := factory.NewIngress(params)
	assert.NotNil(t, ingress)
}

func TestRegisterIngressTools(t *testing.T) {
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	RegisterIngressTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

func TestRegisterIngressToolsWithFactory(t *testing.T) {
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()
	mockFactory := new(testmocks.MockIngressFactory)

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	RegisterIngressToolsWithFactory(mockServer, mockCM, mockFactory)

	mockServer.AssertExpectations(t)
}

func TestParseIngressRules(t *testing.T) {
	t.Run("Valid rules", func(t *testing.T) {
		rulesSlice := []interface{}{
			map[string]interface{}{
				"host": "example.com",
				"paths": []interface{}{
					map[string]interface{}{
						"path":         "/",
						"path_type":    "Prefix",
						"service_name": "backend",
						"service_port": float64(80),
					},
				},
			},
		}

		rules, err := parseIngressRules(rulesSlice)
		assert.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "example.com", rules[0].Host)
		assert.Len(t, rules[0].Paths, 1)
		assert.Equal(t, "/", rules[0].Paths[0].Path)
		assert.Equal(t, "backend", rules[0].Paths[0].ServiceName)
	})

	t.Run("Missing paths", func(t *testing.T) {
		rulesSlice := []interface{}{
			map[string]interface{}{
				"host": "example.com",
			},
		}

		_, err := parseIngressRules(rulesSlice)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'paths' must be a non-empty array")
	})

	t.Run("Invalid path format", func(t *testing.T) {
		rulesSlice := []interface{}{
			map[string]interface{}{
				"host":  "example.com",
				"paths": []interface{}{"invalid"},
			},
		}

		_, err := parseIngressRules(rulesSlice)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be an object")
	})

	t.Run("Missing service port", func(t *testing.T) {
		rulesSlice := []interface{}{
			map[string]interface{}{
				"host": "example.com",
				"paths": []interface{}{
					map[string]interface{}{
						"path":         "/",
						"service_name": "backend",
					},
				},
			},
		}

		_, err := parseIngressRules(rulesSlice)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'service_port' is required")
	})
}

func TestParseIngressTLS(t *testing.T) {
	t.Run("Valid TLS", func(t *testing.T) {
		tlsSlice := []interface{}{
			map[string]interface{}{
				"hosts":       []interface{}{"example.com", "www.example.com"},
				"secret_name": "tls-secret",
			},
		}

		tls, err := parseIngressTLS(tlsSlice)
		assert.NoError(t, err)
		assert.Len(t, tls, 1)
		assert.Equal(t, []string{"example.com", "www.example.com"}, tls[0].Hosts)
		assert.Equal(t, "tls-secret", tls[0].SecretName)
	})

	t.Run("Invalid TLS format", func(t *testing.T) {
		tlsSlice := []interface{}{"invalid"}

		_, err := parseIngressTLS(tlsSlice)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be an object")
	})
}
