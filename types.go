package kai

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
