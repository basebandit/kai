package clustermanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

// Individual test functions

func testNewClusterManager(t *testing.T) {
	cm := New()
	assert.NotNil(t, cm)
	assert.Equal(t, "default", cm.GetCurrentNamespace())
	assert.Empty(t, cm.ListClusters())
}

func testNamespaceOperations(t *testing.T) {
	cm := New()

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
	cm := New()

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
	cm := New()

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
	cm := New()

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
		cm := New()
		err := cm.LoadKubeConfig("", kubeconfigPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cluster name cannot be empty")
	})

	t.Run("EmptyPath", func(t *testing.T) {
		// This test might pass or fail depending on whether you have a valid kubeconfig in the default location
		// So we'll just make sure it doesn't panic
		cm := New()
		_ = cm.LoadKubeConfig("default", "")
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		cm := New()
		err := cm.LoadKubeConfig("test", "/path/does/not/exist")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error accessing file")
	})

	// For a full integration test, we'd need to mock the k8s client creation and API calls,
	// which would require significant refactoring of the original code.
}
