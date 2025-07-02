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

	errMissingName          = "Required parameter 'name' is missing"
	errMissingImage         = "Required parameter 'image' is missing"
	errMissingPod           = "Required parameter 'pod' is missing"
	errMissingPorts         = "Required parameter 'ports' is missing"
	errMissingLabels        = "Parameter 'labels' must be an object"
	errEmptyName            = "Parameter 'name' must be a non-empty string"
	errEmptyImage           = "Parameter 'image' must be a non-empty string"
	errEmptyPod             = "Parameter 'pod' must be a non-empty string"
	errEmptyPorts           = "Parameter 'ports' must be a non-empty array"
	errEmptyLabels          = "Parameter 'labels' must be a non-empty object"
	errConnectionFailed     = "connection failed"
	errQuotaExceeded        = "failed to create deployment: resource quota exceeded"
	errNoUpdateParams       = "At least one field to update must be specified"
	errNoNameOrLabelsParams = "Either 'name' or 'labels' parameter must be provided"
	deleteSuccessMsgFmt     = "Successfully delete pod %q in namespace %q"
	descImagePullPolicy     = "Image pull policy (Always, IfNotPresent, Never)"
	descContainerPortFormat = "Container port to expose (format: 'port' or 'port/protocol')"

	// errMissingName          = "Required parameter 'name' is missing"
	// errInvalidName          = "Parameter 'name' must be a non-empty string"
	// errMissingImage         = "Required parameter 'image' is missing"
	// errInvalidImage         = "Parameter 'image' must be a non-empty string"
	//

)
