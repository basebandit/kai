package cluster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basebandit/kai"
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

func TestExtendedClusterManager(t *testing.T) {
	t.Run("DeleteContext", testDeleteContext)
	t.Run("GetContextInfo", testGetContextInfo)
	t.Run("RenameContext", testRenameContext)
	t.Run("ListContexts", testListContexts)
	t.Run("LoadKubeConfigDuplicateName", testLoadKubeConfigDuplicateName)
	t.Run("SetCurrentContextUpdatesActiveStatus", testSetCurrentContextUpdatesActiveStatus)
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

func testDeleteContext(t *testing.T) {
	cm := New()

	t.Run("DeleteNonexistentContext", func(t *testing.T) {
		err := cm.DeleteContext("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context nonexistent not found")
	})

	t.Run("DeleteActiveContext", func(t *testing.T) {
		fakeClient1 := fake.NewSimpleClientset()
		fakeClient2 := fake.NewSimpleClientset()

		contextInfo1 := &kai.ContextInfo{
			Name:      "context1",
			Cluster:   "cluster1",
			User:      "user1",
			Namespace: "default",
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      "context2",
			Cluster:   "cluster2",
			User:      "user2",
			Namespace: "default",
			IsActive:  false,
		}

		cm.clients["context1"] = fakeClient1
		cm.clients["context2"] = fakeClient2
		cm.contexts["context1"] = contextInfo1
		cm.contexts["context2"] = contextInfo2
		cm.currentContext = "context1"

		err := cm.DeleteContext("context1")
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, "context1")
		assert.NotContains(t, cm.clients, "context1")
		assert.NotEqual(t, "context1", cm.currentContext)
		assert.True(t, cm.contexts[cm.currentContext].IsActive)
	})

	t.Run("DeleteInactiveContext", func(t *testing.T) {
		fakeClient1 := fake.NewSimpleClientset()
		fakeClient2 := fake.NewSimpleClientset()

		contextInfo1 := &kai.ContextInfo{
			Name:      "context1",
			Cluster:   "cluster1",
			User:      "user1",
			Namespace: "default",
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      "context2",
			Cluster:   "cluster2",
			User:      "user2",
			Namespace: "default",
			IsActive:  false,
		}

		cm.clients["context1"] = fakeClient1
		cm.clients["context2"] = fakeClient2
		cm.contexts["context1"] = contextInfo1
		cm.contexts["context2"] = contextInfo2
		cm.currentContext = "context1"

		err := cm.DeleteContext("context2")
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, "context2")
		assert.NotContains(t, cm.clients, "context2")
		assert.Equal(t, "context1", cm.currentContext)
		assert.True(t, cm.contexts["context1"].IsActive)
	})
}

func testGetContextInfo(t *testing.T) {
	cm := New()

	t.Run("GetNonexistentContext", func(t *testing.T) {
		_, err := cm.GetContextInfo("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context nonexistent not found")
	})

	t.Run("GetExistingContext", func(t *testing.T) {
		expectedInfo := &kai.ContextInfo{
			Name:       "test-context",
			Cluster:    "test-cluster",
			User:       "test-user",
			Namespace:  "test-namespace",
			ServerURL:  "https://example.com:6443",
			ConfigPath: "/path/to/config",
			IsActive:   true,
		}

		cm.contexts["test-context"] = expectedInfo

		actualInfo, err := cm.GetContextInfo("test-context")
		assert.NoError(t, err)
		assert.Equal(t, expectedInfo.Name, actualInfo.Name)
		assert.Equal(t, expectedInfo.Cluster, actualInfo.Cluster)
		assert.Equal(t, expectedInfo.User, actualInfo.User)
		assert.Equal(t, expectedInfo.Namespace, actualInfo.Namespace)
		assert.Equal(t, expectedInfo.ServerURL, actualInfo.ServerURL)
		assert.Equal(t, expectedInfo.ConfigPath, actualInfo.ConfigPath)
		assert.Equal(t, expectedInfo.IsActive, actualInfo.IsActive)

		actualInfo.Name = "modified"
		assert.Equal(t, "test-context", expectedInfo.Name)
	})
}

