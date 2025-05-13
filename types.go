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
