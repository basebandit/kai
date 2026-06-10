package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RBAC provides read access to RBAC resources. Kind selects the resource:
// "role", "rolebinding", "clusterrole", "clusterrolebinding" or
// "serviceaccount". Roles, RoleBindings and ServiceAccounts are namespaced.
type RBAC struct {
	Name      string
	Namespace string
}

func (r *RBAC) namespace(cm kai.ClusterManager) string {
	if r.Namespace != "" {
		return r.Namespace
	}
	return cm.GetCurrentNamespace()
}

// ---- Roles ----

func (r *RBAC) ListRoles(ctx context.Context, cm kai.ClusterManager, allNamespaces bool) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := ""
	if !allNamespaces {
		ns = r.namespace(cm)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	roles, err := client.RbacV1().Roles(ns).List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list roles: %w", err)
	}
	if len(roles.Items) == 0 {
		return "No roles found", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Roles (%d):\n", len(roles.Items))
	for i := range roles.Items {
		role := roles.Items[i]
		name := role.Name
		if allNamespaces {
			name = fmt.Sprintf("%s/%s", role.Namespace, role.Name)
		}
		fmt.Fprintf(&sb, "• %s\trules: %d\tage: %s\n", name, len(role.Rules), formatDuration(time.Since(role.CreationTimestamp.Time)))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (r *RBAC) GetRole(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("role name is required")
	}
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := r.namespace(cm)
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	role, err := client.RbacV1().Roles(ns).Get(timeoutCtx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get role %q: %w", r.Name, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Role: %s\nNamespace: %s\n", role.Name, role.Namespace)
	sb.WriteString(formatPolicyRules(role.Rules))
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ---- ClusterRoles ----

func (r *RBAC) ListClusterRoles(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	roles, err := client.RbacV1().ClusterRoles().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list cluster roles: %w", err)
	}
	if len(roles.Items) == 0 {
		return "No cluster roles found", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ClusterRoles (%d):\n", len(roles.Items))
	for i := range roles.Items {
		role := roles.Items[i]
		fmt.Fprintf(&sb, "• %s\trules: %d\tage: %s\n", role.Name, len(role.Rules), formatDuration(time.Since(role.CreationTimestamp.Time)))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (r *RBAC) GetClusterRole(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("cluster role name is required")
	}
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	role, err := client.RbacV1().ClusterRoles().Get(timeoutCtx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster role %q: %w", r.Name, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ClusterRole: %s\n", role.Name)
	sb.WriteString(formatPolicyRules(role.Rules))
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ---- RoleBindings ----

func (r *RBAC) ListRoleBindings(ctx context.Context, cm kai.ClusterManager, allNamespaces bool) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := ""
	if !allNamespaces {
		ns = r.namespace(cm)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	bindings, err := client.RbacV1().RoleBindings(ns).List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list role bindings: %w", err)
	}
	if len(bindings.Items) == 0 {
		return "No role bindings found", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "RoleBindings (%d):\n", len(bindings.Items))
	for i := range bindings.Items {
		b := bindings.Items[i]
		name := b.Name
		if allNamespaces {
			name = fmt.Sprintf("%s/%s", b.Namespace, b.Name)
		}
		fmt.Fprintf(&sb, "• %s\trole: %s/%s\tsubjects: %s\n", name, b.RoleRef.Kind, b.RoleRef.Name, formatSubjects(b.Subjects))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (r *RBAC) GetRoleBinding(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("role binding name is required")
	}
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := r.namespace(cm)
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	b, err := client.RbacV1().RoleBindings(ns).Get(timeoutCtx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get role binding %q: %w", r.Name, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "RoleBinding: %s\nNamespace: %s\nRole: %s/%s\nSubjects: %s\n",
		b.Name, b.Namespace, b.RoleRef.Kind, b.RoleRef.Name, formatSubjects(b.Subjects))
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ---- ClusterRoleBindings ----

func (r *RBAC) ListClusterRoleBindings(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	bindings, err := client.RbacV1().ClusterRoleBindings().List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list cluster role bindings: %w", err)
	}
	if len(bindings.Items) == 0 {
		return "No cluster role bindings found", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ClusterRoleBindings (%d):\n", len(bindings.Items))
	for i := range bindings.Items {
		b := bindings.Items[i]
		fmt.Fprintf(&sb, "• %s\trole: %s/%s\tsubjects: %s\n", b.Name, b.RoleRef.Kind, b.RoleRef.Name, formatSubjects(b.Subjects))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (r *RBAC) GetClusterRoleBinding(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("cluster role binding name is required")
	}
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	b, err := client.RbacV1().ClusterRoleBindings().Get(timeoutCtx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster role binding %q: %w", r.Name, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ClusterRoleBinding: %s\nRole: %s/%s\nSubjects: %s\n",
		b.Name, b.RoleRef.Kind, b.RoleRef.Name, formatSubjects(b.Subjects))
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ---- ServiceAccounts ----

func (r *RBAC) ListServiceAccounts(ctx context.Context, cm kai.ClusterManager, allNamespaces bool) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := ""
	if !allNamespaces {
		ns = r.namespace(cm)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	sas, err := client.CoreV1().ServiceAccounts(ns).List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list service accounts: %w", err)
	}
	if len(sas.Items) == 0 {
		return "No service accounts found", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ServiceAccounts (%d):\n", len(sas.Items))
	for i := range sas.Items {
		sa := sas.Items[i]
		name := sa.Name
		if allNamespaces {
			name = fmt.Sprintf("%s/%s", sa.Namespace, sa.Name)
		}
		fmt.Fprintf(&sb, "• %s\tsecrets: %d\tage: %s\n", name, len(sa.Secrets), formatDuration(time.Since(sa.CreationTimestamp.Time)))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (r *RBAC) GetServiceAccount(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("service account name is required")
	}
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	ns := r.namespace(cm)
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	sa, err := client.CoreV1().ServiceAccounts(ns).Get(timeoutCtx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account %q: %w", r.Name, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "ServiceAccount: %s\nNamespace: %s\n", sa.Name, sa.Namespace)
	if len(sa.Secrets) > 0 {
		names := make([]string, 0, len(sa.Secrets))
		for _, s := range sa.Secrets {
			names = append(names, s.Name)
		}
		fmt.Fprintf(&sb, "Secrets: %s\n", strings.Join(names, ", "))
	}
	if sa.AutomountServiceAccountToken != nil {
		fmt.Fprintf(&sb, "Automount Token: %t\n", *sa.AutomountServiceAccountToken)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func formatPolicyRules(rules []rbacv1.PolicyRule) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Rules (%d):\n", len(rules))
	for _, rule := range rules {
		fmt.Fprintf(&sb, "  apiGroups: [%s] resources: [%s] verbs: [%s]",
			strings.Join(rule.APIGroups, ","), strings.Join(rule.Resources, ","), strings.Join(rule.Verbs, ","))
		if len(rule.ResourceNames) > 0 {
			fmt.Fprintf(&sb, " resourceNames: [%s]", strings.Join(rule.ResourceNames, ","))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatSubjects(subjects []rbacv1.Subject) string {
	if len(subjects) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(subjects))
	for _, s := range subjects {
		if s.Namespace != "" {
			parts = append(parts, fmt.Sprintf("%s:%s/%s", s.Kind, s.Namespace, s.Name))
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s", s.Kind, s.Name))
		}
	}
	return strings.Join(parts, ", ")
}
