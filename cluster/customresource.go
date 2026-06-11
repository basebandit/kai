package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/basebandit/kai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// CustomResource provides access to CRDs and arbitrary custom resources via
// the dynamic client.
type CustomResource struct {
	Group     string
	Version   string
	Resource  string
	Name      string
	Namespace string
}

// ListCRDs lists all CustomResourceDefinitions registered in the cluster.
func (c *CustomResource) ListCRDs(ctx context.Context, cm kai.ClusterManager) (string, error) {
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	list, err := dyn.Resource(crdGVR).List(timeoutCtx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list CRDs: %w", err)
	}
	if len(list.Items) == 0 {
		return "No custom resource definitions found", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Custom Resource Definitions (%d):\n", len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		group, _, _ := unstructured.NestedString(item.Object, "spec", "group")
		scope, _, _ := unstructured.NestedString(item.Object, "spec", "scope")
		kind, _, _ := unstructured.NestedString(item.Object, "spec", "names", "kind")
		fmt.Fprintf(&sb, "• %s\tgroup: %s\tkind: %s\tscope: %s\n", item.GetName(), group, kind, scope)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// GetCRD returns details for a single CustomResourceDefinition.
func (c *CustomResource) GetCRD(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if c.Name == "" {
		return "", fmt.Errorf("CRD name is required")
	}
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	crd, err := dyn.Resource(crdGVR).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get CRD %q: %w", c.Name, err)
	}

	group, _, _ := unstructured.NestedString(crd.Object, "spec", "group")
	scope, _, _ := unstructured.NestedString(crd.Object, "spec", "scope")
	kind, _, _ := unstructured.NestedString(crd.Object, "spec", "names", "kind")
	plural, _, _ := unstructured.NestedString(crd.Object, "spec", "names", "plural")
	versions, _, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")

	var sb strings.Builder
	fmt.Fprintf(&sb, "CRD: %s\nGroup: %s\nKind: %s\nPlural: %s\nScope: %s\n", crd.GetName(), group, kind, plural, scope)
	if len(versions) > 0 {
		names := make([]string, 0, len(versions))
		for _, v := range versions {
			if vm, ok := v.(map[string]interface{}); ok {
				if name, ok := vm["name"].(string); ok {
					served, _ := vm["served"].(bool)
					names = append(names, fmt.Sprintf("%s(served=%t)", name, served))
				}
			}
		}
		fmt.Fprintf(&sb, "Versions: %s\n", strings.Join(names, ", "))
	}
	if group != "" && plural != "" {
		sb.WriteString("\nQuery instances with list_custom_resources using:\n")
		fmt.Fprintf(&sb, "  group=%s, resource=%s, version=<one of the versions above>\n", group, plural)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// List lists instances of a custom resource identified by group/version/resource.
func (c *CustomResource) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool) (string, error) {
	if c.Version == "" || c.Resource == "" {
		return "", fmt.Errorf("version and resource are required")
	}
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{Group: c.Group, Version: c.Version, Resource: c.Resource}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	var list *unstructured.UnstructuredList
	if allNamespaces {
		list, err = dyn.Resource(gvr).List(timeoutCtx, metav1.ListOptions{})
	} else {
		ns := c.Namespace
		if ns == "" {
			ns = cm.GetCurrentNamespace()
		}
		list, err = dyn.Resource(gvr).Namespace(ns).List(timeoutCtx, metav1.ListOptions{})
	}
	if err != nil {
		return "", fmt.Errorf("failed to list custom resources: %w", err)
	}
	if len(list.Items) == 0 {
		return "No custom resources found", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%d):\n", c.Resource, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		if ns := item.GetNamespace(); ns != "" {
			fmt.Fprintf(&sb, "• %s/%s\n", ns, item.GetName())
		} else {
			fmt.Fprintf(&sb, "• %s\n", item.GetName())
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// Get returns a single custom resource instance as YAML-ish key listing.
func (c *CustomResource) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if c.Version == "" || c.Resource == "" || c.Name == "" {
		return "", fmt.Errorf("version, resource and name are required")
	}
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{Group: c.Group, Version: c.Version, Resource: c.Resource}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var (
		obj    *unstructured.Unstructured
		getErr error
	)
	ns := c.Namespace
	if ns == "" {
		ns = cm.GetCurrentNamespace()
	}
	obj, getErr = dyn.Resource(gvr).Namespace(ns).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if getErr != nil {
		// Retry cluster-scoped if namespaced lookup failed.
		obj, err = dyn.Resource(gvr).Get(timeoutCtx, c.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get custom resource %q: %w", c.Name, getErr)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s: %s\n", obj.GetKind(), obj.GetName())
	if obj.GetNamespace() != "" {
		fmt.Fprintf(&sb, "Namespace: %s\n", obj.GetNamespace())
	}
	fmt.Fprintf(&sb, "API Version: %s\n", obj.GetAPIVersion())
	if labels := obj.GetLabels(); len(labels) > 0 {
		fmt.Fprintf(&sb, "Labels: %v\n", labels)
	}
	if status, found, _ := unstructured.NestedMap(obj.Object, "status"); found && len(status) > 0 {
		keys := make([]string, 0, len(status))
		for k := range status {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&sb, "Status fields: %s\n", strings.Join(keys, ", "))
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// Delete removes a single custom resource instance identified by
// group/version/resource/name. It tries the namespaced scope first, then falls
// back to cluster scope, mirroring Get.
func (c *CustomResource) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if c.Version == "" || c.Resource == "" || c.Name == "" {
		return "", fmt.Errorf("version, resource and name are required")
	}
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{Group: c.Group, Version: c.Version, Resource: c.Resource}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	ns := c.Namespace
	if ns == "" {
		ns = cm.GetCurrentNamespace()
	}
	delErr := dyn.Resource(gvr).Namespace(ns).Delete(timeoutCtx, c.Name, metav1.DeleteOptions{})
	if delErr != nil {
		// Retry cluster-scoped if the namespaced delete failed.
		if err = dyn.Resource(gvr).Delete(timeoutCtx, c.Name, metav1.DeleteOptions{}); err != nil {
			return "", fmt.Errorf("failed to delete custom resource %q: %w", c.Name, delErr)
		}
	}
	return fmt.Sprintf("Custom resource %q deleted successfully", c.Name), nil
}

// ListAPIResources lists the server's preferred API resources (discovery).
func (c *CustomResource) ListAPIResources(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	lists, err := client.Discovery().ServerPreferredResources()
	if err != nil && len(lists) == 0 {
		return "", fmt.Errorf("failed to discover API resources: %w", err)
	}

	return formatAPIResources(lists), nil
}

func formatAPIResources(lists []*metav1.APIResourceList) string {
	type apiResource struct{ name, group, kind string }
	var resources []apiResource
	for _, list := range lists {
		if list == nil {
			continue
		}
		gv, _ := schema.ParseGroupVersion(list.GroupVersion)
		for _, res := range list.APIResources {
			if strings.Contains(res.Name, "/") {
				continue // skip subresources
			}
			resources = append(resources, apiResource{name: res.Name, group: gv.Group, kind: res.Kind})
		}
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].group != resources[j].group {
			return resources[i].group < resources[j].group
		}
		return resources[i].name < resources[j].name
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "API Resources (%d):\n", len(resources))
	for _, r := range resources {
		group := r.group
		if group == "" {
			group = "core"
		}
		fmt.Fprintf(&sb, "• %s\tgroup: %s\tkind: %s\n", r.name, group, r.kind)
	}
	return strings.TrimRight(sb.String(), "\n")
}
