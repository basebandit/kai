package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeleteRun(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = applyDiscovery()

	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), applyListKinds)
	_, err := dyn.Resource(cmGVR).Namespace(defaultNamespace).Create(ctx, uObj("v1", "ConfigMap", "cm1", defaultNamespace), metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dyn.Resource(nsGVR).Create(ctx, uObj("v1", "Namespace", "team-a", ""), metav1.CreateOptions{})
	assert.NoError(t, err)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	result, err := (&Delete{Manifest: applyManifest}).Run(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "Deleted 2 object(s)")
	assert.Contains(t, result, "ConfigMap default/cm1 deleted")
	assert.Contains(t, result, "Namespace team-a deleted")

	// Object is gone.
	_, err = dyn.Resource(cmGVR).Namespace(defaultNamespace).Get(ctx, "cm1", metav1.GetOptions{})
	assert.Error(t, err)
}

func TestDeleteNotFound(t *testing.T) {
	ctx := context.Background()

	fakeClient := fake.NewSimpleClientset()
	fakeClient.Resources = applyDiscovery()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), applyListKinds)

	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	// Deleting an absent object is reported, not errored (idempotent).
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: ghost
`
	result, err := (&Delete{Manifest: manifest}).Run(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, "not found (already deleted)")
}

func TestDeleteValidation(t *testing.T) {
	ctx := context.Background()
	mockCM := testmocks.NewMockClusterManager()

	_, err := (&Delete{Manifest: "   "}).Run(ctx, mockCM)
	assert.Error(t, err)

	_, err = (&Delete{Manifest: "---\n---\n"}).Run(ctx, mockCM)
	assert.Error(t, err)
}
