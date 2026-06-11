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
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kfake "k8s.io/client-go/kubernetes/fake"
)

var widgetGVR = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}

func crdObject(name, group, kind, plural, scope string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]interface{}{"name": name},
		"spec": map[string]interface{}{
			"group": group,
			"scope": scope,
			"names": map[string]interface{}{"kind": kind, "plural": plural},
			"versions": []interface{}{
				map[string]interface{}{"name": "v1", "served": true, "storage": true},
			},
		},
	}}
}

func widgetObject(name, namespace string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata":   map[string]interface{}{"name": name},
		"status":     map[string]interface{}{"phase": "Ready"},
	}
	if namespace != "" {
		obj["metadata"].(map[string]interface{})["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: obj}
}

func crListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		crdGVR:    "CustomResourceDefinitionList",
		widgetGVR: "WidgetList",
	}
}

func newCRDynamic(t *testing.T) dynamic.Interface {
	t.Helper()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), crListKinds())
}

func TestCustomResourceCRDs(t *testing.T) {
	ctx := context.Background()

	dyn := newCRDynamic(t)
	_, err := dyn.Resource(crdGVR).Create(ctx, crdObject("widgets.example.com", "example.com", "Widget", "widgets", "Namespaced"), metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

	list, err := (&CustomResource{}).ListCRDs(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, list, "widgets.example.com")
	assert.Contains(t, list, "Widget")

	get, err := (&CustomResource{Name: "widgets.example.com"}).GetCRD(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, get, "Group: example.com")
	assert.Contains(t, get, "v1(served=true)")

	_, err = (&CustomResource{}).GetCRD(ctx, mockCM)
	assert.Error(t, err)
}

func TestCustomResourceInstances(t *testing.T) {
	ctx := context.Background()

	dyn := newCRDynamic(t)
	_, err := dyn.Resource(widgetGVR).Namespace(defaultNamespace).Create(ctx, widgetObject("w1", defaultNamespace), metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	list, err := (&CustomResource{Group: "example.com", Version: "v1", Resource: "widgets"}).List(ctx, mockCM, false)
	assert.NoError(t, err)
	assert.Contains(t, list, "w1")

	all, err := (&CustomResource{Group: "example.com", Version: "v1", Resource: "widgets"}).List(ctx, mockCM, true)
	assert.NoError(t, err)
	assert.Contains(t, all, "w1")

	get, err := (&CustomResource{Group: "example.com", Version: "v1", Resource: "widgets", Name: "w1"}).Get(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, get, "Widget: w1")
	assert.Contains(t, get, "phase")

	del, err := (&CustomResource{Group: "example.com", Version: "v1", Resource: "widgets", Name: "w1"}).Delete(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, del, "deleted successfully")
	_, err = dyn.Resource(widgetGVR).Namespace(defaultNamespace).Get(ctx, "w1", metav1.GetOptions{})
	assert.Error(t, err)

	_, err = (&CustomResource{Version: "v1"}).List(ctx, mockCM, false)
	assert.Error(t, err)
	_, err = (&CustomResource{Resource: "widgets"}).Get(ctx, mockCM)
	assert.Error(t, err)
	_, err = (&CustomResource{Resource: "widgets"}).Delete(ctx, mockCM)
	assert.Error(t, err)
}

func TestListAPIResources(t *testing.T) {
	ctx := context.Background()

	// The fake discovery client returns no preferred resources, so this
	// exercises the discovery call + empty path.
	clientset := kfake.NewSimpleClientset()
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(clientset, nil)

	result, err := (&CustomResource{}).ListAPIResources(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "API Resources")
}

func TestFormatAPIResources(t *testing.T) {
	lists := []*metav1.APIResourceList{
		nil,
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod"},
				{Name: "pods/log", Kind: "Pod"}, // subresource, skipped
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{{Name: "deployments", Kind: "Deployment"}},
		},
	}

	result := formatAPIResources(lists)
	assert.Contains(t, result, "API Resources (2)")
	assert.Contains(t, result, "pods")
	assert.Contains(t, result, "group: core")
	assert.Contains(t, result, "deployments")
	assert.NotContains(t, result, "pods/log")
}
