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

func newPodWithPhase(name string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: defaultNamespace},
		Status:     corev1.PodStatus{Phase: phase},
	}
}

func TestHealthCluster(t *testing.T) {
	ctx := context.Background()

	t.Run("HealthySummary", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			newNode("node-1", true, false),
			newNode("node-2", true, false),
			newPodWithPhase("pod-a", corev1.PodRunning),
			newPodWithPhase("pod-b", corev1.PodRunning),
		)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		health := &Health{}
		result, err := health.Cluster(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "2 total, 2 ready, 0 not ready")
		assert.Contains(t, result, "Running: 2")
		assert.Contains(t, result, "Overall: Healthy")
	})

	t.Run("DegradedWhenNodeNotReady", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			newNode("node-1", true, false),
			newNode("node-2", false, false),
			newPodWithPhase("pod-a", corev1.PodFailed),
		)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		health := &Health{}
		result, err := health.Cluster(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "1 not ready")
		assert.Contains(t, result, "Overall: Degraded")
	})
}
