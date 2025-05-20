package kai

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ServerInterface defines the contract for an mcp server that can register and handle tools.
// Implementations of this interface can register tool handlers and serve MCP Call requests.
type ServerInterface interface {
	AddTool(mcp.Tool, server.ToolHandlerFunc)
	Serve() error
}

// ClusterManager defines the contract for managing Kubernetes clusters.
type ClusterManager interface {
	GetClient(string) (kubernetes.Interface, error)
	GetCurrentClient() (kubernetes.Interface, error)
	GetCurrentContext() string
	GetCurrentDynamicClient() (dynamic.Interface, error)
	GetCurrentNamespace() string
	GetDynamicClient(string) (dynamic.Interface, error)
	ListClusters() []string
	LoadKubeConfig(string, string) error
	SetCurrentContext(string) error
	SetCurrentNamespace(string)
}

// PodOperator defines the operations needed for pod management
type PodOperator interface {
	Create(ctx context.Context, cm ClusterManager) (string, error)
	Get(ctx context.Context, cm ClusterManager) (string, error)
	List(ctx context.Context, cm ClusterManager, limit int64, labelSelector, fieldSelector string) (string, error)
	Delete(ctx context.Context, cm ClusterManager, force bool) (string, error)
	StreamLogs(ctx context.Context, cm ClusterManager, tailLines int64, previous bool, since *time.Duration) (string, error)
}

// DeploymentOperator defines the operations needed for deployment management
type DeploymentOperator interface {
	Create(ctx context.Context, cm ClusterManager) (string, error)
	Get(ctx context.Context, cm ClusterManager) (string, error)
	Update(ctx context.Context, cm ClusterManager) (string, error)
	List(ctx context.Context, cm ClusterManager, allNamespaces bool, labelSelector string) (string, error)
}

// ServiceOperator defines the operations needed for service management
type ServiceOperator interface {
	Create(ctx context.Context, cm ClusterManager) (string, error)
	Get(ctx context.Context, cm ClusterManager) (string, error)
	Delete(ctx context.Context, cm ClusterManager) (string, error)
	List(ctx context.Context, cm ClusterManager, allNamespaces bool, labelSelector string) (string, error)
}
