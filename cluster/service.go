package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

// Service represents a Kubernetes service configuration
type Service struct {
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

// Create creates a new service in the cluster
func (s *Service) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, s.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", s.Namespace, err)
	}

	// Validate service
	if err := s.validate(); err != nil {
		return result, err
	}

	// Convert labels and selector to string maps
	labels := convertToStringMap(s.Labels)
	selector := convertToStringMap(s.Selector)

	// Create the service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
		},
	}

	// Set service type if specified
	if s.Type != "" {
		validTypes := map[string]corev1.ServiceType{
			"ClusterIP":    corev1.ServiceTypeClusterIP,
			"NodePort":     corev1.ServiceTypeNodePort,
			"LoadBalancer": corev1.ServiceTypeLoadBalancer,
			"ExternalName": corev1.ServiceTypeExternalName,
		}
		if serviceType, ok := validTypes[s.Type]; ok {
			service.Spec.Type = serviceType
		} else {
			return result, fmt.Errorf("invalid service type: %s", s.Type)
		}
	}

	// Set ClusterIP if specified
	if s.ClusterIP != "" {
		service.Spec.ClusterIP = s.ClusterIP
	}

	// Set ExternalName if service type is ExternalName
	if s.Type == "ExternalName" && s.ExternalName != "" {
		service.Spec.ExternalName = s.ExternalName
	}

	// Set ExternalIPs if specified
	if len(s.ExternalIPs) > 0 {
		service.Spec.ExternalIPs = s.ExternalIPs
	}

	// Set SessionAffinity if specified
	if s.SessionAffinity != "" {
		validAffinity := map[string]corev1.ServiceAffinity{
			"None":     corev1.ServiceAffinityNone,
			"ClientIP": corev1.ServiceAffinityClientIP,
		}
		if affinity, ok := validAffinity[s.SessionAffinity]; ok {
			service.Spec.SessionAffinity = affinity
		} else {
			return result, fmt.Errorf("invalid session affinity: %s", s.SessionAffinity)
		}
	}

	// Set ports if specified
	if len(s.Ports) > 0 {
		servicePorts := make([]corev1.ServicePort, 0, len(s.Ports))
		for _, port := range s.Ports {
			servicePort := corev1.ServicePort{
				Port:     port.Port,
				Protocol: corev1.ProtocolTCP, // Default to TCP
			}

			if port.Name != "" {
				servicePort.Name = port.Name
			}

			if port.NodePort != 0 {
				if service.Spec.Type != corev1.ServiceTypeNodePort && service.Spec.Type != corev1.ServiceTypeLoadBalancer {
					return result, fmt.Errorf("nodePort can only be specified for NodePort or LoadBalancer service types")
				}
				servicePort.NodePort = port.NodePort
			}

			// Set protocol if specified
			if port.Protocol != "" {
				protocol := corev1.Protocol(strings.ToUpper(port.Protocol))
				if protocol == corev1.ProtocolTCP || protocol == corev1.ProtocolUDP || protocol == corev1.ProtocolSCTP {
					servicePort.Protocol = protocol
				} else {
					return result, fmt.Errorf("invalid protocol: %s", port.Protocol)
				}
			}

			// Set targetPort if specified
			if port.TargetPort != nil {
				switch v := port.TargetPort.(type) {
				case int32:
					servicePort.TargetPort = intstr.FromInt(int(v))
				case int:
					servicePort.TargetPort = intstr.FromInt(v)
				case float64:
					servicePort.TargetPort = intstr.FromInt(int(v))
				case string:
					servicePort.TargetPort = intstr.FromString(v)
				default:
					return result, fmt.Errorf("unsupported targetPort type: %T", v)
				}
			} else {
				// Default targetPort to the same as port
				servicePort.TargetPort = intstr.FromInt(int(port.Port))
			}

			servicePorts = append(servicePorts, servicePort)
		}
		service.Spec.Ports = servicePorts
	} else {
		return result, errors.New("at least one port must be specified")
	}

	createdService, err := client.CoreV1().Services(s.Namespace).Create(timeoutCtx, service, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create service: %w", err)
	}

	result = fmt.Sprintf("Service %q created successfully in namespace %q", createdService.Name, createdService.Namespace)
	result += fmt.Sprintf(" (Type: %s)", createdService.Spec.Type)

	// Add ports to result
	if len(createdService.Spec.Ports) > 0 {
		result += "\nPorts:"
		for _, port := range createdService.Spec.Ports {
			portInfo := fmt.Sprintf("\n- %d", port.Port)
			if port.Name != "" {
				portInfo += fmt.Sprintf(" (%s)", port.Name)
			}

			targetPort := port.TargetPort.String()
			portInfo += fmt.Sprintf(" → %s", targetPort)

			if port.NodePort > 0 {
				portInfo += fmt.Sprintf(" (NodePort: %d)", port.NodePort)
			}

			portInfo += fmt.Sprintf(" [%s]", port.Protocol)
			result += portInfo
		}
	}

	// Add ClusterIP to result
	if createdService.Spec.ClusterIP != "" && createdService.Spec.ClusterIP != "None" {
		result += fmt.Sprintf("\nClusterIP: %s", createdService.Spec.ClusterIP)
	}

	return result, nil
}

