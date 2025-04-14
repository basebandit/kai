package kai

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ServerInterface defines the contract for an mcp server that can register and handle tools.
// Implementations of this interface can register tool handlers and serve MCP Call requests.
type ServerInterface interface {
	AddTool(mcp.Tool, server.ToolHandlerFunc)
	Serve() error
}

// ClusterManagerInterface defines the contract for managing Kubernetes clusters.
// It provides methods for interacting with cluster resources like pods and deployments,
// as well as managing cluster contexts and configurations.
type ClusterManagerInterface interface {
	// DeletePod(context.Context, string, string, bool) (string, error)
	DeleteResource(ctx context.Context, force bool, resourceType, name, namespace, group, version string) error
	GetClient(string) (kubernetes.Interface, error)
	GetCurrentClient() (kubernetes.Interface, error)
	GetCurrentContext() string
	GetCurrentDynamicClient() (dynamic.Interface, error)
	GetCurrentNamespace() string
	GetDynamicClient(string) (dynamic.Interface, error)
	// GetPod(context.Context, string, string) (string, error)
	GetResource(ctx context.Context, resourceType, name, namespace, group, version string) (*unstructured.Unstructured, error)
	ListClusters() []string
	ListDeployments(context.Context, bool, string, string) (string, error)
	CreateDeployment(context.Context, DeploymentParams) (string, error)
	ListResources(ctx context.Context, limit int64, resourceType, namespace, labelSelector, fieldSelector, group, version string) (*unstructured.UnstructuredList, error)
	// ListPods(context.Context, int64, string, string, string) (string, error)
	LoadKubeConfig(string, string) error
	SetCurrentContext(string) error
	SetCurrentNamespace(string)
	StreamPodLogs(context.Context, int64, bool, *time.Duration, string, string, string) (string, error)
}
