package kai

// ContextInfo holds detailed information about the cluster.
type ContextInfo struct {
	Name       string
	Cluster    string
	User       string
	Namespace  string
	ServerURL  string
	ConfigPath string
	IsActive   bool
}

// DeploymentParams holds all possible deployment configuration parameters
type DeploymentParams struct {
	Name             string
	Namespace        string
	Image            string
	Replicas         float64
	Labels           map[string]interface{}
	ContainerPort    string
	Env              map[string]interface{}
	ImagePullPolicy  string
	ImagePullSecrets []interface{}
}

// PodParams holds all possible pod configuration parameters
type PodParams struct {
	Name               string
	Namespace          string
	Image              string
	Command            []interface{}
	Args               []interface{}
	Labels             map[string]interface{}
	ContainerName      string
	ContainerPort      string
	Env                map[string]interface{}
	ImagePullPolicy    string
	ImagePullSecrets   []interface{}
	RestartPolicy      string
	NodeSelector       map[string]interface{}
	ServiceAccountName string
	Volumes            []interface{}
	VolumeMounts       []interface{}
}

// ServiceParams holds all possible service configuration parameters
type ServiceParams struct {
	Name            string
	Namespace       string
	Labels          map[string]interface{}
	Selector        map[string]interface{}
	Type            string
	Ports           []ServicePort
	ClusterIP       string
	ExternalIPs     []string
	ExternalName    string
	SessionAffinity string
}

// ServicePort represents a service port configuration
type ServicePort struct {
	Name       string
	Port       int32
	TargetPort interface{} // Can be int32 or string
	NodePort   int32
	Protocol   string
}

// NamespaceParams holds all possible namespace configuration parameters
type NamespaceParams struct {
	Name        string
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

// ConfigMapParams holds all possible configmap configuration parameters
type ConfigMapParams struct {
	Name        string
	Namespace   string
	Data        map[string]interface{}
	BinaryData  map[string]interface{}
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

// SecretParams holds all possible secret configuration parameters
type SecretParams struct {
	Name        string
	Namespace   string
	Type        string
	Data        map[string]interface{}
	StringData  map[string]interface{}
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

// JobParams holds all possible job configuration parameters
type JobParams struct {
	Name             string
	Namespace        string
	Image            string
	Command          []interface{}
	Args             []interface{}
	RestartPolicy    string
	BackoffLimit     *int32
	Completions      *int32
	Parallelism      *int32
	Labels           map[string]interface{}
	Env              map[string]interface{}
	ImagePullPolicy  string
	ImagePullSecrets []interface{}
}
