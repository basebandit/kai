package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/basebandit/kai"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// Apply applies one or more YAML/JSON manifest documents to the cluster. It
// mirrors `kubectl apply -f`: each document is created if absent or replaced if
// it already exists (upsert). Documents are separated by `---`.
type Apply struct {
	// Manifest is the raw YAML/JSON, optionally multiple `---` separated docs.
	Manifest string

	// Namespace optionally overrides the target namespace for namespaced objects
	// whose manifest omits metadata.namespace. Ignored for cluster-scoped kinds.
	Namespace string
}

// Run applies every document in the manifest and returns a per-object summary.
func (a *Apply) Run(ctx context.Context, cm kai.ClusterManager) (string, error) {
	if strings.TrimSpace(a.Manifest) == "" {
		return "", errors.New("manifest is required")
	}

	objs, err := decodeManifests(a.Manifest)
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
	fmt.Fprintf(&sb, "Applied %d object(s):\n", len(objs))
	for _, obj := range objs {
		line, err := applyObject(ctx, dyn, mapper, obj, a.Namespace, cm)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "• %s\n", line)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// decodeManifests splits a multi-document YAML/JSON stream into unstructured
// objects, validating that each carries apiVersion, kind and metadata.name.
func decodeManifests(manifest string) ([]*unstructured.Unstructured, error) {
	dec := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	var objs []*unstructured.Unstructured
	for {
		raw := map[string]interface{}{}
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
		if len(raw) == 0 {
			continue // empty document between separators
		}
		obj := &unstructured.Unstructured{Object: raw}
		if obj.GetAPIVersion() == "" || obj.GetKind() == "" {
			return nil, errors.New("manifest document missing apiVersion or kind")
		}
		if obj.GetName() == "" {
			return nil, fmt.Errorf("%s document missing metadata.name", obj.GetKind())
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

// newRESTMapper builds a REST mapper from server discovery so arbitrary kinds
// (built-in or CRD) can be resolved to their resource and scope.
func newRESTMapper(disc discovery.DiscoveryInterface) (meta.RESTMapper, error) {
	groupResources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return nil, err
	}
	return restmapper.NewDiscoveryRESTMapper(groupResources), nil
}

// applyObject resolves an object's GVK to a resource via the mapper and applies
// it with server-side apply, honoring namespace scope.
func applyObject(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured, nsOverride string, cm kai.ClusterManager) (string, error) {
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
		obj.SetNamespace(ns)
		ri = dyn.Resource(mapping.Resource).Namespace(ns)
		prefix = ns + "/"
	} else {
		ri = dyn.Resource(mapping.Resource)
	}

	name := obj.GetName()
	existing, err := ri.Get(timeoutCtx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := ri.Create(timeoutCtx, obj, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("failed to create %s %q: %w", gvk.Kind, name, err)
		}
		return fmt.Sprintf("%s %s%s created", gvk.Kind, prefix, name), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get %s %q: %w", gvk.Kind, name, err)
	}

	// Preserve resourceVersion so the update is accepted as a replace.
	obj.SetResourceVersion(existing.GetResourceVersion())
	if _, err := ri.Update(timeoutCtx, obj, metav1.UpdateOptions{}); err != nil {
		return "", fmt.Errorf("failed to update %s %q: %w", gvk.Kind, name, err)
	}
	return fmt.Sprintf("%s %s%s configured", gvk.Kind, prefix, name), nil
}
