package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// Delete removes one or more YAML/JSON manifest documents from the cluster. It
// mirrors `kubectl delete -f`: every document is resolved to its resource and
// deleted. Documents are separated by `---`.
type Delete struct {
	// Manifest is the raw YAML/JSON, optionally multiple `---` separated docs.
	Manifest string
	// Namespace optionally overrides the target namespace for namespaced objects
	// whose manifest omits metadata.namespace. Ignored for cluster-scoped kinds.
	Namespace string
}

// Run deletes every document in the manifest and returns a per-object summary.
func (d *Delete) Run(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if strings.TrimSpace(d.Manifest) == "" {
		return "", errors.New("manifest is required")
	}

	objs, err := decodeManifests(d.Manifest)
	if err != nil {
		return "", err
	}
	if len(objs) == 0 {
		return "", errors.New("no kubernetes objects found in manifest")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}
	dyn, err := cm.GetCurrentDynamicClient()
	if err != nil {
		return "", fmt.Errorf("error getting dynamic client: %w", err)
	}

	mapper, err := newRESTMapper(client.Discovery())
	if err != nil {
		return "", fmt.Errorf("failed to build REST mapper: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Deleted %d object(s):\n", len(objs))
	for _, obj := range objs {
		line, err := deleteObject(ctx, dyn, mapper, obj, d.Namespace, cm)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "• %s\n", line)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// deleteObject resolves an object's GVK to a resource via the mapper and deletes
// it, honoring namespace scope. A missing object is reported, not treated as an
// error, so deleting an already-gone manifest is idempotent.
func deleteObject(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured, nsOverride string, cm kai.ClusterManager) (string, error) {
	gvk := obj.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("unable to resolve %s/%s: %w", gvk.GroupVersion().String(), gvk.Kind, err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var (
		ri     dynamic.ResourceInterface
		prefix string
	)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			if nsOverride != "" {
				ns = nsOverride
			} else {
				ns = cm.GetCurrentNamespace()
			}
		}
		ri = dyn.Resource(mapping.Resource).Namespace(ns)
		prefix = ns + "/"
	} else {
		ri = dyn.Resource(mapping.Resource)
	}

	name := obj.GetName()
	err = ri.Delete(timeoutCtx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return fmt.Sprintf("%s %s%s not found (already deleted)", gvk.Kind, prefix, name), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to delete %s %q: %w", gvk.Kind, name, err)
	}
	return fmt.Sprintf("%s %s%s deleted", gvk.Kind, prefix, name), nil
}
