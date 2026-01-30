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
	"k8s.io/client-go/tools/clientcmd"
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
	t.Run("UpdateKubeconfigCurrentContext", testUpdateKubeconfigCurrentContext)
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
	cm.clients[testCluster] = fakeClient

	err := cm.SetCurrentContext(testCluster)
	assert.NoError(t, err)
	assert.Equal(t, testCluster, cm.GetCurrentContext())

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
	cm.clients[testCluster] = fakeClient
	cm.currentContext = testCluster

	client, err = cm.GetCurrentClient()
	assert.NoError(t, err)
	assert.Equal(t, fakeClient, client)

	client, err = cm.GetClient(testCluster)
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

	cm.clients[testCluster1] = fake.NewSimpleClientset()
	cm.clients[testCluster2] = fake.NewSimpleClientset()

	clusters = cm.ListClusters()
	assert.Len(t, clusters, 2)
	assert.Contains(t, clusters, testCluster1)
	assert.Contains(t, clusters, testCluster2)
}

func testValidateInputs(t *testing.T) {
	err := validateInputs("", "/path/to/config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster name cannot be empty")

	err = validateInputs(testCluster, "/path/to/config")
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
		_ = cm.LoadKubeConfig(testNamespace, "")
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
			Name:      testContext1,
			Cluster:   testCluster1,
			User:      testUser1,
			Namespace: testNamespace,
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      testContext2,
			Cluster:   testCluster2,
			User:      testUser2,
			Namespace: testNamespace,
			IsActive:  false,
		}

		cm.clients[testContext1] = fakeClient1
		cm.clients[testContext2] = fakeClient2
		cm.contexts[testContext1] = contextInfo1
		cm.contexts[testContext2] = contextInfo2
		cm.currentContext = testContext1

		err := cm.DeleteContext(testContext1)
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, testContext1)
		assert.NotContains(t, cm.clients, testContext1)
		assert.NotEqual(t, testContext1, cm.currentContext)
		assert.True(t, cm.contexts[cm.currentContext].IsActive)
	})

	t.Run("DeleteInactiveContext", func(t *testing.T) {
		fakeClient1 := fake.NewSimpleClientset()
		fakeClient2 := fake.NewSimpleClientset()

		contextInfo1 := &kai.ContextInfo{
			Name:      testContext1,
			Cluster:   testCluster1,
			User:      testUser1,
			Namespace: testNamespace,
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      testContext2,
			Cluster:   testCluster2,
			User:      testUser2,
			Namespace: testNamespace,
			IsActive:  false,
		}

		cm.clients[testContext1] = fakeClient1
		cm.clients[testContext2] = fakeClient2
		cm.contexts[testContext1] = contextInfo1
		cm.contexts[testContext2] = contextInfo2
		cm.currentContext = testContext1

		err := cm.DeleteContext(testContext2)
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, testContext2)
		assert.NotContains(t, cm.clients, testContext2)
		assert.Equal(t, testContext1, cm.currentContext)
		assert.True(t, cm.contexts[testContext1].IsActive)
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
			Name:       testContext,
			Cluster:    testCluster,
			User:       testUser,
			Namespace:  testNamespace,
			ServerURL:  "https://example.com:6443",
			ConfigPath: "/path/to/config",
			IsActive:   true,
		}

		cm.contexts[testContext] = expectedInfo

		actualInfo, err := cm.GetContextInfo(testContext)
		assert.NoError(t, err)
		assert.Equal(t, expectedInfo.Name, actualInfo.Name)
		assert.Equal(t, expectedInfo.Cluster, actualInfo.Cluster)
		assert.Equal(t, expectedInfo.User, actualInfo.User)
		assert.Equal(t, expectedInfo.Namespace, actualInfo.Namespace)
		assert.Equal(t, expectedInfo.ServerURL, actualInfo.ServerURL)
		assert.Equal(t, expectedInfo.ConfigPath, actualInfo.ConfigPath)
		assert.Equal(t, expectedInfo.IsActive, actualInfo.IsActive)

		actualInfo.Name = "modified"
		assert.Equal(t, testContext, expectedInfo.Name)
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

		contextInfo1 := &kai.ContextInfo{Name: testContext1}
		contextInfo2 := &kai.ContextInfo{Name: testContext2}

		cm.clients[testContext1] = fakeClient
		cm.clients[testContext2] = fakeClient
		cm.contexts[testContext1] = contextInfo1
		cm.contexts[testContext2] = contextInfo2

		err := cm.RenameContext(testContext1, testContext2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context context2 already exists")
	})

	t.Run("SuccessfulRename", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		contextInfo := &kai.ContextInfo{
			Name:      oldContext,
			Cluster:   testCluster,
			User:      testUser,
			Namespace: testNamespace,
			IsActive:  false,
		}

		cm.clients[oldContext] = fakeClient
		cm.contexts[oldContext] = contextInfo
		cm.kubeconfigs[oldContext] = "/path/to/config"

		err := cm.RenameContext(oldContext, newContext)
		assert.NoError(t, err)

		assert.NotContains(t, cm.contexts, oldContext)
		assert.NotContains(t, cm.clients, oldContext)
		assert.NotContains(t, cm.kubeconfigs, oldContext)

		assert.Contains(t, cm.contexts, newContext)
		assert.Contains(t, cm.clients, newContext)
		assert.Contains(t, cm.kubeconfigs, newContext)

		assert.Equal(t, newContext, cm.contexts[newContext].Name)
	})

	t.Run("RenameActiveContext", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		contextInfo := &kai.ContextInfo{
			Name:     activeContext,
			IsActive: true,
		}

		cm.clients[activeContext] = fakeClient
		cm.contexts[activeContext] = contextInfo
		cm.currentContext = activeContext

		err := cm.RenameContext(activeContext, renamedContext)
		assert.NoError(t, err)

		assert.Equal(t, renamedContext, cm.currentContext)
		assert.Equal(t, renamedContext, cm.contexts[renamedContext].Name)
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
			Name:      testContext1,
			Cluster:   testCluster1,
			User:      testUser1,
			Namespace: testNamespace,
			IsActive:  true,
		}
		contextInfo2 := &kai.ContextInfo{
			Name:      testContext2,
			Cluster:   testCluster2,
			User:      testUser2,
			Namespace: "kube-system",
			IsActive:  false,
		}

		cm.contexts[testContext1] = contextInfo1
		cm.contexts[testContext2] = contextInfo2

		contexts := cm.ListContexts()
		assert.Len(t, contexts, 2)
		assert.Equal(t, testContext1, contexts[0].Name)
		assert.Equal(t, testContext2, contexts[1].Name)

		contextNames := make(map[string]bool)
		for _, ctx := range contexts {
			contextNames[ctx.Name] = true
			ctx.Name = "modified"
		}

		assert.True(t, contextNames[testContext1])
		assert.True(t, contextNames[testContext2])

		assert.Equal(t, testContext1, cm.contexts[testContext1].Name)
		assert.Equal(t, testContext2, cm.contexts[testContext2].Name)
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
	contextInfo := &kai.ContextInfo{Name: existingContext}

	cm.clients[existingContext] = fakeClient
	cm.contexts[existingContext] = contextInfo

	err = cm.LoadKubeConfig(existingContext, kubeconfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context existing-context already exists")
}

