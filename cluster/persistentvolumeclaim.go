package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaim represents an operation target for a namespaced PVC.
type PersistentVolumeClaim struct {
	Name             string
	Namespace        string
	StorageClassName string
	AccessModes      []string
	Storage          string
	VolumeMode       string
	Labels           map[string]interface{}
	Annotations      map[string]interface{}
}

func (p *PersistentVolumeClaim) namespace(cm kai.ClusterManager) string {
	if p.Namespace != "" {
		return p.Namespace
	}
	return cm.GetCurrentNamespace()
}

// Create provisions a new PersistentVolumeClaim.
func (p *PersistentVolumeClaim) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if p.Name == "" {
		return "", fmt.Errorf("persistent volume claim name is required")
	}
	if p.Storage == "" {
		return "", fmt.Errorf("storage request is required (e.g. '1Gi')")
	}

	quantity, err := resource.ParseQuantity(p.Storage)
	if err != nil {
		return "", fmt.Errorf("invalid storage quantity %q: %w", p.Storage, err)
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	ns := p.namespace(cm)

	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	if len(p.AccessModes) > 0 {
		accessModes = accessModes[:0]
		for _, m := range p.AccessModes {
			accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(m))
		}
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: p.Name, Namespace: ns},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: quantity},
			},
		},
	}

	if p.StorageClassName != "" {
		pvc.Spec.StorageClassName = &p.StorageClassName
	}
	if p.VolumeMode != "" {
		mode := corev1.PersistentVolumeMode(p.VolumeMode)
		pvc.Spec.VolumeMode = &mode
	}
	if labels := convertToStringMap(p.Labels); len(labels) > 0 {
		pvc.ObjectMeta.Labels = labels
	}
	if annotations := convertToStringMap(p.Annotations); len(annotations) > 0 {
		pvc.ObjectMeta.Annotations = annotations
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	created, err := client.CoreV1().PersistentVolumeClaims(ns).Create(timeoutCtx, pvc, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create persistent volume claim: %w", err)
	}

	return fmt.Sprintf("PersistentVolumeClaim %q created successfully in namespace %q", created.Name, ns), nil
}

// List returns PVCs in the requested namespace.
func (p *PersistentVolumeClaim) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	ns := ""
	if !allNamespaces {
		ns = p.namespace(cm)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	pvcs, err := client.CoreV1().PersistentVolumeClaims(ns).List(timeoutCtx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	if len(pvcs.Items) == 0 {
		return "No persistent volume claims found", nil
	}

	return formatPVCList(pvcs, allNamespaces), nil
}

// Get returns details for a single PVC.
func (p *PersistentVolumeClaim) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if p.Name == "" {
		return "", fmt.Errorf("persistent volume claim name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	ns := p.namespace(cm)

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	pvc, err := client.CoreV1().PersistentVolumeClaims(ns).Get(timeoutCtx, p.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get persistent volume claim %q: %w", p.Name, err)
	}

	return formatPVC(pvc), nil
}

// Delete removes a PVC.
func (p *PersistentVolumeClaim) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if p.Name == "" {
		return "", fmt.Errorf("persistent volume claim name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	ns := p.namespace(cm)

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := client.CoreV1().PersistentVolumeClaims(ns).Delete(timeoutCtx, p.Name, metav1.DeleteOptions{}); err != nil {
		return "", fmt.Errorf("failed to delete persistent volume claim %q: %w", p.Name, err)
	}

	return fmt.Sprintf("PersistentVolumeClaim %q deleted successfully from namespace %q", p.Name, ns), nil
}

func pvcCapacity(pvc *corev1.PersistentVolumeClaim) string {
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return "<unset>"
}

func formatPVCList(pvcs *corev1.PersistentVolumeClaimList, allNamespaces bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Persistent Volume Claims (%d):\n", len(pvcs.Items))
	for i := range pvcs.Items {
		pvc := pvcs.Items[i]
		sc := "<none>"
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}
		line := fmt.Sprintf("• %s", pvc.Name)
		if allNamespaces {
			line = fmt.Sprintf("• %s/%s", pvc.Namespace, pvc.Name)
		}
		fmt.Fprintf(&sb, "%s\tstatus: %s\tvolume: %s\tcapacity: %s\taccess: %s\tstorageClass: %s\n",
			line, pvc.Status.Phase, pvc.Spec.VolumeName, pvcCapacity(&pvc),
			accessModesToString(pvc.Spec.AccessModes), sc)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatPVC(pvc *corev1.PersistentVolumeClaim) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "PersistentVolumeClaim: %s\n", pvc.Name)
	fmt.Fprintf(&sb, "Namespace: %s\n", pvc.Namespace)
	fmt.Fprintf(&sb, "Status: %s\n", pvc.Status.Phase)
	fmt.Fprintf(&sb, "Capacity: %s\n", pvcCapacity(pvc))
	fmt.Fprintf(&sb, "Access Modes: %s\n", accessModesToString(pvc.Spec.AccessModes))
	if pvc.Spec.StorageClassName != nil {
		fmt.Fprintf(&sb, "Storage Class: %s\n", *pvc.Spec.StorageClassName)
	}
	if pvc.Spec.VolumeName != "" {
		fmt.Fprintf(&sb, "Bound Volume: %s\n", pvc.Spec.VolumeName)
	}
	if pvc.Spec.VolumeMode != nil {
		fmt.Fprintf(&sb, "Volume Mode: %s\n", *pvc.Spec.VolumeMode)
	}
	fmt.Fprintf(&sb, "Age: %s\n", formatDuration(time.Since(pvc.CreationTimestamp.Time)))
	return strings.TrimRight(sb.String(), "\n")
}
