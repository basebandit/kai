package cluster

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func formatPod(pod *corev1.Pod) string {
	// Format the pod details
	result := fmt.Sprintf("Pod: %s\n", pod.Name)
	result += fmt.Sprintf("Namespace: %s\n", pod.Namespace)
	result += fmt.Sprintf("Status: %s\n", pod.Status.Phase)
	result += fmt.Sprintf("Node: %s\n", pod.Spec.NodeName)
	result += fmt.Sprintf("IP: %s\n", pod.Status.PodIP)
	result += fmt.Sprintf("Created: %s\n", pod.CreationTimestamp.Time.Format(time.RFC3339))

	result += "\nContainers:\n"
	for i, container := range pod.Spec.Containers {
		result += fmt.Sprintf("%d. %s (Image: %s)\n", i+1, container.Name, container.Image)

		// Add container status if available
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container.Name {
				ready := "Not Ready"
				if status.Ready {
					ready = "Ready"
				}
				result += fmt.Sprintf("   Status: %s, Restarts: %d\n", ready, status.RestartCount)

				// Add state details
				switch {
				case status.State.Running != nil:
					result += fmt.Sprintf("   Started At: %s\n", status.State.Running.StartedAt.Format(time.RFC3339))
				case status.State.Waiting != nil:
					result += fmt.Sprintf("   Waiting: %s - %s\n", status.State.Waiting.Reason, status.State.Waiting.Message)
				case status.State.Terminated != nil:
					result += fmt.Sprintf("   Terminated: %s - %s (Exit Code: %d)\n",
						status.State.Terminated.Reason,
						status.State.Terminated.Message,
						status.State.Terminated.ExitCode)
				}
				break
			}
		}
	}

	// Add labels
	if len(pod.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range pod.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

func formatPodList(pods *corev1.PodList, allNamespaces bool, limit int64, resultText string) string {
	// Format the pods list
	for _, pod := range pods.Items {
		status := pod.Status.Phase
		ready := "0"
		total := "0"

		if len(pod.Status.ContainerStatuses) > 0 {
			readyCount := 0
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Ready {
					readyCount++
				}
			}
			ready = fmt.Sprintf("%d", readyCount)
			total = fmt.Sprintf("%d", len(pod.Status.ContainerStatuses))
		}

		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)

		// Format the pod info
		podInfo := ""
		if allNamespaces {
			podInfo = fmt.Sprintf("• %s/%s: %s (%s/%s) - IP: %s - Age: %s",
				pod.Namespace, pod.Name, status, ready, total, pod.Status.PodIP, formatDuration(age))
		} else {
			podInfo = fmt.Sprintf("• %s: %s (%s/%s) - IP: %s - Age: %s",
				pod.Name, status, ready, total, pod.Status.PodIP, formatDuration(age))
		}

		// Add node info if available
		if pod.Spec.NodeName != "" {
			podInfo += fmt.Sprintf(" - Node: %s", pod.Spec.NodeName)
		}

		// Add restart count
		restarts := 0
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += int(cs.RestartCount)
		}
		if restarts > 0 {
			podInfo += fmt.Sprintf(" - Restarts: %d", restarts)
		}

		resultText += podInfo + "\n"
	}

	// Add total count
	resultText += fmt.Sprintf("\nTotal: %d pod(s)", len(pods.Items))
	if limit > 0 && int64(len(pods.Items)) == limit {
		resultText += fmt.Sprintf(" (limited to %d results)", limit)
	}

	return resultText
}

func formatDeploymentList(deployments *appsv1.DeploymentList) string {
	var resultText string
	for _, deployment := range deployments.Items {

		age := time.Since(deployment.CreationTimestamp.Time).Round(time.Second)

		resultText += fmt.Sprintf("• %s/%s: %d/%d replicas ready - Age: %s\n",
			deployment.Namespace,
			deployment.Name,
			deployment.Status.ReadyReplicas,
			deployment.Status.Replicas,
			formatDuration(age),
		)
	}

	return resultText
}

