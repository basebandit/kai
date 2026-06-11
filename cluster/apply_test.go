package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// applyDiscovery advertises configmaps (namespaced) and namespaces (cluster)
// so the REST mapper can resolve both scopes during apply.
func applyDiscovery() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
		},
	}}
}

func uObj(apiVersion, kind, name, ns string) *unstructured.Unstructured {
	md := map[string]interface{}{"name": name}
	if ns != "" {
		md["namespace"] = ns
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   md,
	}}
}

var applyListKinds = map[schema.GroupVersionResource]string{
	{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
	{Group: "", Version: "v1", Resource: "namespaces"}: "NamespaceList",
}

const applyManifest = `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: value
---
apiVersion: v1
kind: Namespace
metadata:
  name: team-a
`

func TestApplyRun(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = applyDiscovery()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), applyListKinds)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	result, err := (&Apply{Manifest: applyManifest}).Run(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "Applied 2 object(s)")
	assert.Contains(t, result, "ConfigMap default/cm1 created")
	assert.Contains(t, result, "Namespace team-a created")

	// Namespaced object landed in the current namespace.
	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	got, err := dyn.Resource(cmGVR).Namespace(defaultNamespace).Get(ctx, "cm1", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "cm1", got.GetName())
}

func TestApplyUpdate(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = applyDiscovery()

	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), applyListKinds)
	_, err := dyn.Resource(cmGVR).Namespace(defaultNamespace).Create(ctx, uObj("v1", "ConfigMap", "cm1", defaultNamespace), metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	// Re-applying an existing object takes the update branch.
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: changed
`
	result, err := (&Apply{Manifest: manifest}).Run(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "ConfigMap default/cm1 configured")
}

func TestApplyNamespaceOverride(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = applyDiscovery()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), applyListKinds)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

	// Override applies to a namespaced doc that omits metadata.namespace; current
	// namespace must not be consulted.
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
`
	result, err := (&Apply{Manifest: manifest, Namespace: otherNamespace}).Run(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "ConfigMap "+otherNamespace+"/cm2 created")
}

func TestApplyValidation(t *testing.T) {
	ctx := context.Background()
	mockCM := testmocks.NewMockClusterManager()

	_, err := (&Apply{Manifest: "   "}).Run(ctx, mockCM)
	assert.Error(t, err)

	_, err = (&Apply{Manifest: "---\n---\n"}).Run(ctx, mockCM)
	assert.Error(t, err)
}

func TestDecodeManifests(t *testing.T) {
	objs, err := decodeManifests(applyManifest)
	assert.NoError(t, err)
	assert.Len(t, objs, 2)

	_, err = decodeManifests("foo: bar")
	assert.Error(t, err) // missing apiVersion/kind

	_, err = decodeManifests(`apiVersion: v1
kind: ConfigMap
`)
	assert.Error(t, err) // missing metadata.name
}
