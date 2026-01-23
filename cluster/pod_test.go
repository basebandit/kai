package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	shellCommand = []interface{}{"/bin/sh", "-c"}
	sleepArgs    = []interface{}{"echo hello; sleep 3600"}
	testLabels   = map[string]interface{}{
		"app": "test",
		"env": "dev",
	}
	testEnv = map[string]interface{}{
		"DEBUG": "true",
		"ENV":   "test",
	}
	ssdNodeSelector = map[string]interface{}{
		"disktype": "ssd",
	}
)

// createNamespace creates a namespace object for testing
func TestPodOperations(t *testing.T) {
	t.Run("CreatePod", testCreatePods)
	t.Run("GetPod", testGetPod)
	t.Run("ListPods", testListPods)
	t.Run("DeletePod", testDeletePod)
	t.Run("StreamPodLogs", testStreamPodLogs)
}

func testCreatePods(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		pod            *Pod
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateCreate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Create basic pod",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: testNamespace,
				Image:     "nginx:latest",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Pod \"test-pod\" created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "test-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test-pod", pod.Name)
				assert.Equal(t, testNamespace, pod.Namespace)
				assert.Equal(t, "nginx:latest", pod.Spec.Containers[0].Image)
			},
		},
		{
			name: "Create pod with custom container name",
			pod: &Pod{
				Name:          "custom-pod",
				Namespace:     testNamespace,
				Image:         "nginx:latest",
				ContainerName: "web-server",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "custom-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "web-server", pod.Spec.Containers[0].Name)
			},
		},
		{
			name: "Create pod with container port TCP",
			pod: &Pod{
				Name:          "port-pod",
				Namespace:     testNamespace,
				Image:         "nginx:latest",
				ContainerPort: "8080/TCP",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "port-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Len(t, pod.Spec.Containers[0].Ports, 1)
				assert.Equal(t, int32(8080), pod.Spec.Containers[0].Ports[0].ContainerPort)
				assert.Equal(t, corev1.ProtocolTCP, pod.Spec.Containers[0].Ports[0].Protocol)
			},
		},
		{
			name: "Create pod with container port UDP",
			pod: &Pod{
				Name:          "udp-pod",
				Namespace:     testNamespace,
				Image:         "nginx:latest",
				ContainerPort: "53/UDP",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "udp-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.ProtocolUDP, pod.Spec.Containers[0].Ports[0].Protocol)
			},
		},
		{
			name: "Create pod with image pull policy",
			pod: &Pod{
				Name:            "policy-pod",
				Namespace:       testNamespace,
				Image:           "nginx:latest",
				ImagePullPolicy: "Always",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "policy-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.PullAlways, pod.Spec.Containers[0].ImagePullPolicy)
			},
		},
		{
			name: "Create pod with restart policy",
			pod: &Pod{
				Name:          "restart-pod",
				Namespace:     testNamespace,
				Image:         "nginx:latest",
				RestartPolicy: "OnFailure",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "restart-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, corev1.RestartPolicyOnFailure, pod.Spec.RestartPolicy)
			},
		},
		{
			name: "Create pod with service account",
			pod: &Pod{
				Name:           "sa-pod",
				Namespace:      testNamespace,
				Image:          "nginx:latest",
				ServiceAccount: "test-sa",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "sa-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test-sa", pod.Spec.ServiceAccountName)
			},
		},
		{
			name: "Create pod with command and args",
			pod: &Pod{
				Name:      "cmd-pod",
				Namespace: testNamespace,
				Image:     "busybox:latest",
				Command:   shellCommand,
				Args:      sleepArgs,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "cmd-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, []string{"/bin/sh", "-c"}, pod.Spec.Containers[0].Command)
				assert.Equal(t, []string{"echo hello; sleep 3600"}, pod.Spec.Containers[0].Args)
			},
		},
		{
			name: "Create pod with labels",
			pod: &Pod{
				Name:      "labeled-pod",
				Namespace: testNamespace,
				Image:     "nginx:latest",
				Labels:    testLabels,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "labeled-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "test", pod.Labels["app"])
				assert.Equal(t, "dev", pod.Labels["env"])
			},
		},
		{
			name: "Create pod with environment variables",
			pod: &Pod{
				Name:      "env-pod",
				Namespace: testNamespace,
				Image:     "nginx:latest",
				Env:       testEnv,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "env-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Len(t, pod.Spec.Containers[0].Env, 2)
				envMap := make(map[string]string)
				for _, env := range pod.Spec.Containers[0].Env {
					envMap[env.Name] = env.Value
				}
				assert.Equal(t, "true", envMap["DEBUG"])
				assert.Equal(t, "test", envMap["ENV"])
			},
		},
		{
			name: "Create pod with node selector",
			pod: &Pod{
				Name:         "node-selector-pod",
				Namespace:    testNamespace,
				Image:        "nginx:latest",
				NodeSelector: ssdNodeSelector,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "node-selector-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "ssd", pod.Spec.NodeSelector["disktype"])
			},
		},
		{
			name: "Create pod with image pull secrets",
			pod: &Pod{
				Name:             "pull-secret-pod",
				Namespace:        testNamespace,
				Image:            "nginx:latest",
				ImagePullSecrets: []interface{}{"my-registry-secret"},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "pull-secret-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Len(t, pod.Spec.ImagePullSecrets, 1)
				assert.Equal(t, "my-registry-secret", pod.Spec.ImagePullSecrets[0].Name)
			},
		},
		{
			name: "Create pod with all attributes",
			pod: &Pod{
				Name:             "full-pod",
				Namespace:        testNamespace,
				Image:            "nginx:latest",
				ContainerName:    "custom-container",
				ContainerPort:    "8080/TCP",
				ImagePullPolicy:  "Always",
				RestartPolicy:    "OnFailure",
				ServiceAccount:   "test-sa",
				Command:          shellCommand,
				Args:             sleepArgs,
				Labels:           testLabels,
				Env:              testEnv,
				NodeSelector:     ssdNodeSelector,
				ImagePullSecrets: []interface{}{"registry-secret"},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "created successfully",
			validateCreate: func(t *testing.T, client kubernetes.Interface) {
				pod, err := client.CoreV1().Pods(testNamespace).Get(ctx, "full-pod", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "full-pod", pod.Name)
				assert.Equal(t, "custom-container", pod.Spec.Containers[0].Name)
				assert.Equal(t, int32(8080), pod.Spec.Containers[0].Ports[0].ContainerPort)
				assert.Equal(t, corev1.PullAlways, pod.Spec.Containers[0].ImagePullPolicy)
				assert.Equal(t, corev1.RestartPolicyOnFailure, pod.Spec.RestartPolicy)
				assert.Equal(t, "test-sa", pod.Spec.ServiceAccountName)
			},
		},
		{
			name: "Missing image",
			pod: &Pod{
				Name:      "no-image-pod",
				Namespace: testNamespace,
				Image:     "",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "failed to create pod: image cannot be empty",
		},
		{
			name: "Namespace not found",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: nonexistentNS,
				Image:     "nginx:latest",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.pod.Create(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateCreate != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateCreate(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testGetPod(t *testing.T) {
	ctx := context.Background()

	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: testNamespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	testCases := []struct {
		name           string
		pod            *Pod
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing pod",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(existingPod, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "test-pod",
			expectedError:  "",
		},
		{
			name: "Pod not found",
			pod: &Pod{
				Name:      "nonexistent-pod",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "pod 'nonexistent-pod' not found",
		},
		{
			name: "Namespace not found",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: nonexistentNS,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace 'nonexistent-namespace' not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.pod.Get(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testListPods(t *testing.T) {
	ctx := context.Background()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: testNamespace,
			Labels:    map[string]string{"app": "test"},
		},
	}
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod3",
			Namespace: "other-namespace",
			Labels:    map[string]string{"app": "other"},
		},
	}

	testCases := []struct {
		name              string
		pod               *Pod
		labelSelector     string
		fieldSelector     string
		limit             int64
		setupMock         func(*testmocks.MockClusterManager)
		expectedContent   []string
		unexpectedContent []string
		expectedError     string
	}{
		{
			name: "List pods in namespace",
			pod: &Pod{
				Namespace: testNamespace,
			},
			limit: 10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(pod1, pod2, pod3, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent:   []string{"pod1", "pod2"},
			unexpectedContent: []string{"pod3"},
		},
		{
			name: "List pods with label selector",
			pod: &Pod{
				Namespace: testNamespace,
			},
			labelSelector: "app=test",
			limit:         10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(pod1, pod2, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"pod1", "pod2"},
		},
		{
			name: "List pods in all namespaces",
			pod: &Pod{
				Namespace: "",
			},
			limit: 10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(pod1, pod2, pod3)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"pod1", "pod2", "pod3"},
		},
		{
			name: "List pods with limit",
			pod: &Pod{
				Namespace: testNamespace,
			},
			limit: 1,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(pod1, pod2, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedContent: []string{"pod"},
		},
		{
			name: "Namespace not found",
			pod: &Pod{
				Namespace: nonexistentNS,
			},
			limit: 10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "No pods found",
			pod: &Pod{
				Namespace: testNamespace,
			},
			limit: 10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no pods found",
		},
		{
			name: "No pods match label selector",
			pod: &Pod{
				Namespace: testNamespace,
			},
			labelSelector: "app=nonexistent",
			limit:         10,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(pod1, pod2, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no pods found matching the specified selectors",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.pod.List(ctx, mockCM, tc.limit, tc.labelSelector, tc.fieldSelector)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)

				for _, expected := range tc.expectedContent {
					assert.Contains(t, result, expected)
				}

				for _, unexpected := range tc.unexpectedContent {
					assert.NotContains(t, result, unexpected)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testDeletePod(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		pod            *Pod
		force          bool
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing pod",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: testNamespace,
			},
			force: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: testNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(pod, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Successfully delete pod \"test-pod\"",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().Pods(testNamespace).Get(ctx, "test-pod", metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Force delete pod",
			pod: &Pod{
				Name:      "force-pod",
				Namespace: testNamespace,
			},
			force: true,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "force-pod",
						Namespace: testNamespace,
					},
				}
				fakeClient := fake.NewSimpleClientset(pod, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Successfully delete pod \"force-pod\"",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.CoreV1().Pods(testNamespace).Get(ctx, "force-pod", metav1.GetOptions{})
				assert.Error(t, err)
			},
		},
		{
			name: "Pod not found",
			pod: &Pod{
				Name:      "nonexistent-pod",
				Namespace: testNamespace,
			},
			force: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "pod \"nonexistent-pod\" not found",
		},
		{
			name: "Namespace not found",
			pod: &Pod{
				Name:      "test-pod",
				Namespace: nonexistentNS,
			},
			force: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.pod.Delete(ctx, mockCM, tc.force)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateDelete != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateDelete(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testStreamPodLogs(t *testing.T) {
	ctx := context.Background()

	runningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-pod",
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "container1"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pendingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-pod",
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "container1"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	podWithNoContainers := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-container-pod",
			Namespace: testNamespace,
		},
		Spec:   corev1.PodSpec{},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	testCases := []struct {
		name          string
		pod           *Pod
		tailLines     int64
		previous      bool
		since         *time.Duration
		setupMock     func(*testmocks.MockClusterManager)
		expectedError string
	}{
		{
			name: "Pod not found",
			pod: &Pod{
				Name:          "nonexistent-pod",
				Namespace:     testNamespace,
				ContainerName: "container1",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "pod 'nonexistent-pod' not found",
		},
		{
			name: "Namespace not found",
			pod: &Pod{
				Name:          "test-pod",
				Namespace:     nonexistentNS,
				ContainerName: "container1",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent-namespace\" not found",
		},
		{
			name: "Pod not running without previous flag",
			pod: &Pod{
				Name:          "pending-pod",
				Namespace:     testNamespace,
				ContainerName: "container1",
			},
			previous: false,
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(pendingPod, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "pod 'pending-pod' is in 'Pending' state",
		},
		{
			name: "Container not found",
			pod: &Pod{
				Name:          "running-pod",
				Namespace:     testNamespace,
				ContainerName: "nonexistent-container",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(runningPod, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "container 'nonexistent-container' not found",
		},
		{
			name: "Pod has no containers",
			pod: &Pod{
				Name:          "no-container-pod",
				Namespace:     testNamespace,
				ContainerName: "",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(podWithNoContainers, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no containers found in pod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			_, err := tc.pod.StreamLogs(ctx, mockCM, tc.tailLines, tc.previous, tc.since)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}

			mockCM.AssertExpectations(t)
		})
	}
}
