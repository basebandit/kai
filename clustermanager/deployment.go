package clustermanager

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

func (cm *Cluster) CreateDeployment(ctx context.Context, deploymentParams kai.DeploymentParams) (string, error) {

	var result string

	labels := map[string]interface{}{
		"app": deploymentParams.Name,
	}

	if deploymentParams.Labels != nil {
		for k, v := range deploymentParams.Labels {
			labels[k] = v
		}
	}
	// Container definition
	container := map[string]interface{}{
		"name":  deploymentParams.Name,
		"image": deploymentParams.Image,
	}

	parts := strings.Split(deploymentParams.ContainerPort, "/")
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

	if len(deploymentParams.Env) > 0 {
		envVars := make([]interface{}, 0, len(deploymentParams.Env))
		for k, v := range deploymentParams.Env {
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

	if deploymentParams.ImagePullPolicy != "" {
		validPolicies := map[string]bool{"Always": true, "IfNotPresent": true, "Never": true}
		if _, ok := validPolicies[deploymentParams.ImagePullPolicy]; ok {
			container["imagePullPolicy"] = deploymentParams.ImagePullPolicy
		}
	}

	podSpec := map[string]interface{}{
		"containers": []interface{}{container},
	}

	if len(deploymentParams.ImagePullSecrets) > 0 {
		pullSecrets := make([]interface{}, 0, len(deploymentParams.ImagePullSecrets))
		for _, v := range deploymentParams.ImagePullSecrets {
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

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      deploymentParams.Name,
				"namespace": deploymentParams.Namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"replicas": deploymentParams.Replicas,
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
		return result, fmt.Errorf("failed to get a dynamic client: %v", err)
	}

	_, err = client.Resource(gvr).Namespace(deploymentParams.Namespace).Create(timeoutCtx, deployment, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create deployment: %v", err)
	}

	result = fmt.Sprintf("Deployment %q created successfully in namespace %q with %f replica(s)", deploymentParams.Name, deploymentParams.Namespace, deploymentParams.Replicas)

	return result, nil
}

func (cm *Cluster) ListDeployments(ctx context.Context, allNamespaces bool, labelSelector, namespace string) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error: %v", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if allNamespaces {
		deployments, err := client.AppsV1().Deployments("").List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list deployments: %v", err)
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
			return result, fmt.Errorf("failed to list deployments: %v", err)
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
