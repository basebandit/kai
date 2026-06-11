package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterStorageTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(9)
	RegisterStorageTools(mockServer, mockCM)
	mockServer.AssertExpectations(t)
}

func TestStorageHandlers(t *testing.T) {
	ctx := context.Background()

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-1"},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:    corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	scReclaim := corev1.PersistentVolumeReclaimDelete
	sc := &storagev1.StorageClass{
		ObjectMeta:    metav1.ObjectMeta{Name: "standard"},
		Provisioner:   "p",
		ReclaimPolicy: &scReclaim,
	}

	t.Run("PVHandlers", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(pv)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		r, err := listPVHandler(mockCM)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "pv-1")

		r, err = getPVHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "pv-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "PersistentVolume: pv-1")

		r, err = getPVHandler(mockCM)(ctx, toolRequest(map[string]interface{}{}))
		assert.NoError(t, err)
		assert.Equal(t, errMissingName, resultText(t, r))

		r, err = deletePVHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "pv-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "deleted")
	})

	t.Run("PVCHandlers", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		r, err := createPVCHandler(mockCM)(ctx, toolRequest(map[string]interface{}{
			"name": "pvc-1", "storage": "1Gi", "storage_class": "standard",
			"volume_mode": "Filesystem", "access_modes": []interface{}{"ReadWriteOnce"},
		}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "pvc-1")

		r, err = listPVCHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"all_namespaces": true}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "pvc-1")

		r, err = getPVCHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "pvc-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "PersistentVolumeClaim: pvc-1")

		r, err = deletePVCHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "pvc-1"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "deleted")
	})

	t.Run("StorageClassHandlers", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(sc)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		r, err := listStorageClassHandler(mockCM)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "standard")

		r, err = getStorageClassHandler(mockCM)(ctx, toolRequest(map[string]interface{}{"name": "standard"}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), "StorageClass: standard")
	})
}
