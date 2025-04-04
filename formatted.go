package kai

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
				if status.State.Running != nil {
					result += fmt.Sprintf("   Started At: %s\n", status.State.Running.StartedAt.Format(time.RFC3339))
				} else if status.State.Waiting != nil {
					result += fmt.Sprintf("   Waiting: %s - %s\n", status.State.Waiting.Reason, status.State.Waiting.Message)
				} else if status.State.Terminated != nil {
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

		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second).String()

		// Format the pod info
		podInfo := ""
		if allNamespaces {
			podInfo = fmt.Sprintf("• %s/%s: %s (%s/%s) - IP: %s - Age: %s",
				pod.Namespace, pod.Name, status, ready, total, pod.Status.PodIP, age)
		} else {
			podInfo = fmt.Sprintf("• %s: %s (%s/%s) - IP: %s - Age: %s",
				pod.Name, status, ready, total, pod.Status.PodIP, age)
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
		resultText += fmt.Sprintf("• %s/%s: %d/%d replicas ready - Age: %s\n",
			deployment.Namespace,
			deployment.Name,
			deployment.Status.ReadyReplicas,
			deployment.Status.Replicas,
			time.Since(deployment.CreationTimestamp.Time).Round(time.Second).String())
	}

	return resultText
}