// formatDeployment formats a deployment for display
func formatDeployment(deployment *appsv1.Deployment) string {
	result := fmt.Sprintf("Deployment: %s\n", deployment.Name)
	result += fmt.Sprintf("Namespace: %s\n", deployment.Namespace)

	// Basic information
	var replicas int32 = 0
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	result += fmt.Sprintf("Replicas: %d/%d (available/total)\n", deployment.Status.AvailableReplicas, replicas)
	result += fmt.Sprintf("Created: %s\n", deployment.CreationTimestamp.Format(time.RFC3339))

	result += fmt.Sprintf("Ready: %d\n", deployment.Status.ReadyReplicas)
	// Status conditions
	if len(deployment.Status.Conditions) > 0 {
		result += "\nConditions:\n"
		for _, condition := range deployment.Status.Conditions {
			result += fmt.Sprintf("- Type: %s, Status: %s, Last Update: %s\n",
				condition.Type,
				condition.Status,
				condition.LastUpdateTime.Format(time.RFC3339))
			if condition.Message != "" {
				result += fmt.Sprintf("  Message: %s\n", condition.Message)
			}
		}
	}

	// Selectors
	if len(deployment.Spec.Selector.MatchLabels) > 0 {
		result += "\nSelector:\n"
		for k, v := range deployment.Spec.Selector.MatchLabels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Labels
	if len(deployment.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range deployment.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Strategy
	result += fmt.Sprintf("\nStrategy: %s\n", deployment.Spec.Strategy.Type)
	if deployment.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType && deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			result += fmt.Sprintf("Max Unavailable: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			result += fmt.Sprintf("Max Surge: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxSurge.String())
		}
	}

	// Containers
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		result += "\nContainers:\n"
		for i, container := range deployment.Spec.Template.Spec.Containers {
			result += fmt.Sprintf("%d. %s (Image: %s)\n", i+1, container.Name, container.Image)

			// Container ports
			if len(container.Ports) > 0 {
				result += "   Ports:\n"
				for _, port := range container.Ports {
					result += fmt.Sprintf("   - %d/%s\n", port.ContainerPort, port.Protocol)
				}
			}

			// Environment variables
			if len(container.Env) > 0 {
				result += "   Environment:\n"
				for _, env := range container.Env {
					if env.ValueFrom != nil {
						result += fmt.Sprintf("   - %s: <set from source>\n", env.Name)
					} else {
						result += fmt.Sprintf("   - %s: %s\n", env.Name, env.Value)
					}
				}
			}

			// Resources
			if container.Resources.Limits != nil || container.Resources.Requests != nil {
				result += "   Resources:\n"
				if container.Resources.Limits != nil {
					result += "     Limits:\n"
					for resource, quantity := range container.Resources.Limits {
						result += fmt.Sprintf("     - %s: %s\n", resource, quantity.String())
					}
				}
				if container.Resources.Requests != nil {
					result += "     Requests:\n"
					for resource, quantity := range container.Resources.Requests {
						result += fmt.Sprintf("     - %s: %s\n", resource, quantity.String())
					}
				}
			}

			// Image pull policy
			result += fmt.Sprintf("   Image Pull Policy: %s\n", container.ImagePullPolicy)
		}
	}

	// Volume mounts
	if len(deployment.Spec.Template.Spec.Volumes) > 0 {
		result += "\nVolumes:\n"
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			result += fmt.Sprintf("- %s\n", volume.Name)

			// Add volume type information
			switch {
			case volume.PersistentVolumeClaim != nil:
				result += fmt.Sprintf("  Type: PersistentVolumeClaim (Claim: %s)\n", volume.PersistentVolumeClaim.ClaimName)
			case volume.ConfigMap != nil:
				result += fmt.Sprintf("  Type: ConfigMap (Name: %s)\n", volume.ConfigMap.Name)
			case volume.Secret != nil:
				result += fmt.Sprintf("  Type: Secret (Name: %s)\n", volume.Secret.SecretName)
			case volume.EmptyDir != nil:
				result += "  Type: EmptyDir\n"
			default:
				result += "  Type: Other\n"
			}
		}
	}

	// Pod labels
	if len(deployment.Spec.Template.Labels) > 0 {
		result += "\nPod Labels:\n"
		for k, v := range deployment.Spec.Template.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

// formatService formats a service for display
func formatService(svc *corev1.Service) string {
	result := fmt.Sprintf("Service: %s\n", svc.Name)
	result += fmt.Sprintf("Namespace: %s\n", svc.Namespace)
	result += fmt.Sprintf("Type: %s\n", svc.Spec.Type)

	// Get cluster IP
	result += fmt.Sprintf("ClusterIP: %s\n", svc.Spec.ClusterIP)

	// Get external IP
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ips := []string{}
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, ingress.IP)
			} else if ingress.Hostname != "" {
				ips = append(ips, ingress.Hostname)
			}
		}
		if len(ips) > 0 {
			result += fmt.Sprintf("External IP(s): %s\n", strings.Join(ips, ", "))
		}
	} else if len(svc.Spec.ExternalIPs) > 0 {
		result += fmt.Sprintf("External IP(s): %s\n", strings.Join(svc.Spec.ExternalIPs, ", "))
	}

	// Add creation timestamp
	result += fmt.Sprintf("Created: %s\n", svc.CreationTimestamp.Format(time.RFC3339))

	// Get ports
	if len(svc.Spec.Ports) > 0 {
		result += "\nPorts:\n"
		for i, port := range svc.Spec.Ports {
			portInfo := fmt.Sprintf("%d. ", i+1)
			if port.Name != "" {
				portInfo += fmt.Sprintf("%s: ", port.Name)
			}

			portInfo += fmt.Sprintf("%d", port.Port)

			targetPort := port.TargetPort.String()
			portInfo += fmt.Sprintf(" → %s", targetPort)

			if port.NodePort > 0 {
				portInfo += fmt.Sprintf(" (NodePort: %d)", port.NodePort)
			}

			portInfo += fmt.Sprintf(" [%s]", port.Protocol)
			result += portInfo + "\n"
		}
	}

	// Add selector
	if len(svc.Spec.Selector) > 0 {
		result += "\nSelector:\n"
		for k, v := range svc.Spec.Selector {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Add labels
	if len(svc.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range svc.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

// formatServiceList formats a list of services for display
func formatServiceList(services *corev1.ServiceList, includeNamespace bool) string {
	var result strings.Builder

	for _, svc := range services.Items {
		// Get service type
		serviceType := string(svc.Spec.Type)

		// Get cluster IP
		clusterIP := svc.Spec.ClusterIP
		if clusterIP == "" {
			clusterIP = "<none>"
		}

		// Get external IP
		externalIP := "<none>"
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ips := []string{}
			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				if ingress.IP != "" {
					ips = append(ips, ingress.IP)
				} else if ingress.Hostname != "" {
					ips = append(ips, ingress.Hostname)
				}
			}
			if len(ips) > 0 {
				externalIP = strings.Join(ips, ",")
			}
		} else if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = strings.Join(svc.Spec.ExternalIPs, ",")
		}

		// Get port(s)
		ports := []string{}
		for _, port := range svc.Spec.Ports {
			if port.NodePort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
			}
		}
		portsStr := "<none>"
		if len(ports) > 0 {
			portsStr = strings.Join(ports, ",")
		}

		// Get age
		age := time.Since(svc.CreationTimestamp.Time).Round(time.Second)

		// Format the output
		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: Type=%s, ClusterIP=%s, ExternalIP=%s, Ports=%s, Age=%s\n",
				svc.Namespace,
				svc.Name,
				serviceType,
				clusterIP,
				externalIP,
				portsStr,
				formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: Type=%s, ClusterIP=%s, ExternalIP=%s, Ports=%s, Age=%s\n",
				svc.Name,
				serviceType,
				clusterIP,
				externalIP,
				portsStr,
				formatDuration(age)))
		}
	}

	// Add total count
	result.WriteString(fmt.Sprintf("\nTotal: %d service(s)", len(services.Items)))

	return result.String()
}

