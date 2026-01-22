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

type listConfigMapsTestCase struct {
	name           string
	args           map[string]interface{}
	expectedParams kai.ConfigMapParams
	mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockConfigMapFactory, *testmocks.MockConfigMap)
	expectedOutput string
}

type getConfigMapTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.ConfigMapParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockConfigMapFactory, *testmocks.MockConfigMap)
	expectedOutput          string
	expectConfigMapCreation bool
}

type createConfigMapTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.ConfigMapParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockConfigMapFactory, *testmocks.MockConfigMap)
	expectedOutput          string
	expectConfigMapCreation bool
}

type deleteConfigMapTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.ConfigMapParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockConfigMapFactory, *testmocks.MockConfigMap)
	expectedOutput          string
	expectConfigMapCreation bool
}

type updateConfigMapTestCase struct {
	name                    string
	args                    map[string]interface{}
	expectedParams          kai.ConfigMapParams
	mockSetup               func(*testmocks.MockClusterManager, *testmocks.MockConfigMapFactory, *testmocks.MockConfigMap)
	expectedOutput          string
	expectConfigMapCreation bool
}

func TestRegisterConfigMapTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)
	RegisterConfigMapTools(mockServer, mockClusterMgr)
	mockServer.AssertExpectations(t)
}

func TestRegisterConfigMapToolsWithFactory(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockClusterMgr := testmocks.NewMockClusterManager()
	mockFactory := testmocks.NewMockConfigMapFactory()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)
	RegisterConfigMapToolsWithFactory(mockServer, mockClusterMgr, mockFactory)
	mockServer.AssertExpectations(t)
}

func TestListConfigMapsHandler(t *testing.T) {
	testCases := []listConfigMapsTestCase{
		{
			name: "DefaultNamespace",
			args: map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("ConfigMaps in namespace %q:\n- configmap1\n- configmap2", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("ConfigMaps in namespace %q:", defaultNamespace),
		},
		{
			name: "AllNamespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockConfigMap.On("List", mock.Anything, mockCM, true, "").
					Return("ConfigMaps across all namespaces:\n- ns1/configmap1\n- ns2/configmap2", nil)
			},
			expectedOutput: "ConfigMaps across all namespaces:",
		},
		{
			name: "SpecificNamespace",
			args: map[string]interface{}{
				"namespace": testNamespace,
			},
			expectedParams: kai.ConfigMapParams{
				Namespace: testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockConfigMap.On("List", mock.Anything, mockCM, false, "").
					Return(fmt.Sprintf("ConfigMaps in namespace %q:\n- configmap1", testNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("ConfigMaps in namespace %q:", testNamespace),
		},
		{
			name: "WithLabelSelector",
			args: map[string]interface{}{
				"label_selector": "app=backend",
			},
			expectedParams: kai.ConfigMapParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("List", mock.Anything, mockCM, false, "app=backend").
					Return(fmt.Sprintf("ConfigMaps in namespace %q with label 'app=backend':\n- backend-config", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("ConfigMaps in namespace %q with label 'app=backend':", defaultNamespace),
		},
		{
			name: "Error",
			args: map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("List", mock.Anything, mockCM, false, "").
					Return("", errors.New(errConnectionFailed))
			},
			expectedOutput: errConnectionFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockConfigMapFactory()
			mockConfigMap := testmocks.NewMockConfigMap(tc.expectedParams)

			mockFactory.On("NewConfigMap", tc.expectedParams).Return(mockConfigMap)
			tc.mockSetup(mockCM, mockFactory, mockConfigMap)

			handler := listConfigMapsHandler(mockCM, mockFactory)

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
			mockConfigMap.AssertExpectations(t)
		})
	}
}

func TestGetConfigMapHandler(t *testing.T) {
	configMapName := "test-configmap"

	testCases := []getConfigMapTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": configMapName,
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("ConfigMap %q in namespace %q:\nData: key1=value1", configMapName, defaultNamespace), nil)
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q in namespace %q:", configMapName, defaultNamespace),
			expectConfigMapCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
			},
			expectedOutput:          errMissingName,
			expectConfigMapCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
			},
			expectedOutput:          errEmptyName,
			expectConfigMapCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "nonexistent-configmap",
			},
			expectedParams: kai.ConfigMapParams{
				Name:      "nonexistent-configmap",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Get", mock.Anything, mockCM).
					Return("", fmt.Errorf("ConfigMap %q not found in namespace %q", "nonexistent-configmap", defaultNamespace))
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q not found in namespace %q", "nonexistent-configmap", defaultNamespace),
			expectConfigMapCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockConfigMapFactory()

			var mockConfigMap *testmocks.MockConfigMap
			if tc.expectConfigMapCreation {
				mockConfigMap = testmocks.NewMockConfigMap(tc.expectedParams)
				mockFactory.On("NewConfigMap", tc.expectedParams).Return(mockConfigMap)
			}

			tc.mockSetup(mockCM, mockFactory, mockConfigMap)

			handler := getConfigMapHandler(mockCM, mockFactory)

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
			if mockConfigMap != nil {
				mockConfigMap.AssertExpectations(t)
			}
		})
	}
}

