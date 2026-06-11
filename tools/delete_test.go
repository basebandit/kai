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
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterDeleteTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"),
		mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(1)
	RegisterDeleteTools(mockServer, mockCM)
	mockServer.AssertExpectations(t)
}

func TestDeleteYAMLHandler(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"}},
	}}
	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	listKinds := map[schema.GroupVersionResource]string{cmGVR: "ConfigMapList"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds)
	_, err := dyn.Resource(cmGVR).Namespace(defaultNamespace).Create(ctx, &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "cm1", "namespace": defaultNamespace},
	}}, metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
`
	r, err := deleteYAMLHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"manifest": manifest}))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "ConfigMap default/cm1 deleted")

	// Missing manifest argument.
	r, err = deleteYAMLHandler(mockCM)(ctx, toolRequest(nil))
	assert.NoError(t, err)
	assert.Contains(t, resultText(t, r), "manifest")
}