// formatDeploymentDetailed formats a deployment with detailed information for display
func formatDeploymentDetailed(deployment *appsv1.Deployment) string {
	result := fmt.Sprintf("Deployment: %s\n", deployment.Name)
	result += fmt.Sprintf("Namespace: %s\n", deployment.Namespace)

	// Basic information
	var replicas int32 = 0
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	result += fmt.Sprintf("Replicas: %d/%d (available/total)\n", deployment.Status.AvailableReplicas, replicas)
	result += fmt.Sprintf("Created: %s\n", deployment.CreationTimestamp.Format(time.RFC3339))

	// Status conditions
	if len(deployment.Status.Conditions) > 0 {
		result += "\nConditions:\n"
		for _, condition := range deployment.Status.Conditions {
			result += fmt.Sprintf("- Type: %s, Status: %s, Last Update: %s\n",
				condition.Type,
				condition.Status,
				condition.LastUpdateTime.Format(time.RFC3339))
			if condition.Message != "" {
				result += fmt.Sprintf("  Message: %s\n", condition.Message)
			}
			if condition.Reason != "" {
				result += fmt.Sprintf("  Reason: %s\n", condition.Reason)
			}
		}
	}

	// Selectors
	if len(deployment.Spec.Selector.MatchLabels) > 0 {
		result += "\nSelector:\n"
		for k, v := range deployment.Spec.Selector.MatchLabels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Strategy
	result += fmt.Sprintf("\nStrategy: %s\n", deployment.Spec.Strategy.Type)
	if deployment.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType && deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			result += fmt.Sprintf("Max Unavailable: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			result += fmt.Sprintf("Max Surge: %s\n", deployment.Spec.Strategy.RollingUpdate.MaxSurge.String())
		}
	}

	// Containers
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		result += "\nContainers:\n"
		for i, container := range deployment.Spec.Template.Spec.Containers {
			result += fmt.Sprintf("%d. %s (Image: %s)\n", i+1, container.Name, container.Image)

			// Container ports
			if len(container.Ports) > 0 {
				result += "   Ports:\n"
				for _, port := range container.Ports {
					result += fmt.Sprintf("   - %d/%s\n", port.ContainerPort, port.Protocol)
				}
			}

			// Environment variables
			if len(container.Env) > 0 {
				result += "   Environment:\n"
				for _, env := range container.Env {
					if env.ValueFrom != nil {
						result += fmt.Sprintf("   - %s: <set from source>\n", env.Name)
					} else {
						result += fmt.Sprintf("   - %s: %s\n", env.Name, env.Value)
					}
				}
			}

			// Resources
			if container.Resources.Limits != nil || container.Resources.Requests != nil {
				result += "   Resources:\n"
				if container.Resources.Limits != nil {
					result += "     Limits:\n"
					for resource, quantity := range container.Resources.Limits {
						result += fmt.Sprintf("     - %s: %s\n", resource, quantity.String())
					}
				}
				if container.Resources.Requests != nil {
					result += "     Requests:\n"
					for resource, quantity := range container.Resources.Requests {
						result += fmt.Sprintf("     - %s: %s\n", resource, quantity.String())
					}
				}
			}

			// Image pull policy
			result += fmt.Sprintf("   Image Pull Policy: %s\n", container.ImagePullPolicy)
		}
	}

	// Volume mounts
	if len(deployment.Spec.Template.Spec.Volumes) > 0 {
		result += "\nVolumes:\n"
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			result += fmt.Sprintf("- %s\n", volume.Name)

			// Add volume type information
			switch {
			case volume.PersistentVolumeClaim != nil:
				result += fmt.Sprintf("  Type: PersistentVolumeClaim (Claim: %s)\n", volume.PersistentVolumeClaim.ClaimName)
			case volume.ConfigMap != nil:
				result += fmt.Sprintf("  Type: ConfigMap (Name: %s)\n", volume.ConfigMap.Name)
			case volume.Secret != nil:
				result += fmt.Sprintf("  Type: Secret (Name: %s)\n", volume.Secret.SecretName)
			case volume.EmptyDir != nil:
				result += "  Type: EmptyDir\n"
			default:
				result += "  Type: Other\n"
			}
		}
	}

	// Pod labels
	if len(deployment.Spec.Template.Labels) > 0 {
		result += "\nPod Labels:\n"
		for k, v := range deployment.Spec.Template.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Add status summary
	result += "\nStatus Summary:\n"
	result += fmt.Sprintf("- Ready: %d/%d\n", deployment.Status.ReadyReplicas, replicas)
	result += fmt.Sprintf("- Up-to-date: %d\n", deployment.Status.UpdatedReplicas)
	result += fmt.Sprintf("- Available: %d\n", deployment.Status.AvailableReplicas)

	return result
}