func TestCreateConfigMapHandler(t *testing.T) {
	configMapName := "test-configmap"

	testCases := []createConfigMapTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": configMapName,
				"data": map[string]interface{}{
					"key1": "value1",
				},
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"key1": "value1",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("ConfigMap %q created successfully in namespace %q", configMapName, defaultNamespace), nil)
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q created successfully in namespace %q", configMapName, defaultNamespace),
			expectConfigMapCreation: true,
		},
		{
			name: "WithLabelsAndAnnotations",
			args: map[string]interface{}{
				"name":      configMapName,
				"namespace": testNamespace,
				"data": map[string]interface{}{
					"config": "value",
				},
				"labels": map[string]interface{}{
					"app": "test",
				},
				"annotations": map[string]interface{}{
					"description": "Test ConfigMap",
				},
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"config": "value",
				},
				Labels: map[string]interface{}{
					"app": "test",
				},
				Annotations: map[string]interface{}{
					"description": "Test ConfigMap",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Create", mock.Anything, mockCM).
					Return(fmt.Sprintf("ConfigMap %q created successfully in namespace %q", configMapName, testNamespace), nil)
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q created successfully in namespace %q", configMapName, testNamespace),
			expectConfigMapCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
			},
			expectedOutput:          errMissingName,
			expectConfigMapCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": configMapName,
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Create", mock.Anything, mockCM).
					Return("", errors.New("namespace not found"))
			},
			expectedOutput:          "namespace not found",
			expectConfigMapCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockConfigMapFactory()

			var mockConfigMap *testmocks.MockConfigMap
			if tc.expectConfigMapCreation {
				mockConfigMap = testmocks.NewMockConfigMap(tc.expectedParams)
				mockFactory.On("NewConfigMap", tc.expectedParams).Return(mockConfigMap)
			}

			tc.mockSetup(mockCM, mockFactory, mockConfigMap)

			handler := createConfigMapHandler(mockCM, mockFactory)

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
			if mockConfigMap != nil {
				mockConfigMap.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteConfigMapHandler(t *testing.T) {
	configMapName := "test-configmap"

	testCases := []deleteConfigMapTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": configMapName,
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Delete", mock.Anything, mockCM).
					Return(fmt.Sprintf("ConfigMap %q deleted successfully from namespace %q", configMapName, defaultNamespace), nil)
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q deleted successfully from namespace %q", configMapName, defaultNamespace),
			expectConfigMapCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
			},
			expectedOutput:          errMissingName,
			expectConfigMapCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "nonexistent-configmap",
			},
			expectedParams: kai.ConfigMapParams{
				Name:      "nonexistent-configmap",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Delete", mock.Anything, mockCM).
					Return("", fmt.Errorf("ConfigMap %q not found", "nonexistent-configmap"))
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q not found", "nonexistent-configmap"),
			expectConfigMapCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockConfigMapFactory()

			var mockConfigMap *testmocks.MockConfigMap
			if tc.expectConfigMapCreation {
				mockConfigMap = testmocks.NewMockConfigMap(tc.expectedParams)
				mockFactory.On("NewConfigMap", tc.expectedParams).Return(mockConfigMap)
			}

			tc.mockSetup(mockCM, mockFactory, mockConfigMap)

			handler := deleteConfigMapHandler(mockCM, mockFactory)

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
			if mockConfigMap != nil {
				mockConfigMap.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateConfigMapHandler(t *testing.T) {
	configMapName := "test-configmap"

	testCases := []updateConfigMapTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": configMapName,
				"data": map[string]interface{}{
					"new-key": "new-value",
				},
			},
			expectedParams: kai.ConfigMapParams{
				Name:      configMapName,
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"new-key": "new-value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Update", mock.Anything, mockCM).
					Return(fmt.Sprintf("ConfigMap %q updated successfully in namespace %q", configMapName, defaultNamespace), nil)
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q updated successfully in namespace %q", configMapName, defaultNamespace),
			expectConfigMapCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.ConfigMapParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
			},
			expectedOutput:          errMissingName,
			expectConfigMapCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "nonexistent-configmap",
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			expectedParams: kai.ConfigMapParams{
				Name:      "nonexistent-configmap",
				Namespace: defaultNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockConfigMapFactory, mockConfigMap *testmocks.MockConfigMap) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockConfigMap.On("Update", mock.Anything, mockCM).
					Return("", fmt.Errorf("ConfigMap %q not found", "nonexistent-configmap"))
			},
			expectedOutput:          fmt.Sprintf("ConfigMap %q not found", "nonexistent-configmap"),
			expectConfigMapCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := testmocks.NewMockConfigMapFactory()

			var mockConfigMap *testmocks.MockConfigMap
			if tc.expectConfigMapCreation {
				mockConfigMap = testmocks.NewMockConfigMap(tc.expectedParams)
				mockFactory.On("NewConfigMap", tc.expectedParams).Return(mockConfigMap)
			}

			tc.mockSetup(mockCM, mockFactory, mockConfigMap)

			handler := updateConfigMapHandler(mockCM, mockFactory)

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
			if mockConfigMap != nil {
				mockConfigMap.AssertExpectations(t)
			}
		})
	}
}
