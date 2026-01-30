package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServiceFactory is an interface for creating service operators
type ServiceFactory interface {
	NewService(params kai.ServiceParams) kai.ServiceOperator
}

// DefaultServiceFactory implements the ServiceFactory interface
type DefaultServiceFactory struct{}

// NewDefaultServiceFactory creates a new DefaultServiceFactory
func NewDefaultServiceFactory() *DefaultServiceFactory {
	return &DefaultServiceFactory{}
}

// NewService creates a new service operator
func (f *DefaultServiceFactory) NewService(params kai.ServiceParams) kai.ServiceOperator {
	ports := make([]cluster.ServicePort, 0, len(params.Ports))
	for _, p := range params.Ports {
		ports = append(ports, cluster.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort,
			NodePort:   p.NodePort,
			Protocol:   p.Protocol,
		})
	}

	return &cluster.Service{
		Name:            params.Name,
		Namespace:       params.Namespace,
		Labels:          params.Labels,
		Selector:        params.Selector,
		Type:            params.Type,
		Ports:           ports,
		ClusterIP:       params.ClusterIP,
		ExternalIPs:     params.ExternalIPs,
		ExternalName:    params.ExternalName,
		SessionAffinity: params.SessionAffinity,
	}
}

// RegisterServiceTools registers all service-related tools with the server
func RegisterServiceTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultServiceFactory()
	RegisterServiceToolsWithFactory(s, cm, factory)
}

// RegisterServiceToolsWithFactory registers all service-related tools with the server using the provided factory
func RegisterServiceToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory ServiceFactory) {
	listServiceTool := mcp.NewTool("list_services",
		mcp.WithDescription("List services in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list services across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list services from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter services"),
		),
	)

	s.AddTool(listServiceTool, listServicesHandler(cm, factory))

	getServiceTool := mcp.NewTool("get_service",
		mcp.WithDescription("Get detailed information about a specific service"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the service"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the service (defaults to current namespace)"),
		),
	)

	s.AddTool(getServiceTool, getServiceHandler(cm, factory))

	createServiceTool := mcp.NewTool("create_service",
		mcp.WithDescription("Create a new service in the current namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the service"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the service (defaults to current namespace)"),
		),
		mcp.WithString("type",
			mcp.Description("Service type (ClusterIP, NodePort, LoadBalancer, ExternalName)"),
		),
		mcp.WithObject("selector",
			mcp.Description("Pod selector as key-value pairs to route traffic to"),
		),
		mcp.WithArray("ports",
			mcp.Required(),
			mcp.Description("Ports to expose, each defined as an object with port, targetPort, etc."),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the service"),
		),
		mcp.WithString("cluster_ip",
			mcp.Description("ClusterIP to assign to the service (leave empty for auto-assignment)"),
		),
		mcp.WithArray("external_ips",
			mcp.Description("External IPs for the service"),
		),
		mcp.WithString("external_name",
			mcp.Description("External name for ExternalName service type"),
		),
		mcp.WithString("session_affinity",
			mcp.Description("Session affinity (None, ClientIP)"),
		),
	)

	s.AddTool(createServiceTool, createServiceHandler(cm, factory))

	deleteServiceTool := mcp.NewTool("delete_service",
		mcp.WithDescription("Delete a service or multiple services matching criteria from the current namespace"),
		mcp.WithString("name",
			mcp.Description("Name of the specific service to delete (either name or labels must be provided)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Label selector as key-value pairs to delete services matching these labels"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the service(s) (defaults to current namespace)"),
		),
	)

	s.AddTool(deleteServiceTool, deleteServiceHandler(cm, factory))

	updateServiceTool := mcp.NewTool("update_service",
		mcp.WithDescription("Update an existing service"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the service to update"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the service (defaults to current namespace)"),
		),
		mcp.WithString("type",
			mcp.Description("Service type (ClusterIP, NodePort, LoadBalancer, ExternalName)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to add or update"),
		),
		mcp.WithObject("selector",
			mcp.Description("Selector labels"),
		),
		mcp.WithArray("ports",
			mcp.Description("Service ports configuration"),
		),
		mcp.WithString("cluster_ip",
			mcp.Description("ClusterIP address"),
		),
		mcp.WithArray("external_ips",
			mcp.Description("External IP addresses"),
		),
		mcp.WithString("external_name",
			mcp.Description("External name for ExternalName service type"),
		),
		mcp.WithString("session_affinity",
			mcp.Description("Session affinity (None or ClientIP)"),
		),
	)

	s.AddTool(updateServiceTool, updateServiceHandler(cm, factory))

	patchServiceTool := mcp.NewTool("patch_service",
		mcp.WithDescription("Apply a partial update to an existing service"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the service to patch"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the service (defaults to current namespace)"),
		),
		mcp.WithObject("patch",
			mcp.Required(),
			mcp.Description("Patch data as key-value pairs (e.g., labels, selector, type, externalIPs)"),
		),
	)

	s.AddTool(patchServiceTool, patchServiceHandler(cm, factory))
}

