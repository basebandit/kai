package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// IngressFactory is an interface for creating Ingress operators.
type IngressFactory interface {
	NewIngress(params kai.IngressParams) kai.IngressOperator
}

// DefaultIngressFactory implements the IngressFactory interface.
type DefaultIngressFactory struct{}

// NewDefaultIngressFactory creates a new DefaultIngressFactory.
func NewDefaultIngressFactory() *DefaultIngressFactory {
	return &DefaultIngressFactory{}
}

// NewIngress creates a new Ingress operator.
func (f *DefaultIngressFactory) NewIngress(params kai.IngressParams) kai.IngressOperator {
	return &cluster.Ingress{
		Name:             params.Name,
		Namespace:        params.Namespace,
		IngressClassName: params.IngressClassName,
		Labels:           params.Labels,
		Annotations:      params.Annotations,
		Rules:            params.Rules,
		TLS:              params.TLS,
		DefaultBackend:   params.DefaultBackend,
	}
}

// RegisterIngressTools registers all Ingress-related tools with the server.
func RegisterIngressTools(s kai.ServerInterface, cm kai.ClusterManager) {
	factory := NewDefaultIngressFactory()
	RegisterIngressToolsWithFactory(s, cm, factory)
}

// RegisterIngressToolsWithFactory registers all Ingress-related tools using the provided factory.
func RegisterIngressToolsWithFactory(s kai.ServerInterface, cm kai.ClusterManager, factory IngressFactory) {
	createIngressTool := mcp.NewTool("create_ingress",
		mcp.WithDescription("Create a new Ingress in the specified namespace for HTTP/HTTPS routing"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Ingress"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace for the Ingress (defaults to current namespace)"),
		),
		mcp.WithString("ingress_class",
			mcp.Description("Ingress class name (e.g., 'nginx', 'traefik')"),
		),
		mcp.WithArray("rules",
			mcp.Description("Ingress rules as array of objects with 'host' and 'paths'. Each path has 'path', 'path_type' (Prefix/Exact), 'service_name', and 'service_port'"),
		),
		mcp.WithObject("default_backend",
			mcp.Description("Default backend as an object with 'service_name' and 'service_port'. Provide this if no rules are specified"),
		),
		mcp.WithArray("tls",
			mcp.Description("TLS configuration as array of objects with 'hosts' (array) and 'secret_name'"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to apply to the Ingress"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to apply to the Ingress (e.g., for ingress controller configuration)"),
		),
	)
	s.AddTool(createIngressTool, createIngressHandler(cm, factory))

	getIngressTool := mcp.NewTool("get_ingress",
		mcp.WithDescription("Get information about a specific Ingress"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Ingress"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Ingress (defaults to current namespace)"),
		),
	)
	s.AddTool(getIngressTool, getIngressHandler(cm, factory))

	listIngressesTool := mcp.NewTool("list_ingresses",
		mcp.WithDescription("List Ingresses in the current namespace or across all namespaces"),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Whether to list Ingresses across all namespaces"),
		),
		mcp.WithString("namespace",
			mcp.Description("Specific namespace to list Ingresses from (defaults to current namespace)"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Label selector to filter Ingresses (e.g., 'app=nginx,env=prod')"),
		),
	)
	s.AddTool(listIngressesTool, listIngressesHandler(cm, factory))

	updateIngressTool := mcp.NewTool("update_ingress",
		mcp.WithDescription("Update an existing Ingress"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Ingress to update"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Ingress (defaults to current namespace)"),
		),
		mcp.WithString("ingress_class",
			mcp.Description("New Ingress class name"),
		),
		mcp.WithArray("rules",
			mcp.Description("New Ingress rules (replaces existing rules)"),
		),
		mcp.WithObject("default_backend",
			mcp.Description("Default backend as an object with 'service_name' and 'service_port'"),
		),
		mcp.WithArray("tls",
			mcp.Description("New TLS configuration (replaces existing TLS)"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to add/update on the Ingress"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to add/update on the Ingress"),
		),
	)
	s.AddTool(updateIngressTool, updateIngressHandler(cm, factory))

	deleteIngressTool := mcp.NewTool("delete_ingress",
		mcp.WithDescription("Delete an Ingress from the specified namespace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Ingress to delete"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Ingress (defaults to current namespace)"),
		),
	)
	s.AddTool(deleteIngressTool, deleteIngressHandler(cm, factory))
}