func testSetCurrentContextUpdatesActiveStatus(t *testing.T) {
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	// Create a test kubeconfig file
	kubeconfigContent := `
apiVersion: v1
kind: Config
current-context: context1
contexts:
- name: context1
  context:
    cluster: cluster1
    user: user1
- name: context2
  context:
    cluster: cluster2
    user: user2
clusters:
- name: cluster1
  cluster:
    server: https://example1.com
- name: cluster2
  cluster:
    server: https://example2.com
users:
- name: user1
  user:
    token: token1
- name: user2
  user:
    token: token2
`
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	require.NoError(t, err)

	cm := New()

	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	contextInfo1 := &kai.ContextInfo{
		Name:       testContext1,
		ConfigPath: kubeconfigPath,
		IsActive:   true,
	}
	contextInfo2 := &kai.ContextInfo{
		Name:       testContext2,
		ConfigPath: kubeconfigPath,
		IsActive:   false,
	}

	cm.clients[testContext1] = fakeClient1
	cm.clients[testContext2] = fakeClient2
	cm.contexts[testContext1] = contextInfo1
	cm.contexts[testContext2] = contextInfo2
	cm.currentContext = testContext1

	err = cm.SetCurrentContext(testContext2)
	assert.NoError(t, err)

	assert.Equal(t, testContext2, cm.currentContext)
	assert.False(t, cm.contexts[testContext1].IsActive)
	assert.True(t, cm.contexts[testContext2].IsActive)

	// #nosec G304 - we are writing in a temp dir
	updatedBytes, err := os.ReadFile(kubeconfigPath)
	assert.NoError(t, err)

	config, err := clientcmd.Load(updatedBytes)
	assert.NoError(t, err)
	assert.Equal(t, testContext2, config.CurrentContext)
}