// Get retrieves information about a specific service
func (s *Service) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, err
	}

	// Verify the namespace exists
	_, err = client.CoreV1().Namespaces().Get(ctx, s.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace '%s' not found: %v", s.Namespace, err)
	}

	// Use retry for potential transient issues
	var service *corev1.Service
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		// Only retry on transient errors
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		service, getErr = client.CoreV1().Services(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("service '%s' not found in namespace '%s'", s.Name, s.Namespace)
		}
		return result, fmt.Errorf("failed to get service '%s' in namespace '%s': %v", s.Name, s.Namespace, err)
	}

	result = formatService(service)

	return result, nil
}

// List lists services in the specified namespace or across all namespaces
func (s *Service) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
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
	namespace := s.Namespace
	if namespace == "" && !allNamespaces {
		namespace = cm.GetCurrentNamespace()
	}

	if allNamespaces {
		services, err := client.CoreV1().Services("").List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list services: %w", err)
		}

		if len(services.Items) == 0 {
			result = "No services found across all namespaces"
			return result, nil
		}
		result = "Services across all namespaces:\n"
		result += formatServiceList(services, true)
	} else {
		// First verify the namespace exists
		_, err = client.CoreV1().Namespaces().Get(timeoutCtx, namespace, metav1.GetOptions{})
		if err != nil {
			return result, fmt.Errorf("namespace %q not found: %w", namespace, err)
		}

		services, err := client.CoreV1().Services(namespace).List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list services: %w", err)
		}

		if len(services.Items) == 0 {
			result = fmt.Sprintf("No services found in namespace %q", namespace)
			return result, nil
		}

		result = fmt.Sprintf("Services in namespace %q:\n", namespace)
		result += formatServiceList(services, false)
	}

	return result, nil
}

