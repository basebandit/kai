package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNodeName = "node-1"

func newNode(name string, ready, unschedulable bool) *corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		Spec: corev1.NodeSpec{Unschedulable: unschedulable},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}},
			NodeInfo:   corev1.NodeSystemInfo{KubeletVersion: "v1.30.0"},
		},
	}
}

func TestNodeOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("List", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{}
		result, err := node.List(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, testNodeName)
		assert.Contains(t, result, "Ready")
		assert.Contains(t, result, "control-plane")
	})

	t.Run("GetRequiresName", func(t *testing.T) {
		mockCM := testmocks.NewMockClusterManager()
		node := &Node{}
		_, err := node.Get(ctx, mockCM)
		assert.Error(t, err)
	})

	t.Run("Cordon", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Cordon(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "cordoned successfully")

		updated, _ := fakeClient.CoreV1().Nodes().Get(ctx, testNodeName, metav1.GetOptions{})
		assert.True(t, updated.Spec.Unschedulable)
	})

	t.Run("CordonAlreadyCordoned", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, true))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Cordon(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "already cordoned")
	})

	t.Run("Uncordon", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, true))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Uncordon(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "uncordoned successfully")

		updated, _ := fakeClient.CoreV1().Nodes().Get(ctx, testNodeName, metav1.GetOptions{})
		assert.False(t, updated.Spec.Unschedulable)
	})

	t.Run("DrainNoPods", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, false))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Drain(ctx, mockCM, true, false, -1)

		assert.NoError(t, err)
		assert.Contains(t, result, "drained")
		assert.Contains(t, result, "Evicted 0 pod(s)")
	})
}
