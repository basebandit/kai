package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNodeName = "node-1"

func resourceQty(s string) resource.Quantity { return resource.MustParse(s) }

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

	t.Run("GetSuccess", func(t *testing.T) {
		n := newNode(testNodeName, true, false)
		n.Status.NodeInfo.OSImage = "Ubuntu 22.04"
		n.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "10.0.0.1"}}
		n.Status.Capacity = corev1.ResourceList{
			corev1.ResourceCPU:    resourceQty("4"),
			corev1.ResourceMemory: resourceQty("8Gi"),
			corev1.ResourcePods:   resourceQty("110"),
		}
		fakeClient := fake.NewSimpleClientset(n)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Get(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "Node: "+testNodeName)
		assert.Contains(t, result, "Ubuntu 22.04")
		assert.Contains(t, result, "10.0.0.1")
		assert.Contains(t, result, "Conditions:")
	})

	t.Run("DrainSkipsManagedPods", func(t *testing.T) {
		dsPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ds-pod", Namespace: defaultNamespace,
				OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Name: "ds"}},
			},
			Spec: corev1.PodSpec{NodeName: testNodeName},
		}
		mirrorPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mirror-pod", Namespace: defaultNamespace,
				Annotations: map[string]string{corev1.MirrorPodAnnotationKey: "x"},
			},
			Spec: corev1.PodSpec{NodeName: testNodeName},
		}
		emptyDirPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "data-pod", Namespace: defaultNamespace},
			Spec: corev1.PodSpec{
				NodeName: testNodeName,
				Volumes:  []corev1.Volume{{Name: "cache", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
			},
		}
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, false), dsPod, mirrorPod, emptyDirPod)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Drain(ctx, mockCM, true, false, 30)

		assert.NoError(t, err)
		assert.Contains(t, result, "Skipped 3 pod(s)")
		assert.Contains(t, result, "Evicted 0 pod(s)")
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

	t.Run("DrainEvictsNormalPod", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: defaultNamespace},
			Spec:       corev1.PodSpec{NodeName: testNodeName},
		}
		fakeClient := fake.NewSimpleClientset(newNode(testNodeName, true, false), pod)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		node := &Node{Name: testNodeName}
		result, err := node.Drain(ctx, mockCM, true, true, -1)

		assert.NoError(t, err)
		assert.Contains(t, result, "app-pod")
	})
}