// Delete deletes a service or services that match the given criteria from the cluster
func (s *Service) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, s.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", s.Namespace, err)
	}

	if s.Name != "" {
		// Check if the service exists first
		_, err = client.CoreV1().Services(s.Namespace).Get(timeoutCtx, s.Name, metav1.GetOptions{})
		if err != nil {
			return result, fmt.Errorf("failed to find service %q in namespace %q: %w", s.Name, s.Namespace, err)
		}

		// Delete the specific service
		deleteOptions := metav1.DeleteOptions{}
		err = client.CoreV1().Services(s.Namespace).Delete(timeoutCtx, s.Name, deleteOptions)
		if err != nil {
			return result, fmt.Errorf("failed to delete service %q from namespace %q: %w", s.Name, s.Namespace, err)
		}

		result = fmt.Sprintf("Service %q deleted successfully from namespace %q", s.Name, s.Namespace)
		return result, nil
	}

	// If name is not provided but labels are, delete services matching the label selector
	if len(s.Labels) > 0 {
		// Convert labels to string map for the selector
		labelSelector := ""
		for k, v := range s.Labels {
			if labelSelector != "" {
				labelSelector += ","
			}
			switch val := v.(type) {
			case string:
				labelSelector += fmt.Sprintf("%s=%s", k, val)
			default:
				labelSelector += fmt.Sprintf("%s=%v", k, val)
			}
		}

		// List services matching the selector
		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
		}

		serviceList, err := client.CoreV1().Services(s.Namespace).List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list services with label selector %q in namespace %q: %w", labelSelector, s.Namespace, err)
		}

		if len(serviceList.Items) == 0 {
			return result, fmt.Errorf("no services found with label selector %q in namespace %q", labelSelector, s.Namespace)
		}

		// Delete each matching service
		deleteOptions := metav1.DeleteOptions{}
		deletedCount := 0
		deletedNames := []string{}

		for _, service := range serviceList.Items {
			err = client.CoreV1().Services(s.Namespace).Delete(timeoutCtx, service.Name, deleteOptions)
			if err != nil {
				// Continue trying to delete other services even if one fails
				result += fmt.Sprintf("Failed to delete service %q: %v\n", service.Name, err)
			} else {
				deletedCount++
				deletedNames = append(deletedNames, service.Name)
			}
		}

		if deletedCount == 0 {
			return result, fmt.Errorf("failed to delete any services with label selector %q in namespace %q", labelSelector, s.Namespace)
		}

		result = fmt.Sprintf("Deleted %d services with label selector %q from namespace %q:\n- %s",
			deletedCount, labelSelector, s.Namespace, strings.Join(deletedNames, "\n- "))
		return result, nil
	}

	return result, errors.New("either service name or label selector must be provided")
}

// validate validates the service parameters
func (s *Service) validate() error {
	// Name is required
	if s.Name == "" {
		return errors.New("service name is required")
	}

	// Namespace is required
	if s.Namespace == "" {
		return errors.New("namespace is required")
	}

	// Service type validation
	if s.Type != "" {
		validTypes := map[string]bool{
			"ClusterIP":    true,
			"NodePort":     true,
			"LoadBalancer": true,
			"ExternalName": true,
		}
		if !validTypes[s.Type] {
			return fmt.Errorf("invalid service type: %s", s.Type)
		}
	}

	// Validate ports
	for i, port := range s.Ports {
		if port.Port <= 0 {
			return fmt.Errorf("port %d: port number must be positive", i)
		}

		if port.Protocol != "" {
			protocol := strings.ToUpper(port.Protocol)
			if protocol != "TCP" && protocol != "UDP" && protocol != "SCTP" {
				return fmt.Errorf("port %d: invalid protocol: %s", i, port.Protocol)
			}
		}

		if port.NodePort < 0 {
			return fmt.Errorf("port %d: nodePort must be non-negative", i)
		}
	}

	// If service type is ExternalName, ExternalName must be provided
	if s.Type == "ExternalName" && s.ExternalName == "" {
		return errors.New("externalName must be specified for ExternalName service type")
	}

	return nil
}

