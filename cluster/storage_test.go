package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newPV(name string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:                      corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              "standard",
		},
		Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
	}
}

func TestPersistentVolumeOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("List", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newPV("pv-1"))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		pv := &PersistentVolume{}
		result, err := pv.List(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "pv-1")
		assert.Contains(t, result, "RWO")
	})

	t.Run("ListEmpty", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		pv := &PersistentVolume{}
		result, err := pv.List(ctx, mockCM)
		assert.NoError(t, err)
		assert.Equal(t, "No persistent volumes found", result)
	})

	t.Run("GetAndValidate", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newPV("pv-1"))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		pv := &PersistentVolume{Name: "pv-1"}
		result, err := pv.Get(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "PersistentVolume: pv-1")

		_, err = (&PersistentVolume{}).Get(ctx, mockCM)
		assert.Error(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(newPV("pv-1"))
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		pv := &PersistentVolume{Name: "pv-1"}
		result, err := pv.Delete(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "deleted successfully")
	})
}

func TestPersistentVolumeClaimOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		pvc := &PersistentVolumeClaim{
			Name:             "pvc-1",
			Storage:          "1Gi",
			StorageClassName: "standard",
			AccessModes:      []string{"ReadWriteOnce"},
			VolumeMode:       "Filesystem",
			Labels:           map[string]interface{}{"app": "db"},
		}
		result, err := pvc.Create(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "pvc-1")

		got, err := fakeClient.CoreV1().PersistentVolumeClaims(defaultNamespace).Get(ctx, "pvc-1", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "db", got.Labels["app"])
	})

	t.Run("CreateValidation", func(t *testing.T) {
		mockCM := testmocks.NewMockClusterManager()
		_, err := (&PersistentVolumeClaim{}).Create(ctx, mockCM)
		assert.Error(t, err)
		_, err = (&PersistentVolumeClaim{Name: "x"}).Create(ctx, mockCM)
		assert.Error(t, err)
		_, err = (&PersistentVolumeClaim{Name: "x", Storage: "bad-qty"}).Create(ctx, mockCM)
		assert.Error(t, err)
	})

	t.Run("ListGetDelete", func(t *testing.T) {
		existing := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "pvc-1", Namespace: defaultNamespace},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")}},
			},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		}
		fakeClient := fake.NewSimpleClientset(existing)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

		list, err := (&PersistentVolumeClaim{}).List(ctx, mockCM, false, "")
		assert.NoError(t, err)
		assert.Contains(t, list, "pvc-1")

		all, err := (&PersistentVolumeClaim{}).List(ctx, mockCM, true, "")
		assert.NoError(t, err)
		assert.Contains(t, all, "pvc-1")

		got, err := (&PersistentVolumeClaim{Name: "pvc-1"}).Get(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, got, "PersistentVolumeClaim: pvc-1")

		del, err := (&PersistentVolumeClaim{Name: "pvc-1"}).Delete(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, del, "deleted successfully")
	})
}

func TestStorageFormattingHelpers(t *testing.T) {
	assert.Equal(t, "RWO,ROX,RWX,RWOP", accessModesToString([]corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce, corev1.ReadOnlyMany, corev1.ReadWriteMany, corev1.ReadWriteOncePod,
	}))
	assert.Equal(t, "<none>", accessModesToString(nil))
	assert.Equal(t, "Custom", accessModesToString([]corev1.PersistentVolumeAccessMode{"Custom"}))

	assert.Equal(t, "<unknown>", pvCapacity(&corev1.PersistentVolume{}))
	assert.Equal(t, "<unset>", pvcCapacity(&corev1.PersistentVolumeClaim{}))
}

func TestPVCNamespaceOverride(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)

	// Explicit namespace must be honored without consulting GetCurrentNamespace.
	pvc := &PersistentVolumeClaim{Name: "pvc-x", Namespace: otherNamespace, Storage: "1Gi"}
	result, err := pvc.Create(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, result, otherNamespace)
}

func TestRBACNamespaceOverride(t *testing.T) {
	ctx := context.Background()
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: defaultNamespace}}
	fakeClient := fake.NewSimpleClientset(role)
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)

	// Explicit namespace exercises the non-default branch of RBAC.namespace.
	_, err := (&RBAC{Namespace: defaultNamespace}).ListRoles(ctx, mockCM, false)
	assert.NoError(t, err)
}

func TestStorageClassOperations(t *testing.T) {
	ctx := context.Background()

	reclaim := corev1.PersistentVolumeReclaimDelete
	binding := storagev1.VolumeBindingWaitForFirstConsumer
	expand := true
	sc := &storagev1.StorageClass{
		ObjectMeta:           metav1.ObjectMeta{Name: "standard", Annotations: map[string]string{defaultStorageClassAnnotation: "true"}},
		Provisioner:          "kubernetes.io/aws-ebs",
		ReclaimPolicy:        &reclaim,
		VolumeBindingMode:    &binding,
		AllowVolumeExpansion: &expand,
		Parameters:           map[string]string{"type": "gp3"},
	}

	t.Run("List", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(sc)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := (&StorageClass{}).List(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "standard (default)")
	})

	t.Run("Get", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(sc)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)

		result, err := (&StorageClass{Name: "standard"}).Get(ctx, mockCM)
		assert.NoError(t, err)
		assert.Contains(t, result, "Default: true")
		assert.Contains(t, result, "gp3")

		_, err = (&StorageClass{}).Get(ctx, mockCM)
		assert.Error(t, err)
	})
}
