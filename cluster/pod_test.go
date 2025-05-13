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
	t.Run("CreatePod", testCreatePods)
	t.Run("GetPod", testGetPod)
	t.Run("ListPods", testListPods)
	t.Run("DeletePod", testDeletePod)
	t.Run("StreamPodLogs", testStreamPodLogs)
}

// Pod Operations Tests
func testCreatePods(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		pod          Pod
		setupObjects []runtime.Object
		expectedText string
		expectError  bool
		errorMsg     string
	}{
		{
			name: "Create basic pod",
			pod: Pod{
				Name:      podName,
				Namespace: testNamespace,
				Image:     "nginx:latest",
			},
			setupObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectedText: "Pod \"test-pod\" created successfully",
			expectError:  false,
		},
		{
			name: "Create pod with all attributes",
			pod: Pod{
				Name:            "full-pod",
				Namespace:       testNamespace,
				Image:           "nginx:latest",
				ContainerName:   "custom-container",
				ContainerPort:   "8080/TCP",
				ImagePullPolicy: "Always",
				RestartPolicy:   "OnFailure",
				ServiceAccount:  "test-sa",
				Command:         []interface{}{"/bin/sh", "-c"},
				Args:            []interface{}{"echo hello; sleep 3600"},
				Labels: map[string]interface{}{
					"app": "test",
					"env": "dev",
				},
				Env: map[string]interface{}{
					"DEBUG": "true",
					"ENV":   "test",
				},
				NodeSelector: map[string]interface{}{
					"disktype": "ssd",
				},
				ImagePullSecrets: []interface{}{"registry-secret"},
			},
			setupObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectedText: "Pod \"full-pod\" created successfully",
			expectError:  false,
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      podName,
				Namespace: nonexistentNS,
				Image:     "nginx:latest",
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "Missing image",
			pod: Pod{
				Name:      "no-image-pod",
				Namespace: testNamespace,
				Image:     "", // Empty image
			},
			setupObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectError: true,
			errorMsg:    "failed to create pod", // The actual error would come from the k8s API
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			cm := New()
			fakeClient := fake.NewSimpleClientset(tc.setupObjects...)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			// Execute
			result, err := tc.pod.Create(ctx, cm)

			// Verify result
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedText)

				// Verify pod was created
				pod, err := fakeClient.CoreV1().Pods(tc.pod.Namespace).Get(ctx, tc.pod.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.pod.Name, pod.Name)
				assert.Equal(t, tc.pod.Namespace, pod.Namespace)

				// Check container details
				if len(pod.Spec.Containers) > 0 {
					container := pod.Spec.Containers[0]
					assert.Equal(t, tc.pod.Image, container.Image)

					if tc.pod.ContainerName != "" {
						assert.Equal(t, tc.pod.ContainerName, container.Name)
					}

					// Verify command if set
					if tc.pod.Command != nil && len(tc.pod.Command) > 0 {
						expectedCmd := make([]string, 0)
						for _, cmd := range tc.pod.Command {
							if cmdStr, ok := cmd.(string); ok {
								expectedCmd = append(expectedCmd, cmdStr)
							}
						}
						assert.Equal(t, expectedCmd, container.Command)
					}

					// Verify args if set
					if tc.pod.Args != nil && len(tc.pod.Args) > 0 {
						expectedArgs := make([]string, 0)
						for _, arg := range tc.pod.Args {
							if argStr, ok := arg.(string); ok {
								expectedArgs = append(expectedArgs, argStr)
							}
						}
						assert.Equal(t, expectedArgs, container.Args)
					}
				}

				// Check pod level details
				if tc.pod.RestartPolicy != "" {
					expectedPolicy := "Always" // Default
					switch tc.pod.RestartPolicy {
					case "OnFailure":
						expectedPolicy = "OnFailure"
					case "Never":
						expectedPolicy = "Never"
					}
					assert.Equal(t, expectedPolicy, string(pod.Spec.RestartPolicy))
				}

				// Check service account
				if tc.pod.ServiceAccount != "" {
					assert.Equal(t, tc.pod.ServiceAccount, pod.Spec.ServiceAccountName)
				}
			}
		})
	}
}

func testGetPod(t *testing.T) {
	ctx := context.Background()

	// Create test pods and namespaces
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: testNamespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	_testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
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
				Name:      podName,
				Namespace: testNamespace,
			},
			expectError: false,
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:      nonexistentPodName,
				Namespace: testNamespace,
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      podName,
				Namespace: nonexistentNS,
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cm := New()
			fakeClient := fake.NewSimpleClientset(testPod, _testNamespace)
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
				assert.Contains(t, result, podName)
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
				Namespace: testNamespace,
				Labels:    map[string]string{"app": "test"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: testNamespace,
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
				Name: testNamespace,
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
		labelSelector     string
		limit             int64
		expectError       bool
		errorMsg          string
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name: "List pods in namespace",
			pod: Pod{
				Namespace: testNamespace,
			},
			limit:             10,
			expectError:       false,
			expectedContent:   []string{"pod1", "pod2"},
			unexpectedContent: []string{"pod3"},
		},
		{
			name: "List pods with label selector",
			pod: Pod{
				Namespace: testNamespace,
			},
			labelSelector:     "app=test",
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
				Namespace: nonexistentNS,
			},
			limit:       10,
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "No pods found with label selector",
			pod: Pod{
				Namespace: testNamespace,
			},
			labelSelector: "app=nonexistent",
			limit:         10,
			expectError:   true,
			errorMsg:      "no pods found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := New()
			fakeClient := fake.NewSimpleClientset(testPods...)
			cm.clients["test-cluster"] = fakeClient
			cm.currentContext = "test-cluster"

			result, err := tc.pod.List(ctx, cm, tc.limit, tc.labelSelector, "")

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
				Name:      podName,
				Namespace: testNamespace,
			},
			force: false,
			setupObjects: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: testNamespace,
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Force delete pod",
			pod: Pod{
				Name:      "force-pod",
				Namespace: testNamespace,
			},
			force: true,
			setupObjects: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "force-pod",
						Namespace: testNamespace,
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:      nonexistentPodName,
				Namespace: testNamespace,
			},
			force: false,
			setupObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      podName,
				Namespace: nonexistentNS,
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
			Name:      podName,
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: containerName,
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pendingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pendingPodName,
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: containerName,
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	_testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
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
				Name:          podName,
				Namespace:     testNamespace,
				ContainerName: nonexistentContainer,
			},
			setupObjects:  []runtime.Object{runningPod, _testNamespace},
			expectError:   true,
			errorContains: "container 'nonexistent-container' not found",
		},
		{
			name: "Pod not running",
			pod: Pod{
				Name:          pendingPodName,
				Namespace:     testNamespace,
				ContainerName: containerName,
			},
			setupObjects:  []runtime.Object{pendingPod, _testNamespace},
			expectError:   true,
			errorContains: "pod 'pending-pod' is in 'Pending' state",
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:          nonexistentPodName,
				Namespace:     testNamespace,
				ContainerName: containerName,
			},
			setupObjects:  []runtime.Object{_testNamespace},
			expectError:   true,
			errorContains: "pod 'nonexistent-pod' not found",
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:          podName,
				Namespace:     nonexistentNS,
				ContainerName: containerName,
			},
			setupObjects:  []runtime.Object{},
			expectError:   true,
			errorContains: "namespace",
		},
		{
			name: "Previous logs for non-running pod",
			pod: Pod{
				Name:          pendingPodName,
				Namespace:     testNamespace,
				ContainerName: containerName,
			},
			setupObjects: []runtime.Object{pendingPod, _testNamespace},
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
