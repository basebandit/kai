package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
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

	// Add default app label for when no labels provided
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

// Get retrieves information about a specific deployment
func (d *Deployment) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// If namespace is empty, use current namespace
	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	// Get the deployment
	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	result = formatDeployment(deployment)
	return result, nil
}

// Update updates an existing deployment in the cluster
func (d *Deployment) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// If namespace is empty, use current namespace
	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	// Get the current deployment
	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update replicas if specified
	if d.Replicas > 0 {
		replicas := int32(d.Replicas)
		deployment.Spec.Replicas = &replicas
	}

	// Update image if specified
	if d.Image != "" {
		// Find the container with the same name as the deployment or the first container
		containerIndex := -1
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == d.Name || i == 0 {
				containerIndex = i
				break
			}
		}

		if containerIndex >= 0 {
			deployment.Spec.Template.Spec.Containers[containerIndex].Image = d.Image
		} else {
			return result, fmt.Errorf("no suitable container found to update image")
		}
	}

	// Update labels if specified
	if d.Labels != nil {
		// Convert to string map
		labels := make(map[string]string)
		for k, v := range d.Labels {
			if strVal, ok := v.(string); ok {
				labels[k] = strVal
			} else if strVal, ok := fmt.Sprintf("%v", v), true; ok {
				labels[k] = strVal
			}
		}

		// Update deployment labels
		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}
		for k, v := range labels {
			deployment.Labels[k] = v
		}

		// Update template labels
		if deployment.Spec.Template.Labels == nil {
			deployment.Spec.Template.Labels = make(map[string]string)
		}
		for k, v := range labels {
			deployment.Spec.Template.Labels[k] = v
		}

		// Update selector labels (carefully, as this is immutable for most fields)
		// Only add new labels, don't modify existing ones
		if deployment.Spec.Selector.MatchLabels == nil {
			deployment.Spec.Selector.MatchLabels = make(map[string]string)
		}
		for k, v := range labels {
			if _, exists := deployment.Spec.Selector.MatchLabels[k]; !exists {
				deployment.Spec.Selector.MatchLabels[k] = v
			}
		}
	}

	// Update environment variables if specified
	if len(d.Env) > 0 {
		// Find the container with the same name as the deployment or the first container
		containerIndex := -1
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == d.Name || i == 0 {
				containerIndex = i
				break
			}
		}

		if containerIndex >= 0 {
			// Convert env map to Kubernetes env vars
			newEnvVars := make([]corev1.EnvVar, 0, len(d.Env))
			for k, v := range d.Env {
				if strVal, ok := v.(string); ok {
					newEnvVars = append(newEnvVars, corev1.EnvVar{
						Name:  k,
						Value: strVal,
					})
				}
			}

			// Create a map of existing env vars for easy lookup
			existingEnvVars := make(map[string]int)
			for i, env := range deployment.Spec.Template.Spec.Containers[containerIndex].Env {
				existingEnvVars[env.Name] = i
			}

			// Update or add environment variables
			for _, env := range newEnvVars {
				if i, exists := existingEnvVars[env.Name]; exists {
					// Update existing env var
					deployment.Spec.Template.Spec.Containers[containerIndex].Env[i] = env
				} else {
					// Add new env var
					deployment.Spec.Template.Spec.Containers[containerIndex].Env = append(
						deployment.Spec.Template.Spec.Containers[containerIndex].Env, env)
				}
			}
		} else {
			return result, fmt.Errorf("no suitable container found to update environment variables")
		}
	}

	// Update container port if specified
	if d.ContainerPort != "" {
		// Find the container with the same name as the deployment or the first container
		containerIndex := -1
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == d.Name || i == 0 {
				containerIndex = i
				break
			}
		}

		if containerIndex >= 0 {
			parts := strings.Split(d.ContainerPort, "/")
			var portVal int32
			if _, err := fmt.Sscanf(parts[0], "%d", &portVal); err == nil {
				portDefinition := corev1.ContainerPort{
					ContainerPort: portVal,
				}

				// Add protocol if specified
				if len(parts) > 1 {
					protocol := strings.ToUpper(parts[1])
					if protocol == "TCP" || protocol == "UDP" || protocol == "SCTP" {
						portDefinition.Protocol = corev1.Protocol(protocol)
					}
				} else {
					// Default to TCP
					portDefinition.Protocol = corev1.ProtocolTCP
				}

				// Check if we need to update an existing port or add a new one
				portUpdated := false
				for i, port := range deployment.Spec.Template.Spec.Containers[containerIndex].Ports {
					if port.ContainerPort == portVal {
						deployment.Spec.Template.Spec.Containers[containerIndex].Ports[i] = portDefinition
						portUpdated = true
						break
					}
				}

				if !portUpdated {
					deployment.Spec.Template.Spec.Containers[containerIndex].Ports = append(
						deployment.Spec.Template.Spec.Containers[containerIndex].Ports, portDefinition)
				}
			}
		} else {
			return result, fmt.Errorf("no suitable container found to update container port")
		}
	}

	// Update image pull policy if specified
	if d.ImagePullPolicy != "" {
		// Find the container with the same name as the deployment or the first container
		containerIndex := -1
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == d.Name || i == 0 {
				containerIndex = i
				break
			}
		}

		if containerIndex >= 0 {
			validPolicies := map[string]corev1.PullPolicy{
				"Always":       corev1.PullAlways,
				"IfNotPresent": corev1.PullIfNotPresent,
				"Never":        corev1.PullNever,
			}
			if policy, ok := validPolicies[d.ImagePullPolicy]; ok {
				deployment.Spec.Template.Spec.Containers[containerIndex].ImagePullPolicy = policy
			}
		} else {
			return result, fmt.Errorf("no suitable container found to update image pull policy")
		}
	}

	// Update image pull secrets if specified
	if len(d.ImagePullSecrets) > 0 {
		pullSecrets := make([]corev1.LocalObjectReference, 0, len(d.ImagePullSecrets))
		for _, v := range d.ImagePullSecrets {
			if strVal, ok := v.(string); ok && strVal != "" {
				pullSecrets = append(pullSecrets, corev1.LocalObjectReference{
					Name: strVal,
				})
			}
		}
		if len(pullSecrets) > 0 {
			deployment.Spec.Template.Spec.ImagePullSecrets = pullSecrets
		}
	}

	// Update the deployment
	updatedDeployment, err := client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q updated successfully in namespace %q", updatedDeployment.Name, updatedDeployment.Namespace)
	if updatedDeployment.Spec.Replicas != nil {
		result += fmt.Sprintf(" with %d replica(s)", *updatedDeployment.Spec.Replicas)
	}

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

