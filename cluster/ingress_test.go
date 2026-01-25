package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIngressOperations(t *testing.T) {
	t.Run("CreateIngress", testCreateIngress)
	t.Run("GetIngress", testGetIngress)
	t.Run("ListIngresses", testListIngresses)
	t.Run("UpdateIngress", testUpdateIngress)
	t.Run("DeleteIngress", testDeleteIngress)
}

func testCreateIngress(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		ingress        *Ingress
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Create basic Ingress",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
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
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"test-ingress\" created successfully",
			expectedError:  "",
		},
		{
			name: "Create Ingress with TLS",
			ingress: &Ingress{
				Name:             "tls-ingress",
				Namespace:        testNamespace,
				IngressClassName: "nginx",
				Rules: []kai.IngressRule{
					{
						Host: "secure.example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/api",
								PathType:    "Exact",
								ServiceName: "api-service",
								ServicePort: "http",
							},
						},
					},
				},
				TLS: []kai.IngressTLS{
					{
						Hosts:      []string{"secure.example.com"},
						SecretName: "tls-secret",
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"tls-ingress\" created successfully",
			expectedError:  "",
		},
		{
			name: "Create Ingress with labels and annotations",
			ingress: &Ingress{
				Name:      "annotated-ingress",
				Namespace: testNamespace,
				Labels:    map[string]interface{}{"app": "web"},
				Annotations: map[string]interface{}{
					"nginx.ingress.kubernetes.io/rewrite-target": "/",
				},
				Rules: []kai.IngressRule{
					{
						Host: "app.example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								PathType:    "Prefix",
								ServiceName: "web-service",
								ServicePort: float64(8080),
							},
						},
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"annotated-ingress\" created successfully",
			expectedError:  "",
		},
		{
			name: "Missing Ingress name",
			ingress: &Ingress{
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServiceName: "backend",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Ingress name is required",
		},
		{
			name: "Missing namespace",
			ingress: &Ingress{
				Name: "test-ingress",
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServiceName: "backend",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace is required",
		},
		{
			name: "Namespace not found",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: "nonexistent",
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServiceName: "backend",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent\" not found",
		},
		{
			name: "GetCurrentClient error",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServiceName: "backend",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, assert.AnError)
			},
			expectedError: "error getting client",
		},
		{
			name: "Invalid path type",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								PathType:    "Invalid",
								ServiceName: "backend",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "invalid path type",
		},
		{
			name: "Missing service name in path",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServicePort: 80,
							},
						},
					},
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "service name is required",
		},
		{
			name: "Missing service port in path",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/",
								ServiceName: "backend",
							},
						},
					},
				},
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "service port is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.ingress.Create(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testGetIngress(t *testing.T) {
	ctx := context.Background()

	pathType := networkingv1.PathTypePrefix
	existingIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: testNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "backend",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		ingress        *Ingress
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing Ingress",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "test-ingress",
			expectedError:  "",
		},
		{
			name: "Ingress not found",
			ingress: &Ingress{
				Name:      "nonexistent",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "GetCurrentClient error",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, assert.AnError)
			},
			expectedError: "error getting client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.ingress.Get(ctx, mockCM)

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

func testListIngresses(t *testing.T) {
	ctx := context.Background()

	pathType := networkingv1.PathTypePrefix
	ingress1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ingress1", Namespace: testNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "app1.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "service1",
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingress2 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ingress2", Namespace: testNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "app2.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "service2",
											Port: networkingv1.ServiceBackendPort{Number: 8080},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		ingress        *Ingress
		allNamespaces  bool
		labelSelector  string
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult []string
		expectedError  string
	}{
		{
			name: "List Ingresses in namespace",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ingress1, ingress2)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: []string{"ingress1", "ingress2"},
			expectedError:  "",
		},
		{
			name: "List Ingresses all namespaces",
			ingress: &Ingress{
				Namespace: "",
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ingress1, ingress2)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: []string{"ingress1", "ingress2"},
			expectedError:  "",
		},
		{
			name: "No Ingresses in empty namespace",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Ingresses found",
		},
		{
			name: "No Ingresses with label selector",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "app=nonexistent",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ingress1)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Ingresses found matching the specified label selector",
		},
		{
			name: "No Ingresses in any namespace",
			ingress: &Ingress{
				Namespace: "",
			},
			allNamespaces: true,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Ingresses found in any namespace",
		},
		{
			name: "GetCurrentClient error",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, assert.AnError)
			},
			expectedError: "error getting client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.ingress.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tc.expectedResult {
					assert.Contains(t, result, expected)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testUpdateIngress(t *testing.T) {
	ctx := context.Background()

	pathType := networkingv1.PathTypePrefix
	existingIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: testNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "backend",
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		ingress        *Ingress
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Update Ingress class",
			ingress: &Ingress{
				Name:             "test-ingress",
				Namespace:        testNamespace,
				IngressClassName: "nginx",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"test-ingress\" updated successfully",
			expectedError:  "",
		},
		{
			name: "Update Ingress rules",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				Rules: []kai.IngressRule{
					{
						Host: "new.example.com",
						Paths: []kai.IngressPath{
							{
								Path:        "/api",
								PathType:    "Prefix",
								ServiceName: "api-service",
								ServicePort: 8080,
							},
						},
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"test-ingress\" updated successfully",
			expectedError:  "",
		},
		{
			name: "Update Ingress with TLS",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
				TLS: []kai.IngressTLS{
					{
						Hosts:      []string{"example.com"},
						SecretName: "tls-secret",
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"test-ingress\" updated successfully",
			expectedError:  "",
		},
		{
			name: "Update Ingress labels and annotations",
			ingress: &Ingress{
				Name:        "test-ingress",
				Namespace:   testNamespace,
				Labels:      map[string]interface{}{"env": "prod"},
				Annotations: map[string]interface{}{"key": "value"},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Ingress \"test-ingress\" updated successfully",
			expectedError:  "",
		},
		{
			name: "Ingress not found",
			ingress: &Ingress{
				Name:             "nonexistent",
				Namespace:        testNamespace,
				IngressClassName: "nginx",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing Ingress name",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Ingress name is required",
		},
		{
			name: "Missing namespace",
			ingress: &Ingress{
				Name: "test-ingress",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace is required",
		},
		{
			name: "GetCurrentClient error",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, assert.AnError)
			},
			expectedError: "error getting client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.ingress.Update(ctx, mockCM)

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

func testDeleteIngress(t *testing.T) {
	ctx := context.Background()

	pathType := networkingv1.PathTypePrefix
	existingIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: testNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "backend",
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		ingress        *Ingress
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing Ingress",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingIngress)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "deleted successfully",
			expectedError:  "",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.NetworkingV1().Ingresses(testNamespace).Get(ctx, "test-ingress", metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Ingress not found",
			ingress: &Ingress{
				Name:      "nonexistent",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing Ingress name",
			ingress: &Ingress{
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Ingress name is required",
		},
		{
			name: "GetCurrentClient error",
			ingress: &Ingress{
				Name:      "test-ingress",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				mockCM.On("GetCurrentClient").Return(nil, assert.AnError)
			},
			expectedError: "error getting client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.ingress.Delete(ctx, mockCM)

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