func testUpdateKubeconfigCurrentContext(t *testing.T) {
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	// Create initial kubeconfig with context1 as current
	kubeconfigContent := `
apiVersion: v1
kind: Config
current-context: context1
contexts:
- name: context1
  context:
    cluster: cluster1
    user: user1
- name: context2
  context:
    cluster: cluster2
    user: user2
clusters:
- name: cluster1
  cluster:
    server: https://example1.com
- name: cluster2
  cluster:
    server: https://example2.com
users:
- name: user1
  user:
    token: token1
- name: user2
  user:
    token: token2
`
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	require.NoError(t, err)

	cm := New()

	t.Run("UpdateToExistingContext", func(t *testing.T) {
		err := cm.updateKubeconfigCurrentContext(testContext2, kubeconfigPath)
		assert.NoError(t, err)

		// #nosec G304
		updatedBytes, err := os.ReadFile(kubeconfigPath)
		assert.NoError(t, err)

		config, err := clientcmd.Load(updatedBytes)
		assert.NoError(t, err)
		assert.Equal(t, testContext2, config.CurrentContext)
	})

	t.Run("UpdateToNonexistentContext", func(t *testing.T) {
		err := cm.updateKubeconfigCurrentContext("nonexistent", kubeconfigPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context nonexistent not found in kubeconfig")
	})

	t.Run("UpdateWithPrefixedContextName", func(t *testing.T) {
		// Test when our internal context name has a prefix
		err := cm.updateKubeconfigCurrentContext("prefix-context1", kubeconfigPath)
		assert.NoError(t, err)

		// #nosec G304
		updatedBytes, err := os.ReadFile(kubeconfigPath)
		assert.NoError(t, err)

		config, err := clientcmd.Load(updatedBytes)
		assert.NoError(t, err)
		assert.Equal(t, testContext1, config.CurrentContext)
	})

	t.Run("UpdateNonexistentFile", func(t *testing.T) {
		err := cm.updateKubeconfigCurrentContext(testContext1, "/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading kubeconfig file")
	})
}

func TestPortForwardOperations(t *testing.T) {
	t.Run("ListPortForwards", testListPortForwards)
	t.Run("StopPortForward", testStopPortForward)
	t.Run("StartPortForwardErrors", testStartPortForwardErrors)
}

func testListPortForwards(t *testing.T) {
	cm := New()

	// Clear any existing sessions
	pfMutex.Lock()
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex.Unlock()

	// Test empty list
	sessions := cm.ListPortForwards()
	assert.Empty(t, sessions)

	// Add some sessions
	pfMutex.Lock()
	portForwardSessions["pf-test-1"] = &PortForwardSession{
		ID:         "pf-test-1",
		Namespace:  "default",
		Target:     "nginx",
		TargetType: "pod",
		LocalPort:  8080,
		RemotePort: 80,
		PodName:    "nginx",
		stopChan:   make(chan struct{}),
	}
	portForwardSessions["pf-test-2"] = &PortForwardSession{
		ID:         "pf-test-2",
		Namespace:  "web",
		Target:     "my-service",
		TargetType: "service",
		LocalPort:  3000,
		RemotePort: 80,
		PodName:    "my-service-pod",
		stopChan:   make(chan struct{}),
	}
	pfMutex.Unlock()

	// Test list with sessions
	sessions = cm.ListPortForwards()
	assert.Len(t, sessions, 2)

	// Verify session contents
	foundSession1 := false
	foundSession2 := false
	for _, s := range sessions {
		if s.ID == "pf-test-1" {
			foundSession1 = true
			assert.Equal(t, "default", s.Namespace)
			assert.Equal(t, "nginx", s.Target)
			assert.Equal(t, "pod", s.TargetType)
			assert.Equal(t, 8080, s.LocalPort)
			assert.Equal(t, 80, s.RemotePort)
		}
		if s.ID == "pf-test-2" {
			foundSession2 = true
			assert.Equal(t, "web", s.Namespace)
			assert.Equal(t, "my-service", s.Target)
			assert.Equal(t, "service", s.TargetType)
		}
	}
	assert.True(t, foundSession1, "Session pf-test-1 should be found")
	assert.True(t, foundSession2, "Session pf-test-2 should be found")

	// Cleanup
	pfMutex.Lock()
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex.Unlock()
}

func testStopPortForward(t *testing.T) {
	cm := New()

	// Clear any existing sessions
	pfMutex.Lock()
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex.Unlock()

	// Test stopping non-existent session
	err := cm.StopPortForward("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port forward session \"nonexistent\" not found")

	// Add a session
	stopChan := make(chan struct{})
	pfMutex.Lock()
	portForwardSessions["pf-stop-test"] = &PortForwardSession{
		ID:         "pf-stop-test",
		Namespace:  "default",
		Target:     "test-pod",
		TargetType: "pod",
		LocalPort:  9090,
		RemotePort: 90,
		PodName:    "test-pod",
		stopChan:   stopChan,
	}
	pfMutex.Unlock()

	// Verify session exists
	sessions := cm.ListPortForwards()
	assert.Len(t, sessions, 1)

	// Stop the session
	err = cm.StopPortForward("pf-stop-test")
	assert.NoError(t, err)

	// Verify session is removed
	sessions = cm.ListPortForwards()
	assert.Empty(t, sessions)

	// Verify stop channel was closed
	select {
	case <-stopChan:
		// Channel is closed, as expected
	default:
		t.Error("Stop channel should be closed")
	}

	// Stopping again should fail
	err = cm.StopPortForward("pf-stop-test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func testStartPortForwardErrors(t *testing.T) {
	cm := New()

	// Clear any existing sessions
	pfMutex.Lock()
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex.Unlock()

	t.Run("NoKubeconfigPath", func(t *testing.T) {
		// Manager without kubeconfig path should fail
		_, err := cm.StartPortForward(
			t.Context(),
			"default",
			"pod",
			"nginx",
			8080,
			80,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kubeconfig path not found")
	})

	t.Run("InvalidKubeconfigPath", func(t *testing.T) {
		// Set invalid kubeconfig path
		cm.kubeconfigs["test-context"] = "/nonexistent/path/config"
		cm.currentContext = "test-context"

		_, err := cm.StartPortForward(
			t.Context(),
			"default",
			"pod",
			"nginx",
			8080,
			80,
		)
		assert.Error(t, err)
		// Should fail when building config
		assert.Contains(t, err.Error(), "failed to build config")
	})

	// Cleanup
	pfMutex.Lock()
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex.Unlock()
}
