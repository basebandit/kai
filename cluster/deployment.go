package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Deployment represents a Kubernetes deployment configuration
type Deployment struct {
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

// Create creates a new deployment in the cluster
func (d *Deployment) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	// Add default app label if no labels provided
	labels := map[string]interface{}{
		"app": d.Name,
	}

	if d.Labels != nil {
		for k, v := range d.Labels {
			labels[k] = v
		}
	}

	// Container definition
	container := map[string]interface{}{
		"name":  d.Name,
		"image": d.Image,
	}

	// Process container port if specified
	if d.ContainerPort != "" {
		parts := strings.Split(d.ContainerPort, "/")
		var portVal int64
		if _, err := fmt.Sscanf(parts[0], "%d", &portVal); err == nil {
			portDefinition := map[string]interface{}{
				"containerPort": portVal,
			}

			// Add protocol if specified
			if len(parts) > 1 && (parts[1] == "TCP" || parts[1] == "UDP" || parts[1] == "SCTP") {
				portDefinition["protocol"] = parts[1]
			}

			container["ports"] = []interface{}{portDefinition}
		}
	}

	// Add environment variables if specified
	if len(d.Env) > 0 {
		envVars := make([]interface{}, 0, len(d.Env))
		for k, v := range d.Env {
			if strVal, ok := v.(string); ok {
				envVars = append(envVars, map[string]interface{}{
					"name":  k,
					"value": strVal,
				})
			}
		}
		if len(envVars) > 0 {
			container["env"] = envVars
		}
	}

	// Set image pull policy if specified
	if d.ImagePullPolicy != "" {
		validPolicies := map[string]bool{"Always": true, "IfNotPresent": true, "Never": true}
		if _, ok := validPolicies[d.ImagePullPolicy]; ok {
			container["imagePullPolicy"] = d.ImagePullPolicy
		}
	}

	// Create pod spec
	podSpec := map[string]interface{}{
		"containers": []interface{}{container},
	}

	// Add image pull secrets if specified
	if len(d.ImagePullSecrets) > 0 {
		pullSecrets := make([]interface{}, 0, len(d.ImagePullSecrets))
		for _, v := range d.ImagePullSecrets {
			if strVal, ok := v.(string); ok && strVal != "" {
				pullSecrets = append(pullSecrets, map[string]interface{}{
					"name": strVal,
				})
			}
		}
		if len(pullSecrets) > 0 {
			podSpec["imagePullSecrets"] = pullSecrets
		}
	}

	// Create the deployment resource
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      d.Name,
				"namespace": d.Namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"replicas": d.Replicas,
				"selector": map[string]interface{}{
					"matchLabels": labels,
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": labels,
					},
					"spec": podSpec,
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return result, fmt.Errorf("failed to get a dynamic client: %w", err)
	}

	_, err = client.Resource(gvr).Namespace(d.Namespace).Create(timeoutCtx, deployment, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q created successfully in namespace %q with %g replica(s)", d.Name, d.Namespace, d.Replicas)

	return result, nil
}

// List lists deployments in the specified namespace or across all namespaces
func (d *Deployment) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		//TODO: add fieldSelector option as well
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// If namespace is empty but allNamespaces is false, use the current namespace
	namespace := d.Namespace
	if namespace == "" && !allNamespaces {
		namespace = cm.GetCurrentNamespace()
	}

	if allNamespaces {
		deployments, err := client.AppsV1().Deployments("").List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list deployments: %w", err)
		}

		if len(deployments.Items) == 0 {
			result = "No deployments found across all namespaces"
			return result, nil
		}
		result = "Deployments across all namespaces:\n"
		result += formatDeploymentList(deployments)
	} else {
		deployments, err := client.AppsV1().Deployments(namespace).List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list deployments: %w", err)
		}

		if len(deployments.Items) == 0 {
			result = fmt.Sprintf("No deployments found in namespace %q.", namespace)
			return result, nil
		}

		result = fmt.Sprintf("Deployments in namespace %q:\n", namespace)
		result += formatDeploymentList(deployments)
	}

	return result, nil
}
