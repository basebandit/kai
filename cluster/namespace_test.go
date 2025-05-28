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

func TestNamespaceOperations(t *testing.T) {
	t.Run("CreateNamespace", testCreateNamespaces)
	t.Run("GetNamespace", testGetNamespace)
	t.Run("ListNamespaces", testListNamespaces)
	t.Run("DeleteNamespace", testDeleteNamespace)
}

func createNamespaceObj(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
}

func testCreateNamespaces(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		namespace    Namespace
		setupObjects []runtime.Object
		expectedText string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Create basic namespace",
			namespace: Namespace{
				Name: testNamespace,
			},
			setupObjects: []runtime.Object{},
			expectedText: fmt.Sprintf("Namespace %q created successfully", testNamespace),
			expectError:  false,
		},
		{
			name: "Create namespace with labels",
			namespace: Namespace{
				Name: "labeled-namespace",
				Labels: map[string]interface{}{
					"env":  "test",
					"team": "dev",
				},
			},
			setupObjects: []runtime.Object{},
			expectedText: "Namespace \"labeled-namespace\" created successfully",
			expectError:  false,
		},
		{
			name: "Create namespace with annotations",
			namespace: Namespace{
				Name: "annotated-namespace",
				Annotations: map[string]interface{}{
					"description": "Test namespace",
					"owner":       "test-team",
				},
			},
			setupObjects: []runtime.Object{},
			expectedText: "Namespace \"annotated-namespace\" created successfully",
			expectError:  false,
		},
		{
			name: "Missing namespace name",
			namespace: Namespace{
				Name: "",
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "namespace name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.namespace.Create(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedText)

				fakeClient := cm.clients[testCluster]
				namespace, err := fakeClient.CoreV1().Namespaces().Get(ctx, tc.namespace.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.namespace.Name, namespace.Name)

				if tc.namespace.Labels != nil {
					assert.NotNil(t, namespace.Labels)
					for k, v := range tc.namespace.Labels {
						if strVal, ok := v.(string); ok {
							assert.Equal(t, strVal, namespace.Labels[k])
						}
					}
				}

				if tc.namespace.Annotations != nil {
					assert.NotNil(t, namespace.Annotations)
					for k, v := range tc.namespace.Annotations {
						if strVal, ok := v.(string); ok {
							assert.Equal(t, strVal, namespace.Annotations[k])
						}
					}
				}
			}
		})
	}
}

func testGetNamespace(t *testing.T) {
	ctx := context.Background()
	testNs := createNamespaceObj(testNamespace, nil)

	testCases := []struct {
		name        string
		namespace   Namespace
		expectError bool
		errorMsg    string
	}{
		{
			name: "Get existing namespace",
			namespace: Namespace{
				Name: testNamespace,
			},
			expectError: false,
		},
		{
			name: "Namespace not found",
			namespace: Namespace{
				Name: nonexistentNS,
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testNs)

			result, err := tc.namespace.Get(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, testNamespace)
			}
		})
	}
}

func testListNamespaces(t *testing.T) {
	ctx := context.Background()

	ns1 := createNamespaceObj("namespace1", map[string]string{"env": "dev"})
	ns2 := createNamespaceObj("namespace2", map[string]string{"env": "dev"})
	ns3 := createNamespaceObj("namespace3", map[string]string{"env": "prod"})

	testCases := []struct {
		name              string
		namespace         Namespace
		labelSelector     string
		expectError       bool
		errorMsg          string
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name:            "List all namespaces",
			namespace:       Namespace{},
			labelSelector:   "",
			expectError:     false,
			expectedContent: []string{"namespace1", "namespace2", "namespace3"},
		},
		{
			name:              "List namespaces with label selector",
			namespace:         Namespace{},
			labelSelector:     "env=dev",
			expectError:       false,
			expectedContent:   []string{"namespace1", "namespace2"},
			unexpectedContent: []string{"namespace3"},
		},
		{
			name:          "No namespaces match label selector",
			namespace:     Namespace{},
			labelSelector: "env=nonexistent",
			expectError:   true,
			errorMsg:      "no namespaces found matching the specified selectors",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(ns1, ns2, ns3)

			result, err := tc.namespace.List(ctx, cm, tc.labelSelector)

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

func testDeleteNamespace(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		namespace    Namespace
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
		validate     func(*testing.T, context.Context, *Manager)
	}{
		{
			name: "Delete existing namespace by name",
			namespace: Namespace{
				Name: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceObj(testNamespace, nil),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				_, err = client.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Namespace not found",
			namespace: Namespace{
				Name: nonexistentNS,
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "failed to find namespace",
		},
		{
			name: "Delete namespaces by label selector",
			namespace: Namespace{
				Labels: map[string]interface{}{
					"env": "test",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceObj(testNamespace1, map[string]string{"env": "test"}),
				createNamespaceObj(testNamespace2, map[string]string{"env": "test"}),
				createNamespaceObj(testNamespace3, map[string]string{"env": "prod"}),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				_, err1 := client.CoreV1().Namespaces().Get(ctx, testNamespace1, metav1.GetOptions{})
				_, err2 := client.CoreV1().Namespaces().Get(ctx, testNamespace2, metav1.GetOptions{})
				assert.Error(t, err1)
				assert.Error(t, err2)

				prodNs, err3 := client.CoreV1().Namespaces().Get(ctx, testNamespace3, metav1.GetOptions{})
				assert.NoError(t, err3)
				assert.Equal(t, testNamespace3, prodNs.Name)
			},
		},
		{
			name: "No namespaces match label selector",
			namespace: Namespace{
				Labels: map[string]interface{}{
					"env": "nonexistent",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceObj("test-ns", map[string]string{"env": "test"}),
			},
			expectError: true,
			errorMsg:    "no namespaces found with label selector",
		},
		{
			name: "Missing name and labels",
			namespace: Namespace{
				Name: "",
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "either namespace name or label selector must be provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.namespace.Delete(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tc.namespace.Name != "" {
					assert.Contains(t, result, "deleted successfully")
				} else {
					assert.Contains(t, result, "Deleted")
				}

				if tc.validate != nil {
					tc.validate(t, ctx, cm)
				}
			}
		})
	}
}
