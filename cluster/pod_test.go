package cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testClusterName = "test-cluster"
)

var (
	shellCommand = []interface{}{"/bin/sh", "-c"}
	sleepArgs    = []interface{}{"echo hello; sleep 3600"}
	testLabels   = map[string]interface{}{
		"app": "test",
		"env": "dev",
	}
	testEnv = map[string]interface{}{
		"DEBUG": "true",
		"ENV":   "test",
	}
	ssdNodeSelector = map[string]interface{}{
		"disktype": "ssd",
	}
)

// TestPodOperations groups all pod-related operations tests
func TestPodOperations(t *testing.T) {
	t.Run("CreatePod", testCreatePods)
	t.Run("GetPod", testGetPod)
	t.Run("ListPods", testListPods)
	t.Run("DeletePod", testDeletePod)
	t.Run("StreamPodLogs", testStreamPodLogs)
}

// createNamespace creates a namespace object for testing
func createNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// createPod creates a pod object for testing
func createPod(name, namespace string, phase corev1.PodPhase, containers ...string) *corev1.Pod {
	containerList := make([]corev1.Container, 0, len(containers))
	for _, c := range containers {
		containerList = append(containerList, corev1.Container{
			Name: c,
		})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: containerList,
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
}

// createPodWithLabels creates a pod with labels for testing
func createPodWithLabels(name, namespace string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

// setupTestCluster creates a test cluster manager with the given objects
func setupTestCluster(objects ...runtime.Object) *Cluster {
	cm := New()
	fakeClient := fake.NewSimpleClientset(objects...)
	cm.clients[testClusterName] = fakeClient
	cm.currentContext = testClusterName
	return cm
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
				Image:     nginxImage,
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: fmt.Sprintf(podCreatedFmt, podName),
			expectError:  false,
		},
		{
			name: "Create pod with all attributes",
			pod: Pod{
				Name:             fullPodName,
				Namespace:        testNamespace,
				Image:            nginxImage,
				ContainerName:    customContainer,
				ContainerPort:    "8080/TCP",
				ImagePullPolicy:  alwaysPullPolicy,
				RestartPolicy:    onFailurePolicy,
				ServiceAccount:   testServiceAccount,
				Command:          shellCommand,
				Args:             sleepArgs,
				Labels:           testLabels,
				Env:              testEnv,
				NodeSelector:     ssdNodeSelector,
				ImagePullSecrets: []interface{}{registrySecret},
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectedText: fmt.Sprintf(podCreatedFmt, fullPodName),
			expectError:  false,
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      podName,
				Namespace: nonexistentNS,
				Image:     nginxImage,
			},
			setupObjects: []runtime.Object{},
			expectError:  true,
			errorMsg:     fmt.Sprintf("namespace %q not found", nonexistentNS),
		},
		{
			name: "Missing image",
			pod: Pod{
				Name:      noImagePodName,
				Namespace: testNamespace,
				Image:     "", // Empty image
			},
			setupObjects: []runtime.Object{
				createNamespace(testNamespace),
			},
			expectError: true,
			errorMsg:    "failed to create pod", // The actual error would come from the k8s API
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			cm := setupTestCluster(tc.setupObjects...)

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
				fakeClient := cm.clients[testClusterName]
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
					if len(tc.pod.Command) > 0 {
						expectedCmd := make([]string, 0)
						for _, cmd := range tc.pod.Command {
							if cmdStr, ok := cmd.(string); ok {
								expectedCmd = append(expectedCmd, cmdStr)
							}
						}
						assert.Equal(t, expectedCmd, container.Command)
					}

					// Verify args if set
					if len(tc.pod.Args) > 0 {
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
					expectedPolicy := string(corev1.RestartPolicyAlways) // Default
					switch tc.pod.RestartPolicy {
					case onFailurePolicy:
						expectedPolicy = string(corev1.RestartPolicyOnFailure)
					case neverPolicy:
						expectedPolicy = string(corev1.RestartPolicyNever)
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

	// Create common test objects
	runningPod := createPod(podName, testNamespace, corev1.PodRunning)
	testNamespaceObj := createNamespace(testNamespace)

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
			errorMsg:    notFoundErrMsg,
		},
		{
			name: "Namespace not found",
			pod: Pod{
				Name:      podName,
				Namespace: nonexistentNS,
			},
			expectError: true,
			errorMsg:    notFoundErrMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(runningPod, testNamespaceObj)

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

	// Create test objects
	pod1WithLabel := createPodWithLabels(pod1Name, testNamespace, map[string]string{"app": "test"})
	pod2WithLabel := createPodWithLabels(pod2Name, testNamespace, map[string]string{"app": "test"})
	pod3DiffNS := createPodWithLabels(pod3Name, otherNamespace, nil)
	testNamespaceObj := createNamespace(testNamespace)
	otherNamespaceObj := createNamespace(otherNamespace)

	// Create test pods collection
	testPods := []runtime.Object{
		pod1WithLabel,
		pod2WithLabel,
		pod3DiffNS,
		testNamespaceObj,
		otherNamespaceObj,
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
			expectedContent:   []string{pod1Name, pod2Name},
			unexpectedContent: []string{pod3Name},
		},
		{
			name: "List pods with label selector",
			pod: Pod{
				Namespace: testNamespace,
			},
			labelSelector:     "app=test",
			limit:             10,
			expectError:       false,
			expectedContent:   []string{pod1Name, pod2Name},
			unexpectedContent: []string{pod3Name},
		},
		{
			name: "List pods in all namespaces",
			pod: Pod{
				Namespace: "",
			},
			limit:           10,
			expectError:     false,
			expectedContent: []string{pod1Name, pod2Name, pod3Name},
		},
		{
			name: "List pods in non-existent namespace",
			pod: Pod{
				Namespace: nonexistentNS,
			},
			limit:       10,
			expectError: true,
			errorMsg:    notFoundErrMsg,
		},
		{
			name: "No pods found with label selector",
			pod: Pod{
				Namespace: testNamespace,
			},
			labelSelector: "app=nonexistent",
			limit:         10,
			expectError:   true,
			errorMsg:      noPodsFoundMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(testPods...)

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
				createPod(podName, testNamespace, corev1.PodRunning),
				createNamespace(testNamespace),
			},
			expectError: false,
		},
		{
			name: "Force delete pod",
			pod: Pod{
				Name:      forcePodName,
				Namespace: testNamespace,
			},
			force: true,
			setupObjects: []runtime.Object{
				createPod(forcePodName, testNamespace, corev1.PodRunning),
				createNamespace(testNamespace),
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
				createNamespace(testNamespace),
			},
			expectError: true,
			errorMsg:    notFoundErrMsg,
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
			errorMsg:     notFoundErrMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)
			fakeClient := cm.clients[testClusterName]

			result, err := tc.pod.Delete(ctx, cm, tc.force)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, deleteSuccessMsg)

				// Verify the pod was deleted
				_, err = fakeClient.CoreV1().Pods(tc.pod.Namespace).Get(ctx, tc.pod.Name, metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), notFoundErrMsg)
			}
		})
	}
}

