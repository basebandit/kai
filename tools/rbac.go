package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterRBACTools registers read-only RBAC inspection tools.
func RegisterRBACTools(s kai.ServerInterface, cm kai.ClusterManager) {
	nsArg := mcp.WithString("namespace", mcp.Description("Namespace (defaults to current)"))
	allNsArg := mcp.WithBoolean("all_namespaces", mcp.Description("List across all namespaces"))
	nameArg := mcp.WithString("name", mcp.Required(), mcp.Description("Resource name"))

	s.AddTool(mcp.NewTool("list_roles", mcp.WithDescription("List RBAC roles in a namespace"),
		readOnlyAnnotation("List roles"), nsArg, allNsArg), rbacListHandler(cm, "role"))
	s.AddTool(mcp.NewTool("get_role", mcp.WithDescription("Get an RBAC role with its rules"),
		readOnlyAnnotation("Get role"), nameArg, nsArg), rbacGetHandler(cm, "role"))

	s.AddTool(mcp.NewTool("list_role_bindings", mcp.WithDescription("List RBAC role bindings in a namespace"),
		readOnlyAnnotation("List role bindings"), nsArg, allNsArg), rbacListHandler(cm, "rolebinding"))
	s.AddTool(mcp.NewTool("get_role_binding", mcp.WithDescription("Get an RBAC role binding"),
		readOnlyAnnotation("Get role binding"), nameArg, nsArg), rbacGetHandler(cm, "rolebinding"))

	s.AddTool(mcp.NewTool("list_cluster_roles", mcp.WithDescription("List cluster roles"),
		readOnlyAnnotation("List cluster roles")), rbacListHandler(cm, "clusterrole"))
	s.AddTool(mcp.NewTool("get_cluster_role", mcp.WithDescription("Get a cluster role with its rules"),
		readOnlyAnnotation("Get cluster role"), nameArg), rbacGetHandler(cm, "clusterrole"))

	s.AddTool(mcp.NewTool("list_cluster_role_bindings", mcp.WithDescription("List cluster role bindings"),
		readOnlyAnnotation("List cluster role bindings")), rbacListHandler(cm, "clusterrolebinding"))
	s.AddTool(mcp.NewTool("get_cluster_role_binding", mcp.WithDescription("Get a cluster role binding"),
		readOnlyAnnotation("Get cluster role binding"), nameArg), rbacGetHandler(cm, "clusterrolebinding"))

	s.AddTool(mcp.NewTool("list_service_accounts", mcp.WithDescription("List service accounts in a namespace"),
		readOnlyAnnotation("List service accounts"), nsArg, allNsArg), rbacListHandler(cm, "serviceaccount"))
	s.AddTool(mcp.NewTool("get_service_account", mcp.WithDescription("Get a service account"),
		readOnlyAnnotation("Get service account"), nameArg, nsArg), rbacGetHandler(cm, "serviceaccount"))
}

func rbacListHandler(cm kai.ClusterManager, kind string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_"+kind))
		rbac := cluster.RBAC{}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			rbac.Namespace = ns
		}
		allNamespaces := false
		if all, ok := request.GetArguments()["all_namespaces"].(bool); ok {
			allNamespaces = all
		}

		var (
			result string
			err    error
		)
		switch kind {
		case "role":
			result, err = rbac.ListRoles(ctx, cm, allNamespaces)
		case "rolebinding":
			result, err = rbac.ListRoleBindings(ctx, cm, allNamespaces)
		case "clusterrole":
			result, err = rbac.ListClusterRoles(ctx, cm)
		case "clusterrolebinding":
			result, err = rbac.ListClusterRoleBindings(ctx, cm)
		case "serviceaccount":
			result, err = rbac.ListServiceAccounts(ctx, cm, allNamespaces)
		}
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list %s: %s", kind, err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func rbacGetHandler(cm kai.ClusterManager, kind string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_"+kind))
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		rbac := cluster.RBAC{Name: name}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			rbac.Namespace = ns
		}

		var (
			result string
			err    error
		)
		switch kind {
		case "role":
			result, err = rbac.GetRole(ctx, cm)
		case "rolebinding":
			result, err = rbac.GetRoleBinding(ctx, cm)
		case "clusterrole":
			result, err = rbac.GetClusterRole(ctx, cm)
		case "clusterrolebinding":
			result, err = rbac.GetClusterRoleBinding(ctx, cm)
		case "serviceaccount":
			result, err = rbac.GetServiceAccount(ctx, cm)
		}
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get %s: %s", kind, err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
