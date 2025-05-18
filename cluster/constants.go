package cluster

// Constants for common test values
const (
	// Namespace constants
	testNamespace    = "test-namespace"
	defaultNamespace = "default"
	otherNamespace   = "other-namespace"
	nonexistentNS    = "nonexistent-namespace"
	emptyNamespace   = "empty-namespace"

	// Pod constants
	podName            = "test-pod"
	pod1Name           = "pod1"
	pod2Name           = "pod2"
	pod3Name           = "pod3"
	fullPodName        = "full-pod"
	noImagePodName     = "no-image-pod"
	forcePodName       = "force-pod"
	pendingPodName     = "pending-pod"
	nonexistentPodName = "nonexistent-pod"

	// Container constants
	containerName        = "test-container"
	customContainer      = "custom-container"
	nonexistentContainer = "nonexistent-container"

	// Image constants
	nginxImage = "nginx:latest"

	// Policy constants
	alwaysPullPolicy = "Always"
	onFailurePolicy  = "OnFailure"
	neverPolicy      = "Never"

	// Service account and secrets
	testServiceAccount = "test-sa"
	registrySecret     = "registry-secret"

	// Test cluster constant
	testClusterName = "test-cluster"

	// Message templates
	podCreatedFmt    = "Pod %q created successfully"
	notFoundErrMsg   = "not found"
	deleteSuccessMsg = "Successfully delete pod"
	noPodsFoundMsg   = "no pods found"

	// Deployment constants
	deploymentName1 = "deployment1"
	deploymentName2 = "deployment2"
	deploymentName3 = "deployment3"
)
