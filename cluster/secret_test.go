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

func TestSecretOperations(t *testing.T) {
	t.Run("CreateSecret", testCreateSecret)
	t.Run("GetSecret", testGetSecret)
	t.Run("ListSecrets", testListSecrets)
	t.Run("DeleteSecret", testDeleteSecret)
	t.Run("UpdateSecret", testUpdateSecret)
}

func testCreateSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		secret         *Secret
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateCreate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Create basic Secret",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"username": "admin",
					"password": "secret123",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, secretName, secret.Name)
				assert.Equal(t, testNamespace, secret.Namespace)
				assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
				assert.Equal(t, []byte("admin"), secret.Data["username"])
				assert.Equal(t, []byte("secret123"), secret.Data["password"])
			},
		},
		{
			name: "Create Secret with type",
			secret: &Secret{
				Name:      "tls-secret",
				Namespace: testNamespace,
				Type:      secretTypeTLS,
				Data: map[string]interface{}{
					"tls.crt": "cert-data",
					"tls.key": "key-data",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"tls-secret\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "tls-secret", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.SecretType(secretTypeTLS), secret.Type)
			},
		},
		{
			name: "Create Secret with StringData",
			secret: &Secret{
				Name:      "stringdata-secret",
				Namespace: testNamespace,
				StringData: map[string]interface{}{
					"config": "plain-text-config",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"stringdata-secret\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "stringdata-secret", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "plain-text-config", secret.StringData["config"])
			},
		},
		{
			name: "Create Secret with labels and annotations",
			secret: &Secret{
				Name:      "labeled-secret",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
				Labels: map[string]interface{}{
					"app": "test",
					"env": "dev",
				},
				Annotations: map[string]interface{}{
					"description": "Test Secret",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"labeled-secret\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "labeled-secret", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test", secret.Labels["app"])
				assert.Equal(t, "dev", secret.Labels["env"])
				assert.Equal(t, "Test Secret", secret.Annotations["description"])
			},
		},
		{
			name: "Create Secret with byte data",
			secret: &Secret{
				Name:      "byte-secret",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"binary": []byte{0x01, 0x02, 0x03},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"byte-secret\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "byte-secret", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, []byte{0x01, 0x02, 0x03}, secret.Data["binary"])
			},
		},
		{
			name: "Namespace not found",
			secret: &Secret{
				Name:      secretName,
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
			name: "Missing Secret name",
			secret: &Secret{
				Name:      "",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Secret name is required",
		},
		{
			name: "Missing namespace",
			secret: &Secret{
				Name:      secretName,
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

			result, err := tc.secret.Create(ctx, mockCM)

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

func testGetSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		secret         *Secret
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing Secret",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret: test-secret",
		},
		{
			name: "Secret not found",
			secret: &Secret{
				Name:      nonexistentSecret,
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

			result, err := tc.secret.Get(ctx, mockCM)

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

func testListSecrets(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name              string
		secret            *Secret
		allNamespaces     bool
		labelSelector     string
		setupMock         func(*testmocks.MockClusterManager)
		expectedContent   []string
		unexpectedContent []string
		expectedError     string
	}{
		{
			name: "List Secrets in namespace",
			secret: &Secret{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				sec1 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName1,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec2 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName2,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec3 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName3,
						Namespace: otherNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(sec1, sec2, sec3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{secretName1, secretName2},
			unexpectedContent: []string{secretName3},
		},
		{
			name: "List Secrets in all namespaces",
			secret: &Secret{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				sec1 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName1,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec2 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName2,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec3 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName3,
						Namespace: otherNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(sec1, sec2, sec3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{secretName1, secretName2, secretName3},
		},
		{
			name: "List Secrets with label selector",
			secret: &Secret{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "env=dev",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				sec1 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName1,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec2 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName2,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				sec3 := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName3,
						Namespace: otherNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(sec1, sec2, sec3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{secretName1, secretName2},
			unexpectedContent: []string{secretName3},
		},
		{
			name: "No Secrets match label selector",
			secret: &Secret{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "env=nonexistent",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Secrets found matching the specified label selector",
		},
		{
			name: "No Secrets in empty namespace",
			secret: &Secret{
				Namespace: emptyNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Secrets found in namespace \"empty-namespace\"",
		},
		{
			name: "No Secrets in any namespace",
			secret: &Secret{
				Namespace: testNamespace,
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Secrets found in any namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.secret.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

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

func testDeleteSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		secret         *Secret
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing Secret",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" deleted successfully",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Secret not found",
			secret: &Secret{
				Name:      nonexistentSecret,
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing Secret name",
			secret: &Secret{
				Name:      "",
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Secret name is required for deletion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.secret.Delete(ctx, mockCM)

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

func testUpdateSecret(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		secret         *Secret
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateUpdate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Update existing Secret data",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"username": "newuser",
					"password": "newpass",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"username": []byte("olduser"),
					},
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, []byte("newuser"), secret.Data["username"])
				assert.Equal(t, []byte("newpass"), secret.Data["password"])
			},
		},
		{
			name: "Update Secret with StringData",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
				StringData: map[string]interface{}{
					"config": "updated-config",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "updated-config", secret.StringData["config"])
			},
		},
		{
			name: "Update Secret type",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
				Type:      secretTypeTLS,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.SecretType(secretTypeTLS), secret.Type)
			},
		},
		{
			name: "Update Secret labels and annotations",
			secret: &Secret{
				Name:      secretName,
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"version": "v2",
				},
				Annotations: map[string]interface{}{
					"updated": "true",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
						Labels: map[string]string{
							"version": "v1",
						},
					},
					Type: corev1.SecretTypeOpaque,
				}
				fakeClient := fake.NewSimpleClientset(existingSecret)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Secret \"test-secret\" updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, secretName, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "v2", secret.Labels["version"])
				assert.Equal(t, "true", secret.Annotations["updated"])
			},
		},
		{
			name: "Secret not found",
			secret: &Secret{
				Name:      nonexistentSecret,
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
			name: "Missing Secret name",
			secret: &Secret{
				Name:      "",
				Namespace: testNamespace,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Secret name is required for update",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.secret.Update(ctx, mockCM)

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
