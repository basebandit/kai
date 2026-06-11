package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterRBACTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(10)
	RegisterRBACTools(mockServer, mockCM)
	mockServer.AssertExpectations(t)
}

func TestRBACHandlers(t *testing.T) {
	ctx := context.Background()

	newCM := func() (*testmocks.MockClusterManager, *fake.Clientset) {
		fakeClient := fake.NewSimpleClientset(
			&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: defaultNamespace}},
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "rb1", Namespace: defaultNamespace}, RoleRef: rbacv1.RoleRef{Kind: "Role", Name: "r1"}},
			&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "cr1"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "crb1"}, RoleRef: rbacv1.RoleRef{Kind: "ClusterRole", Name: "cr1"}},
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa1", Namespace: defaultNamespace}},
		)
		mockCM := testmocks.NewMockClusterManager()
		mockCM.On("GetCurrentClient").Return(fakeClient, nil)
		mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
		return mockCM, fakeClient
	}

	listCases := []struct {
		kind, want string
	}{
		{"role", "r1"},
		{"rolebinding", "rb1"},
		{"clusterrole", "cr1"},
		{"clusterrolebinding", "crb1"},
		{"serviceaccount", "sa1"},
	}
	for _, tc := range listCases {
		mockCM, _ := newCM()
		r, err := rbacListHandler(mockCM, tc.kind)(ctx, toolRequest(nil))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), tc.want)
	}

	getCases := []struct {
		kind, name, want string
	}{
		{"role", "r1", "Role: r1"},
		{"rolebinding", "rb1", "RoleBinding: rb1"},
		{"clusterrole", "cr1", "ClusterRole: cr1"},
		{"clusterrolebinding", "crb1", "ClusterRoleBinding: crb1"},
		{"serviceaccount", "sa1", "ServiceAccount: sa1"},
	}
	for _, tc := range getCases {
		mockCM, _ := newCM()
		r, err := rbacGetHandler(mockCM, tc.kind)(ctx, toolRequest(map[string]interface{}{"name": tc.name}))
		assert.NoError(t, err)
		assert.Contains(t, resultText(t, r), tc.want)
	}

	t.Run("GetMissingName", func(t *testing.T) {
		mockCM, _ := newCM()
		r, err := rbacGetHandler(mockCM, "role")(ctx, toolRequest(map[string]interface{}{}))
		assert.NoError(t, err)
		assert.Equal(t, errMissingName, resultText(t, r))
	})
}
