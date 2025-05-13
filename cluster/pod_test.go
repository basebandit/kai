package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestPodOperations groups all pod-related operations tests
func TestPodOperations(t *testing.T) {
	t.Run("GetPod", testGetPod)
	t.Run("ListPods", testListPods)
	t.Run("DeletePod", testDeletePod)
	t.Run("StreamPodLogs", testStreamPodLogs)
}

// Pod Operations Tests
func testGetPod(t *testing.T) {
	ctx := context.Background()

	// Create test pods and namespaces
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Define test cases
	testCases := []struct {
		name        string
		pod         Pod
		expectError bool
		errorMsg    string
	}{
		{
			name: "Get existing pod",
			pod: Pod{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			expectError: false,
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:      "nonexistent-pod",
				Namespace: "test-namespace",
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      "test-pod",
				Namespace: "nonexistent-namespace",
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cm := New()
			fakeClient := fake.NewSimpleClientset(testPod, testNamespace)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			result, err := tc.pod.Get(ctx, cm)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "test-pod")
			}
		})
	}
}

func testListPods(t *testing.T) {
	ctx := context.Background()

	// Create test pods
	testPods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "test-namespace",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "test-namespace",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "other-namespace",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other-namespace",
			},
		},
	}

	// Define test cases
	testCases := []struct {
		name              string
		pod               Pod
		limit             int64
		expectError       bool
		errorMsg          string
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name: "List pods in namespace",
			pod: Pod{
				Namespace: "test-namespace",
			},
			limit:             10,
			expectError:       false,
			expectedContent:   []string{"pod1", "pod2"},
			unexpectedContent: []string{"pod3"},
		},
		{
			name: "List pods with label selector",
			pod: Pod{
				Namespace:     "test-namespace",
				LabelSelector: "app=test",
			},
			limit:             10,
			expectError:       false,
			expectedContent:   []string{"pod1", "pod2"},
			unexpectedContent: []string{"pod3"},
		},
		{
			name: "List pods in all namespaces",
			pod: Pod{
				Namespace: "",
			},
			limit:           10,
			expectError:     false,
			expectedContent: []string{"pod1", "pod2", "pod3"},
		},
		{
			name: "List pods in non-existent namespace",
			pod: Pod{
				Namespace: "nonexistent-namespace",
			},
			limit:       10,
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "No pods found with label selector",
			pod: Pod{
				Namespace:     "test-namespace",
				LabelSelector: "app=nonexistent",
			},
			limit:       10,
			expectError: true,
			errorMsg:    "no pods found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := New()
			fakeClient := fake.NewSimpleClientset(testPods...)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			result, err := tc.pod.List(ctx, cm, tc.limit)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Check for expected content
				for _, expected := range tc.expectedContent {
					assert.Contains(t, result, expected)
				}

				// Check for unexpected content
				for _, unexpected := range tc.unexpectedContent {
					assert.NotContains(t, result, unexpected)
				}
			}
		})
	}
}

func testDeletePod(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		pod          Pod
		force        bool
		setupObjects []runtime.Object
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Delete existing pod",
			pod: Pod{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			force: false,
			setupObjects: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "test-namespace",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Force delete pod",
			pod: Pod{
				Name:      "force-pod",
				Namespace: "test-namespace",
			},
			force: true,
			setupObjects: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "force-pod",
						Namespace: "test-namespace",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:      "nonexistent-pod",
				Namespace: "test-namespace",
			},
			force: false,
			setupObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      "test-pod",
				Namespace: "nonexistent-namespace",
			},
			force:        false,
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := New()
			fakeClient := fake.NewSimpleClientset(tc.setupObjects...)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			result, err := tc.pod.Delete(ctx, cm, tc.force)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "Successfully delete pod")

				// Verify the pod was deleted
				_, err = fakeClient.CoreV1().Pods(tc.pod.Namespace).Get(ctx, tc.pod.Name, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			}
		})
	}
}

func testStreamPodLogs(t *testing.T) {
	ctx := context.Background()

	// Create test pods for reuse
	runningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pendingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Define test cases
	testCases := []struct {
		name          string
		pod           Pod
		setupObjects  []runtime.Object
		tailLines     int64
		previous      bool
		since         *time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name: "Container not found",
			pod: Pod{
				Name:          "test-pod",
				Namespace:     "test-namespace",
				ContainerName: "nonexistent-container",
			},
			setupObjects:  []runtime.Object{runningPod, testNamespace},
			expectError:   true,
			errorContains: "container 'nonexistent-container' not found",
		},
		{
			name: "Pod not running",
			pod: Pod{
				Name:          "pending-pod",
				Namespace:     "test-namespace",
				ContainerName: "test-container",
			},
			setupObjects:  []runtime.Object{pendingPod, testNamespace},
			expectError:   true,
			errorContains: "pod 'pending-pod' is in 'Pending' state",
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:          "nonexistent-pod",
				Namespace:     "test-namespace",
				ContainerName: "test-container",
			},
			setupObjects:  []runtime.Object{testNamespace},
			expectError:   true,
			errorContains: "pod 'nonexistent-pod' not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:          "test-pod",
				Namespace:     "nonexistent-namespace",
				ContainerName: "test-container",
			},
			setupObjects:  []runtime.Object{},
			expectError:   true,
			errorContains: "namespace",
		},
		{
			name: "Previous logs for non-running pod",
			pod: Pod{
				Name:          "pending-pod",
				Namespace:     "test-namespace",
				ContainerName: "test-container",
			},
			setupObjects: []runtime.Object{pendingPod, testNamespace},
			previous:     true,  // Should bypass running state check
			expectError:  false, // No error in validation, but will fail in the fake client
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cm := New()
			fakeClient := fake.NewSimpleClientset(tc.setupObjects...)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			_, err := tc.pod.StreamLogs(ctx, cm, tc.tailLines, tc.previous, tc.since)

			// Note that with fake client, the actual streaming will fail
			// so we're mainly testing the validation logic
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.name == "Previous logs for non-running pod" {
				// This will likely fail with fake client during streaming
				// but it should pass the state validation check
				if err != nil {
					// Ensure it's not failing due to pod state
					assert.NotContains(t, err.Error(), "is in 'Pending' state")
				}
			}
		})
	}
}