func testRenameContext(t *testing.T) {
	cm := New()

	t.Run("RenameSameNames", func(t *testing.T) {
		err := cm.RenameContext("test", "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "old and new context names cannot be the same")
	})

	t.Run("RenameNonexistentContext", func(t *testing.T) {
		err := cm.RenameContext("nonexistent", "new-name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context nonexistent not found")
	})

	t.Run("RenameToExistingName", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		contextInfo1 := &kai.ContextInfo{Name: "context1"}
		contextInfo2 := &kai.ContextInfo{Name: "context2"}

		cm.clients["context1"] = fakeClient
		cm.clients["context2"] = fakeClient
		cm.contexts["context1"] = contextInfo1
		cm.contexts["context2"] = contextInfo2

		err := cm.RenameContext("context1", "context2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context context2 already exists")
	})

	t.Run("SuccessfulRename", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		contextInfo := &kai.ContextInfo{
			Name:      "old-context",
			Cluster:   "test-cluster",
			User:      "test-user",
			Namespace: "default",
			IsActive:  false,
		}

		cm.clients["old-context"] = fakeClient
		cm.contexts["old-context"] = contextInfo
		cm.kubeconfigs["old-context"] = "/path/to/config"

		err := cm.RenameContext("old-context", "new-context")
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, "old-context")
		assert.NotContains(t, cm.clients, "old-context")
		assert.NotContains(t, cm.kubeconfigs, "old-context")

		assert.Contains(t, cm.contexts, "new-context")
		assert.Contains(t, cm.clients, "new-context")
		assert.Contains(t, cm.kubeconfigs, "new-context")

		assert.Equal(t, "new-context", cm.contexts["new-context"].Name)
	})

	t.Run("RenameActiveContext", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		contextInfo := &kai.ContextInfo{
			Name:     "active-context",
			IsActive: true,
		}

		cm.clients["active-context"] = fakeClient
		cm.contexts["active-context"] = contextInfo
		cm.currentContext = "active-context"

		err := cm.RenameContext("active-context", "renamed-context")
		assert.NoError(t, err)

		assert.Equal(t, "renamed-context", cm.currentContext)
		assert.Equal(t, "renamed-context", cm.contexts["renamed-context"].Name)
	})
}

func testListContexts(t *testing.T) {
	cm := New()

	t.Run("EmptyContexts", func(t *testing.T) {
		contexts := cm.ListContexts()
		assert.Empty(t, contexts)
	})

	t.Run("MultipleContexts", func(t *testing.T) {
		contextInfo1 := &kai.ContextInfo{
			Name:      "context1",
			Cluster:   "cluster1",
			User:      "user1",
			Namespace: "default",
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      "context2",
			Cluster:   "cluster2",
			User:      "user2",
			Namespace: "kube-system",
			IsActive:  false,
		}

		cm.contexts["context1"] = contextInfo1
		cm.contexts["context2"] = contextInfo2

		contexts := cm.ListContexts()
		assert.Len(t, contexts, 2)

		contextNames := make(map[string]bool)
		for _, ctx := range contexts {
			contextNames[ctx.Name] = true
			ctx.Name = "modified"
		}

		assert.True(t, contextNames["context1"])
		assert.True(t, contextNames["context2"])

		assert.Equal(t, "context1", cm.contexts["context1"].Name)
		assert.Equal(t, "context2", cm.contexts["context2"].Name)
	})
}

func testLoadKubeConfigDuplicateName(t *testing.T) {
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

	cm := New()

	fakeClient := fake.NewSimpleClientset()
	contextInfo := &kai.ContextInfo{Name: "existing-context"}

	cm.clients["existing-context"] = fakeClient
	cm.contexts["existing-context"] = contextInfo

	err = cm.LoadKubeConfig("existing-context", kubeconfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context existing-context already exists")
}

func testSetCurrentContextUpdatesActiveStatus(t *testing.T) {
	cm := New()

	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	contextInfo1 := &kai.ContextInfo{
		Name:     "context1",
		IsActive: true,
	}
	contextInfo2 := &kai.ContextInfo{
		Name:     "context2",
		IsActive: false,
	}

	cm.clients["context1"] = fakeClient1
	cm.clients["context2"] = fakeClient2
	cm.contexts["context1"] = contextInfo1
	cm.contexts["context2"] = contextInfo2
	cm.currentContext = "context1"

	err := cm.SetCurrentContext("context2")
	assert.NoError(t, err)

	assert.Equal(t, "context2", cm.currentContext)
	assert.False(t, cm.contexts["context1"].IsActive)
	assert.True(t, cm.contexts["context2"].IsActive)
}