// Describe provides detailed information about a deployment
func (d *Deployment) Describe(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get deployment: %w", err)
	}

	result := formatDeploymentDetailed(deployment)
	return result, nil
}

// Delete removes a deployment from the cluster
func (d *Deployment) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	err = client.AppsV1().Deployments(namespace).Delete(timeoutCtx, d.Name, metav1.DeleteOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to delete deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q deleted successfully from namespace %q", d.Name, namespace)
	return result, nil
}

// Scale adjusts the number of replicas for a deployment
func (d *Deployment) Scale(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	replicas := int32(d.Replicas)
	deployment.Spec.Replicas = &replicas

	_, err = client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to scale deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q scaled to %d replica(s) in namespace %q", d.Name, replicas, namespace)
	return result, nil
}

// RolloutStatus checks the status of a deployment rollout
func (d *Deployment) RolloutStatus(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q rollout status:\n", d.Name)
	result += fmt.Sprintf("  Replicas: %d desired | %d updated | %d total | %d available | %d unavailable\n",
		*deployment.Spec.Replicas,
		deployment.Status.UpdatedReplicas,
		deployment.Status.Replicas,
		deployment.Status.AvailableReplicas,
		deployment.Status.UnavailableReplicas)

	for _, condition := range deployment.Status.Conditions {
		result += fmt.Sprintf("  %s: %s (Reason: %s) - %s\n",
			condition.Type,
			condition.Status,
			condition.Reason,
			condition.Message)
	}

	if deployment.Status.Replicas == deployment.Status.UpdatedReplicas &&
		deployment.Status.UpdatedReplicas == deployment.Status.AvailableReplicas &&
		deployment.Status.ObservedGeneration >= deployment.Generation {
		result += "\nRollout complete!"
	} else {
		result += "\nRollout in progress..."
	}

	return result, nil
}

// RolloutHistory shows the revision history of a deployment
func (d *Deployment) RolloutHistory(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	replicaSets, err := client.AppsV1().ReplicaSets(namespace).List(timeoutCtx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		return result, fmt.Errorf("failed to list replica sets: %w", err)
	}

	result = fmt.Sprintf("Rollout history for deployment %q:\n\n", d.Name)
	result += "REVISION  CHANGE-CAUSE\n"

	for _, rs := range replicaSets.Items {
		if revision, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
			changeCause := rs.Annotations["kubernetes.io/change-cause"]
			if changeCause == "" {
				changeCause = "<none>"
			}
			result += fmt.Sprintf("%s         %s\n", revision, changeCause)
		}
	}

	return result, nil
}

// RolloutUndo rolls back a deployment to a previous revision
func (d *Deployment) RolloutUndo(ctx context.Context, cm kai.ClusterManager, revision int64) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	if revision > 0 {
		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}
		deployment.Annotations["deployment.kubernetes.io/revision"] = fmt.Sprintf("%d", revision)
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to rollback deployment: %w", err)
	}

	if revision > 0 {
		result = fmt.Sprintf("Deployment %q rolled back to revision %d in namespace %q", d.Name, revision, namespace)
	} else {
		result = fmt.Sprintf("Deployment %q rolled back to previous revision in namespace %q", d.Name, namespace)
	}

	return result, nil
}

// RolloutRestart restarts a deployment
func (d *Deployment) RolloutRestart(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to restart deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q restarted in namespace %q", d.Name, namespace)
	return result, nil
}

// RolloutPause pauses a deployment rollout
func (d *Deployment) RolloutPause(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	deployment.Spec.Paused = true

	_, err = client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to pause deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q paused in namespace %q", d.Name, namespace)
	return result, nil
}

// RolloutResume resumes a paused deployment rollout
func (d *Deployment) RolloutResume(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	namespace := d.Namespace
	if namespace == "" {
		namespace = cm.GetCurrentNamespace()
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(timeoutCtx, d.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get deployment: %w", err)
	}

	deployment.Spec.Paused = false

	_, err = client.AppsV1().Deployments(namespace).Update(timeoutCtx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to resume deployment: %w", err)
	}

	result = fmt.Sprintf("Deployment %q resumed in namespace %q", d.Name, namespace)
	return result, nil
}
