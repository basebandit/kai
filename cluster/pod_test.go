package clustermanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
	cm := New()
	ctx := context.Background()

	// Create a fake client with a test pod
	fakeClient := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		},
	)

	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test getting an existing pod
	pod, err := cm.GetPod(ctx, "test-pod", "test-namespace")
	assert.NoError(t, err)
	assert.Contains(t, pod, "test-pod")

	// Test getting a non-existent pod
	_, err = cm.GetPod(ctx, "nonexistent-pod", "test-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting a pod in a non-existent namespace
	_, err = cm.GetPod(ctx, "test-pod", "nonexistent-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func testListPods(t *testing.T) {
	cm := New()
	ctx := context.Background()

	// Create test pods
	objects := []runtime.Object{
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

	fakeClient := fake.NewSimpleClientset(objects...)
	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test listing pods in a specific namespace
	result, err := cm.ListPods(ctx, 10, "test-namespace", "", "")
	assert.NoError(t, err)
	assert.Contains(t, result, "pod1")
	assert.Contains(t, result, "pod2")
	assert.NotContains(t, result, "pod3")

	// Test listing pods with a label selector
	result, err = cm.ListPods(ctx, 10, "test-namespace", "app=test", "")
	assert.NoError(t, err)
	assert.Contains(t, result, "pod1")
	assert.Contains(t, result, "pod2")

	// Test listing pods in all namespaces
	result, err = cm.ListPods(ctx, 10, "", "", "")
	assert.NoError(t, err)
	assert.Contains(t, result, "pod1")
	assert.Contains(t, result, "pod2")
	assert.Contains(t, result, "pod3")

	// Test listing pods in a non-existent namespace
	_, err = cm.ListPods(ctx, 10, "nonexistent-namespace", "", "")
	assert.Error(t, err)
}

func testDeletePod(t *testing.T) {
	cm := New()
	ctx := context.Background()

	// Create a fake client with a test pod
	fakeClient := fake.NewSimpleClientset(
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
	)

	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test deleting an existing pod
	result, err := cm.DeletePod(ctx, "test-pod", "test-namespace", false)
	assert.NoError(t, err)
	assert.Contains(t, result, "Successfully delete pod")

	// Verify the pod was deleted
	_, err = fakeClient.CoreV1().Pods("test-namespace").Get(ctx, "test-pod", metav1.GetOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting a non-existent pod
	_, err = cm.DeletePod(ctx, "nonexistent-pod", "test-namespace", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting a pod with force option
	fakeClient = fake.NewSimpleClientset(
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
	)
	cm.clients["test-cluster"] = fakeClient

	result, err = cm.DeletePod(ctx, "force-pod", "test-namespace", true)
	assert.NoError(t, err)
	assert.Contains(t, result, "Successfully delete pod")
}

func testStreamPodLogs(t *testing.T) {
	// NOTE: This would require a more sophisticated mock to test properly
	// as the fake client doesn't support streaming logs
	// For now, we just test the error cases with a basic fake client

	cm := New()
	ctx := context.Background()

	// Create a fake client with a test pod
	fakeClient := fake.NewSimpleClientset(
		&corev1.Pod{
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
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		},
	)

	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test with non-existent namespace
	_, err := cm.StreamPodLogs(ctx, 10, false, nil, "test-pod", "test-container", "nonexistent-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "namespace")

	// Test with non-existent pod
	_, err = cm.StreamPodLogs(ctx, 10, false, nil, "nonexistent-pod", "test-container", "test-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pod")

	// Test with non-existent container
	_, err = cm.StreamPodLogs(ctx, 10, false, nil, "test-pod", "nonexistent-container", "test-namespace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container")

	// Note: We can't fully test the streaming logs functionality with the fake client
}

// Helper for mocking the stream logs functionality
// This would be used to properly test the log streaming feature
type mockPodLogStream struct {
	content string
}

func (m *mockPodLogStream) Read(p []byte) (n int, err error) {
	copy(p, []byte(m.content))
	return len(m.content), nil
}

func (m *mockPodLogStream) Close() error {
	return nil
}
