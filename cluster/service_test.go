package cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestServiceOperations groups all service-related operations tests
func TestServiceOperations(t *testing.T) {
	t.Run("CreateService", testCreateServices)
	t.Run("GetService", testGetService)
	t.Run("ListServices", testListServices)
	t.Run("DeleteService", testDeleteService)
}

// createService creates a service object for testing
func createServiceObj(name, namespace string, serviceType corev1.ServiceType) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			CreationTimestamp: metav1.Time{
				Time: time.Now().Add(-24 * time.Hour), // 1 day old
			},
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func testCreateServices(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		service      Service
		setupObjects []runtime.Object
		expectedText string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Create ClusterIP service",
			service: Service{
				Name:      "test-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: "Service \"test-service\" created successfully",
			expectError:  false,
		},
		{
			name: "Create NodePort service",
			service: Service{
				Name:      "nodeport-service",
				Namespace: testNamespace,
				Type:      "NodePort",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						NodePort:   30080,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: "Service \"nodeport-service\" created successfully",
			expectError:  false,
		},
		{
			name: "Create service with multiple ports",
			service: Service{
				Name:      "multi-port-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: 8080,
						Protocol:   "TCP",
					},
					{
						Name:       "https",
						Port:       443,
						TargetPort: 8443,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: "Service \"multi-port-service\" created successfully",
			expectError:  false,
		},
		{
			name: "Create service with named targetPort",
			service: Service{
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
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: "Service \"named-target-service\" created successfully",
			expectError:  false,
		},
		{
			name: "Namespace not found",
			service: Service{
				Name:      "test-service",
				Namespace: nonexistentNS,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						Protocol:   "TCP",
					},
				},
				Selector: map[string]interface{}{
					"app": "test",
				},
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     fmt.Sprintf("namespace %q not found", nonexistentNS),
		},
		{
			name: "Invalid service type",
			service: Service{
				Name:      "invalid-type-service",
				Namespace: testNamespace,
				Type:      "InvalidType",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						Protocol:   "TCP",
					},
				},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectError: true,
			errorMsg:    "invalid service type",
		},
		{
			name: "NodePort specified for ClusterIP service",
			service: Service{
				Name:      "invalid-nodeport-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports: []ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						NodePort:   30080, // Invalid for ClusterIP
						Protocol:   "TCP",
					},
				},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectError: true,
			errorMsg:    "nodePort can only be specified for NodePort or LoadBalancer",
		},
		{
			name: "No ports specified",
			service: Service{
				Name:      "no-port-service",
				Namespace: testNamespace,
				Type:      "ClusterIP",
				Ports:     []ServicePort{}, // Empty ports
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectError: true,
			errorMsg:    "at least one port must be specified",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.service.Create(ctx, cm)

			// Verify result
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedText)

				// Verify service was created
				fakeClient := cm.clients[testClusterName]
				service, err := fakeClient.CoreV1().Services(tc.service.Namespace).Get(ctx, tc.service.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.service.Name, service.Name)
				assert.Equal(t, tc.service.Namespace, service.Namespace)

				// Check service type
				if tc.service.Type != "" {
					assert.Equal(t, corev1.ServiceType(tc.service.Type), service.Spec.Type)
				}

				// Check ports
				assert.Equal(t, len(tc.service.Ports), len(service.Spec.Ports))
				if len(tc.service.Ports) > 0 {
					for i, expectedPort := range tc.service.Ports {
						actualPort := service.Spec.Ports[i]
						assert.Equal(t, expectedPort.Port, actualPort.Port)

						// Check NodePort if specified
						if expectedPort.NodePort != 0 {
							assert.Equal(t, expectedPort.NodePort, actualPort.NodePort)
						}
					}
				}

				// Check selector
				if tc.service.Selector != nil {
					assert.NotNil(t, service.Spec.Selector)
					for k, v := range tc.service.Selector {
						if strVal, ok := v.(string); ok {
							assert.Equal(t, strVal, service.Spec.Selector[k])
						}
					}
				}
			}
		})
	}
}

func testGetService(t *testing.T) {
	ctx := context.Background()

	serviceName := "test-service"
	nonexistentServiceName := "nonexistent-service"

	clusterIPService := createServiceObj(serviceName, testNamespace, corev1.ServiceTypeClusterIP)
	testNamespaceObj := createNamespace(testNamespace)

	testCases := []struct {
		name        string
		service     Service
		expectError bool
		errorMsg    string
	}{
		{
			name: "Get existing service",
			service: Service{
				Name:      serviceName,
				Namespace: testNamespace,
			},
			expectError: false,
		},
		{
			name: "Service not found",
			service: Service{
				Name:      nonexistentServiceName,
				Namespace: testNamespace,
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Namespace not found",
			service: Service{
				Name:      serviceName,
				Namespace: nonexistentNS,
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(clusterIPService, testNamespaceObj)

			result, err := tc.service.Get(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, serviceName)
			}
		})
	}
}

func testListServices(t *testing.T) {
	ctx := context.Background()

	service1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service1",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "test"},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	service2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service2",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "test"},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// Service with different label - should not be matched
	service3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service3",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "other"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "other"},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	testNamespaceObj := createNamespace(testNamespace)

	// "List services with label selector" test case
	testServices := []runtime.Object{
		service1, service2, service3, testNamespaceObj,
	}

	cm := setupTestCluster(testServices...)

	service := &Service{
		Namespace: testNamespace,
	}

	result, err := service.List(ctx, cm, false, "app=test")

	assert.NoError(t, err)
	assert.Contains(t, result, "service1")
	assert.Contains(t, result, "service2")
	assert.NotContains(t, result, "service3")
}

