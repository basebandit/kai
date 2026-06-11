package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Health reports overall cluster status and resource usage.
type Health struct{}

var (
	nodeMetricsGVR = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	podMetricsGVR  = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
)

// Cluster summarises node readiness and pod phase distribution.
func (h *Health) Cluster(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	nodes, err := client.CoreV1().Nodes().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	pods, err := client.CoreV1().Pods("").List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	var ready, notReady, unschedulable int
	for i := range nodes.Items {
		node := nodes.Items[i]
		if nodeReadyStatus(&node) == "Ready" {
			ready++
		} else {
			notReady++
		}
		if node.Spec.Unschedulable {
			unschedulable++
		}
	}

	phases := map[corev1.PodPhase]int{}
	for i := range pods.Items {
		phases[pods.Items[i].Status.Phase]++
	}

	var sb strings.Builder
	sb.WriteString("Cluster Health\n")
	fmt.Fprintf(&sb, "Nodes: %d total, %d ready, %d not ready", len(nodes.Items), ready, notReady)
	if unschedulable > 0 {
		fmt.Fprintf(&sb, ", %d unschedulable", unschedulable)
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Pods: %d total\n", len(pods.Items))

	phaseOrder := []corev1.PodPhase{corev1.PodRunning, corev1.PodPending, corev1.PodSucceeded, corev1.PodFailed, corev1.PodUnknown}
	for _, phase := range phaseOrder {
		if count, ok := phases[phase]; ok {
			fmt.Fprintf(&sb, "  %s: %d\n", phase, count)
			delete(phases, phase)
		}
	}
	for phase, count := range phases {
		fmt.Fprintf(&sb, "  %s: %d\n", phase, count)
	}

	overall := "Healthy"
	if notReady > 0 || phases[corev1.PodFailed] > 0 {
		overall = "Degraded"
	}
	fmt.Fprintf(&sb, "Overall: %s", overall)

	return strings.TrimRight(sb.String(), "\n"), nil
}

// NodeMetrics reports CPU/memory usage per node via the metrics API.
func (h *Health) NodeMetrics(ctx context.Context, cm kai.ClusterManager) (string, error) {
	return h.resourceMetrics(ctx, cm, nodeMetricsGVR, "", "Node metrics")
}

// PodMetrics reports CPU/memory usage per pod via the metrics API.
func (h *Health) PodMetrics(ctx context.Context, cm kai.ClusterManager, namespace string, allNamespaces bool) (string, error) {
	ns := ""
	if !allNamespaces {
		ns = namespace
		if ns == "" {
			ns = cm.GetCurrentNamespace()
		}
	}
	return h.resourceMetrics(ctx, cm, podMetricsGVR, ns, "Pod metrics")
}

func (h *Health) resourceMetrics(ctx context.Context, cm kai.ClusterManager, gvr schema.GroupVersionResource, namespace, title string) (string, error) {
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = dyn.Resource(gvr).Namespace(namespace).List(timeoutCtx, metav1.ListOptions{})
	} else {
		list, err = dyn.Resource(gvr).List(timeoutCtx, metav1.ListOptions{})
	}
	if err != nil {
		// metrics-server may not be installed; degrade gracefully.
		return fmt.Sprintf("%s unavailable: %v\n(Is metrics-server installed in the cluster?)", title, err), nil
	}

	if len(list.Items) == 0 {
		return fmt.Sprintf("No %s available", strings.ToLower(title)), nil
	}

	type usage struct{ name, ns, cpu, mem string }
	rows := make([]usage, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		u := usage{name: item.GetName(), ns: item.GetNamespace()}
		if c, found, _ := unstructured.NestedString(item.Object, "usage", "cpu"); found {
			u.cpu = c
		}
		if m, found, _ := unstructured.NestedString(item.Object, "usage", "memory"); found {
			u.mem = m
		}
		rows = append(rows, u)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%d):\n", title, len(rows))
	for _, r := range rows {
		if r.ns != "" {
			fmt.Fprintf(&sb, "• %s/%s\tcpu: %s\tmemory: %s\n", r.ns, r.name, r.cpu, r.mem)
		} else {
			fmt.Fprintf(&sb, "• %s\tcpu: %s\tmemory: %s\n", r.name, r.cpu, r.mem)
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