// listServicesHandler handles the list_services tool
func listServicesHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_services"))

		var allNamespaces bool

		if allNamespacesArg, ok := request.Params.Arguments["all_namespaces"].(bool); ok {
			allNamespaces = allNamespacesArg
		}

		var namespace string
		if !allNamespaces {
			if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
				namespace = namespaceArg
			} else {
				namespace = cm.GetCurrentNamespace()
			}
		}

		var labelSelector string
		if labelSelectorArg, ok := request.Params.Arguments["label_selector"].(string); ok {
			labelSelector = labelSelectorArg
		}

		params := kai.ServiceParams{
			Namespace: namespace, // will be used if allNamespaces is false
		}
		service := factory.NewService(params)

		resultText, err := service.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			slog.Warn("failed to list services",
				slog.Bool("all_namespaces", allNamespaces),
				slog.String("namespace", namespace),
				slog.String("label_selector", labelSelector),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// getServiceHandler handles the get_service tool
func getServiceHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_service"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ServiceParams{
			Name:      name,
			Namespace: namespace,
		}

		service := factory.NewService(params)

		resultText, err := service.Get(ctx, cm)
		if err != nil {
			slog.Warn("failed to get service",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// createServiceHandler handles the create_service tool
func createServiceHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_service"))

		params := kai.ServiceParams{}

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		portsArg, ok := request.Params.Arguments["ports"]
		if !ok || portsArg == nil {
			return mcp.NewToolResultText(errMissingPorts), nil
		}

		portsArray, ok := portsArg.([]interface{})
		if !ok || len(portsArray) == 0 {
			return mcp.NewToolResultText(errEmptyPorts), nil
		}

		ports, err := processPortsArray(portsArray)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Invalid ports configuration: %v", err)), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		var serviceType string
		if typeArg, ok := request.Params.Arguments["type"].(string); ok && typeArg != "" {
			validTypes := map[string]bool{
				"ClusterIP":    true,
				"NodePort":     true,
				"LoadBalancer": true,
				"ExternalName": true,
			}
			if !validTypes[typeArg] {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid service type: %s", typeArg)), nil
			}
			serviceType = typeArg
		} else {
			serviceType = "ClusterIP" // Default to ClusterIP
		}

		var selector map[string]interface{}
		if selectorArg, ok := request.Params.Arguments["selector"].(map[string]interface{}); ok && len(selectorArg) > 0 {
			selector = selectorArg
		}

		var labels map[string]interface{}
		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok && len(labelsArg) > 0 {
			labels = labelsArg
		}

		var clusterIP string
		if clusterIPArg, ok := request.Params.Arguments["cluster_ip"].(string); ok && clusterIPArg != "" {
			clusterIP = clusterIPArg
		}

		var externalIPs []string
		if externalIPsArg, ok := request.Params.Arguments["external_ips"].([]interface{}); ok && len(externalIPsArg) > 0 {
			for _, ip := range externalIPsArg {
				if ipStr, ok := ip.(string); ok && ipStr != "" {
					externalIPs = append(externalIPs, ipStr)
				}
			}
		}

		var externalName string
		if externalNameArg, ok := request.Params.Arguments["external_name"].(string); ok && externalNameArg != "" {
			externalName = externalNameArg
		}

		var sessionAffinity string
		if sessionAffinityArg, ok := request.Params.Arguments["session_affinity"].(string); ok && sessionAffinityArg != "" {
			validAffinities := map[string]bool{
				"None":     true,
				"ClientIP": true,
			}
			if !validAffinities[sessionAffinityArg] {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid session affinity: %s", sessionAffinityArg)), nil
			}
			sessionAffinity = sessionAffinityArg
		}

		params.Name = name
		params.Namespace = namespace
		params.Type = serviceType
		params.Selector = selector
		params.Labels = labels
		params.Ports = ports
		params.ClusterIP = clusterIP
		params.ExternalIPs = externalIPs
		params.ExternalName = externalName
		params.SessionAffinity = sessionAffinity

		if params.Type == "ExternalName" && params.ExternalName == "" {
			return mcp.NewToolResultText("ExternalName must be specified for ExternalName service type"), nil
		}

		service := factory.NewService(params)
		resultText, err := service.Create(ctx, cm)
		if err != nil {
			slog.Warn("failed to create service",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// deleteServiceHandler handles the delete_service tool
func deleteServiceHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_service"))

		params := kai.ServiceParams{}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}
		params.Namespace = namespace

		nameArg, nameOk := request.Params.Arguments["name"]
		if nameOk && nameArg != nil {
			name, ok := nameArg.(string)
			if !ok || name == "" {
				return mcp.NewToolResultText(errEmptyName), nil
			}
			params.Name = name
		}

		labelsArg, labelsOk := request.Params.Arguments["labels"]
		if labelsOk && labelsArg != nil {
			labels, ok := labelsArg.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultText(errMissingLabels), nil
			}
			if len(labels) == 0 {
				return mcp.NewToolResultText(errEmptyLabels), nil
			}
			params.Labels = labels
		}

		if !nameOk && !labelsOk {
			return mcp.NewToolResultText(errNoNameOrLabelsParams), nil
		}

		service := factory.NewService(params)

		resultText, err := service.Delete(ctx, cm)
		if err != nil {
			slog.Warn("failed to delete service",
				slog.String("name", params.Name),
				slog.String("namespace", params.Namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

// processPortsArray processes the ports array from the request
func processPortsArray(portsArray []interface{}) ([]kai.ServicePort, error) {
	var ports []kai.ServicePort

	for i, port := range portsArray {
		portObj, ok := port.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("port %d: must be an object", i)
		}

		portArg, ok := portObj["port"]
		if !ok || portArg == nil {
			return nil, fmt.Errorf("port %d: required field 'port' is missing", i)
		}

		var portNum int32
		switch p := portArg.(type) {
		case float64:
			portNum = int32(p)
		case int:
			portNum = int32(p)
		case string:
			pNum, err := strconv.ParseInt(p, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("port %d: invalid port number: %v", i, err)
			}
			portNum = int32(pNum)
		default:
			return nil, fmt.Errorf("port %d: unsupported port type: %T", i, p)
		}

		if portNum <= 0 || portNum > 65535 {
			return nil, fmt.Errorf("port %d: port number must be between 1 and 65535", i)
		}

		servicePort := kai.ServicePort{
			Port: portNum,
		}

		if nameArg, ok := portObj["name"].(string); ok && nameArg != "" {
			servicePort.Name = nameArg
		}

		if targetPortArg, ok := portObj["targetPort"]; ok && targetPortArg != nil {
			switch tp := targetPortArg.(type) {
			case float64:
				if tp <= 0 || tp > 65535 {
					return nil, fmt.Errorf("port %d: targetPort must be between 1 and 65535", i)
				}
				servicePort.TargetPort = int32(tp)
			case int:
				if tp <= 0 || tp > 65535 {
					return nil, fmt.Errorf("port %d: targetPort must be between 1 and 65535", i)
				}
				servicePort.TargetPort = int32(tp)
			case string:
				// If it's a number string, validate range
				if num, err := strconv.ParseInt(tp, 10, 32); err == nil {
					if num <= 0 || num > 65535 {
						return nil, fmt.Errorf("port %d: targetPort must be between 1 and 65535", i)
					}
					servicePort.TargetPort = int32(num)
				} else {
					// It's a named port
					servicePort.TargetPort = tp
				}
			default:
				return nil, fmt.Errorf("port %d: unsupported targetPort type: %T", i, tp)
			}
		} else {
			// Default to the same as port
			servicePort.TargetPort = portNum
		}

		if nodePortArg, ok := portObj["nodePort"]; ok && nodePortArg != nil {
			var nodePort int32
			switch np := nodePortArg.(type) {
			case float64:
				nodePort = int32(np)
			case int:
				nodePort = int32(np)
			case string:
				npNum, err := strconv.ParseInt(np, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("port %d: invalid nodePort: %v", i, err)
				}
				nodePort = int32(npNum)
			default:
				return nil, fmt.Errorf("port %d: unsupported nodePort type: %T", i, np)
			}

			// Validate nodePort range (Kubernetes uses 30000-32767 by default)
			// https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport
			if nodePort < 30000 || nodePort > 32767 {
				return nil, fmt.Errorf("port %d: nodePort must be between 30000 and 32767", i)
			}
			servicePort.NodePort = nodePort
		}

		if protocolArg, ok := portObj["protocol"].(string); ok && protocolArg != "" {
			protocol := protocolArg
			validProtocols := map[string]bool{
				"TCP":  true,
				"UDP":  true,
				"SCTP": true,
				"tcp":  true,
				"udp":  true,
				"sctp": true,
			}
			if !validProtocols[protocol] {
				return nil, fmt.Errorf("port %d: invalid protocol: %s", i, protocol)
			}
			servicePort.Protocol = protocol
		} else {
			servicePort.Protocol = "TCP" // Default to TCP
		}

		ports = append(ports, servicePort)
	}

	return ports, nil
}

func updateServiceHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ServiceParams{
			Name:      name,
			Namespace: namespace,
		}

		if serviceType, ok := request.Params.Arguments["type"].(string); ok && serviceType != "" {
			params.Type = serviceType
		}

		if labels, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labels
		}

		if selector, ok := request.Params.Arguments["selector"].(map[string]interface{}); ok {
			params.Selector = selector
		}

		if portsArg, ok := request.Params.Arguments["ports"].([]interface{}); ok && len(portsArg) > 0 {
			ports, err := processPortsArray(portsArg)
			if err != nil {
				return mcp.NewToolResultText(err.Error()), nil
			}
			params.Ports = ports
		}

		if clusterIP, ok := request.Params.Arguments["cluster_ip"].(string); ok && clusterIP != "" {
			params.ClusterIP = clusterIP
		}

		if externalIPsArg, ok := request.Params.Arguments["external_ips"].([]interface{}); ok {
			externalIPs := make([]string, 0, len(externalIPsArg))
			for _, ip := range externalIPsArg {
				if ipStr, ok := ip.(string); ok {
					externalIPs = append(externalIPs, ipStr)
				}
			}
			params.ExternalIPs = externalIPs
		}

		if externalName, ok := request.Params.Arguments["external_name"].(string); ok && externalName != "" {
			params.ExternalName = externalName
		}

		if sessionAffinity, ok := request.Params.Arguments["session_affinity"].(string); ok && sessionAffinity != "" {
			params.SessionAffinity = sessionAffinity
		}

		service := factory.NewService(params)
		resultText, err := service.Update(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}

func patchServiceHandler(cm kai.ClusterManager, factory ServiceFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		patchArg, ok := request.Params.Arguments["patch"]
		if !ok || patchArg == nil {
			return mcp.NewToolResultText("missing required parameter: patch"), nil
		}

		patchData, ok := patchArg.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultText("patch parameter must be an object"), nil
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.ServiceParams{
			Name:      name,
			Namespace: namespace,
		}

		service := factory.NewService(params)
		resultText, err := service.Patch(ctx, cm, patchData)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}

		return mcp.NewToolResultText(resultText), nil
	}
}
