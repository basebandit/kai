package kai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func formatServiceInfo(service *corev1.Service, includeNamespace bool) string {
	var result string
	if includeNamespace {
		result = fmt.Sprintf("• %s/%s: ", service.Namespace, service.Name)
	} else {
		result = fmt.Sprintf("• %s: ", service.Name)
	}

	// add service type
	result += fmt.Sprintf("%s", service.Spec.Type)

	// add cluster IP if available
	if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
		result += fmt.Sprintf(" - ClusterIPL %s", service.Spec.ClusterIP)
	}

	// add external IPs if available
	if len(service.Spec.ExternalIPs) > 0 {
		result += fmt.Sprintf("- External IPs: %s", strings.Join(service.Spec.ExternalIPs, ", "))
	}

	// add ports
	if len(service.Spec.Ports) > 0 {
		ports := make([]string, 0, len(service.Spec.Ports))
		for _, port := range service.Spec.Ports {
			if port.NodePort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
			}
		}
		result += fmt.Sprintf(" - Ports: %s", strings.Join(ports, ", "))
	}

	// add age
	result += fmt.Sprintf("- Age: %s\n", time.Since(service.CreationTimestamp.Time).Round(time.Second).String())

	return result
}

func jsonMapToString(m map[string]string) string {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}

	return string(jsonBytes)
}

func parseContainerPort(portStr string) (int32, error) {
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(port), nil
}
