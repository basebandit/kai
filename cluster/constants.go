package cluster

// Constants for common test values
const (
	testNamespace    = "test-namespace"
	defaultNamespace = "default"
	otherNamespace   = "other-namespace"
	nonexistentNS    = "nonexistent-namespace"

	podName            = "test-pod"
	pod1Name           = "pod1"
	pod2Name           = "pod2"
	pod3Name           = "pod3"
	fullPodName        = "full-pod"
	noImagePodName     = "no-image-pod"
	forcePodName       = "force-pod"
	pendingPodName     = "pending-pod"
	nonexistentPodName = "nonexistent-pod"

	containerName        = "test-container"
	customContainer      = "custom-container"
	nonexistentContainer = "nonexistent-container"

	nginxImage = "nginx:latest"

	alwaysPullPolicy = "Always"
	onFailurePolicy  = "OnFailure"
	neverPolicy      = "Never"

	testServiceAccount = "test-sa"
	registrySecret     = "registry-secret"

	// Message templates
	podCreatedFmt    = "Pod %q created successfully"
	notFoundErrMsg   = "not found"
	deleteSuccessMsg = "Successfully delete pod"
	noPodsFoundMsg   = "no pods found"
)
