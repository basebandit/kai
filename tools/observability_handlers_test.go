package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"

	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var metricsListKinds = map[schema.GroupVersionResource]string{
	{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}: "NodeMetricsList",
	{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}:  "PodMetricsList",
}

func toolRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	assert.NotNil(t, result)
	return result.Content[0].(mcp.TextContent).Text
}

func makeNode(name string, ready, unschedulable bool) *corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{Unschedulable: unschedulable},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}},
			NodeInfo:   corev1.NodeSystemInfo{KubeletVersion: "v1.30.0"},
		},
	}
}

func TestListEventsHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e1", Namespace: defaultNamespace},
			Type:           "Warning",
			Reason:         "BackOff",
			Message:        "back-off restarting",
			Count:          3,
			LastTimestamp:  metav1.Now(),
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "pod-a"},
		})
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		result, err := listEventsHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
			"type": "Warning", "involved_object": "pod-a", "limit": float64(10),
		}))

		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "BackOff")
	})

	t.Run("AllNamespaces", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := listEventsHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
			"all_namespaces": true,
		}))

		assert.NoError(t, err)
		assert.Equal(t, "No events found", resultText(t, result))
	})
}

func TestNodeHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("ListNodes", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(makeNode("node-1", true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := listNodesHandler(mockCM)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "node-1")
	})

	t.Run("GetNodeMissingName", func(t *testing.T) {
		mockCM := testmocks.NewMockClusterManager()
		result, err := getNodeHandler(mockCM)(ctx, toolRequest(map[string]interface{}{}))
		assert.NoError(t, err)
		assert.Equal(t, errMissingNode, resultText(t, result))
	})

	t.Run("GetNodeEmptyName", func(t *testing.T) {
		mockCM := testmocks.NewMockClusterManager()
		result, err := getNodeHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": ""}))
		assert.NoError(t, err)
		assert.Equal(t, errEmptyName, resultText(t, result))
	})

	t.Run("GetNodeSuccess", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(makeNode("node-1", true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := getNodeHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "node-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "Node: node-1")
	})

	t.Run("Cordon", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(makeNode("node-1", true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := cordonNodeHandler(mockCM, false)(ctx, toolRequest(map[string]interface{}{"name": "node-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "cordoned")
	})

	t.Run("Uncordon", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(makeNode("node-1", true, true))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := cordonNodeHandler(mockCM, true)(ctx, toolRequest(map[string]interface{}{"name": "node-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "uncordoned")
	})

	t.Run("DrainSkipsManagedPods", func(t *testing.T) {
		dsPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ds-pod", Namespace: defaultNamespace,
				OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Name: "ds"}},
			},
			Spec: corev1.PodSpec{NodeName: "node-1"},
		}
		fakeClient := fake.NewSimpleClientset(makeNode("node-1", true, false), dsPod)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := drainNodeHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
			"name": "node-1", "ignore_daemonsets": true, "grace_period": float64(30),
		}))
		assert.NoError(t, err)
		text := resultText(t, result)
		assert.Contains(t, text, "drained")
		assert.Contains(t, text, "Skipped")
	})
}

func TestHealthHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("ClusterHealth", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			makeNode("node-1", true, false),
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: defaultNamespace}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := clusterHealthHandler(mockCM)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, result), "Cluster Health")
	})

	t.Run("NodeMetricsDegradesGracefully", func(t *testing.T) {
		dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), metricsListKinds)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentDynamicClient").Return(dynClient, nil)

		result, err := nodeMetricsHandler(mockCM)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.NotEmpty(t, resultText(t, result))
	})

	t.Run("PodMetricsDegradesGracefully", func(t *testing.T) {
		dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), metricsListKinds)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
		mockCM.On("GetCurrentDynamicClient").Return(dynClient, nil)

		result, err := podMetricsHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"namespace": defaultNamespace}))
		assert.NoError(t, err)
		assert.NotEmpty(t, resultText(t, result))
	})
}
