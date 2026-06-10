package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterCustomResourceTools registers CRD, custom resource and API discovery tools.
func RegisterCustomResourceTools(s kai.ServerInterface, cm kai.ClusterManager) {
	s.AddTool(mcp.NewTool("list_crds",
		mcp.WithDescription("List all CustomResourceDefinitions registered in the cluster"),
		readOnlyAnnotation("List CRDs"),
	), listCRDsHandler(cm))

	s.AddTool(mcp.NewTool("get_crd",
		mcp.WithDescription("Get details about a CustomResourceDefinition, including how to query its instances"),
		readOnlyAnnotation("Get CRD"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the CRD (e.g. 'widgets.example.com')")),
	), getCRDHandler(cm))

	s.AddTool(mcp.NewTool("list_custom_resources",
		mcp.WithDescription("List instances of a custom resource by group/version/resource"),
		readOnlyAnnotation("List custom resources"),
		mcp.WithString("group", mcp.Description("API group (e.g. 'example.com'; empty for core)")),
		mcp.WithString("version", mcp.Required(), mcp.Description("API version (e.g. 'v1')")),
		mcp.WithString("resource", mcp.Required(), mcp.Description("Plural resource name (e.g. 'widgets')")),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current; ignored for cluster-scoped)")),
		mcp.WithBoolean("all_namespaces", mcp.Description("List across all namespaces")),
	), listCustomResourcesHandler(cm))

	s.AddTool(mcp.NewTool("get_custom_resource",
		mcp.WithDescription("Get a single custom resource instance by group/version/resource/name"),
		readOnlyAnnotation("Get custom resource"),
		mcp.WithString("group", mcp.Description("API group (e.g. 'example.com'; empty for core)")),
		mcp.WithString("version", mcp.Required(), mcp.Description("API version (e.g. 'v1')")),
		mcp.WithString("resource", mcp.Required(), mcp.Description("Plural resource name (e.g. 'widgets')")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the resource instance")),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current; ignored for cluster-scoped)")),
	), getCustomResourceHandler(cm))

	s.AddTool(mcp.NewTool("list_api_resources",
		mcp.WithDescription("List the server's preferred API resources (like 'kubectl api-resources')"),
		readOnlyAnnotation("List API resources"),
	), listAPIResourcesHandler(cm))
}

func listCRDsHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_crds"))
		cr := cluster.CustomResource{}
		result, err := cr.ListCRDs(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list CRDs: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getCRDHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		cr := cluster.CustomResource{Name: name}
		result, err := cr.GetCRD(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get CRD: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func customResourceFromRequest(request mcp.CallToolRequest) (cluster.CustomResource, *mcp.CallToolResult) {
	cr := cluster.CustomResource{}
	if g, ok := request.GetArguments()["group"].(string); ok {
		cr.Group = g
	}
	version, ok := request.GetArguments()["version"].(string)
	if !ok || version == "" {
		return cr, mcp.NewToolResultText("Required parameter 'version' is missing")
	}
	cr.Version = version
	resource, ok := request.GetArguments()["resource"].(string)
	if !ok || resource == "" {
		return cr, mcp.NewToolResultText("Required parameter 'resource' is missing")
	}
	cr.Resource = resource
	if ns, ok := request.GetArguments()["namespace"].(string); ok {
		cr.Namespace = ns
	}
	return cr, nil
}

func listCustomResourcesHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_custom_resources"))
		cr, errResult := customResourceFromRequest(request)
		if errResult != nil {
			return errResult, nil
		}
		allNamespaces := false
		if all, ok := request.GetArguments()["all_namespaces"].(bool); ok {
			allNamespaces = all
		}
		result, err := cr.List(ctx, cm, allNamespaces)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list custom resources: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getCustomResourceHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_custom_resource"))
		cr, errResult := customResourceFromRequest(request)
		if errResult != nil {
			return errResult, nil
		}
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		cr.Name = name
		result, err := cr.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get custom resource: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func listAPIResourcesHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_api_resources"))
		cr := cluster.CustomResource{}
		result, err := cr.ListAPIResources(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list API resources: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