func createIngressHandler(cm kai.ClusterManager, factory IngressFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_ingress"))

		nameArg, ok := request.Params.Arguments["name"]
		if !ok || nameArg == nil {
			return mcp.NewToolResultText(errMissingName), nil
		}

		name, ok := nameArg.(string)
		if !ok || name == "" {
			return mcp.NewToolResultText(errEmptyName), nil
		}

		rulesArg, hasRules := request.Params.Arguments["rules"]
		defaultBackendArg, hasDefaultBackend := request.Params.Arguments["default_backend"]
		if !hasRules && !hasDefaultBackend {
			return mcp.NewToolResultText("Required parameter 'rules' or 'default_backend' is missing"), nil
		}

		var rulesSlice []interface{}
		if hasRules {
			var ok bool
			rulesSlice, ok = rulesArg.([]interface{})
			if !ok || len(rulesSlice) == 0 {
				if !hasDefaultBackend {
					return mcp.NewToolResultText("Parameter 'rules' must be a non-empty array"), nil
				}
				rulesSlice = nil
			}
		}

		namespace := cm.GetCurrentNamespace()
		if namespaceArg, ok := request.Params.Arguments["namespace"].(string); ok && namespaceArg != "" {
			namespace = namespaceArg
		}

		params := kai.IngressParams{
			Name:      name,
			Namespace: namespace,
		}

		if ingressClassArg, ok := request.Params.Arguments["ingress_class"].(string); ok && ingressClassArg != "" {
			params.IngressClassName = ingressClassArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		// Parse rules
		if len(rulesSlice) > 0 {
			rules, err := parseIngressRules(rulesSlice)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid rules: %s", err.Error())), nil
			}
			params.Rules = rules
		}

		// Parse default backend
		if hasDefaultBackend {
			backend, err := parseIngressBackend(defaultBackendArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid default backend: %s", err.Error())), nil
			}
			params.DefaultBackend = backend
		}

		// Parse TLS
		if tlsArg, ok := request.Params.Arguments["tls"].([]interface{}); ok {
			tls, err := parseIngressTLS(tlsArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid TLS configuration: %s", err.Error())), nil
			}
			params.TLS = tls
		}

		ingress := factory.NewIngress(params)
		result, err := ingress.Create(ctx, cm)
		if err != nil {
			slog.Warn("failed to create Ingress",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create Ingress: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func getIngressHandler(cm kai.ClusterManager, factory IngressFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_ingress"))

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

		params := kai.IngressParams{
			Name:      name,
			Namespace: namespace,
		}

		ingress := factory.NewIngress(params)
		result, err := ingress.Get(ctx, cm)
		if err != nil {
			slog.Warn("failed to get Ingress",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get Ingress: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func listIngressesHandler(cm kai.ClusterManager, factory IngressFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_ingresses"))

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

		params := kai.IngressParams{
			Namespace: namespace,
		}

		ingress := factory.NewIngress(params)
		result, err := ingress.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			slog.Warn("failed to list Ingresses",
				slog.Bool("all_namespaces", allNamespaces),
				slog.String("namespace", namespace),
				slog.String("label_selector", labelSelector),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list Ingresses: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func updateIngressHandler(cm kai.ClusterManager, factory IngressFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "update_ingress"))

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

		params := kai.IngressParams{
			Name:      name,
			Namespace: namespace,
		}

		if ingressClassArg, ok := request.Params.Arguments["ingress_class"].(string); ok && ingressClassArg != "" {
			params.IngressClassName = ingressClassArg
		}

		if labelsArg, ok := request.Params.Arguments["labels"].(map[string]interface{}); ok {
			params.Labels = labelsArg
		}

		if annotationsArg, ok := request.Params.Arguments["annotations"].(map[string]interface{}); ok {
			params.Annotations = annotationsArg
		}

		// Parse rules if provided
		if rulesArg, ok := request.Params.Arguments["rules"].([]interface{}); ok && len(rulesArg) > 0 {
			rules, err := parseIngressRules(rulesArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid rules: %s", err.Error())), nil
			}
			params.Rules = rules
		}

		// Parse default backend if provided
		if defaultBackendArg, ok := request.Params.Arguments["default_backend"]; ok {
			backend, err := parseIngressBackend(defaultBackendArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid default backend: %s", err.Error())), nil
			}
			params.DefaultBackend = backend
		}

		// Parse TLS if provided
		if tlsArg, ok := request.Params.Arguments["tls"].([]interface{}); ok {
			tls, err := parseIngressTLS(tlsArg)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Invalid TLS configuration: %s", err.Error())), nil
			}
			params.TLS = tls
		}

		ingress := factory.NewIngress(params)
		result, err := ingress.Update(ctx, cm)
		if err != nil {
			slog.Warn("failed to update Ingress",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to update Ingress: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func deleteIngressHandler(cm kai.ClusterManager, factory IngressFactory) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "delete_ingress"))

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

		params := kai.IngressParams{
			Name:      name,
			Namespace: namespace,
		}

		ingress := factory.NewIngress(params)
		result, err := ingress.Delete(ctx, cm)
		if err != nil {
			slog.Warn("failed to delete Ingress",
				slog.String("name", name),
				slog.String("namespace", namespace),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete Ingress: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func parseIngressRules(rulesSlice []interface{}) ([]kai.IngressRule, error) {
	rules := make([]kai.IngressRule, 0, len(rulesSlice))

	for i, ruleItem := range rulesSlice {
		ruleMap, ok := ruleItem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule %d: must be an object", i)
		}

		rule := kai.IngressRule{}

		if host, ok := ruleMap["host"].(string); ok {
			rule.Host = host
		}

		pathsArg, ok := ruleMap["paths"].([]interface{})
		if !ok || len(pathsArg) == 0 {
			return nil, fmt.Errorf("rule %d: 'paths' must be a non-empty array", i)
		}

		paths := make([]kai.IngressPath, 0, len(pathsArg))
		for j, pathItem := range pathsArg {
			pathMap, ok := pathItem.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("rule %d, path %d: must be an object", i, j)
			}

			path := kai.IngressPath{}

			if pathStr, ok := pathMap["path"].(string); ok {
				path.Path = pathStr
			} else {
				path.Path = "/"
			}

			if pathType, ok := pathMap["path_type"].(string); ok {
				path.PathType = pathType
			} else {
				path.PathType = "Prefix"
			}

			serviceName, ok := pathMap["service_name"].(string)
			if !ok || serviceName == "" {
				return nil, fmt.Errorf("rule %d, path %d: 'service_name' is required", i, j)
			}
			path.ServiceName = serviceName

			servicePort := pathMap["service_port"]
			if servicePort == nil {
				return nil, fmt.Errorf("rule %d, path %d: 'service_port' is required", i, j)
			}
			path.ServicePort = servicePort

			paths = append(paths, path)
		}

		rule.Paths = paths
		rules = append(rules, rule)
	}

	return rules, nil
}

func parseIngressBackend(backendArg interface{}) (*kai.IngressBackend, error) {
	backendMap, ok := backendArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("default_backend must be an object")
	}

	serviceName, ok := backendMap["service_name"].(string)
	if !ok || serviceName == "" {
		return nil, fmt.Errorf("default_backend: 'service_name' is required")
	}

	servicePort := backendMap["service_port"]
	if servicePort == nil {
		return nil, fmt.Errorf("default_backend: 'service_port' is required")
	}

	return &kai.IngressBackend{
		ServiceName: serviceName,
		ServicePort: servicePort,
	}, nil
}

func parseIngressTLS(tlsSlice []interface{}) ([]kai.IngressTLS, error) {
	tls := make([]kai.IngressTLS, 0, len(tlsSlice))

	for i, tlsItem := range tlsSlice {
		tlsMap, ok := tlsItem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("TLS %d: must be an object", i)
		}

		tlsConfig := kai.IngressTLS{}

		if hostsArg, ok := tlsMap["hosts"].([]interface{}); ok {
			hosts := make([]string, 0, len(hostsArg))
			for _, h := range hostsArg {
				if host, ok := h.(string); ok {
					hosts = append(hosts, host)
				}
			}
			tlsConfig.Hosts = hosts
		}

		if secretName, ok := tlsMap["secret_name"].(string); ok {
			tlsConfig.SecretName = secretName
		}

		tls = append(tls, tlsConfig)
	}

	return tls, nil
}
