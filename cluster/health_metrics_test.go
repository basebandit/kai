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
)

var testMetricsListKinds = map[schema.GroupVersionResource]string{
	nodeMetricsGVR: "NodeMetricsList",
	podMetricsGVR:  "PodMetricsList",
}

func nodeMetric(name, cpu, mem string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "metrics.k8s.io/v1beta1",
		"kind":       "NodeMetrics",
		"metadata":   map[string]interface{}{"name": name},
		"usage":      map[string]interface{}{"cpu": cpu, "memory": mem},
	}}
}

func podMetric(name, namespace, cpu, mem string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "metrics.k8s.io/v1beta1",
		"kind":       "PodMetrics",
		"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
		"usage":      map[string]interface{}{"cpu": cpu, "memory": mem},
	}}
}

func newMetricsClient(t *testing.T) dynamic.Interface {
	t.Helper()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testMetricsListKinds)
}

func TestHealthMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("NodeMetricsWithData", func(t *testing.T) {
		dyn := newMetricsClient(t)
		_, err := dyn.Resource(nodeMetricsGVR).Create(ctx, nodeMetric("node-b", "200m", "300Mi"), metav1.CreateOptions{})
		assert.NoError(t, err)
		_, err = dyn.Resource(nodeMetricsGVR).Create(ctx, nodeMetric("node-a", "100m", "200Mi"), metav1.CreateOptions{})
		assert.NoError(t, err)

		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

		health := &Health{}
		result, err := health.NodeMetrics(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "Node metrics (2)")
		assert.Contains(t, result, "node-a")
		assert.Contains(t, result, "cpu: 100m")
	})

	t.Run("NodeMetricsEmpty", func(t *testing.T) {
		dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), testMetricsListKinds)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

		health := &Health{}
		result, err := health.NodeMetrics(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "No node metrics available")
	})

	t.Run("PodMetricsWithData", func(t *testing.T) {
		dyn := newMetricsClient(t)
		_, err := dyn.Resource(podMetricsGVR).Namespace(defaultNamespace).Create(ctx, podMetric("pod-a", defaultNamespace, "10m", "20Mi"), metav1.CreateOptions{})
		assert.NoError(t, err)

		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
		mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

		health := &Health{}
		result, err := health.PodMetrics(ctx, mockCM, "", false)

		assert.NoError(t, err)
		assert.Contains(t, result, "Pod metrics (1)")
		assert.Contains(t, result, "default/pod-a")
	})

	t.Run("PodMetricsAllNamespaces", func(t *testing.T) {
		dyn := newMetricsClient(t)
		_, err := dyn.Resource(podMetricsGVR).Namespace(defaultNamespace).Create(ctx, podMetric("pod-a", defaultNamespace, "10m", "20Mi"), metav1.CreateOptions{})
		assert.NoError(t, err)

		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(dyn, nil)

		health := &Health{}
		result, err := health.PodMetrics(ctx, mockCM, "", true)

		assert.NoError(t, err)
		assert.Contains(t, result, "Pod metrics (1)")
	})
}
