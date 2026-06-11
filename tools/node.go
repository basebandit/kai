package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

const errMissingNode = "Required parameter 'name' (node name) is missing"

// RegisterNodeTools registers node management tools.
func RegisterNodeTools(s kai.ServerInterface, cm kai.ClusterManager) {
	listNodesTool := mcp.NewTool("list_nodes",
		mcp.WithDescription("List all nodes in the cluster with status, roles and version"),
		readOnlyAnnotation("List nodes"),
	)
	s.AddTool(listNodesTool, listNodesHandler(cm))

	getNodeTool := mcp.NewTool("get_node",
		mcp.WithDescription("Get detailed information about a specific node"),
		readOnlyAnnotation("Get node"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the node")),
	)
	s.AddTool(getNodeTool, getNodeHandler(cm))

	cordonNodeTool := mcp.NewTool("cordon_node",
		mcp.WithDescription("Mark a node as unschedulable so no new pods are scheduled onto it"),
		idempotentMutationAnnotation("Cordon node"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the node")),
	)
	s.AddTool(cordonNodeTool, cordonNodeHandler(cm, false))

	uncordonNodeTool := mcp.NewTool("uncordon_node",
		mcp.WithDescription("Mark a node as schedulable again"),
		idempotentMutationAnnotation("Uncordon node"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the node")),
	)
	s.AddTool(uncordonNodeTool, cordonNodeHandler(cm, true))

	drainNodeTool := mcp.NewTool("drain_node",
		mcp.WithDescription("Cordon a node and evict its pods (DaemonSet and mirror pods are skipped)"),
		destructiveAnnotation("Drain node"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the node")),
		mcp.WithBoolean("ignore_daemonsets",
			mcp.Description("Skip DaemonSet-managed pods instead of failing (default true)"),
		),
		mcp.WithBoolean("delete_local_data",
			mcp.Description("Evict pods using emptyDir volumes, losing their local data (default false)"),
		),
		mcp.WithNumber("grace_period",
			mcp.Description("Eviction grace period in seconds (-1 uses the pod default)"),
		),
	)
	s.AddTool(drainNodeTool, drainNodeHandler(cm))
}

func nodeNameFromRequest(request mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	nameArg, ok := request.GetArguments()["name"]
	if !ok || nameArg == nil {
		return "", mcp.NewToolResultText(errMissingNode)
	}
	name, ok := nameArg.(string)
	if !ok || name == "" {
		return "", mcp.NewToolResultText(errEmptyName)
	}
	return name, nil
}

func listNodesHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_nodes"))
		node := cluster.Node{}
		result, err := node.List(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list nodes: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getNodeHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "get_node"))
		name, errResult := nodeNameFromRequest(request)
		if errResult != nil {
			return errResult, nil
		}
		node := cluster.Node{Name: name}
		result, err := node.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get node: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func cordonNodeHandler(cm kai.ClusterManager, uncordon bool) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := nodeNameFromRequest(request)
		if errResult != nil {
			return errResult, nil
		}
		node := cluster.Node{Name: name}

		var (
			result string
			err    error
		)
		if uncordon {
			result, err = node.Uncordon(ctx, cm)
		} else {
			result, err = node.Cordon(ctx, cm)
		}
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to update node: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func drainNodeHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "drain_node"))
		name, errResult := nodeNameFromRequest(request)
		if errResult != nil {
			return errResult, nil
		}
		node := cluster.Node{Name: name}

		ignoreDaemonSets := true
		if v, ok := request.GetArguments()["ignore_daemonsets"].(bool); ok {
			ignoreDaemonSets = v
		}
		deleteLocalData := false
		if v, ok := request.GetArguments()["delete_local_data"].(bool); ok {
			deleteLocalData = v
		}
		gracePeriod := int64(-1)
		if v, ok := request.GetArguments()["grace_period"].(float64); ok {
			gracePeriod = int64(v)
		}

		result, err := node.Drain(ctx, cm, ignoreDaemonSets, deleteLocalData, gracePeriod)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to drain node: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
