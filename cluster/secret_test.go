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

func TestSecretOperations(t *testing.T) {
	t.Run("CreateSecret", testCreateSecret)
	t.Run("GetSecret", testGetSecret)
	t.Run("ListSecrets", testListSecrets)
	t.Run("DeleteSecret", testDeleteSecret)
}

func createSecretObj(name, namespace string, secretType corev1.SecretType, data map[string][]byte, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: secretType,
		Data: data,
	}
}

func testCreateSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		secret       Secret
		setupObjects []runtime.Object
		expectedText string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Create basic Secret",
			secret: Secret{
				Name:      secretName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: fmt.Sprintf("Secret %q created successfully in namespace %q", secretName, testNamespace),
			expectError:  false,
		},
		{
			name: "Create Secret with type",
			secret: Secret{
				Name:      "tls-secret",
				Namespace: testNamespace,
				Type:      secretTypeTLS,
				Data: map[string]interface{}{
					"tls.crt": "cert-data",
					"tls.key": "key-data",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: "Secret \"tls-secret\" created successfully",
			expectError:  false,
		},
		{
			name: "Create Secret with StringData",
			secret: Secret{
				Name:      "stringdata-secret",
				Namespace: testNamespace,
				StringData: map[string]interface{}{
					"config": "plain-text-config",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: "Secret \"stringdata-secret\" created successfully",
			expectError:  false,
		},
		{
			name: "Create Secret with labels and annotations",
			secret: Secret{
				Name:      "labeled-secret",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
				Labels: map[string]interface{}{
					"app": "test",
				},
				Annotations: map[string]interface{}{
					"description": "Test Secret",
				},
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectedText: "Secret \"labeled-secret\" created successfully",
			expectError:  false,
		},
		{
			name: "Namespace not found",
			secret: Secret{
				Name:      secretName,
				Namespace: nonexistentNS,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     fmt.Sprintf("namespace %q not found", nonexistentNS),
		},
		{
			name: "Missing Secret name",
			secret: Secret{
				Name:      "",
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "Secret name is required",
		},
		{
			name: "Missing namespace",
			secret: Secret{
				Name:      secretName,
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

			result, err := tc.secret.Create(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedText)

				fakeClient := cm.clients[testCluster]
				secret, err := fakeClient.CoreV1().Secrets(tc.secret.Namespace).Get(ctx, tc.secret.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.secret.Name, secret.Name)
				assert.Equal(t, tc.secret.Namespace, secret.Namespace)

				if tc.secret.Labels != nil {
					assert.NotNil(t, secret.Labels)
					for k, v := range tc.secret.Labels {
						if strVal, ok := v.(string); ok {
							assert.Equal(t, strVal, secret.Labels[k])
						}
					}
				}
			}
		})
	}
}

func testGetSecret(t *testing.T) {
	ctx := context.Background()
	testSec := createSecretObj(secretName, testNamespace, corev1.SecretTypeOpaque, map[string][]byte{"key": []byte("value")}, nil)
	testNS := createNamespaceForTest(testNamespace)

	testCases := []struct {
		name        string
		secret      Secret
		expectError bool
		errorMsg    string
	}{
		{
			name: "Get existing Secret",
			secret: Secret{
				Name:      secretName,
				Namespace: testNamespace,
			},
			expectError: false,
		},
		{
			name: "Secret not found",
			secret: Secret{
				Name:      nonexistentSecret,
				Namespace: testNamespace,
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testNS, testSec)

			result, err := tc.secret.Get(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, secretName)
			}
		})
	}
}

func testListSecrets(t *testing.T) {
	ctx := context.Background()

	testNS := createNamespaceForTest(testNamespace)
	otherNS := createNamespaceForTest(otherNamespace)
	sec1 := createSecretObj(secretName1, testNamespace, corev1.SecretTypeOpaque, map[string][]byte{"key": []byte("value1")}, map[string]string{"env": "dev"})
	sec2 := createSecretObj(secretName2, testNamespace, corev1.SecretTypeOpaque, map[string][]byte{"key": []byte("value2")}, map[string]string{"env": "dev"})
	sec3 := createSecretObj(secretName3, otherNamespace, corev1.SecretTypeOpaque, map[string][]byte{"key": []byte("value3")}, map[string]string{"env": "prod"})

	testCases := []struct {
		name              string
		secret            Secret
		allNamespaces     bool
		labelSelector     string
		expectError       bool
		errorMsg          string
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name: "List Secrets in namespace",
			secret: Secret{
				Namespace: testNamespace,
			},
			allNamespaces:    false,
			labelSelector:    "",
			expectError:      false,
			expectedContent:  []string{secretName1, secretName2},
			unexpectedContent: []string{secretName3},
		},
		{
			name: "List Secrets in all namespaces",
			secret: Secret{
				Namespace: testNamespace,
			},
			allNamespaces:   true,
			labelSelector:   "",
			expectError:     false,
			expectedContent: []string{secretName1, secretName2, secretName3},
		},
		{
			name: "List Secrets with label selector",
			secret: Secret{
				Namespace: testNamespace,
			},
			allNamespaces:    true,
			labelSelector:    "env=dev",
			expectError:      false,
			expectedContent:  []string{secretName1, secretName2},
			unexpectedContent: []string{secretName3},
		},
		{
			name: "No Secrets match label selector",
			secret: Secret{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "env=nonexistent",
			expectError:   true,
			errorMsg:      "no Secrets found matching the specified label selector",
		},
		{
			name: "No Secrets in empty namespace",
			secret: Secret{
				Namespace: emptyNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			expectError:   true,
			errorMsg:      fmt.Sprintf("no Secrets found in namespace %q", emptyNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testNS, otherNS, sec1, sec2, sec3)

			result, err := tc.secret.List(ctx, cm, tc.allNamespaces, tc.labelSelector)

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

func testDeleteSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		secret       Secret
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
		validate     func(*testing.T, context.Context, *Manager)
	}{
		{
			name: "Delete existing Secret",
			secret: Secret{
				Name:      secretName,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
				createSecretObj(secretName, testNamespace, corev1.SecretTypeOpaque, map[string][]byte{"key": []byte("value")}, nil),
			},
			expectError: false,
			validate: func(t *testing.T, ctx context.Context, cm *Manager) {
				client, err := cm.GetCurrentClient()
				require.NoError(t, err)

				_, err = client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Secret not found",
			secret: Secret{
				Name:      nonexistentSecret,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Missing Secret name",
			secret: Secret{
				Name:      "",
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{
				createNamespaceForTest(testNamespace),
			},
			expectError: true,
			errorMsg:    "Secret name is required for deletion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.secret.Delete(ctx, cm)

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
