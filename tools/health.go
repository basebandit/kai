package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterHealthTools registers cluster health and metrics tools.
func RegisterHealthTools(s kai.ServerInterface, cm kai.ClusterManager) {
	clusterHealthTool := mcp.NewTool("cluster_health",
		mcp.WithDescription("Summarize cluster health: node readiness and pod phase distribution"),
		readOnlyAnnotation("Cluster health"),
	)
	s.AddTool(clusterHealthTool, clusterHealthHandler(cm))

	nodeMetricsTool := mcp.NewTool("node_metrics",
		mcp.WithDescription("Show CPU and memory usage per node (requires metrics-server)"),
		readOnlyAnnotation("Node metrics"),
	)
	s.AddTool(nodeMetricsTool, nodeMetricsHandler(cm))

	podMetricsTool := mcp.NewTool("pod_metrics",
		mcp.WithDescription("Show CPU and memory usage per pod (requires metrics-server)"),
		readOnlyAnnotation("Pod metrics"),
		mcp.WithString("namespace",
			mcp.Description("Namespace to report (defaults to current namespace)"),
		),
		mcp.WithBoolean("all_namespaces",
			mcp.Description("Report pods across all namespaces"),
		),
	)
	s.AddTool(podMetricsTool, podMetricsHandler(cm))
}

func clusterHealthHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "cluster_health"))
		health := cluster.Health{}
		result, err := health.Cluster(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get cluster health: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func nodeMetricsHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "node_metrics"))
		health := cluster.Health{}
		result, err := health.NodeMetrics(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get node metrics: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func podMetricsHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "pod_metrics"))
		namespace := ""
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			namespace = ns
		}
		allNamespaces := false
		if all, ok := request.GetArguments()["all_namespaces"].(bool); ok {
			allNamespaces = all
		}

		health := cluster.Health{}
		result, err := health.PodMetrics(ctx, cm, namespace, allNamespaces)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get pod metrics: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
