package kai

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// MockKubernetesInterface is a mock for kubernetes.Interface
type MockKubernetesInterface struct {
	mock.Mock
	kubernetes.Interface
}

// TestClusterManager groups all ClusterManager tests
func TestClusterManager(t *testing.T) {
	t.Run("Creation", testNewClusterManager)
	t.Run("Namespace", testNamespaceOperations)
	t.Run("Context", testContextOperations)
	t.Run("Clients", testClientOperations)
	t.Run("ListClusters", testListClusters)
}

// TestKubeConfigLoading groups all kubeconfig loading related tests
func TestKubeConfigLoading(t *testing.T) {
	t.Run("ValidateInputs", testValidateInputs)
	t.Run("ResolvePath", testResolvePath)
	t.Run("ValidateFile", testValidateFile)
	t.Run("LoadKubeConfig", testLoadKubeConfig)
}

// TestPodOperations groups all pod-related operations tests
func TestPodOperations(t *testing.T) {
	t.Run("GetPod", testGetPod)
	t.Run("ListPods", testListPods)
	t.Run("DeletePod", testDeletePod)
	t.Run("StreamPodLogs", testStreamPodLogs)
}

// TestDeploymentOperations groups all deployment-related tests
func TestDeploymentOperations(t *testing.T) {
	t.Run("ListDeployments", testListDeployments)
}

// Individual test functions

func testNewClusterManager(t *testing.T) {
	cm := NewClusterManager()
	assert.NotNil(t, cm)
	assert.Equal(t, "default", cm.GetCurrentNamespace())
	assert.Empty(t, cm.ListClusters())
}

func testNamespaceOperations(t *testing.T) {
	cm := NewClusterManager()

	// Test default
	assert.Equal(t, "default", cm.GetCurrentNamespace())

	// Test setting to a new value
	cm.SetCurrentNamespace("test-namespace")
	assert.Equal(t, "test-namespace", cm.GetCurrentNamespace())

	// Test setting to empty (should revert to default)
	cm.SetCurrentNamespace("")
	assert.Equal(t, "default", cm.GetCurrentNamespace())
}

func testContextOperations(t *testing.T) {
	cm := NewClusterManager()

	// Add a fake client
	fakeClient := fake.NewSimpleClientset()
	cm.clients["test-cluster"] = fakeClient

	// Test setting to a valid context
	err := cm.SetCurrentContext("test-cluster")
	assert.NoError(t, err)
	assert.Equal(t, "test-cluster", cm.GetCurrentContext())

	// Test setting to an invalid context
	err = cm.SetCurrentContext("nonexistent-cluster")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster nonexistent-cluster not found")
}

func testClientOperations(t *testing.T) {
	cm := NewClusterManager()

	// Test with no clients
	client, err := cm.GetCurrentClient()
	assert.Error(t, err)
	assert.Nil(t, client)

	// Add a fake client
	fakeClient := fake.NewSimpleClientset()
	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test getting the current client
	client, err = cm.GetCurrentClient()
	assert.NoError(t, err)
	assert.Equal(t, fakeClient, client)

	// Test getting a specific client
	client, err = cm.GetClient("test-cluster")
	assert.NoError(t, err)
	assert.Equal(t, fakeClient, client)

	// Test getting a non-existent client
	client, err = cm.GetClient("nonexistent-cluster")
	assert.Error(t, err)
	assert.Nil(t, client)

	// Test getting dynamic clients
	dynamicClient, err := cm.GetCurrentDynamicClient()
	assert.Error(t, err) // We haven't set any dynamic clients
	assert.Nil(t, dynamicClient)
}

func testListClusters(t *testing.T) {
	cm := NewClusterManager()

	// Test with no clients
	clusters := cm.ListClusters()
	assert.Empty(t, clusters)

	// Add some fake clients
	cm.clients["cluster1"] = fake.NewSimpleClientset()
	cm.clients["cluster2"] = fake.NewSimpleClientset()

	// Test listing clusters
	clusters = cm.ListClusters()
	assert.Len(t, clusters, 2)
	assert.Contains(t, clusters, "cluster1")
	assert.Contains(t, clusters, "cluster2")
}

// KubeConfig Loading Tests

func testValidateInputs(t *testing.T) {
	// Test with empty cluster name
	err := validateInputs("", "/path/to/config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name cannot be empty")

	// Test with valid inputs
	err = validateInputs("test-cluster", "/path/to/config")
	assert.NoError(t, err)
}

func testResolvePath(t *testing.T) {
	// Test with empty path
	path, err := resolvePath("")
	assert.NoError(t, err)
	assert.Contains(t, path, ".kube/config")

	// Test with provided path
	testPath := "/path/to/config"
	path, err = resolvePath(testPath)
	assert.NoError(t, err)
	assert.Equal(t, testPath, path)
}

func testValidateFile(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "config")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Create a directory
	dirPath := filepath.Join(tempDir, "configdir")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Test with non-existent file
	err = validateFile("/nonexistent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error accessing file")

	// Test with directory
	err = validateFile(dirPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory, not a file")

	// Test with valid file
	err = validateFile(filePath)
	assert.NoError(t, err)
}

func testLoadKubeConfig(t *testing.T) {
	// Create a temporary kubeconfig file for testing
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	// Sample minimal kubeconfig content
	kubeconfigContent := `
apiVersion: v1
kind: Config
current-context: test-context
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
clusters:
- name: test-cluster
  cluster:
    server: https://example.com
users:
- name: test-user
  user:
    token: test-token
`
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	require.NoError(t, err)

	t.Run("EmptyClusterName", func(t *testing.T) {
		cm := NewClusterManager()
		err := cm.LoadKubeConfig("", kubeconfigPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cluster name cannot be empty")
	})

	t.Run("EmptyPath", func(t *testing.T) {
		// This test might pass or fail depending on whether you have a valid kubeconfig in the default location
		// So we'll just make sure it doesn't panic
		cm := NewClusterManager()
		_ = cm.LoadKubeConfig("default", "")
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		cm := NewClusterManager()
		err := cm.LoadKubeConfig("test", "/path/does/not/exist")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error accessing file")
	})

	// For a full integration test, we'd need to mock the k8s client creation and API calls,
	// which would require significant refactoring of the original code.
}

// Pod Operations Tests

func testGetPod(t *testing.T) {
	cm := NewClusterManager()
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
	cm := NewClusterManager()
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
	cm := NewClusterManager()
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

	cm := NewClusterManager()
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

func testListDeployments(t *testing.T) {
	cm := NewClusterManager()
	ctx := context.Background()

	// Create test deployments
	objects := []runtime.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment1",
				Namespace: "test-namespace",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment2",
				Namespace: "test-namespace",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment3",
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

	// Test listing deployments in a specific namespace
	result, err := cm.ListDeployments(ctx, false, "", "test-namespace")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.Contains(t, result, "deployment2")
	assert.NotContains(t, result, "deployment3")

	// Test listing deployments with a label selector
	result, err = cm.ListDeployments(ctx, false, "app=test", "test-namespace")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.NotContains(t, result, "deployment2")

	// Test listing deployments in all namespaces
	result, err = cm.ListDeployments(ctx, true, "", "")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.Contains(t, result, "deployment2")
	assert.Contains(t, result, "deployment3")
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
