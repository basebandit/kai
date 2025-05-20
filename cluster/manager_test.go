package cluster

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

func testNewClusterManager(t *testing.T) {
	cm := New()
	assert.NotNil(t, cm)
	assert.Equal(t, defaultNamespace, cm.GetCurrentNamespace())
	assert.Empty(t, cm.ListClusters())
}

func testNamespaceOperations(t *testing.T) {
	cm := New()
	assert.Equal(t, defaultNamespace, cm.GetCurrentNamespace())

	cm.SetCurrentNamespace(testNamespace)
	assert.Equal(t, testNamespace, cm.GetCurrentNamespace())

	cm.SetCurrentNamespace("")
	assert.Equal(t, defaultNamespace, cm.GetCurrentNamespace())
}

func testContextOperations(t *testing.T) {
	cm := New()
	fakeClient := fake.NewSimpleClientset()
	cm.clients[testClusterName] = fakeClient

	err := cm.SetCurrentContext(testClusterName)
	assert.NoError(t, err)
	assert.Equal(t, testClusterName, cm.GetCurrentContext())

	err = cm.SetCurrentContext("nonexistent-cluster")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster nonexistent-cluster not found")
}

func testClientOperations(t *testing.T) {
	cm := New()
	client, err := cm.GetCurrentClient()
	assert.Error(t, err)
	assert.Nil(t, client)

	fakeClient := fake.NewSimpleClientset()
	cm.clients[testClusterName] = fakeClient
	cm.currentContext = testClusterName

	client, err = cm.GetCurrentClient()
	assert.NoError(t, err)
	assert.Equal(t, fakeClient, client)

	client, err = cm.GetClient(testClusterName)
	assert.NoError(t, err)
	assert.Equal(t, fakeClient, client)

	client, err = cm.GetClient("nonexistent-cluster")
	assert.Error(t, err)
	assert.Nil(t, client)

	dynamicClient, err := cm.GetCurrentDynamicClient()
	assert.Error(t, err) // We haven't set any dynamic clients
	assert.Nil(t, dynamicClient)
}

func testListClusters(t *testing.T) {
	cm := New()
	clusters := cm.ListClusters()
	assert.Empty(t, clusters)

	cm.clients["cluster1"] = fake.NewSimpleClientset()
	cm.clients["cluster2"] = fake.NewSimpleClientset()

	clusters = cm.ListClusters()
	assert.Len(t, clusters, 2)
	assert.Contains(t, clusters, "cluster1")
	assert.Contains(t, clusters, "cluster2")
}

func testValidateInputs(t *testing.T) {
	err := validateInputs("", "/path/to/config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name cannot be empty")

	err = validateInputs(testClusterName, "/path/to/config")
	assert.NoError(t, err)
}

func testResolvePath(t *testing.T) {
	path, err := resolvePath("")
	assert.NoError(t, err)
	assert.Contains(t, path, ".kube/config")

	testPath := "/path/to/config"
	path, err = resolvePath(testPath)
	assert.NoError(t, err)
	assert.Equal(t, testPath, path)
}

func testValidateFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "config")
	err := os.WriteFile(filePath, []byte("test"), 0600)
	require.NoError(t, err)

	dirPath := filepath.Join(tempDir, "configdir")
	err = os.Mkdir(dirPath, 0700)
	require.NoError(t, err)

	err = validateFile("/nonexistent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error accessing file")

	err = validateFile(dirPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory, not a file")

	err = validateFile(filePath)
	assert.NoError(t, err)
}

func testLoadKubeConfig(t *testing.T) {
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

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
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
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
}
