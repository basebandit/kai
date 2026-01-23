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

type createSecretTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.SecretParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockSecretFactory, *testmocks.MockSecret)
	expectedOutput          string
	expectSecretCreation    bool
}

type getSecretTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.SecretParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockSecretFactory, *testmocks.MockSecret)
	expectedOutput          string
	expectSecretCreation    bool
}

type listSecretsTestCase struct {
	name           string
	args           map[string]interface{}
	expectedParams kai.SecretParams
	mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockSecretFactory, *testmocks.MockSecret)
	expectedOutput string
}

type deleteSecretTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.SecretParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockSecretFactory, *testmocks.MockSecret)
	expectedOutput          string
	expectSecretCreation    bool
}

func TestRegisterSecretTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)
	RegisterSecretTools(mockServer, mockClusterMgr)
	mockServer.AssertExpectations(t)
}

func TestRegisterSecretToolsWithFactory(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()
	mockFactory := testmocks.NewMockSecretFactory()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)
	RegisterSecretToolsWithFactory(mockServer, mockClusterMgr, mockFactory)
	mockServer.AssertExpectations(t)
}

func TestCreateSecretHandler(t *testing.T) {
	testCases := []createSecretTestCase{
		{
			name: "Create basic Secret",
			args: map[string]interface{}{
				"name": testSecretName,
				"data": map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Create Secret with type",
			args: map[string]interface{}{
				"name": testSecretName,
				"type": tlsSecretType,
				"data": map[string]interface{}{
					"tls.crt": "cert-data",
					"tls.key": "key-data",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Type:      tlsSecretType,
				Data: map[string]interface{}{
					"tls.crt": "cert-data",
					"tls.key": "key-data",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Create Secret with string_data",
			args: map[string]interface{}{
				"name": testSecretName,
				"string_data": map[string]interface{}{
					"api-key": "my-api-key",
					"token":   "my-token",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				StringData: map[string]interface{}{
					"api-key": "my-api-key",
					"token":   "my-token",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Create Secret with labels and annotations",
			args: map[string]interface{}{
				"name": testSecretName,
				"data": map[string]interface{}{
					"key": "value",
				},
				"labels": map[string]interface{}{
					"app": "backend",
					"env": "production",
				},
				"annotations": map[string]interface{}{
					"description": "Production database credentials",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
				Labels: map[string]interface{}{
					"app": "backend",
					"env": "production",
				},
				Annotations: map[string]interface{}{
					"description": "Production database credentials",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Create Secret in specific namespace",
			args: map[string]interface{}{
				"name":      testSecretName,
				"namespace": testNamespace,
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, testNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q created successfully in namespace %q", testSecretName, testNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Invalid secret type",
			args: map[string]interface{}{
				"name": testSecretName,
				"type": invalidSecretType,
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:       "invalid secret type",
			expectSecretCreation: false,
		},
		{
			name:         "Missing Secret name",
			args:         map[string]interface{}{},
			mockSetup:    func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errMissingName,
			expectSecretCreation: false,
		},
		{
			name: "Empty Secret name",
			args: map[string]interface{}{
				"name": "",
			},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errEmptyName,
			expectSecretCreation: false,
		},
		{
			name: "Secret creation failure",
			args: map[string]interface{}{
				"name": testSecretName,
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Create", mock.Anything, mockCM).
					Return("", errors.New("namespace not found"))
			},
			expectedOutput:       "Failed to create Secret: namespace not found",
			expectSecretCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockSecretFactory()
			mockSecret := testmocks.NewMockSecret(tc.expectedParams)

			tc.mockSetup(mockCM, mockFactory, mockSecret)

			if tc.expectSecretCreation {
				mockFactory.On("NewSecret", mock.MatchedBy(func(params kai.SecretParams) bool {
					return params.Name == tc.expectedParams.Name &&
						params.Namespace == tc.expectedParams.Namespace
				})).Return(mockSecret)
			}

			handler := createSecretHandler(mockCM, mockFactory)
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
			mockSecret.AssertExpectations(t)
		})
	}
}

func TestGetSecretHandler(t *testing.T) {
	testCases := []getSecretTestCase{
		{
			name: "Get existing Secret",
			args: map[string]interface{}{
				"name": testSecretName,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret: %s\nNamespace: %s\nType: Opaque", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret: %s", testSecretName),
			expectSecretCreation: true,
		},
		{
			name: "Get Secret from specific namespace",
			args: map[string]interface{}{
				"name":      testSecretName,
				"namespace": testNamespace,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret: %s\nNamespace: %s\nType: Opaque", testSecretName, testNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret: %s", testSecretName),
			expectSecretCreation: true,
		},
		{
			name:                 "Missing Secret name",
			args:                 map[string]interface{}{},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errMissingName,
			expectSecretCreation: false,
		},
		{
			name: "Empty Secret name",
			args: map[string]interface{}{
				"name": "",
			},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errEmptyName,
			expectSecretCreation: false,
		},
		{
			name: "Secret not found",
			args: map[string]interface{}{
				"name": testSecretName,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Get", mock.Anything, mockCM).
					Return("", errors.New("secret not found"))
			},
			expectedOutput:       "Failed to get Secret: secret not found",
			expectSecretCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockSecretFactory()
			mockSecret := testmocks.NewMockSecret(tc.expectedParams)

			tc.mockSetup(mockCM, mockFactory, mockSecret)

			if tc.expectSecretCreation {
				mockFactory.On("NewSecret", mock.MatchedBy(func(params kai.SecretParams) bool {
					return params.Name == tc.expectedParams.Name &&
						params.Namespace == tc.expectedParams.Namespace
				})).Return(mockSecret)
			}

			handler := getSecretHandler(mockCM, mockFactory)
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
			mockSecret.AssertExpectations(t)
		})
	}
}

func TestListSecretsHandler(t *testing.T) {
	testCases := []listSecretsTestCase{
		{
			name: "List Secrets in default namespace",
			args: map[string]interface{}{},
			expectedParams: kai.SecretParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Secrets in namespace %q:\n- secret1\n- secret2", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Secrets in namespace %q:", defaultNamespace),
		},
		{
			name: "List Secrets across all namespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.SecretParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockSecret.On("List", mock.Anything, mockCM, true, "").
					Return("Secrets across all namespaces:\n- ns1/secret1\n- ns2/secret2", nil)
			},
			expectedOutput: "Secrets across all namespaces:",
		},
		{
			name: "List Secrets in specific namespace",
			args: map[string]interface{}{
				"namespace": testNamespace,
			},
			expectedParams: kai.SecretParams{
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockSecret.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("Secrets in namespace %q:\n- secret1", testNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Secrets in namespace %q:", testNamespace),
		},
		{
			name: "List Secrets with label selector",
			args: map[string]interface{}{
				"label_selector": "app=backend",
			},
			expectedParams: kai.SecretParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("List", mock.Anything, mockCM, false, "app=backend").
					Return(fmt.Sprintf("Secrets in namespace %q with label 'app=backend':\n- backend-secret", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Secrets in namespace %q with label 'app=backend':", defaultNamespace),
		},
		{
			name: "List Secrets error",
			args: map[string]interface{}{},
			expectedParams: kai.SecretParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("List", mock.Anything, mockCM, false, "").
					Return("", errors.New("connection failed"))
			},
			expectedOutput: "Failed to list Secrets: connection failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockSecretFactory()
			mockSecret := testmocks.NewMockSecret(tc.expectedParams)

			tc.mockSetup(mockCM, mockFactory, mockSecret)

			mockFactory.On("NewSecret", mock.MatchedBy(func(params kai.SecretParams) bool {
				return params.Namespace == tc.expectedParams.Namespace
			})).Return(mockSecret)

			handler := listSecretsHandler(mockCM, mockFactory)
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
			mockSecret.AssertExpectations(t)
		})
	}
}

func TestDeleteSecretHandler(t *testing.T) {
	testCases := []deleteSecretTestCase{
		{
			name: "Delete existing Secret",
			args: map[string]interface{}{
				"name": testSecretName,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q deleted successfully from namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q deleted successfully from namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Delete Secret from specific namespace",
			args: map[string]interface{}{
				"name":      testSecretName,
				"namespace": testNamespace,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q deleted successfully from namespace %q", testSecretName, testNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q deleted successfully from namespace %q", testSecretName, testNamespace),
			expectSecretCreation: true,
		},
		{
			name:                 "Missing Secret name",
			args:                 map[string]interface{}{},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errMissingName,
			expectSecretCreation: false,
		},
		{
			name: "Empty Secret name",
			args: map[string]interface{}{
				"name": "",
			},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errEmptyName,
			expectSecretCreation: false,
		},
		{
			name: "Secret not found",
			args: map[string]interface{}{
				"name": testSecretName,
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Delete", mock.Anything, mockCM).
					Return("", errors.New("secret not found"))
			},
			expectedOutput:       "Failed to delete Secret: secret not found",
			expectSecretCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockSecretFactory()
			mockSecret := testmocks.NewMockSecret(tc.expectedParams)

			tc.mockSetup(mockCM, mockFactory, mockSecret)

			if tc.expectSecretCreation {
				mockFactory.On("NewSecret", mock.MatchedBy(func(params kai.SecretParams) bool {
					return params.Name == tc.expectedParams.Name &&
						params.Namespace == tc.expectedParams.Namespace
				})).Return(mockSecret)
			}

			handler := deleteSecretHandler(mockCM, mockFactory)
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
			mockSecret.AssertExpectations(t)
		})
	}
}

func TestUpdateSecretHandler(t *testing.T) {
	type updateSecretTestCase struct {
		name                 string
		args                 map[string]interface{}
		expectedParams       kai.SecretParams
		mockSetup            func(*testmocks.MockClusterManager, *testmocks.MockSecretFactory, *testmocks.MockSecret)
		expectedOutput       string
		expectSecretCreation bool
	}

	testCases := []updateSecretTestCase{
		{
			name: "Update Secret data",
			args: map[string]interface{}{
				"name": testSecretName,
				"data": map[string]interface{}{
					"username": "newuser",
					"password": "newpass",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"username": "newuser",
					"password": "newpass",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Update Secret with new type",
			args: map[string]interface{}{
				"name": testSecretName,
				"type": tlsSecretType,
				"data": map[string]interface{}{
					"tls.crt": "new-cert",
					"tls.key": "new-key",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Type:      tlsSecretType,
				Data: map[string]interface{}{
					"tls.crt": "new-cert",
					"tls.key": "new-key",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Update Secret labels",
			args: map[string]interface{}{
				"name": testSecretName,
				"labels": map[string]interface{}{
					"env":     "staging",
					"version": "v2",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Labels: map[string]interface{}{
					"env":     "staging",
					"version": "v2",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, defaultNamespace),
			expectSecretCreation: true,
		},
		{
			name: "Update Secret in specific namespace",
			args: map[string]interface{}{
				"name":      testSecretName,
				"namespace": testNamespace,
				"data": map[string]interface{}{
					"key": "updated-value",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "updated-value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, testNamespace), nil)
			},
			expectedOutput:       fmt.Sprintf("Secret %q updated successfully in namespace %q", testSecretName, testNamespace),
			expectSecretCreation: true,
		},
		{
			name:                 "Missing Secret name for update",
			args:                 map[string]interface{}{},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errMissingName,
			expectSecretCreation: false,
		},
		{
			name: "Empty Secret name for update",
			args: map[string]interface{}{
				"name": "",
			},
			mockSetup:            func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {},
			expectedOutput:       errEmptyName,
			expectSecretCreation: false,
		},
		{
			name: "Invalid secret type for update",
			args: map[string]interface{}{
				"name": testSecretName,
				"type": invalidSecretType,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:       "invalid secret type",
			expectSecretCreation: false,
		},
		{
			name: "Secret not found for update",
			args: map[string]interface{}{
				"name": testSecretName,
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			expectedParams: kai.SecretParams{
				Name:      testSecretName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockSecretFactory, mockSecret *testmocks.MockSecret) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockSecret.On("Update", mock.Anything, mockCM).
					Return("", errors.New("secret not found"))
			},
			expectedOutput:       "Failed to update Secret: secret not found",
			expectSecretCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockSecretFactory()
			mockSecret := testmocks.NewMockSecret(tc.expectedParams)

			tc.mockSetup(mockCM, mockFactory, mockSecret)

			if tc.expectSecretCreation {
				mockFactory.On("NewSecret", mock.MatchedBy(func(params kai.SecretParams) bool {
					return params.Name == tc.expectedParams.Name &&
						params.Namespace == tc.expectedParams.Namespace
				})).Return(mockSecret)
			}

			handler := updateSecretHandler(mockCM, mockFactory)
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
			mockSecret.AssertExpectations(t)
		})
	}
}

func TestValidateSecretType(t *testing.T) {
	testCases := []struct {
		name        string
		secretType  string
		expectError bool
	}{
		{"Valid Opaque type", "Opaque", false},
		{"Valid TLS type", "kubernetes.io/tls", false},
		{"Valid dockerconfigjson type", "kubernetes.io/dockerconfigjson", false},
		{"Valid service account token", "kubernetes.io/service-account-token", false},
		{"Valid dockercfg type", "kubernetes.io/dockercfg", false},
		{"Valid basic auth type", "kubernetes.io/basic-auth", false},
		{"Valid SSH auth type", "kubernetes.io/ssh-auth", false},
		{"Invalid type", "invalid/type", true},
		{"Invalid type with special chars", "my-custom-type!", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSecretType(tc.secretType)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid secret type")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
