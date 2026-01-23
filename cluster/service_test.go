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

func TestServiceOperations(t *testing.T) {
	t.Run("CreateService", testCreateServices)
	t.Run("GetService", testGetService)
	t.Run("ListServices", testListServices)
	t.Run("DeleteService", testDeleteService)
}

func testCreateServices(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		service        *Service
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateCreate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Create basic ClusterIP service",
			service: &Service{
				Name:      "test-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Service \"test-service\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "test-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test-service", svc.Name)
				assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
				assert.Len(t, svc.Spec.Ports, 1)
				assert.Equal(t, int32(80), svc.Spec.Ports[0].Port)
			},
		},
		{
			name: "Create NodePort service",
			service: &Service{
				Name:      "nodeport-service",
				Namespace: testNamespace,
				Type:      "NodePort",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
						NodePort:   30080,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Service \"nodeport-service\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "nodeport-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceTypeNodePort, svc.Spec.Type)
				assert.Equal(t, int32(30080), svc.Spec.Ports[0].NodePort)
			},
		},
		{
			name: "Create LoadBalancer service",
			service: &Service{
				Name:      "lb-service",
				Namespace: testNamespace,
				Type:      "LoadBalancer",
				Ports: []ServicePort{
					{
						Port:       443,
						TargetPort: int32(8443),
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "web",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Service \"lb-service\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "lb-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceTypeLoadBalancer, svc.Spec.Type)
			},
		},
		{
			name: "Create service with multiple ports",
			service: &Service{
				Name:      "multi-port-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: int32(8080),
						Protocol:   "TCP",
					},
					{
						Name:       "https",
						Port:       443,
						TargetPort: int32(8443),
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Service \"multi-port-service\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "multi-port-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Len(t, svc.Spec.Ports, 2)
				assert.Equal(t, "http", svc.Spec.Ports[0].Name)
				assert.Equal(t, "https", svc.Spec.Ports[1].Name)
			},
		},
		{
			name: "Create service with named target port",
			service: &Service{
				Name:      "named-target-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: "http",
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "named-target-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "http", svc.Spec.Ports[0].TargetPort.StrVal)
			},
		},
		{
			name: "Create service with UDP protocol",
			service: &Service{
				Name:      "udp-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       53,
						TargetPort: int32(53),
						Protocol:   "UDP",
					},
				},
				Selector: map[string]interface{}{
					"app": "dns",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "udp-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ProtocolUDP, svc.Spec.Ports[0].Protocol)
			},
		},
		{
			name: "Create service with labels",
			service: &Service{
				Name:      "labeled-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Labels: map[string]interface{}{
					"env":  "test",
					"tier": "backend",
				},
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "labeled-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test", svc.Labels["env"])
				assert.Equal(t, "backend", svc.Labels["tier"])
			},
		},
		{
			name: "Create service with session affinity",
			service: &Service{
				Name:            "affinity-service",
				Namespace:       testNamespace,
				Type:            "ClusterIP",
				SessionAffinity: "ClientIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "affinity-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceAffinityClientIP, svc.Spec.SessionAffinity)
			},
		},
		{
			name: "Create ExternalName service",
			service: &Service{
				Name:         "external-service",
				Namespace:    testNamespace,
				Type:         "ExternalName",
				ExternalName: "example.com",
				Ports: []ServicePort{
					{
						Port: 80,
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
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				svc, err := client.CoreV1().Services(testNamespace).Get(ctx, "external-service", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceTypeExternalName, svc.Spec.Type)
				assert.Equal(t, "example.com", svc.Spec.ExternalName)
			},
		},
		{
			name: "Missing service name",
			service: &Service{
				Name:      "",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{Port: 80},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "service name is required",
		},
		{
			name: "Missing namespace",
			service: &Service{
				Name:      "test-service",
				Namespace: "",
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{Port: 80},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"\" not found",
		},
		{
			name: "Missing ports",
			service: &Service{
				Name:      "test-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports:     []ServicePort{},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "at least one port must be specified",
		},
		{
			name: "Namespace not found",
			service: &Service{
				Name:      "test-service",
				Namespace: nonexistentNS,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: int32(8080),
					},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "Invalid service type",
			service: &Service{
				Name:      "invalid-service",
				Namespace: testNamespace,
				Type:      "InvalidType",
				Ports: []ServicePort{
					{Port: 80},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "invalid service type",
		},
		{
			name: "Invalid protocol",
			service: &Service{
				Name:      "invalid-protocol",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:     80,
						Protocol: "INVALID",
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
			expectedError: "invalid protocol",
		},
		{
			name: "ExternalName service without external name",
			service: &Service{
				Name:      "external-service",
				Namespace: testNamespace,
				Type:      "ExternalName",
				Ports: []ServicePort{
					{Port: 80},
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "externalName must be specified for ExternalName service type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.service.Create(ctx, mockCM)

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

func testGetService(t *testing.T) {
	ctx := context.Background()

	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	testCases := []struct {
		name           string
		service        *Service
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing service",
			service: &Service{
				Name:      "test-service",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(existingService, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "test-service",
		},
		{
			name: "Service not found",
			service: &Service{
				Name:      "nonexistent-service",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "service 'nonexistent-service' not found",
		},
		{
			name: "Namespace not found",
			service: &Service{
				Name:      "test-service",
				Namespace: nonexistentNS,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace 'nonexistent-namespace' not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.service.Get(ctx, mockCM)

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

func testListServices(t *testing.T) {
	ctx := context.Background()

	svc1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service1",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	svc2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service2",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
		},
	}
	svc3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service3",
			Namespace: "other-namespace",
			Labels:    map[string]string{"app": "other"},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	testCases := []struct {
		name              string
		service           *Service
		allNamespaces     bool
		labelSelector     string
		setupMock         func(*testmocks.MockClusterManager)
		expectedContent   []string
		unexpectedContent []string
		expectedError     string
	}{
		{
			name: "List services in namespace",
			service: &Service{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(svc1, svc2, svc3, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{"service1", "service2"},
			unexpectedContent: []string{"service3"},
		},
		{
			name: "List services with label selector",
			service: &Service{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "app=test",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(svc1, svc2, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"service1", "service2"},
		},
		{
			name: "List services in all namespaces",
			service: &Service{
				Namespace: "",
			},
			allNamespaces: true,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(svc1, svc2, svc3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"service1", "service2", "service3"},
		},
		{
			name: "No services found in namespace",
			service: &Service{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"No services found"},
		},
		{
			name: "Namespace not found",
			service: &Service{
				Namespace: nonexistentNS,
			},
			allNamespaces: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.service.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

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

func testDeleteService(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		service        *Service
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing service by name",
			service: &Service{
				Name:      "test-service",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				svc := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: testNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(svc, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Service \"test-service\" deleted successfully",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().Services(testNamespace).Get(ctx, "test-service", metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Delete services by label selector",
			service: &Service{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				svc1 := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": "test"},
					},
				}
				svc2 := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service2",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": "test"},
					},
				}
				fakeClient := fake.NewSimpleClientset(svc1, svc2, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Deleted 2 services",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err1 := client.CoreV1().Services(testNamespace).Get(ctx, "service1", metav1.GetOptions{})
				_, err2 := client.CoreV1().Services(testNamespace).Get(ctx, "service2", metav1.GetOptions{})
				assert.Error(t, err1)
				assert.Error(t, err2)
			},
		},
		{
			name: "Service not found",
			service: &Service{
				Name:      "nonexistent-service",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to find service",
		},
		{
			name: "Namespace not found",
			service: &Service{
				Name:      "test-service",
				Namespace: nonexistentNS,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "No services match label selector",
			service: &Service{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "nonexistent",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				svc := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service1",
						Namespace: testNamespace,
						Labels:    map[string]string{"app": "test"},
					},
				}
				fakeClient := fake.NewSimpleClientset(svc, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no services found with label selector",
		},
		{
			name: "Missing name and labels",
			service: &Service{
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "either service name or label selector must be provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.service.Delete(ctx, mockCM)

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
