package cluster

// This file contains test-only constants shared across test files.
// These constants are not used in production code.

const (
	// Namespace constants
	testNamespace    = "test-namespace"
	defaultNamespace = "default"
	otherNamespace   = "other-namespace"
	nonexistentNS    = "nonexistent-namespace"
	emptyNamespace   = "empty-namespace"

	// Pod constants
	nonexistentPodName = "nonexistent-pod"

	// Container constants
	nonexistentContainer = "nonexistent-container"

	// Image constants
	nginxImage = "nginx:latest"

	// Test cluster constant
	testCluster  = "test-cluster"
	testCluster1 = "cluster1"
	testCluster2 = "cluster2"

	// Test cluster context
	testContext     = "test-context"
	testContext1    = "context1"
	testContext2    = "context2"
	newContext      = "new-context"
	oldContext      = "old-context"
	activeContext   = "active-context"
	renamedContext  = "rename-context"
	existingContext = "existing-context"

	// Test cluster user
	testUser  = "test-user"
	testUser1 = "user1"
	testUser2 = "user2"

	// Deployment constants
	deploymentName1 = "deployment1"
	deploymentName2 = "deployment2"
	deploymentName3 = "deployment3"

	// Namespace test constants
	testNamespace1 = "test-ns-1"
	testNamespace2 = "test-ns-2"
	testNamespace3 = "prod-ns"

	// ConfigMap constants
	configMapName        = "test-configmap"
	configMapName1       = "configmap1"
	configMapName2       = "configmap2"
	configMapName3       = "configmap3"
	nonexistentConfigMap = "nonexistent-configmap"

	// Secret constants
	secretName          = "test-secret"
	secretName1         = "secret1"
	secretName2         = "secret2"
	secretName3         = "secret3"
	nonexistentSecret   = "nonexistent-secret"
	secretTypeOpaque    = "Opaque"
	secretTypeTLS       = "kubernetes.io/tls"              // #nosec G101
	secretTypeDockerCfg = "kubernetes.io/dockerconfigjson" // #nosec G101
)
