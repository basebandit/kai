package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// Node represents an operation target for a cluster node.
type Node struct {
	Name string
}

func (n *Node) validate() error {
	if n.Name == "" {
		return fmt.Errorf("node name is required")
	}
	return nil
}

// List returns a summary of all nodes in the cluster.
func (n *Node) List(ctx context.Context, cm kai.ClusterManager) (string, error) {
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

	if len(nodes.Items) == 0 {
		return "No nodes found", nil
	}

	return formatNodeList(nodes), nil
}

// Get returns detailed information about a single node.
func (n *Node) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if err := n.validate(); err != nil {
		return "", err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(timeoutCtx, n.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %q: %w", n.Name, err)
	}

	return formatNode(node), nil
}

// Cordon marks the node unschedulable.
func (n *Node) Cordon(ctx context.Context, cm kai.ClusterManager) (string, error) {
	return n.setSchedulable(ctx, cm, true)
}

// Uncordon marks the node schedulable again.
func (n *Node) Uncordon(ctx context.Context, cm kai.ClusterManager) (string, error) {
	return n.setSchedulable(ctx, cm, false)
}

func (n *Node) setSchedulable(ctx context.Context, cm kai.ClusterManager, unschedulable bool) (string, error) {
	if err := n.validate(); err != nil {
		return "", err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(timeoutCtx, n.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node %q: %w", n.Name, err)
	}

	verb := "cordoned"
	if !unschedulable {
		verb = "uncordoned"
	}

	if node.Spec.Unschedulable == unschedulable {
		return fmt.Sprintf("Node %q already %s", n.Name, verb), nil
	}

	node.Spec.Unschedulable = unschedulable
	if _, err := client.CoreV1().Nodes().Update(timeoutCtx, node, metav1.UpdateOptions{}); err != nil {
		return "", fmt.Errorf("failed to update node %q: %w", n.Name, err)
	}

	slog.Info("node schedulability changed", slog.String("node", n.Name), slog.Bool("unschedulable", unschedulable))
	return fmt.Sprintf("Node %q %s successfully", n.Name, verb), nil
}

// Drain cordons the node and evicts its pods. DaemonSet-managed and
// mirror (static) pods are skipped, matching kubectl drain behaviour.
func (n *Node) Drain(ctx context.Context, cm kai.ClusterManager, ignoreDaemonSets, deleteLocalData bool, gracePeriod int64) (string, error) {
	if err := n.validate(); err != nil {
		return "", err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	if _, err := n.Cordon(ctx, cm); err != nil {
		return "", err
	}

	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", n.Name).String(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods on node %q: %w", n.Name, err)
	}

	var (
		evicted []string
		skipped []string
		failed  []string
	)

	for i := range pods.Items {
		pod := pods.Items[i]
		if reason, skip := shouldSkipPod(&pod, ignoreDaemonSets, deleteLocalData); skip {
			skipped = append(skipped, fmt.Sprintf("%s/%s (%s)", pod.Namespace, pod.Name, reason))
			continue
		}

		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace},
		}
		if gracePeriod >= 0 {
			eviction.DeleteOptions = &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
		}

		if err := client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction); err != nil {
			failed = append(failed, fmt.Sprintf("%s/%s: %v", pod.Namespace, pod.Name, err))
			continue
		}
		evicted = append(evicted, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Node %q drained (cordoned).\n", n.Name)
	fmt.Fprintf(&sb, "Evicted %d pod(s)", len(evicted))
	if len(evicted) > 0 {
		sb.WriteString(":\n- " + strings.Join(evicted, "\n- "))
	}
	sb.WriteString("\n")
	if len(skipped) > 0 {
		fmt.Fprintf(&sb, "Skipped %d pod(s):\n- %s\n", len(skipped), strings.Join(skipped, "\n- "))
	}
	if len(failed) > 0 {
		fmt.Fprintf(&sb, "Failed to evict %d pod(s):\n- %s\n", len(failed), strings.Join(failed, "\n- "))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func shouldSkipPod(pod *corev1.Pod, ignoreDaemonSets, deleteLocalData bool) (string, bool) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			if ignoreDaemonSets {
				return "DaemonSet-managed", true
			}
			return "", false
		}
	}
	// Mirror (static) pods cannot be evicted.
	if _, ok := pod.Annotations[corev1.MirrorPodAnnotationKey]; ok {
		return "mirror pod", true
	}
	if !deleteLocalData {
		for _, vol := range pod.Spec.Volumes {
			if vol.EmptyDir != nil {
				return "uses emptyDir (set delete_local_data=true to evict)", true
			}
		}
	}
	return "", false
}

func nodeReadyStatus(node *corev1.Node) string {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodeRoles(node *corev1.Node) string {
	var roles []string
	for label := range node.Labels {
		if role, ok := strings.CutPrefix(label, "node-role.kubernetes.io/"); ok {
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

func formatNodeList(nodes *corev1.NodeList) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Nodes (%d):\n", len(nodes.Items))
	for i := range nodes.Items {
		node := nodes.Items[i]
		status := nodeReadyStatus(&node)
		if node.Spec.Unschedulable {
			status += ",SchedulingDisabled"
		}
		age := formatDuration(time.Since(node.CreationTimestamp.Time))
		fmt.Fprintf(&sb, "• %s\tstatus: %s\troles: %s\tversion: %s\tage: %s\n",
			node.Name, status, nodeRoles(&node), node.Status.NodeInfo.KubeletVersion, age)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatNode(node *corev1.Node) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Node: %s\n", node.Name)
	status := nodeReadyStatus(node)
	if node.Spec.Unschedulable {
		status += ",SchedulingDisabled"
	}
	fmt.Fprintf(&sb, "Status: %s\n", status)
	fmt.Fprintf(&sb, "Roles: %s\n", nodeRoles(node))
	fmt.Fprintf(&sb, "Kubelet Version: %s\n", node.Status.NodeInfo.KubeletVersion)
	fmt.Fprintf(&sb, "OS Image: %s\n", node.Status.NodeInfo.OSImage)
	fmt.Fprintf(&sb, "Kernel: %s\n", node.Status.NodeInfo.KernelVersion)
	fmt.Fprintf(&sb, "Container Runtime: %s\n", node.Status.NodeInfo.ContainerRuntimeVersion)
	fmt.Fprintf(&sb, "Age: %s\n", formatDuration(time.Since(node.CreationTimestamp.Time)))

	for _, addr := range node.Status.Addresses {
		fmt.Fprintf(&sb, "%s: %s\n", addr.Type, addr.Address)
	}

	if cpu, ok := node.Status.Capacity[corev1.ResourceCPU]; ok {
		fmt.Fprintf(&sb, "Capacity: cpu=%s, memory=%s, pods=%s\n",
			cpu.String(), node.Status.Capacity.Memory().String(), node.Status.Capacity.Pods().String())
	}

	sb.WriteString("Conditions:\n")
	for _, cond := range node.Status.Conditions {
		fmt.Fprintf(&sb, "  %s: %s (%s)\n", cond.Type, cond.Status, cond.Reason)
	}

	return strings.TrimRight(sb.String(), "\n")
}
