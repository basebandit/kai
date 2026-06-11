package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kfake "k8s.io/client-go/kubernetes/fake"
)

var (
	crdGVRTest    = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	widgetGVRTest = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	crListKinds   = map[schema.GroupVersionResource]string{
		crdGVRTest:    "CustomResourceDefinitionList",
		widgetGVRTest: "WidgetList",
	}
)

func TestRegisterCustomResourceTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(6)
	RegisterCustomResourceTools(mockServer, mockCM)
	mockServer.AssertExpectations(t)
}

func TestCustomResourceHandlers(t *testing.T) {
	ctx := context.Background()

	crd := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]interface{}{"name": "widgets.example.com"},
		"spec": map[string]interface{}{
			"group": "example.com", "scope": "Namespaced",
			"names":    map[string]interface{}{"kind": "Widget", "plural": "widgets"},
			"versions": []interface{}{map[string]interface{}{"name": "v1", "served": true}},
		},
	}}
	widget := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "example.com/v1", "kind": "Widget",
		"metadata": map[string]interface{}{"name": "w1", "namespace": defaultNamespace},
	}}

	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), crListKinds)
	_, err := dyn.Resource(crdGVRTest).Create(ctx, crd, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dyn.Resource(widgetGVRTest).Namespace(defaultNamespace).Create(ctx, widget, metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	r, err := listCRDsHandler(mockCM)(ctx, toolRequest(nil))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "widgets.example.com")

	r, err = getCRDHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "widgets.example.com"}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "Group: example.com")

	r, err = listCustomResourcesHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
		"group": "example.com", "version": "v1", "resource": "widgets",
	}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "w1")

	r, err = getCustomResourceHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
		"group": "example.com", "version": "v1", "resource": "widgets", "name": "w1",
	}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "Widget: w1")

	r, err = deleteCustomResourceHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
		"group": "example.com", "version": "v1", "resource": "widgets", "name": "w1",
	}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "deleted successfully")

	// Missing required version.
	r, err = listCustomResourcesHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"resource": "widgets"}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "version")

	// API discovery (fake returns none).
	clientset := kfake.NewSimpleClientset()
	discCM := testmocks.NewMockClusterManager()
	discCM.On("GetCurrentClient").Return(clientset, nil)
	r, err = listAPIResourcesHandler(discCM)(ctx, toolRequest(nil))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "API Resources")
}