// formatDuration formats a time.Duration in a human-readable way similar to kubectl
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

// convert interface map to string map
func convertToStringMap(input map[string]interface{}) map[string]string {
	if input == nil {
		return nil
	}

	result := make(map[string]string, len(input))
	for k, v := range input {
		if strValue, ok := v.(string); ok {
			result[k] = strValue
		} else if strValue, ok := fmt.Sprintf("%v", v), true; ok {
			result[k] = strValue
		}
	}
	return result
}

func formatNamespace(ns *corev1.Namespace) string {
	result := fmt.Sprintf("Namespace: %s\n", ns.Name)
	result += fmt.Sprintf("Status: %s\n", ns.Status.Phase)
	result += fmt.Sprintf("Created: %s\n", ns.CreationTimestamp.Time.Format(time.RFC3339))

	if len(ns.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range ns.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	if len(ns.Annotations) > 0 {
		result += "\nAnnotations:\n"
		for k, v := range ns.Annotations {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

func formatNamespaceList(namespaces *corev1.NamespaceList, labelSelector string) string {
	var result strings.Builder

	if labelSelector != "" {
		result.WriteString(fmt.Sprintf("Namespaces matching label selector '%s':\n", labelSelector))
	} else {
		result.WriteString("Namespaces:\n")
	}

	for _, ns := range namespaces.Items {
		age := time.Since(ns.CreationTimestamp.Time).Round(time.Second)
		status := string(ns.Status.Phase)
		if status == "" {
			status = "Active"
		}

		result.WriteString(fmt.Sprintf("• %s: Status=%s, Age=%s",
			ns.Name, status, formatDuration(age)))

		if len(ns.Labels) > 0 {
			labelCount := len(ns.Labels)
			result.WriteString(fmt.Sprintf(" - Labels: %d", labelCount))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d namespace(s)", len(namespaces.Items)))

	return result.String()
}

func formatConfigMap(cm *corev1.ConfigMap) string {
	result := fmt.Sprintf("ConfigMap: %s\n", cm.Name)
	result += fmt.Sprintf("Namespace: %s\n", cm.Namespace)
	result += fmt.Sprintf("Created: %s\n", cm.CreationTimestamp.Time.Format(time.RFC3339))

	if len(cm.Data) > 0 {
		result += "\nData:\n"
		for k, v := range cm.Data {
			if len(v) > 100 {
				result += fmt.Sprintf("- %s: %s... (%d bytes)\n", k, v[:100], len(v))
			} else {
				result += fmt.Sprintf("- %s: %s\n", k, v)
			}
		}
	}

	if len(cm.BinaryData) > 0 {
		result += "\nBinary Data:\n"
		for k, v := range cm.BinaryData {
			result += fmt.Sprintf("- %s: (%d bytes)\n", k, len(v))
		}
	}

	if len(cm.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range cm.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	if len(cm.Annotations) > 0 {
		result += "\nAnnotations:\n"
		for k, v := range cm.Annotations {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

func formatConfigMapList(configMaps *corev1.ConfigMapList, includeNamespace bool) string {
	var result strings.Builder

	for _, cm := range configMaps.Items {
		age := time.Since(cm.CreationTimestamp.Time).Round(time.Second)
		dataCount := len(cm.Data) + len(cm.BinaryData)

		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: Data=%d, Age=%s",
				cm.Namespace, cm.Name, dataCount, formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: Data=%d, Age=%s",
				cm.Name, dataCount, formatDuration(age)))
		}

		if len(cm.Labels) > 0 {
			result.WriteString(fmt.Sprintf(" - Labels: %d", len(cm.Labels)))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d ConfigMap(s)", len(configMaps.Items)))

	return result.String()
}

func formatSecret(secret *corev1.Secret) string {
	result := fmt.Sprintf("Secret: %s\n", secret.Name)
	result += fmt.Sprintf("Namespace: %s\n", secret.Namespace)
	result += fmt.Sprintf("Type: %s\n", secret.Type)
	result += fmt.Sprintf("Created: %s\n", secret.CreationTimestamp.Time.Format(time.RFC3339))

	if len(secret.Data) > 0 {
		result += "\nData (keys only - values masked):\n"
		for k, v := range secret.Data {
			result += fmt.Sprintf("- %s: (%d bytes)\n", k, len(v))
		}
	}

	if len(secret.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range secret.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	if len(secret.Annotations) > 0 {
		result += "\nAnnotations:\n"
		for k, v := range secret.Annotations {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

func formatSecretList(secrets *corev1.SecretList, includeNamespace bool) string {
	var result strings.Builder

	for _, secret := range secrets.Items {
		age := time.Since(secret.CreationTimestamp.Time).Round(time.Second)
		dataCount := len(secret.Data)

		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: Type=%s, Data=%d, Age=%s",
				secret.Namespace, secret.Name, secret.Type, dataCount, formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: Type=%s, Data=%d, Age=%s",
				secret.Name, secret.Type, dataCount, formatDuration(age)))
		}

		if len(secret.Labels) > 0 {
			result.WriteString(fmt.Sprintf(" - Labels: %d", len(secret.Labels)))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d Secret(s)", len(secrets.Items)))

	return result.String()
}

func formatJob(job *batchv1.Job) string {
	result := fmt.Sprintf("Job: %s\n", job.Name)
	result += fmt.Sprintf("Namespace: %s\n", job.Namespace)

	if job.Spec.Completions != nil {
		result += fmt.Sprintf("Succeeded/Completions: %d/%d\n", job.Status.Succeeded, *job.Spec.Completions)
	}
	if job.Spec.Parallelism != nil {
		result += fmt.Sprintf("Parallelism: %d\n", *job.Spec.Parallelism)
	}

	if job.Status.Active > 0 {
		result += fmt.Sprintf("Active: %d\n", job.Status.Active)
	}
	if job.Status.Failed > 0 {
		result += fmt.Sprintf("Failed: %d\n", job.Status.Failed)
	}

	result += fmt.Sprintf("Created: %s\n", job.CreationTimestamp.Time.Format(time.RFC3339))
	result += fmt.Sprintf("Succeeded: %d\n", job.Status.Succeeded)

	if job.Status.StartTime != nil {
		result += fmt.Sprintf("Start Time: %s\n", job.Status.StartTime.Time.Format(time.RFC3339))
	}
	if job.Status.CompletionTime != nil {
		result += fmt.Sprintf("Completion Time: %s\n", job.Status.CompletionTime.Time.Format(time.RFC3339))
		duration := job.Status.CompletionTime.Time.Sub(job.Status.StartTime.Time)
		result += fmt.Sprintf("Duration: %s\n", formatDuration(duration))
	}

	if len(job.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range job.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	result += fmt.Sprintf("\nImage: %s\n", job.Spec.Template.Spec.Containers[0].Image)

	return result
}

func formatJobList(jobs *batchv1.JobList, includeNamespace bool) string {
	var result strings.Builder

	if includeNamespace {
		result.WriteString("Jobs across all namespaces:\n")
	} else {
		result.WriteString(fmt.Sprintf("Jobs in namespace %q:\n", jobs.Items[0].Namespace))
	}

	for _, job := range jobs.Items {
		age := time.Since(job.CreationTimestamp.Time).Round(time.Second)

		completions := "0"
		if job.Spec.Completions != nil {
			completions = fmt.Sprintf("%d", *job.Spec.Completions)
		}

		status := fmt.Sprintf("%d/%s", job.Status.Succeeded, completions)
		if job.Status.Active > 0 {
			status += fmt.Sprintf(" (Active: %d)", job.Status.Active)
		}
		if job.Status.Failed > 0 {
			status += fmt.Sprintf(" (Failed: %d)", job.Status.Failed)
		}

		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: %s - Age: %s",
				job.Namespace, job.Name, status, formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: %s - Age: %s",
				job.Name, status, formatDuration(age)))
		}

		if len(job.Labels) > 0 {
			result.WriteString(fmt.Sprintf(" - Labels: %d", len(job.Labels)))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d Job(s)", len(jobs.Items)))

	return result.String()
}

func convertToStringSlice(input []interface{}) []string {
	if input == nil {
		return nil
	}
	result := make([]string, 0, len(input))
	for _, item := range input {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func convertToEnvVars(input map[string]interface{}) []corev1.EnvVar {
	if input == nil {
		return nil
	}
	envVars := make([]corev1.EnvVar, 0, len(input))
	for key, val := range input {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: fmt.Sprintf("%v", val),
		})
	}
	return envVars
}

func convertToLocalObjectReferences(input []interface{}) []corev1.LocalObjectReference {
	if input == nil {
		return nil
	}
	refs := make([]corev1.LocalObjectReference, 0, len(input))
	for _, item := range input {
		if str, ok := item.(string); ok {
			refs = append(refs, corev1.LocalObjectReference{Name: str})
		}
	}
	return refs
}

func formatCronJob(cronJob *batchv1.CronJob) string {
	result := fmt.Sprintf("CronJob: %s\n", cronJob.Name)
	result += fmt.Sprintf("Namespace: %s\n", cronJob.Namespace)
	result += fmt.Sprintf("Schedule: %s\n", cronJob.Spec.Schedule)

	suspended := "No"
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		suspended = "Yes"
	}
	result += fmt.Sprintf("Suspend: %s\n", suspended)

	result += fmt.Sprintf("Concurrency Policy: %s\n", cronJob.Spec.ConcurrencyPolicy)

	if cronJob.Status.LastScheduleTime != nil {
		result += fmt.Sprintf("Last Schedule: %s\n", cronJob.Status.LastScheduleTime.Time.Format(time.RFC3339))
	}

	if cronJob.Status.LastSuccessfulTime != nil {
		result += fmt.Sprintf("Last Successful: %s\n", cronJob.Status.LastSuccessfulTime.Time.Format(time.RFC3339))
	}

	result += fmt.Sprintf("Active Jobs: %d\n", len(cronJob.Status.Active))
	result += fmt.Sprintf("Created: %s\n", cronJob.CreationTimestamp.Time.Format(time.RFC3339))

	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		result += fmt.Sprintf("Successful Jobs History Limit: %d\n", *cronJob.Spec.SuccessfulJobsHistoryLimit)
	}
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		result += fmt.Sprintf("Failed Jobs History Limit: %d\n", *cronJob.Spec.FailedJobsHistoryLimit)
	}
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		result += fmt.Sprintf("Starting Deadline Seconds: %d\n", *cronJob.Spec.StartingDeadlineSeconds)
	}

	if len(cronJob.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range cronJob.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		result += fmt.Sprintf("\nImage: %s\n", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	}

	return result
}

func formatCronJobList(cronJobs *batchv1.CronJobList, includeNamespace bool) string {
	var result strings.Builder

	if includeNamespace {
		result.WriteString("CronJobs across all namespaces:\n")
	} else {
		result.WriteString(fmt.Sprintf("CronJobs in namespace %q:\n", cronJobs.Items[0].Namespace))
	}

	for _, cronJob := range cronJobs.Items {
		age := time.Since(cronJob.CreationTimestamp.Time).Round(time.Second)

		suspended := "Active"
		if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
			suspended = "Suspended"
		}

		lastSchedule := "<none>"
		if cronJob.Status.LastScheduleTime != nil {
			lastSchedule = formatDuration(time.Since(cronJob.Status.LastScheduleTime.Time))
		}

		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: Schedule=%s, %s, LastSchedule=%s, Active=%d, Age=%s",
				cronJob.Namespace, cronJob.Name, cronJob.Spec.Schedule, suspended, lastSchedule, len(cronJob.Status.Active), formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: Schedule=%s, %s, LastSchedule=%s, Active=%d, Age=%s",
				cronJob.Name, cronJob.Spec.Schedule, suspended, lastSchedule, len(cronJob.Status.Active), formatDuration(age)))
		}

		if len(cronJob.Labels) > 0 {
			result.WriteString(fmt.Sprintf(" - Labels: %d", len(cronJob.Labels)))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d CronJob(s)", len(cronJobs.Items)))

	return result.String()
}

func formatIngress(ingress *networkingv1.Ingress) string {
	result := fmt.Sprintf("Ingress: %s\n", ingress.Name)
	result += fmt.Sprintf("Namespace: %s\n", ingress.Namespace)

	if ingress.Spec.IngressClassName != nil {
		result += fmt.Sprintf("Ingress Class: %s\n", *ingress.Spec.IngressClassName)
	}

	result += fmt.Sprintf("Created: %s\n", ingress.CreationTimestamp.Time.Format(time.RFC3339))

	// Default backend
	if ingress.Spec.DefaultBackend != nil {
		result += "\nDefault Backend:\n"
		if ingress.Spec.DefaultBackend.Service != nil {
			result += fmt.Sprintf("  Service: %s\n", ingress.Spec.DefaultBackend.Service.Name)
			if ingress.Spec.DefaultBackend.Service.Port.Number > 0 {
				result += fmt.Sprintf("  Port: %d\n", ingress.Spec.DefaultBackend.Service.Port.Number)
			} else if ingress.Spec.DefaultBackend.Service.Port.Name != "" {
				result += fmt.Sprintf("  Port: %s\n", ingress.Spec.DefaultBackend.Service.Port.Name)
			}
		}
	}

	// Rules
	if len(ingress.Spec.Rules) > 0 {
		result += "\nRules:\n"
		for _, rule := range ingress.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = "*"
			}
			result += fmt.Sprintf("  Host: %s\n", host)

			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					pathType := "Prefix"
					if path.PathType != nil {
						pathType = string(*path.PathType)
					}
					pathStr := path.Path
					if pathStr == "" {
						pathStr = "/"
					}

					if path.Backend.Service != nil {
						portStr := ""
						if path.Backend.Service.Port.Number > 0 {
							portStr = fmt.Sprintf("%d", path.Backend.Service.Port.Number)
						} else if path.Backend.Service.Port.Name != "" {
							portStr = path.Backend.Service.Port.Name
						}
						result += fmt.Sprintf("    %s (%s) → %s:%s\n", pathStr, pathType, path.Backend.Service.Name, portStr)
					}
				}
			}
		}
	}

	// TLS configuration
	if len(ingress.Spec.TLS) > 0 {
		result += "\nTLS:\n"
		for _, tls := range ingress.Spec.TLS {
			if len(tls.Hosts) > 0 {
				result += fmt.Sprintf("  Hosts: %s\n", strings.Join(tls.Hosts, ", "))
			}
			if tls.SecretName != "" {
				result += fmt.Sprintf("  Secret: %s\n", tls.SecretName)
			}
		}
	}

	// Status - Load Balancer addresses
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		result += "\nLoad Balancer:\n"
		for _, lb := range ingress.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				result += fmt.Sprintf("  IP: %s\n", lb.IP)
			}
			if lb.Hostname != "" {
				result += fmt.Sprintf("  Hostname: %s\n", lb.Hostname)
			}
		}
	}

	// Labels
	if len(ingress.Labels) > 0 {
		result += "\nLabels:\n"
		for k, v := range ingress.Labels {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	// Annotations
	if len(ingress.Annotations) > 0 {
		result += "\nAnnotations:\n"
		for k, v := range ingress.Annotations {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
	}

	return result
}

func formatIngressList(ingresses *networkingv1.IngressList, includeNamespace bool) string {
	var result strings.Builder

	if includeNamespace {
		result.WriteString("Ingresses across all namespaces:\n")
	} else {
		if len(ingresses.Items) > 0 {
			result.WriteString(fmt.Sprintf("Ingresses in namespace %q:\n", ingresses.Items[0].Namespace))
		} else {
			result.WriteString("Ingresses in namespace:\n")
		}
	}

	for _, ingress := range ingresses.Items {
		age := time.Since(ingress.CreationTimestamp.Time).Round(time.Second)

		// Collect hosts
		hosts := []string{}
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		hostStr := "*"
		if len(hosts) > 0 {
			hostStr = strings.Join(hosts, ", ")
		}

		// Get address from status
		address := "<pending>"
		if len(ingress.Status.LoadBalancer.Ingress) > 0 {
			lb := ingress.Status.LoadBalancer.Ingress[0]
			if lb.IP != "" {
				address = lb.IP
			} else if lb.Hostname != "" {
				address = lb.Hostname
			}
		}

		// Get ingress class
		ingressClass := "<none>"
		if ingress.Spec.IngressClassName != nil {
			ingressClass = *ingress.Spec.IngressClassName
		}

		if includeNamespace {
			result.WriteString(fmt.Sprintf("• %s/%s: Class=%s, Hosts=%s, Address=%s, Age=%s",
				ingress.Namespace, ingress.Name, ingressClass, hostStr, address, formatDuration(age)))
		} else {
			result.WriteString(fmt.Sprintf("• %s: Class=%s, Hosts=%s, Address=%s, Age=%s",
				ingress.Name, ingressClass, hostStr, address, formatDuration(age)))
		}

		// TLS indicator
		if len(ingress.Spec.TLS) > 0 {
			result.WriteString(" [TLS]")
		}

		if len(ingress.Labels) > 0 {
			result.WriteString(fmt.Sprintf(" - Labels: %d", len(ingress.Labels)))
		}

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\nTotal: %d Ingress(es)", len(ingresses.Items)))

	return result.String()
}
