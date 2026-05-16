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

func newEvent(name, namespace, evType, reason, objName string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: name, Namespace: namespace},
		Type:           evType,
		Reason:         reason,
		Message:        reason + " message",
		Count:          1,
		LastTimestamp:  metav1.Now(),
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: objName, Namespace: namespace},
	}
}

func TestEventList(t *testing.T) {
	ctx := context.Background()

	t.Run("ListsEventsInNamespace", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			newEvent("e1", defaultNamespace, "Warning", "BackOff", "pod-a"),
			newEvent("e2", defaultNamespace, "Normal", "Pulled", "pod-b"),
		)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		event := &Event{Namespace: defaultNamespace}
		result, err := event.List(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "Events (2)")
		assert.Contains(t, result, "BackOff")
		assert.Contains(t, result, "Pod/pod-a")
	})

	t.Run("NoEvents", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		event := &Event{Namespace: defaultNamespace}
		result, err := event.List(ctx, mockCM)

		assert.NoError(t, err)
		assert.Equal(t, "No events found", result)
	})

	t.Run("DefaultsToCurrentNamespace", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newEvent("e1", defaultNamespace, "Warning", "Failed", "pod-a"))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		event := &Event{}
		result, err := event.List(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "Failed")
	})

	t.Run("AllNamespacesFormatting", func(t *testing.T) {
		e1 := newEvent("e1", defaultNamespace, "Warning", "BackOff", "pod-a")
		e1.Count = 5
		// Event with only EventTime set (no LastTimestamp) exercises eventTime fallback.
		e2 := newEvent("e2", otherNamespace, "Normal", "Pulled", "pod-b")
		e2.LastTimestamp = metav1.Time{}
		e2.EventTime = metav1.NowMicro()
		// Event with only FirstTimestamp set.
		e3 := newEvent("e3", otherNamespace, "Normal", "Created", "pod-c")
		e3.LastTimestamp = metav1.Time{}
		e3.FirstTimestamp = metav1.Now()

		fakeClient := fake.NewSimpleClientset(e1, e2, e3)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		event := &Event{AllNamespaces: true}
		result, err := event.List(ctx, mockCM)

		assert.NoError(t, err)
		assert.Contains(t, result, "ns: "+otherNamespace)
		assert.Contains(t, result, "count: 5")
	})
}
