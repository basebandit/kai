package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConfigMapOperations(t *testing.T) {
	t.Run("CreateConfigMap", testCreateConfigMap)
	t.Run("GetConfigMap", testGetConfigMap)
	t.Run("ListConfigMaps", testListConfigMaps)
	t.Run("DeleteConfigMap", testDeleteConfigMap)
	t.Run("UpdateConfigMap", testUpdateConfigMap)
}

func testCreateConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		configMap      *ConfigMap
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateCreate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Create basic ConfigMap",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"config.yaml": "key: value",
					"app.conf":    "setting=true",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"test-configmap\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, configMapName, cm.Name)
				assert.Equal(t, testNamespace, cm.Namespace)
				assert.Equal(t, "key: value", cm.Data["config.yaml"])
				assert.Equal(t, "setting=true", cm.Data["app.conf"])
			},
		},
		{
			name: "Create ConfigMap with binary data",
			configMap: &ConfigMap{
				Name:      "binary-configmap",
				Namespace: testNamespace,
				BinaryData: map[string]interface{}{
					"binary.dat": []byte{0x01, 0x02, 0x03, 0x04},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"binary-configmap\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, "binary-configmap", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, cm.BinaryData["binary.dat"])
			},
		},
		{
			name: "Create ConfigMap with labels and annotations",
			configMap: &ConfigMap{
				Name:      "labeled-configmap",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
				Labels: map[string]interface{}{
					"app": "myapp",
					"env": "prod",
				},
				Annotations: map[string]interface{}{
					"description": "Production config",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"labeled-configmap\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, "labeled-configmap", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "myapp", cm.Labels["app"])
				assert.Equal(t, "prod", cm.Labels["env"])
				assert.Equal(t, "Production config", cm.Annotations["description"])
			},
		},
		{
			name: "Create ConfigMap with both data and binary data",
			configMap: &ConfigMap{
				Name:      "mixed-configmap",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"text.txt": "plain text",
				},
				BinaryData: map[string]interface{}{
					"data.bin": []byte{0xAA, 0xBB},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"mixed-configmap\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, "mixed-configmap", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "plain text", cm.Data["text.txt"])
				assert.Equal(t, []byte{0xAA, 0xBB}, cm.BinaryData["data.bin"])
			},
		},
		{
			name: "Namespace not found",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: nonexistentNS,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "Missing ConfigMap name",
			configMap: &ConfigMap{
				Name:      "",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "ConfigMap name is required",
		},
		{
			name: "Missing namespace",
			configMap: &ConfigMap{
				Name: configMapName,
				Namespace: "",
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.configMap.Create(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateCreate != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateCreate(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testGetConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		configMap      *ConfigMap
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing ConfigMap",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"key": "value",
					},
				}
				fakeClient := fake.NewSimpleClientset(existingCM)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap: test-configmap",
		},
		{
			name: "ConfigMap not found",
			configMap: &ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.configMap.Get(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testListConfigMaps(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name              string
		configMap         *ConfigMap
		allNamespaces     bool
		labelSelector     string
		setupMock         func(*testmocks.MockClusterManager)
		expectedContent   []string
		unexpectedContent []string
		expectedError     string
	}{
		{
			name: "List ConfigMaps in namespace",
			configMap: &ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				cm1 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName1,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
				}
				cm2 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName2,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
				}
				cm3 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName3,
						Namespace: otherNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				}
				fakeClient := fake.NewSimpleClientset(cm1, cm2, cm3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{configMapName1, configMapName2},
			unexpectedContent: []string{configMapName3},
		},
		{
			name: "List ConfigMaps in all namespaces",
			configMap: &ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				cm1 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName1,
						Namespace: testNamespace,
					},
				}
				cm2 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName2,
						Namespace: testNamespace,
					},
				}
				cm3 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName3,
						Namespace: otherNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(cm1, cm2, cm3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{configMapName1, configMapName2, configMapName3},
		},
		{
			name: "List ConfigMaps with label selector",
			configMap: &ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "env=dev",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				cm1 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName1,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
				}
				cm2 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName2,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
				}
				cm3 := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName3,
						Namespace: otherNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				}
				fakeClient := fake.NewSimpleClientset(cm1, cm2, cm3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{configMapName1, configMapName2},
			unexpectedContent: []string{configMapName3},
		},
		{
			name: "No ConfigMaps match label selector",
			configMap: &ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "env=nonexistent",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no ConfigMaps found matching the specified label selector",
		},
		{
			name: "No ConfigMaps in empty namespace",
			configMap: &ConfigMap{
				Namespace: emptyNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no ConfigMaps found in namespace \"empty-namespace\"",
		},
		{
			name: "No ConfigMaps in any namespace",
			configMap: &ConfigMap{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no ConfigMaps found in any namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.configMap.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)

				for _, expected := range tc.expectedContent {
					assert.Contains(t, result, expected)
				}

				for _, unexpected := range tc.unexpectedContent {
					assert.NotContains(t, result, unexpected)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testDeleteConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		configMap      *ConfigMap
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing ConfigMap",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: testNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(existingCM)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"test-configmap\" deleted successfully",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "ConfigMap not found",
			configMap: &ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing ConfigMap name",
			configMap: &ConfigMap{
				Name:      "",
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "ConfigMap name is required for deletion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.configMap.Delete(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateDelete != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateDelete(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testUpdateConfigMap(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		configMap      *ConfigMap
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateUpdate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Update existing ConfigMap data",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"config.yaml": "updated: true",
					"new.conf":    "added=yes",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"config.yaml": "old: value",
					},
				}
				fakeClient := fake.NewSimpleClientset(existingCM)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"test-configmap\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "updated: true", cm.Data["config.yaml"])
				assert.Equal(t, "added=yes", cm.Data["new.conf"])
			},
		},
		{
			name: "Update ConfigMap with binary data",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				BinaryData: map[string]interface{}{
					"data.bin": []byte{0xFF, 0xEE},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: testNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(existingCM)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"test-configmap\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, []byte{0xFF, 0xEE}, cm.BinaryData["data.bin"])
			},
		},
		{
			name: "Update ConfigMap labels and annotations",
			configMap: &ConfigMap{
				Name:      configMapName,
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"version": "v2",
				},
				Annotations: map[string]interface{}{
					"updated": "true",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
						Namespace: testNamespace,
						Labels: map[string]string{
							"version": "v1",
						},
					},
				}
				fakeClient := fake.NewSimpleClientset(existingCM)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "ConfigMap \"test-configmap\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				cm, err := client.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "v2", cm.Labels["version"])
				assert.Equal(t, "true", cm.Annotations["updated"])
			},
		},
		{
			name: "ConfigMap not found",
			configMap: &ConfigMap{
				Name:      nonexistentConfigMap,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing ConfigMap name",
			configMap: &ConfigMap{
				Name:      "",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "ConfigMap name is required for update",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.configMap.Update(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateUpdate != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateUpdate(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}
