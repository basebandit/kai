package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRBACRoles(t *testing.T) {
	ctx := context.Background()
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: defaultNamespace},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}, ResourceNames: []string{"p1"},
		}},
	}
	fakeClient := fake.NewSimpleClientset(role)
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	list, err := (&RBAC{}).ListRoles(ctx, mockCM, false)
	assert.NoError(t, err)
	assert.Contains(t, list, "reader")

	all, err := (&RBAC{}).ListRoles(ctx, mockCM, true)
	assert.NoError(t, err)
	assert.Contains(t, all, "reader")

	get, err := (&RBAC{Name: "reader"}).GetRole(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, get, "pods")
	assert.Contains(t, get, "p1")

	_, err = (&RBAC{}).GetRole(ctx, mockCM)
	assert.Error(t, err)
}

func TestRBACClusterRoles(t *testing.T) {
	ctx := context.Background()
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "admin"},
		Rules:      []rbacv1.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}}},
	}
	fakeClient := fake.NewSimpleClientset(cr)
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)

	list, err := (&RBAC{}).ListClusterRoles(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, list, "admin")

	get, err := (&RBAC{Name: "admin"}).GetClusterRole(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, get, "ClusterRole: admin")

	_, err = (&RBAC{}).GetClusterRole(ctx, mockCM)
	assert.Error(t, err)
}

func TestRBACBindings(t *testing.T) {
	ctx := context.Background()
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "rb1", Namespace: defaultNamespace},
		RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "sa1", Namespace: defaultNamespace}},
	}
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crb1"},
		RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: "admin"},
		Subjects:   []rbacv1.Subject{{Kind: "User", Name: "alice"}},
	}
	fakeClient := fake.NewSimpleClientset(rb, crb)
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	rbList, err := (&RBAC{}).ListRoleBindings(ctx, mockCM, false)
	assert.NoError(t, err)
	assert.Contains(t, rbList, "rb1")

	_, err = (&RBAC{}).ListRoleBindings(ctx, mockCM, true)
	assert.NoError(t, err)

	rbGet, err := (&RBAC{Name: "rb1"}).GetRoleBinding(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, rbGet, "ServiceAccount:default/sa1")

	_, err = (&RBAC{}).GetRoleBinding(ctx, mockCM)
	assert.Error(t, err)

	crbList, err := (&RBAC{}).ListClusterRoleBindings(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, crbList, "crb1")

	crbGet, err := (&RBAC{Name: "crb1"}).GetClusterRoleBinding(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, crbGet, "User:alice")

	_, err = (&RBAC{}).GetClusterRoleBinding(ctx, mockCM)
	assert.Error(t, err)
}

func TestRBACServiceAccounts(t *testing.T) {
	ctx := context.Background()
	automount := true
	sa := &corev1.ServiceAccount{
		ObjectMeta:                   metav1.ObjectMeta{Name: "sa1", Namespace: defaultNamespace},
		Secrets:                      []corev1.ObjectReference{{Name: "sa1-token"}},
		AutomountServiceAccountToken: &automount,
	}
	fakeClient := fake.NewSimpleClientset(sa)
	mockCM := testmocks.NewMockClusterManager()
	mockCM.On("GetCurrentClient").Return(fakeClient, nil)
	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)

	list, err := (&RBAC{}).ListServiceAccounts(ctx, mockCM, false)
	assert.NoError(t, err)
	assert.Contains(t, list, "sa1")

	_, err = (&RBAC{}).ListServiceAccounts(ctx, mockCM, true)
	assert.NoError(t, err)

	get, err := (&RBAC{Name: "sa1"}).GetServiceAccount(ctx, mockCM)
	assert.NoError(t, err)
	assert.Contains(t, get, "sa1-token")

	_, err = (&RBAC{}).GetServiceAccount(ctx, mockCM)
	assert.Error(t, err)
}