// Update updates an existing service in the cluster
func (s *Service) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	service, err := client.CoreV1().Services(s.Namespace).Get(timeoutCtx, s.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get service: %w", err)
	}

	if len(s.Labels) > 0 {
		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(s.Labels) {
			service.Labels[k] = v
		}
	}

	if len(s.Selector) > 0 {
		service.Spec.Selector = convertToStringMap(s.Selector)
	}

	if s.Type != "" {
		validTypes := map[string]corev1.ServiceType{
			"ClusterIP":    corev1.ServiceTypeClusterIP,
			"NodePort":     corev1.ServiceTypeNodePort,
			"LoadBalancer": corev1.ServiceTypeLoadBalancer,
			"ExternalName": corev1.ServiceTypeExternalName,
		}
		if serviceType, ok := validTypes[s.Type]; ok {
			service.Spec.Type = serviceType
		} else {
			return result, fmt.Errorf("invalid service type: %s", s.Type)
		}
	}

	if s.ClusterIP != "" {
		service.Spec.ClusterIP = s.ClusterIP
	}

	if s.ExternalName != "" {
		service.Spec.ExternalName = s.ExternalName
	}

	if len(s.ExternalIPs) > 0 {
		service.Spec.ExternalIPs = s.ExternalIPs
	}

	if s.SessionAffinity != "" {
		validAffinity := map[string]corev1.ServiceAffinity{
			"None":     corev1.ServiceAffinityNone,
			"ClientIP": corev1.ServiceAffinityClientIP,
		}
		if affinity, ok := validAffinity[s.SessionAffinity]; ok {
			service.Spec.SessionAffinity = affinity
		} else {
			return result, fmt.Errorf("invalid session affinity: %s", s.SessionAffinity)
		}
	}

	if len(s.Ports) > 0 {
		servicePorts := make([]corev1.ServicePort, 0, len(s.Ports))
		for _, port := range s.Ports {
			servicePort := corev1.ServicePort{
				Port:     port.Port,
				Protocol: corev1.ProtocolTCP,
			}

			if port.Name != "" {
				servicePort.Name = port.Name
			}

			if port.NodePort != 0 {
				servicePort.NodePort = port.NodePort
			}

			if port.Protocol != "" {
				protocol := corev1.Protocol(strings.ToUpper(port.Protocol))
				if protocol == corev1.ProtocolTCP || protocol == corev1.ProtocolUDP || protocol == corev1.ProtocolSCTP {
					servicePort.Protocol = protocol
				} else {
					return result, fmt.Errorf("invalid protocol: %s", port.Protocol)
				}
			}

			if port.TargetPort != nil {
				switch v := port.TargetPort.(type) {
				case int32:
					servicePort.TargetPort = intstr.FromInt(int(v))
				case int:
					servicePort.TargetPort = intstr.FromInt(v)
				case float64:
					servicePort.TargetPort = intstr.FromInt(int(v))
				case string:
					servicePort.TargetPort = intstr.FromString(v)
				default:
					return result, fmt.Errorf("unsupported targetPort type: %T", v)
				}
			} else {
				servicePort.TargetPort = intstr.FromInt(int(port.Port))
			}

			servicePorts = append(servicePorts, servicePort)
		}
		service.Spec.Ports = servicePorts
	}

	updatedService, err := client.CoreV1().Services(s.Namespace).Update(timeoutCtx, service, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update service: %w", err)
	}

	result = fmt.Sprintf("Service %q updated successfully in namespace %q (Type: %s)", updatedService.Name, updatedService.Namespace, updatedService.Spec.Type)
	return result, nil
}

// Patch applies a partial update to an existing service
func (s *Service) Patch(ctx context.Context, cm kai.ClusterManager, patchData map[string]interface{}) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	service, err := client.CoreV1().Services(s.Namespace).Get(timeoutCtx, s.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get service: %w", err)
	}

	if labels, ok := patchData["labels"].(map[string]interface{}); ok {
		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(labels) {
			service.Labels[k] = v
		}
	}

	if selector, ok := patchData["selector"].(map[string]interface{}); ok {
		for k, v := range convertToStringMap(selector) {
			service.Spec.Selector[k] = v
		}
	}

	if serviceType, ok := patchData["type"].(string); ok {
		validTypes := map[string]corev1.ServiceType{
			"ClusterIP":    corev1.ServiceTypeClusterIP,
			"NodePort":     corev1.ServiceTypeNodePort,
			"LoadBalancer": corev1.ServiceTypeLoadBalancer,
			"ExternalName": corev1.ServiceTypeExternalName,
		}
		if st, ok := validTypes[serviceType]; ok {
			service.Spec.Type = st
		} else {
			return result, fmt.Errorf("invalid service type: %s", serviceType)
		}
	}

	if externalIPs, ok := patchData["externalIPs"].([]interface{}); ok {
		ips := make([]string, 0, len(externalIPs))
		for _, ip := range externalIPs {
			if ipStr, ok := ip.(string); ok {
				ips = append(ips, ipStr)
			}
		}
		service.Spec.ExternalIPs = ips
	}

	updatedService, err := client.CoreV1().Services(s.Namespace).Update(timeoutCtx, service, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to patch service: %w", err)
	}

	result = fmt.Sprintf("Service %q patched successfully in namespace %q", updatedService.Name, updatedService.Namespace)
	return result, nil
}
