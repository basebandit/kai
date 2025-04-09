package kai

// DeploymentParams for creation of dynamic deployments
type DeploymentParams struct {
	Name,
	Image,
	Namespace,
	ContainerPort,
	ImagePullPolicy string
	ImagePullSecrets []interface{}
	Labels           map[string]interface{}
	Env              map[string]interface{}
	Replicas         float64
}