func testDeleteService(t *testing.T) {
	ctx := context.Background()

	serviceName := "test-service"
	nonexistentServiceName := "nonexistent-service"

	// Create services with specific labels for testing
	service1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service1",
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":     "frontend",
				"tier":    "web",
				"version": "v1",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "frontend"},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	service2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service2",
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":     "backend",
				"tier":    "api",
				"version": "v1",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "backend"},
			Ports: []corev1.ServicePort{
				{
					Port:     8080,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	service3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service3",
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":     "frontend",
				"tier":    "web",
				"version": "v2",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "frontend"},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	clusterIPService := createServiceObj(serviceName, testNamespace, corev1.ServiceTypeClusterIP)
	testNamespaceObj := createNamespace(testNamespace)

	testCases := []struct {
		name         string
		service      Service
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
		expectResult string
		validate     func(*testing.T, context.Context, kai.ClusterManager)
	}{
		{
			name: "Delete specific service by name",
			service: Service{
				Name:      serviceName,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{clusterIPService, testNamespaceObj},
			expectError:  false,
			expectResult: "deleted successfully",
			validate: func(t *testing.T, ctx context.Context, cm kai.ClusterManager) {
				// Verify the service was actually deleted
				client, err := cm.GetCurrentClient()
				require.NoError(t, err, "Should be able to get client")

				_, err = client.CoreV1().Services(testNamespace).Get(ctx, serviceName, metav1.GetOptions{})
				assert.Error(t, err, "Service should have been deleted")
				assert.True(t, errors.IsNotFound(err), "Expected 'not found' error but got: %v", err)
			},
		},
		{
			name: "Service not found",
			service: Service{
				Name:      nonexistentServiceName,
				Namespace: testNamespace,
			},
			setupObjects: []runtime.Object{clusterIPService, testNamespaceObj},
			expectError:  true,
			errorMsg:     "failed to find service",
		},
		{
			name: "Namespace not found",
			service: Service{
				Name:      serviceName,
				Namespace: nonexistentNS,
			},
			setupObjects: []runtime.Object{clusterIPService, testNamespaceObj},
			expectError:  true,
			errorMsg:     "namespace",
		},
		{
			name: "Missing service name and labels",
			service: Service{
				Name:      "", // Empty name
				Namespace: testNamespace,
				// No labels
			},
			setupObjects: []runtime.Object{clusterIPService, testNamespaceObj},
			expectError:  true,
			errorMsg:     "either service name or label selector must be provided",
		},
		{
			name: "Delete services by app label",
			service: Service{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "frontend",
				},
			},
			setupObjects: []runtime.Object{service1, service2, service3, testNamespaceObj},
			expectError:  false,
			expectResult: "Deleted 2 services with label selector",
			validate: func(t *testing.T, ctx context.Context, cm kai.ClusterManager) {
				// Verify the frontend services were deleted
				client, err := cm.GetCurrentClient()
				require.NoError(t, err, "Should be able to get client")

				// service1 and service3 should be gone (frontend)
				_, err1 := client.CoreV1().Services(testNamespace).Get(ctx, "service1", metav1.GetOptions{})
				_, err3 := client.CoreV1().Services(testNamespace).Get(ctx, "service3", metav1.GetOptions{})
				assert.Error(t, err1, "Service1 should have been deleted")
				assert.Error(t, err3, "Service3 should have been deleted")

				// service2 should still exist (backend)
				svc2, err2 := client.CoreV1().Services(testNamespace).Get(ctx, "service2", metav1.GetOptions{})
				assert.NoError(t, err2, "Service2 should still exist")
				assert.Equal(t, "service2", svc2.Name)
			},
		},
		{
			name: "Delete services by multiple labels",
			service: Service{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"tier":    "web",
					"version": "v2",
				},
			},
			setupObjects: []runtime.Object{service1, service2, service3, testNamespaceObj},
			expectError:  false,
			expectResult: "Deleted 1 services with label selector",
			validate: func(t *testing.T, ctx context.Context, cm kai.ClusterManager) {
				// Verify only service3 was deleted (tier=web,version=v2)
				client, err := cm.GetCurrentClient()
				require.NoError(t, err, "Should be able to get client")

				// service3 should be gone
				_, err3 := client.CoreV1().Services(testNamespace).Get(ctx, "service3", metav1.GetOptions{})
				assert.Error(t, err3, "Service3 should have been deleted")

				// service1 and service2 should still exist
				svc1, err1 := client.CoreV1().Services(testNamespace).Get(ctx, "service1", metav1.GetOptions{})
				svc2, err2 := client.CoreV1().Services(testNamespace).Get(ctx, "service2", metav1.GetOptions{})
				assert.NoError(t, err1, "Service1 should still exist")
				assert.NoError(t, err2, "Service2 should still exist")
				assert.Equal(t, "service1", svc1.Name)
				assert.Equal(t, "service2", svc2.Name)
			},
		},
		{
			name: "No services match label selector",
			service: Service{
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"app": "nonexistent",
				},
			},
			setupObjects: []runtime.Object{service1, service2, service3, testNamespaceObj},
			expectError:  true,
			errorMsg:     "no services found with label selector",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a fresh test cluster for each test case
			cm := setupTestCluster(tc.setupObjects...)

			result, err := tc.service.Delete(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectResult)

				// Run custom validation if provided
				if tc.validate != nil {
					tc.validate(t, ctx, cm)
				}
			}
		})
	}
}
