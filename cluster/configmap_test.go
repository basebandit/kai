package cluster

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestConfigMapOperations(t *testing.T) {
	t.Run("CreateConfigMap", testCreateConfigMap)
	t.Run("GetConfigMap", testGetConfigMap)
	t.Run("ListConfigMaps", testListConfigMaps)
	t.Run("DeleteConfigMap", testDeleteConfigMap)
	t.Run("UpdateConfigMap", testUpdateConfigMap)
}

func createConfigMapObj(name, namespace string, data map[string]string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}

func createNamespaceForTest(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
}

func testCreateConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		configMap    ConfigMap
		setupObjects []runtime.Object
		expectedText string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Create basic ConfigMap",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key1": "value1",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: fmt.Sprintf("ConfigMap %q created successfully in namespace %q", configMapName, testNamespace),
			expectError:  false,
		},
		{
			name: "Create ConfigMap with labels and annotations",
			configMap: ConfigMap{
				Name:      "labeled-configmap",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"config": "data",
				},
				Labels: map[string]interface{}{
					"app":  "test",
					"env":  "dev",
				},
				Annotations: map[string]interface{}{
					"description": "Test ConfigMap",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: "ConfigMap \"labeled-configmap\" created successfully",
			expectError:  false,
		},
		{
			name: "Create ConfigMap with binary data",
			configMap: ConfigMap{
				Name:      "binary-configmap",
				Namespace: testNamespace,
				BinaryData: map[string]interface{}{
					"binary-key": "binary-value",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: "ConfigMap \"binary-configmap\" created successfully",
			expectError:  false,
		},
		{
			name: "Namespace not found",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: nonexistentNS,
				Data: map[string]interface{}{
					"key1": "value1",
				},
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     fmt.Sprintf("namespace %q not found", nonexistentNS),
		},
		{
			name: "Missing ConfigMap name",
			configMap: ConfigMap{
				Name:      "",
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "ConfigMap name is required",
		},
		{
			name: "Missing namespace",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: "",
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "namespace is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.configMap.Create(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedText)

				fakeClient := cm.clients[testCluster]
				configMap, err := fakeClient.CoreV1().ConfigMaps(tc.configMap.Namespace).Get(ctx, tc.configMap.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.configMap.Name, configMap.Name)
				assert.Equal(t, tc.configMap.Namespace, configMap.Namespace)

				if tc.configMap.Labels != nil {
					assert.NotNil(t, configMap.Labels)
					for k, v := range tc.configMap.Labels {
						if strVal, ok := v.(string); ok {
							assert.Equal(t, strVal, configMap.Labels[k])
						}
					}
				}
			}
		})
	}
}

func testGetConfigMap(t *testing.T) {
	ctx := context.Background()
	testCM := createConfigMapObj(configMapName, testNamespace, map[string]string{"key": "value"}, nil)
	testNS := createNamespaceForTest(testNamespace)

	testCases := []struct {
		name        string
		configMap   ConfigMap
		expectError bool
		errorMsg    string
	}{
		{
			name: "Get existing ConfigMap",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
			},
			expectError: false,
		},
		{
			name: "ConfigMap not found",
			configMap: ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testNS, testCM)

			result, err := tc.configMap.Get(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, configMapName)
			}
		})
	}
}

func testListConfigMaps(t *testing.T) {
	ctx := context.Background()

	testNS := createNamespaceForTest(testNamespace)
	otherNS := createNamespaceForTest(otherNamespace)
	cm1 := createConfigMapObj(configMapName1, testNamespace, map[string]string{"key": "value1"}, map[string]string{"env": "dev"})
	cm2 := createConfigMapObj(configMapName2, testNamespace, map[string]string{"key": "value2"}, map[string]string{"env": "dev"})
	cm3 := createConfigMapObj(configMapName3, otherNamespace, map[string]string{"key": "value3"}, map[string]string{"env": "prod"})

	testCases := []struct {
		name              string
		configMap         ConfigMap
		allNamespaces     bool
		labelSelector     string
		expectError       bool
		errorMsg          string
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name: "List ConfigMaps in namespace",
			configMap: ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces:    false,
			labelSelector:    "",
			expectError:      false,
			expectedContent:  []string{configMapName1, configMapName2},
			unexpectedContent: []string{configMapName3},
		},
		{
			name: "List ConfigMaps in all namespaces",
			configMap: ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces:   true,
			labelSelector:   "",
			expectError:     false,
			expectedContent: []string{configMapName1, configMapName2, configMapName3},
		},
		{
			name: "List ConfigMaps with label selector",
			configMap: ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces:    true,
			labelSelector:    "env=dev",
			expectError:      false,
			expectedContent:  []string{configMapName1, configMapName2},
			unexpectedContent: []string{configMapName3},
		},
		{
			name: "No ConfigMaps match label selector",
			configMap: ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "env=nonexistent",
			expectError:   true,
			errorMsg:      "no ConfigMaps found matching the specified label selector",
		},
		{
			name: "No ConfigMaps in empty namespace",
			configMap: ConfigMap{
				Namespace: emptyNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			expectError:   true,
			errorMsg:      fmt.Sprintf("no ConfigMaps found in namespace %q", emptyNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testNS, otherNS, cm1, cm2, cm3)

			result, err := tc.configMap.List(ctx, cm, tc.allNamespaces, tc.labelSelector)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				for _, expected := range tc.expectedContent {
					assert.Contains(t, result, expected)
				}

				for _, unexpected := range tc.unexpectedContent {
					assert.NotContains(t, result, unexpected)
				}
			}
		})
	}
}

func testDeleteConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		configMap    ConfigMap
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
		validate     func(*testing.T, context.Context, *Manager)
	}{
		{
			name: "Delete existing ConfigMap",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
				createConfigMapObj(configMapName, testNamespace, map[string]string{"key": "value"}, nil),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				_, err = client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "ConfigMap not found",
			configMap: ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Missing ConfigMap name",
			configMap: ConfigMap{
				Name:      "",
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "ConfigMap name is required for deletion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.configMap.Delete(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "deleted successfully")

				if tc.validate != nil {
					tc.validate(t, ctx, cm)
				}
			}
		})
	}
}

func testUpdateConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		configMap    ConfigMap
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
		validate     func(*testing.T, context.Context, *Manager)
	}{
		{
			name: "Update ConfigMap data",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"new-key": "new-value",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
				createConfigMapObj(configMapName, testNamespace, map[string]string{"old-key": "old-value"}, nil),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				configMap, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, "new-value", configMap.Data["new-key"])
				assert.NotContains(t, configMap.Data, "old-key")
			},
		},
		{
			name: "Update ConfigMap labels",
			configMap: ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "updated",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
				createConfigMapObj(configMapName, testNamespace, map[string]string{"key": "value"}, map[string]string{"app": "original"}),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				configMap, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, "updated", configMap.Labels["app"])
			},
		},
		{
			name: "ConfigMap not found for update",
			configMap: ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Missing ConfigMap name for update",
			configMap: ConfigMap{
				Name:      "",
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "ConfigMap name is required for update",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.configMap.Update(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "updated successfully")

				if tc.validate != nil {
					tc.validate(t, ctx, cm)
				}
			}
		})
	}
}
