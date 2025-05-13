package tools

const (
	testPodName           = "test-pod"
	nonexistentPodName    = "non-existent-pod"
	defaultNamespace      = "default"
	testNamespace         = "test-namespace"
	nginxPodName          = "nginx-pod"
	nginxImage            = "nginx:latest"
	myAppImage            = "myapp:v1.2.3"
	deploymentCreatedFmt  = "Deployment %q created successfully in namespace %q with %d replica(s)"
	defaultContainerPort  = "8080/TCP"
	alwaysImagePullPolicy = "Always"
	registrySecretName    = "registry-secret"

	missingNameError      = "Required parameter 'name' is missing"
	missingImageError     = "Required parameter 'image' is missing"
	emptyNameError        = "Parameter 'name' must be a non-empty string"
	emptyImageError       = "Parameter 'image' must be a non-empty string"
	connectionFailedError = "connection failed"
	quotaExceededError    = "failed to create deployment: resource quota exceeded"

	deleteSuccessMsgFmt = "Successfully delete pod %q in namespace %q"
)
