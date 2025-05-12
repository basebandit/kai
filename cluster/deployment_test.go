package clustermanager

import (
	"context"
	"testing"

	"github.com/basebandit/kai"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// TestDeploymentOperations groups all deployment-related tests
func TestDeploymentOperations(t *testing.T) {
	t.Run("ListDeployments", testListDeployments)
	t.Run("CreateDeployment", testCreateDeployment)
}

func testListDeployments(t *testing.T) {
	cm := New()
	ctx := context.Background()

	// Create test deployments
	objects := []runtime.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment1",
				Namespace: "test-namespace",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment2",
				Namespace: "test-namespace",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment3",
				Namespace: "other-namespace",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other-namespace",
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(objects...)
	cm.clients["test-cluster"] = fakeClient
	cm.currentContext = "test-cluster"

	// Test listing deployments in a specific namespace
	result, err := cm.ListDeployments(ctx, false, "", "test-namespace")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.Contains(t, result, "deployment2")
	assert.NotContains(t, result, "deployment3")

	// Test listing deployments with a label selector
	result, err = cm.ListDeployments(ctx, false, "app=test", "test-namespace")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.NotContains(t, result, "deployment2")

	// Test listing deployments in all namespaces
	result, err = cm.ListDeployments(ctx, true, "", "")
	assert.NoError(t, err)
	assert.Contains(t, result, "deployment1")
	assert.Contains(t, result, "deployment2")
	assert.Contains(t, result, "deployment3")
}

func testCreateDeployment(t *testing.T) {
	cm := New()
	ctx := context.Background()

	// Create a test scheme with needed types registered
	scheme := runtime.NewScheme()

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "DeploymentList",
	}

	// Create fake client with custom list kinds
	listKinds := map[schema.GroupVersionResource]string{
		gvr: gvk.Kind,
	}

	// Register the list kinds for deployments since the fake client doesn't know how to
	// list Deployments by default
	fakeDynamicClient := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds)

	// Add the dynamic client to the cluster manager
	cm.dynamicClients["test-cluster"] = fakeDynamicClient
	cm.currentContext = "test-cluster"

	// Test basic deployment creation
	deploymentParams := kai.DeploymentParams{
		Name:            "test-deployment",
		Image:           "nginx:latest",
		Namespace:       "default",
		ContainerPort:   "80/TCP",
		ImagePullPolicy: "IfNotPresent",
		Replicas:        3,
	}

	result, err := cm.CreateDeployment(ctx, deploymentParams)
	assert.NoError(t, err)
	assert.Contains(t, result, "created successfully")
	assert.Contains(t, result, deploymentParams.Name)

	// Verify the deployment was created
	deployment, err := fakeDynamicClient.Resource(gvr).Namespace(deploymentParams.Namespace).Get(ctx, deploymentParams.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, deployment)

	// Verify basic properties
	name, exists, err := unstructured.NestedString(deployment.Object, "metadata", "name")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, deploymentParams.Name, name)

	// Verify replicas
	replicas, exists, err := unstructured.NestedFloat64(deployment.Object, "spec", "replicas")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, deploymentParams.Replicas, replicas)

	// Now, let's try listing deployments (this should work with our custom list kinds)
	deploymentList, err := fakeDynamicClient.Resource(gvr).Namespace(deploymentParams.Namespace).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, deploymentList)
	assert.GreaterOrEqual(t, len(deploymentList.Items), 1)

	// Test deployment with environment variables
	envDeploymentParams := kai.DeploymentParams{
		Name:            "test-env-deployment",
		Image:           "app:v1",
		Namespace:       "default",
		ContainerPort:   "8080",
		ImagePullPolicy: "Always",
		Replicas:        1,
		Env: map[string]interface{}{
			"DEBUG": "true",
			"PORT":  "8080",
		},
	}

	result, err = cm.CreateDeployment(ctx, envDeploymentParams)
	assert.NoError(t, err)
	assert.Contains(t, result, "created successfully")

	// Get the deployment directly
	envDeployment, err := fakeDynamicClient.Resource(gvr).Namespace(envDeploymentParams.Namespace).Get(ctx, envDeploymentParams.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// Verify container and env vars
	containers, exists, err := unstructured.NestedSlice(envDeployment.Object, "spec", "template", "spec", "containers")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NotEmpty(t, containers)

	container := containers[0].(map[string]interface{})
	env, exists, err := unstructured.NestedSlice(container, "env")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, len(envDeploymentParams.Env), len(env))

	// Test deployment with pull secrets
	secretsDeploymentParams := kai.DeploymentParams{
		Name:             "test-secrets-deployment",
		Image:            "private/repo:latest",
		Namespace:        "default",
		Replicas:         1,
		ImagePullSecrets: []interface{}{"docker-registry"},
	}

	result, err = cm.CreateDeployment(ctx, secretsDeploymentParams)
	assert.NoError(t, err)
	assert.Contains(t, result, "created successfully")

	// Get the deployment directly
	secretsDeployment, err := fakeDynamicClient.Resource(gvr).Namespace(secretsDeploymentParams.Namespace).Get(ctx, secretsDeploymentParams.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// Verify imagePullSecrets
	secrets, exists, err := unstructured.NestedSlice(secretsDeployment.Object, "spec", "template", "spec", "imagePullSecrets")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, len(secretsDeploymentParams.ImagePullSecrets), len(secrets))

	// Test deployment with custom labels
	labelsDeploymentParams := kai.DeploymentParams{
		Name:      "test-labels-deployment",
		Image:     "busybox:latest",
		Namespace: "default",
		Replicas:  1,
		Labels: map[string]interface{}{
			"environment": "testing",
			"tier":        "backend",
		},
	}

	result, err = cm.CreateDeployment(ctx, labelsDeploymentParams)
	assert.NoError(t, err)
	assert.Contains(t, result, "created successfully")

	// Get the deployment directly
	labelsDeployment, err := fakeDynamicClient.Resource(gvr).Namespace(labelsDeploymentParams.Namespace).Get(ctx, labelsDeploymentParams.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// Verify labels
	labels, exists, err := unstructured.NestedMap(labelsDeployment.Object, "metadata", "labels")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check for app label (default)
	assert.Equal(t, labelsDeploymentParams.Name, labels["app"])

	// Check for custom labels
	for k, v := range labelsDeploymentParams.Labels {
		assert.Equal(t, v, labels[k])
	}

	// Test error case - no dynamic client
	errorCm := New()
	_, err = errorCm.CreateDeployment(ctx, deploymentParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get a dynamic client")
}
