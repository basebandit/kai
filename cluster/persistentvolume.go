package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolume represents an operation target for a cluster-scoped PV.
type PersistentVolume struct {
	Name string
}

func (p *PersistentVolume) validate() error {
	if p.Name == "" {
		return fmt.Errorf("persistent volume name is required")
	}
	return nil
}

// List returns all persistent volumes in the cluster.
func (p *PersistentVolume) List(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	pvs, err := client.CoreV1().PersistentVolumes().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	if len(pvs.Items) == 0 {
		return "No persistent volumes found", nil
	}

	return formatPersistentVolumeList(pvs), nil
}

// Get returns details for a single persistent volume.
func (p *PersistentVolume) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	pv, err := client.CoreV1().PersistentVolumes().Get(timeoutCtx, p.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get persistent volume %q: %w", p.Name, err)
	}

	return formatPersistentVolume(pv), nil
}

// Delete removes a persistent volume.
func (p *PersistentVolume) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := client.CoreV1().PersistentVolumes().Delete(timeoutCtx, p.Name, metav1.DeleteOptions{}); err != nil {
		return "", fmt.Errorf("failed to delete persistent volume %q: %w", p.Name, err)
	}

	return fmt.Sprintf("PersistentVolume %q deleted successfully", p.Name), nil
}

func pvCapacity(pv *corev1.PersistentVolume) string {
	if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return "<unknown>"
}

func accessModesToString(modes []corev1.PersistentVolumeAccessMode) string {
	if len(modes) == 0 {
		return "<none>"
	}
	out := make([]string, 0, len(modes))
	for _, m := range modes {
		switch m {
		case corev1.ReadWriteOnce:
			out = append(out, "RWO")
		case corev1.ReadOnlyMany:
			out = append(out, "ROX")
		case corev1.ReadWriteMany:
			out = append(out, "RWX")
		case corev1.ReadWriteOncePod:
			out = append(out, "RWOP")
		default:
			out = append(out, string(m))
		}
	}
	return strings.Join(out, ",")
}

func formatPersistentVolumeList(pvs *corev1.PersistentVolumeList) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Persistent Volumes (%d):\n", len(pvs.Items))
	for i := range pvs.Items {
		pv := pvs.Items[i]
		claim := "<unbound>"
		if pv.Spec.ClaimRef != nil {
			claim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}
		fmt.Fprintf(&sb, "• %s\tcapacity: %s\taccess: %s\treclaim: %s\tstatus: %s\tclaim: %s\tstorageClass: %s\n",
			pv.Name, pvCapacity(&pv), accessModesToString(pv.Spec.AccessModes),
			pv.Spec.PersistentVolumeReclaimPolicy, pv.Status.Phase, claim, pv.Spec.StorageClassName)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatPersistentVolume(pv *corev1.PersistentVolume) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "PersistentVolume: %s\n", pv.Name)
	fmt.Fprintf(&sb, "Capacity: %s\n", pvCapacity(pv))
	fmt.Fprintf(&sb, "Access Modes: %s\n", accessModesToString(pv.Spec.AccessModes))
	fmt.Fprintf(&sb, "Reclaim Policy: %s\n", pv.Spec.PersistentVolumeReclaimPolicy)
	fmt.Fprintf(&sb, "Status: %s\n", pv.Status.Phase)
	fmt.Fprintf(&sb, "Storage Class: %s\n", pv.Spec.StorageClassName)
	if pv.Spec.ClaimRef != nil {
		fmt.Fprintf(&sb, "Claim: %s/%s\n", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
	}
	if pv.Spec.VolumeMode != nil {
		fmt.Fprintf(&sb, "Volume Mode: %s\n", *pv.Spec.VolumeMode)
	}
	fmt.Fprintf(&sb, "Age: %s\n", formatDuration(time.Since(pv.CreationTimestamp.Time)))
	return strings.TrimRight(sb.String(), "\n")
}
