package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

// StorageClass represents an operation target for a cluster-scoped storage class.
type StorageClass struct {
	Name string
}

// List returns all storage classes in the cluster.
func (s *StorageClass) List(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	classes, err := client.StorageV1().StorageClasses().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list storage classes: %w", err)
	}

	if len(classes.Items) == 0 {
		return "No storage classes found", nil
	}

	return formatStorageClassList(classes), nil
}

// Get returns details for a single storage class.
func (s *StorageClass) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if s.Name == "" {
		return "", fmt.Errorf("storage class name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	sc, err := client.StorageV1().StorageClasses().Get(timeoutCtx, s.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get storage class %q: %w", s.Name, err)
	}

	return formatStorageClass(sc), nil
}

func isDefaultStorageClass(sc *storagev1.StorageClass) bool {
	return sc.Annotations[defaultStorageClassAnnotation] == "true"
}

func formatStorageClassList(classes *storagev1.StorageClassList) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Storage Classes (%d):\n", len(classes.Items))
	for i := range classes.Items {
		sc := classes.Items[i]
		name := sc.Name
		if isDefaultStorageClass(&sc) {
			name += " (default)"
		}
		reclaim := ""
		if sc.ReclaimPolicy != nil {
			reclaim = string(*sc.ReclaimPolicy)
		}
		binding := ""
		if sc.VolumeBindingMode != nil {
			binding = string(*sc.VolumeBindingMode)
		}
		fmt.Fprintf(&sb, "• %s\tprovisioner: %s\treclaim: %s\tbinding: %s\n",
			name, sc.Provisioner, reclaim, binding)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatStorageClass(sc *storagev1.StorageClass) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "StorageClass: %s\n", sc.Name)
	fmt.Fprintf(&sb, "Default: %t\n", isDefaultStorageClass(sc))
	fmt.Fprintf(&sb, "Provisioner: %s\n", sc.Provisioner)
	if sc.ReclaimPolicy != nil {
		fmt.Fprintf(&sb, "Reclaim Policy: %s\n", *sc.ReclaimPolicy)
	}
	if sc.VolumeBindingMode != nil {
		fmt.Fprintf(&sb, "Volume Binding Mode: %s\n", *sc.VolumeBindingMode)
	}
	if sc.AllowVolumeExpansion != nil {
		fmt.Fprintf(&sb, "Allow Volume Expansion: %t\n", *sc.AllowVolumeExpansion)
	}
	if len(sc.Parameters) > 0 {
		sb.WriteString("Parameters:\n")
		for k, v := range sc.Parameters {
			fmt.Fprintf(&sb, "  %s: %s\n", k, v)
		}
	}
	fmt.Fprintf(&sb, "Age: %s\n", formatDuration(time.Since(sc.CreationTimestamp.Time)))
	return strings.TrimRight(sb.String(), "\n")
}
