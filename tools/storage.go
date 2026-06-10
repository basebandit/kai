package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterStorageTools registers persistent volume, PVC and storage class tools.
func RegisterStorageTools(s kai.ServerInterface, cm kai.ClusterManager) {
	s.AddTool(mcp.NewTool("list_persistent_volumes",
		mcp.WithDescription("List all persistent volumes (cluster-scoped)"),
		readOnlyAnnotation("List persistent volumes"),
	), listPVHandler(cm))

	s.AddTool(mcp.NewTool("get_persistent_volume",
		mcp.WithDescription("Get details about a specific persistent volume"),
		readOnlyAnnotation("Get persistent volume"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the persistent volume")),
	), getPVHandler(cm))

	s.AddTool(mcp.NewTool("delete_persistent_volume",
		mcp.WithDescription("Delete a persistent volume"),
		destructiveAnnotation("Delete persistent volume"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the persistent volume")),
	), deletePVHandler(cm))

	s.AddTool(mcp.NewTool("create_persistent_volume_claim",
		mcp.WithDescription("Create a persistent volume claim"),
		creationAnnotation("Create PVC"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the PVC")),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current)")),
		mcp.WithString("storage", mcp.Required(), mcp.Description("Requested storage, e.g. '1Gi'")),
		mcp.WithString("storage_class", mcp.Description("Storage class name")),
		mcp.WithString("volume_mode", mcp.Description("Volume mode: Filesystem (default) or Block")),
		mcp.WithArray("access_modes", mcp.Description("Access modes (ReadWriteOnce, ReadOnlyMany, ReadWriteMany, ReadWriteOncePod)")),
	), createPVCHandler(cm))

	s.AddTool(mcp.NewTool("list_persistent_volume_claims",
		mcp.WithDescription("List persistent volume claims in a namespace"),
		readOnlyAnnotation("List PVCs"),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current)")),
		mcp.WithBoolean("all_namespaces", mcp.Description("List across all namespaces")),
		mcp.WithString("label_selector", mcp.Description("Label selector to filter PVCs")),
	), listPVCHandler(cm))

	s.AddTool(mcp.NewTool("get_persistent_volume_claim",
		mcp.WithDescription("Get details about a specific persistent volume claim"),
		readOnlyAnnotation("Get PVC"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the PVC")),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current)")),
	), getPVCHandler(cm))

	s.AddTool(mcp.NewTool("delete_persistent_volume_claim",
		mcp.WithDescription("Delete a persistent volume claim"),
		destructiveAnnotation("Delete PVC"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the PVC")),
		mcp.WithString("namespace", mcp.Description("Namespace (defaults to current)")),
	), deletePVCHandler(cm))

	s.AddTool(mcp.NewTool("list_storage_classes",
		mcp.WithDescription("List all storage classes in the cluster"),
		readOnlyAnnotation("List storage classes"),
	), listStorageClassHandler(cm))

	s.AddTool(mcp.NewTool("get_storage_class",
		mcp.WithDescription("Get details about a specific storage class"),
		readOnlyAnnotation("Get storage class"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the storage class")),
	), getStorageClassHandler(cm))
}

func requireName(request mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	nameArg, ok := request.GetArguments()["name"]
	if !ok || nameArg == nil {
		return "", mcp.NewToolResultText(errMissingName)
	}
	name, ok := nameArg.(string)
	if !ok || name == "" {
		return "", mcp.NewToolResultText(errEmptyName)
	}
	return name, nil
}

func listPVHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_persistent_volumes"))
		pv := cluster.PersistentVolume{}
		result, err := pv.List(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list persistent volumes: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getPVHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		pv := cluster.PersistentVolume{Name: name}
		result, err := pv.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get persistent volume: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func deletePVHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		pv := cluster.PersistentVolume{Name: name}
		result, err := pv.Delete(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete persistent volume: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func createPVCHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "create_persistent_volume_claim"))
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		pvc := cluster.PersistentVolumeClaim{Name: name}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			pvc.Namespace = ns
		}
		if storage, ok := request.GetArguments()["storage"].(string); ok {
			pvc.Storage = storage
		}
		if sc, ok := request.GetArguments()["storage_class"].(string); ok {
			pvc.StorageClassName = sc
		}
		if vm, ok := request.GetArguments()["volume_mode"].(string); ok {
			pvc.VolumeMode = vm
		}
		if modes, ok := request.GetArguments()["access_modes"].([]interface{}); ok {
			for _, m := range modes {
				if s, ok := m.(string); ok {
					pvc.AccessModes = append(pvc.AccessModes, s)
				}
			}
		}
		result, err := pvc.Create(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to create PVC: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func listPVCHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_persistent_volume_claims"))
		pvc := cluster.PersistentVolumeClaim{}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			pvc.Namespace = ns
		}
		allNamespaces := false
		if all, ok := request.GetArguments()["all_namespaces"].(bool); ok {
			allNamespaces = all
		}
		labelSelector := ""
		if ls, ok := request.GetArguments()["label_selector"].(string); ok {
			labelSelector = ls
		}
		result, err := pvc.List(ctx, cm, allNamespaces, labelSelector)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list PVCs: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getPVCHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		pvc := cluster.PersistentVolumeClaim{Name: name}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			pvc.Namespace = ns
		}
		result, err := pvc.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get PVC: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func deletePVCHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		pvc := cluster.PersistentVolumeClaim{Name: name}
		if ns, ok := request.GetArguments()["namespace"].(string); ok {
			pvc.Namespace = ns
		}
		result, err := pvc.Delete(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to delete PVC: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func listStorageClassHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool invoked", slog.String("tool", "list_storage_classes"))
		sc := cluster.StorageClass{}
		result, err := sc.List(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to list storage classes: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func getStorageClassHandler(cm kai.ClusterManager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, errResult := requireName(request)
		if errResult != nil {
			return errResult, nil
		}
		sc := cluster.StorageClass{Name: name}
		result, err := sc.Get(ctx, cm)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Failed to get storage class: %s", err.Error())), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
