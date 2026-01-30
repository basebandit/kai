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

func TestNamespaceOperations(t *testing.T) {
	t.Run("CreateNamespace", testCreateNamespaces)
	t.Run("GetNamespace", testGetNamespace)
	t.Run("ListNamespaces", testListNamespaces)
	t.Run("DeleteNamespace", testDeleteNamespace)
	t.Run("UpdateNamespace", testUpdateNamespace)
}

func testCreateNamespaces(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		namespace      *Namespace
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateCreate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Create basic namespace",
			namespace: &Namespace{
				Name: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Namespace \"test-namespace\" created successfully",
			expectedError:  "",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, testNamespace, ns.Name)
			},
		},
		{
			name: "Create namespace with labels",
			namespace: &Namespace{
				Name: "labeled-namespace",
				Labels: map[string]interface{}{
					"env":  "test",
					"team": "dev",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Namespace \"labeled-namespace\" created successfully",
			expectedError:  "",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, "labeled-namespace", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test", ns.Labels["env"])
				assert.Equal(t, "dev", ns.Labels["team"])
			},
		},
		{
			name: "Create namespace with annotations",
			namespace: &Namespace{
				Name: "annotated-namespace",
				Annotations: map[string]interface{}{
					"description": "Test namespace",
					"owner":       "test-team",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Namespace \"annotated-namespace\" created successfully",
			expectedError:  "",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, "annotated-namespace", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "Test namespace", ns.Annotations["description"])
				assert.Equal(t, "test-team", ns.Annotations["owner"])
			},
		},
		{
			name: "Missing namespace name",
			namespace: &Namespace{
				Name: "",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.namespace.Create(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				// Validate creation if validator provided
				if tc.validateCreate != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateCreate(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testGetNamespace(t *testing.T) {
	ctx := context.Background()

	existingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	testCases := []struct {
		name           string
		namespace      *Namespace
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing namespace",
			namespace: &Namespace{
				Name: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingNs)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: testNamespace,
			expectedError:  "",
		},
		{
			name: "Namespace not found",
			namespace: &Namespace{
				Name: nonexistentNS,
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

			result, err := tc.namespace.Get(ctx, mockCM)

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

func testListNamespaces(t *testing.T) {
	ctx := context.Background()

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "namespace1",
			Labels: map[string]string{"env": "dev"},
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "namespace2",
			Labels: map[string]string{"env": "dev"},
		},
	}
	ns3 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "namespace3",
			Labels: map[string]string{"env": "prod"},
		},
	}

	testCases := []struct {
		name              string
		namespace         *Namespace
		labelSelector     string
		setupMock         func(*testmocks.MockClusterManager)
		expectedContent   []string
		unexpectedContent []string
		expectedError     string
	}{
		{
			name:          "List all namespaces",
			namespace:     &Namespace{},
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ns1, ns2, ns3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"namespace1", "namespace2", "namespace3"},
		},
		{
			name:          "List namespaces with label selector",
			namespace:     &Namespace{},
			labelSelector: "env=dev",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ns1, ns2, ns3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{"namespace1", "namespace2"},
			unexpectedContent: []string{"namespace3"},
		},
		{
			name:          "No namespaces match label selector",
			namespace:     &Namespace{},
			labelSelector: "env=nonexistent",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ns1, ns2, ns3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no namespaces found matching the specified selectors",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.namespace.List(ctx, mockCM, tc.labelSelector)

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

func testDeleteNamespace(t *testing.T) {
	ctx := context.Background()

	existingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace1,
			Labels: map[string]string{"env": "test"},
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace2,
			Labels: map[string]string{"env": "test"},
		},
	}
	ns3 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace3,
			Labels: map[string]string{"env": "prod"},
		},
	}

	testCases := []struct {
		name           string
		namespace      *Namespace
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing namespace by name",
			namespace: &Namespace{
				Name: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingNs)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "deleted successfully",
			expectedError:  "",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Namespace not found",
			namespace: &Namespace{
				Name: nonexistentNS,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to find namespace",
		},
		{
			name: "Delete namespaces by label selector",
			namespace: &Namespace{
				Labels: map[string]interface{}{
					"env": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(ns1, ns2, ns3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Deleted",
			expectedError:  "",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
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
			namespace: &Namespace{
				Labels: map[string]interface{}{
					"env": "nonexistent",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-ns",
						Labels: map[string]string{"env": "test"},
					},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no namespaces found with label selector",
		},
		{
			name: "Missing name and labels",
			namespace: &Namespace{
				Name: "",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "either namespace name or label selector must be provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.namespace.Delete(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				// Validate deletion if validator provided
				if tc.validateDelete != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateDelete(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testUpdateNamespace(t *testing.T) {
	ctx := context.Background()

	existingNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				"env": "dev",
			},
			Annotations: map[string]string{
				"description": "original",
			},
		},
	}

	testCases := []struct {
		name           string
		namespace      *Namespace
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateUpdate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Update namespace labels",
			namespace: &Namespace{
				Name: "test-namespace",
				Labels: map[string]interface{}{
					"env":     "prod",
					"version": "v2",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingNamespace)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, "test-namespace", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "prod", ns.Labels["env"])
				assert.Equal(t, "v2", ns.Labels["version"])
			},
		},
		{
			name: "Update namespace annotations",
			namespace: &Namespace{
				Name: "test-namespace",
				Annotations: map[string]interface{}{
					"description": "updated",
					"owner":       "team-a",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingNamespace)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, "test-namespace", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "updated", ns.Annotations["description"])
				assert.Equal(t, "team-a", ns.Annotations["owner"])
			},
		},
		{
			name: "Update both labels and annotations",
			namespace: &Namespace{
				Name: "test-namespace",
				Labels: map[string]interface{}{
					"tier": "frontend",
				},
				Annotations: map[string]interface{}{
					"note": "critical",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingNamespace)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				ns, err := client.CoreV1().Namespaces().Get(ctx, "test-namespace", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "frontend", ns.Labels["tier"])
				assert.Equal(t, "critical", ns.Annotations["note"])
			},
		},
		{
			name: "Namespace not found",
			namespace: &Namespace{
				Name: "nonexistent-namespace",
				Labels: map[string]interface{}{
					"env": "test",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to get namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.namespace.Update(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
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