func testStreamPodLogs(t *testing.T) {
	ctx := context.Background()

	// Create test objects
	runningPod := createPod(podName, testNamespace, corev1.PodRunning, containerName)
	pendingPod := createPod(pendingPodName, testNamespace, corev1.PodPending, containerName)
	testNamespaceObj := createNamespace(testNamespace)

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
			setupObjects:  []runtime.Object{runningPod, testNamespaceObj},
			expectError:   true,
			errorContains: fmt.Sprintf("container '%s' not found", nonexistentContainer),
		},
		{
			name: "Pod not running",
			pod: Pod{
				Name:          pendingPodName,
				Namespace:     testNamespace,
				ContainerName: containerName,
			},
			setupObjects:  []runtime.Object{pendingPod, testNamespaceObj},
			expectError:   true,
			errorContains: fmt.Sprintf("pod '%s' is in 'Pending' state", pendingPodName),
		},
		{
			name: "Pod not found",
			pod: Pod{
				Name:          nonexistentPodName,
				Namespace:     testNamespace,
				ContainerName: containerName,
			},
			setupObjects:  []runtime.Object{testNamespaceObj},
			expectError:   true,
			errorContains: fmt.Sprintf("pod '%s' not found", nonexistentPodName),
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
			setupObjects: []runtime.Object{pendingPod, testNamespaceObj},
			previous:     true,  // Should bypass running state check
			expectError:  false, // No error in validation, but will fail in the fake client
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := setupTestCluster(tc.setupObjects...)

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
