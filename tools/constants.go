package tools

const (
	// Common test resource names
	testPodName           = "test-pod"
	nonexistentPodName    = "non-existent-pod"
	testSecretName        = "test-secret"
	testConfigMapName     = "test-configmap"

	// Namespace constants
	defaultNamespace      = "default"
	testNamespace         = "test-namespace"

	// Image constants
	nginxPodName          = "nginx-pod"
	nginxImage            = "nginx:latest"
	myAppImage            = "myapp:v1.2.3"

	// Secret type constants
	testSecretType        = "Opaque"
	tlsSecretType         = "kubernetes.io/tls"
	dockerSecretType      = "kubernetes.io/dockerconfigjson"
	invalidSecretType     = "invalid/secret-type"

	// Format strings
	deploymentCreatedFmt  = "Deployment %q created successfully in namespace %q with %d replica(s)"
	deleteSuccessMsgFmt   = "Successfully delete pod %q in namespace %q"

	// Configuration constants
	defaultContainerPort  = "8080/TCP"
	alwaysImagePullPolicy = "Always"
	registrySecretName    = "registry-secret"

	// Error messages
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

	// Descriptions
	descImagePullPolicy     = "Image pull policy (Always, IfNotPresent, Never)"
	descContainerPortFormat = "Container port to expose (format: 'port' or 'port/protocol')"
)
