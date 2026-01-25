package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{45 * time.Second, "45s"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{2 * 24 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		assert.Equal(t, tt.expected, result)
	}
}

func TestConvertToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "value",
				"int":    123,
				"bool":   true,
			},
			expected: map[string]string{
				"string": "value",
				"int":    "123",
				"bool":   "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToStringMap(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, len(tt.expected), len(result))
				for k, v := range tt.expected {
					assert.Equal(t, v, result[k])
				}
			}
		})
	}
}

func TestFormatNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-ns",
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Labels:            map[string]string{"env": "test"},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	result := formatNamespace(ns)
	assert.Contains(t, result, "Namespace: test-ns")
	assert.Contains(t, result, "Status: Active")
	assert.Contains(t, result, "- env: test")
}

func TestFormatNamespaceList(t *testing.T) {
	nsList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
				},
				Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
			},
		},
	}

	result := formatNamespaceList(nsList, "")
	assert.Contains(t, result, "Namespaces:")
	assert.Contains(t, result, "• default: Status=Active, Age=1d")
	assert.Contains(t, result, "Total: 1 namespace(s)")

	resultWithSelector := formatNamespaceList(nsList, "env=prod")
	assert.Contains(t, resultWithSelector, "matching label selector 'env=prod'")
}

func TestFormatNamespaceListEmptyStatus(t *testing.T) {
	nsList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Status: corev1.NamespaceStatus{Phase: ""},
			},
		},
	}

	result := formatNamespaceList(nsList, "")
	assert.Contains(t, result, "Status=Active")
}

func TestFormatPod(t *testing.T) {
	t.Run("Format basic pod", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				CreationTimestamp: metav1.Time{
					Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-1",
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				PodIP: "10.0.0.1",
			},
		}

		result := formatPod(pod)
		assert.Contains(t, result, "test-pod")
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "Running")
		assert.Contains(t, result, "node-1")
		assert.Contains(t, result, "10.0.0.1")
		assert.Contains(t, result, "nginx")
		assert.Contains(t, result, "nginx:latest")
	})

	t.Run("Format pod with container statuses", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:v1"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "app",
						Ready:        true,
						RestartCount: 2,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{
								StartedAt: metav1.Time{Time: time.Now()},
							},
						},
					},
				},
			},
		}

		result := formatPod(pod)
		assert.Contains(t, result, "Ready")
		assert.Contains(t, result, "Restarts: 2")
		assert.Contains(t, result, "Started At:")
	})

	t.Run("Format pod with waiting container", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "waiting-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:v1"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "app",
						Ready: false,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "ImagePullBackOff",
								Message: "Failed to pull image",
							},
						},
					},
				},
			},
		}

		result := formatPod(pod)
		assert.Contains(t, result, "Waiting:")
		assert.Contains(t, result, "ImagePullBackOff")
	})

	t.Run("Format pod with terminated container", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terminated-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:v1"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "app",
						Ready: false,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:   "Error",
								Message:  "Container crashed",
								ExitCode: 1,
							},
						},
					},
				},
			},
		}

		result := formatPod(pod)
		assert.Contains(t, result, "Terminated:")
		assert.Contains(t, result, "Exit Code: 1")
	})

	t.Run("Format pod with labels", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "labeled-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app":     "myapp",
					"version": "v1",
				},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:v1"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}

		result := formatPod(pod)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "app")
	})
}

func TestFormatPodList(t *testing.T) {
	t.Run("Format pod list", func(t *testing.T) {
		podList := &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
					},
					Spec:   corev1.PodSpec{NodeName: "node-1"},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-2",
						Namespace:         "kube-system",
						CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
					},
					Spec:   corev1.PodSpec{NodeName: "node-2"},
					Status: corev1.PodStatus{Phase: corev1.PodPending},
				},
			},
		}

		result := formatPodList(podList, true, 0, "")
		assert.Contains(t, result, "pod-1")
		assert.Contains(t, result, "pod-2")
		assert.Contains(t, result, "Running")
		assert.Contains(t, result, "Pending")
	})

	t.Run("Format pod list with limit", func(t *testing.T) {
		podList := &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				},
			},
		}

		result := formatPodList(podList, false, 10, "Result header")
		assert.Contains(t, result, "Result header")
		assert.Contains(t, result, "Total: 1 pod(s)")
	})
}

func TestFormatService(t *testing.T) {
	t.Run("Format ClusterIP service", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-service",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.96.0.1",
				Ports: []corev1.ServicePort{
					{Name: "http", Port: 80, Protocol: corev1.ProtocolTCP},
				},
				Selector: map[string]string{"app": "myapp"},
			},
		}

		result := formatService(svc)
		assert.Contains(t, result, "test-service")
		assert.Contains(t, result, "ClusterIP")
		assert.Contains(t, result, "10.96.0.1")
		assert.Contains(t, result, "http")
		assert.Contains(t, result, "80")
	})

	t.Run("Format LoadBalancer service", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "lb-service",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeLoadBalancer,
				ClusterIP: "10.96.0.2",
				Ports:     []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}},
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{{IP: "203.0.113.1"}},
				},
			},
		}

		result := formatService(svc)
		assert.Contains(t, result, "LoadBalancer")
		assert.Contains(t, result, "203.0.113.1")
	})

	t.Run("Format service with session affinity", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "sticky-service",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.ServiceSpec{
				Type:            corev1.ServiceTypeClusterIP,
				SessionAffinity: corev1.ServiceAffinityClientIP,
				Ports:           []corev1.ServicePort{{Port: 80}},
			},
		}

		result := formatService(svc)
		assert.Contains(t, result, "sticky-service")
		assert.Contains(t, result, "ClusterIP")
	})

	t.Run("Format service with labels", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-service",
				Namespace:         "default",
				Labels:            map[string]string{"env": "prod"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.ServiceSpec{
				Type:  corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{{Port: 80}},
			},
		}

		result := formatService(svc)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "env")
	})
}

func TestFormatServiceList(t *testing.T) {
	t.Run("Format service list", func(t *testing.T) {
		svcList := &corev1.ServiceList{
			Items: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "svc-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: corev1.ServiceSpec{
						Type:      corev1.ServiceTypeClusterIP,
						ClusterIP: "10.96.0.1",
						Ports:     []corev1.ServicePort{{Port: 80}},
					},
				},
			},
		}

		result := formatServiceList(svcList, false)
		assert.Contains(t, result, "svc-1")
		assert.Contains(t, result, "ClusterIP")
	})

	t.Run("Format service list all namespaces", func(t *testing.T) {
		svcList := &corev1.ServiceList{
			Items: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "svc-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: corev1.ServiceSpec{
						Type:  corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{{Port: 80}},
					},
				},
			},
		}

		result := formatServiceList(svcList, true)
		// assert.Contains(t, result, "NAMESPACE")
		assert.Contains(t, result, "default")
	})
}

func TestFormatConfigMap(t *testing.T) {
	t.Run("Format basic configmap", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-cm",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		result := formatConfigMap(cm)
		assert.Contains(t, result, "test-cm")
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "key1")
		assert.Contains(t, result, "value1")
	})

	t.Run("Format configmap with binary data", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "binary-cm",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			BinaryData: map[string][]byte{
				"file.bin": {0x01, 0x02, 0x03},
			},
		}

		result := formatConfigMap(cm)
		assert.Contains(t, result, "binary-cm")
		assert.Contains(t, result, "file.bin")
	})

	t.Run("Format configmap with labels", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-cm",
				Namespace:         "default",
				Labels:            map[string]string{"app": "myapp"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Data: map[string]string{"key": "value"},
		}

		result := formatConfigMap(cm)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "app")
	})
}

func TestFormatConfigMapList(t *testing.T) {
	cmList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cm-1",
					Namespace:         "default",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Data: map[string]string{"key": "value"},
			},
		},
	}

	result := formatConfigMapList(cmList, false)
	assert.Contains(t, result, "cm-1")
}

func TestFormatSecret(t *testing.T) {
	t.Run("Format basic secret", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              secretName,
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password": []byte("secret123"),
			},
		}

		result := formatSecret(secret)
		assert.Contains(t, result, secretName)
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "password")
	})

	t.Run("Format secret with labels", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-secret",
				Namespace:         "default",
				Labels:            map[string]string{"env": "prod"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"key": []byte("value")},
		}

		result := formatSecret(secret)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "env")
	})
}

func TestFormatSecretList(t *testing.T) {
	secretList := &corev1.SecretList{
		Items: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "secret-1",
					Namespace:         "default",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}

	result := formatSecretList(secretList, false)
	assert.Contains(t, result, "secret-1")
}

func TestFormatJob(t *testing.T) {
	t.Run("Format basic job", func(t *testing.T) {
		completions := int32(1)
		parallelism := int32(1)
		template := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}, {}},
			},
		}
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Parallelism: &parallelism,
				Template:    template,
			},
			Status: batchv1.JobStatus{
				StartTime: &metav1.Time{Time: time.Now()},
				Active:    0,
				Succeeded: 1,
				Failed:    0,
			},
		}

		result := formatJob(job)
		assert.Contains(t, result, "test-job")
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "Succeeded: 1")
	})

	t.Run("Format job with completion time", func(t *testing.T) {
		completionTime := metav1.Time{Time: time.Now()}
		template := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}, {}},
			},
		}
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "completed-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
			},
			Spec: batchv1.JobSpec{
				Template: template,
			},
			Status: batchv1.JobStatus{
				StartTime:      &metav1.Time{Time: time.Now()},
				Succeeded:      1,
				CompletionTime: &completionTime,
			},
		}

		result := formatJob(job)
		assert.Contains(t, result, "Completion Time:")
	})

	t.Run("Format job with labels", func(t *testing.T) {
		template := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{}, {}},
			},
		}
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-job",
				Namespace:         "default",
				Labels:            map[string]string{"batch": "true"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Template: template,
			},
			Status: batchv1.JobStatus{},
		}

		result := formatJob(job)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "batch")
	})
}

func TestFormatJobList(t *testing.T) {
	t.Run("Format job list", func(t *testing.T) {
		jobList := &batchv1.JobList{
			Items: []batchv1.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "job-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Status: batchv1.JobStatus{Active: 1},
				},
			},
		}

		result := formatJobList(jobList, false)
		assert.Contains(t, result, "job-1")
		assert.Contains(t, result, "Active: 1")
	})

	t.Run("Format job list all namespaces", func(t *testing.T) {
		jobList := &batchv1.JobList{
			Items: []batchv1.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "job-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Status: batchv1.JobStatus{},
				},
			},
		}

		result := formatJobList(jobList, true)
		// assert.Contains(t, result, "NAMESPACE")
		assert.Contains(t, result, "default")
	})
}

func TestFormatDeployment(t *testing.T) {
	t.Run("Format basic deployment", func(t *testing.T) {
		replicas := int32(3)
		labelSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"selector": "test-label",
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-deployment",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &labelSelector,
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				UpdatedReplicas:   3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
			},
		}

		result := formatDeployment(deployment)
		assert.Contains(t, result, "test-deployment")
		assert.Contains(t, result, "default")
		assert.Contains(t, result, "Ready: 3")
	})

	t.Run("Format deployment with conditions", func(t *testing.T) {
		labelSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test-1": "test-label",
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "cond-deployment",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &labelSelector,
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:    appsv1.DeploymentAvailable,
						Status:  corev1.ConditionTrue,
						Reason:  "MinimumReplicasAvailable",
						Message: "Deployment has minimum availability.",
					},
				},
			},
		}

		result := formatDeployment(deployment)
		assert.Contains(t, result, "Conditions:")
		assert.Contains(t, result, "Available")
	})

	t.Run("Format deployment with labels", func(t *testing.T) {
		labelSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"selector": "test-label",
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-deployment",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &labelSelector,
			},
			Status: appsv1.DeploymentStatus{},
		}

		result := formatDeployment(deployment)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "app")
	})
}

func TestFormatDeploymentList(t *testing.T) {
	deploymentList := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "deploy-1",
					Namespace:         "default",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
			},
		},
	}

	result := formatDeploymentList(deploymentList)
	assert.Contains(t, result, "deploy-1")
}

func TestConvertToStringSlice(t *testing.T) {
	t.Run("Convert valid slice", func(t *testing.T) {
		input := []interface{}{"foo", "bar", "baz"}
		result := convertToStringSlice(input)
		assert.Equal(t, []string{"foo", "bar", "baz"}, result)
	})

	t.Run("Convert nil slice", func(t *testing.T) {
		result := convertToStringSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("Convert slice with mixed types", func(t *testing.T) {
		input := []interface{}{"foo", 123, "bar"}
		result := convertToStringSlice(input)
		assert.Equal(t, []string{"foo", "bar"}, result)
	})

	t.Run("Convert empty slice", func(t *testing.T) {
		input := []interface{}{}
		result := convertToStringSlice(input)
		assert.Equal(t, []string{}, result)
	})
}

func TestConvertToEnvVars(t *testing.T) {
	t.Run("Convert valid map", func(t *testing.T) {
		input := map[string]interface{}{
			"KEY1": "value1",
			"KEY2": "value2",
		}
		result := convertToEnvVars(input)
		assert.Len(t, result, 2)

		envMap := make(map[string]string)
		for _, env := range result {
			envMap[env.Name] = env.Value
		}
		assert.Equal(t, "value1", envMap["KEY1"])
		assert.Equal(t, "value2", envMap["KEY2"])
	})

	t.Run("Convert nil map", func(t *testing.T) {
		result := convertToEnvVars(nil)
		assert.Nil(t, result)
	})

	t.Run("Convert map with non-string values", func(t *testing.T) {
		input := map[string]interface{}{
			"PORT":    8080,
			"ENABLED": true,
		}
		result := convertToEnvVars(input)
		assert.Len(t, result, 2)

		envMap := make(map[string]string)
		for _, env := range result {
			envMap[env.Name] = env.Value
		}
		assert.Equal(t, "8080", envMap["PORT"])
		assert.Equal(t, "true", envMap["ENABLED"])
	})

	t.Run("Convert empty map", func(t *testing.T) {
		input := map[string]interface{}{}
		result := convertToEnvVars(input)
		assert.Equal(t, []corev1.EnvVar{}, result)
	})
}

func TestConvertToLocalObjectReferences(t *testing.T) {
	t.Run("Convert valid slice", func(t *testing.T) {
		input := []interface{}{"secret1", "secret2"}
		result := convertToLocalObjectReferences(input)
		assert.Len(t, result, 2)
		assert.Equal(t, "secret1", result[0].Name)
		assert.Equal(t, "secret2", result[1].Name)
	})

	t.Run("Convert nil slice", func(t *testing.T) {
		result := convertToLocalObjectReferences(nil)
		assert.Nil(t, result)
	})

	t.Run("Convert slice with mixed types", func(t *testing.T) {
		input := []interface{}{"secret1", 123, "secret2"}
		result := convertToLocalObjectReferences(input)
		assert.Len(t, result, 2)
		assert.Equal(t, "secret1", result[0].Name)
		assert.Equal(t, "secret2", result[1].Name)
	})

	t.Run("Convert empty slice", func(t *testing.T) {
		input := []interface{}{}
		result := convertToLocalObjectReferences(input)
		assert.Equal(t, []corev1.LocalObjectReference{}, result)
	})
}

func TestFormatCronJob(t *testing.T) {
	t.Run("Format basic cronjob", func(t *testing.T) {
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-cronjob",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Spec: batchv1.CronJobSpec{
				Schedule:          "*/5 * * * *",
				ConcurrencyPolicy: batchv1.AllowConcurrent,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "test", Image: "busybox:latest"}},
							},
						},
					},
				},
			},
			Status: batchv1.CronJobStatus{
				Active: []corev1.ObjectReference{},
			},
		}

		result := formatCronJob(cronJob)
		assert.Contains(t, result, "CronJob: test-cronjob")
		assert.Contains(t, result, "Namespace: default")
		assert.Contains(t, result, "Schedule: */5 * * * *")
		assert.Contains(t, result, "Suspend: No")
		assert.Contains(t, result, "Concurrency Policy: Allow")
		assert.Contains(t, result, "Active Jobs: 0")
		assert.Contains(t, result, "Image: busybox:latest")
	})

	t.Run("Format suspended cronjob", func(t *testing.T) {
		suspend := true
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "suspended-cronjob",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				Suspend:  &suspend,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "test", Image: "nginx"}},
							},
						},
					},
				},
			},
		}

		result := formatCronJob(cronJob)
		assert.Contains(t, result, "Suspend: Yes")
	})

	t.Run("Format cronjob with history limits", func(t *testing.T) {
		successLimit := int32(5)
		failedLimit := int32(3)
		startingDeadline := int64(100)
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "limits-cronjob",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule:                   "0 * * * *",
				SuccessfulJobsHistoryLimit: &successLimit,
				FailedJobsHistoryLimit:     &failedLimit,
				StartingDeadlineSeconds:    &startingDeadline,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "test", Image: "alpine"}},
							},
						},
					},
				},
			},
		}

		result := formatCronJob(cronJob)
		assert.Contains(t, result, "Successful Jobs History Limit: 5")
		assert.Contains(t, result, "Failed Jobs History Limit: 3")
		assert.Contains(t, result, "Starting Deadline Seconds: 100")
	})

	t.Run("Format cronjob with last schedule time", func(t *testing.T) {
		lastSchedule := metav1.Time{Time: time.Now().Add(-5 * time.Minute)}
		lastSuccess := metav1.Time{Time: time.Now().Add(-3 * time.Minute)}
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "scheduled-cronjob",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "*/5 * * * *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
							},
						},
					},
				},
			},
			Status: batchv1.CronJobStatus{
				LastScheduleTime:   &lastSchedule,
				LastSuccessfulTime: &lastSuccess,
			},
		}

		result := formatCronJob(cronJob)
		assert.Contains(t, result, "Last Schedule:")
		assert.Contains(t, result, "Last Successful:")
	})

	t.Run("Format cronjob with labels", func(t *testing.T) {
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-cronjob",
				Namespace:         "default",
				Labels:            map[string]string{"app": "batch", "env": "prod"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
							},
						},
					},
				},
			},
		}

		result := formatCronJob(cronJob)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "app")
		assert.Contains(t, result, "batch")
	})
}

func TestFormatCronJobList(t *testing.T) {
	t.Run("Format cronjob list single namespace", func(t *testing.T) {
		cronJobList := &batchv1.CronJobList{
			Items: []batchv1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "cronjob-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/5 * * * *",
					},
					Status: batchv1.CronJobStatus{
						Active: []corev1.ObjectReference{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "cronjob-2",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
					},
					Status: batchv1.CronJobStatus{
						Active: []corev1.ObjectReference{{Name: "job-1"}},
					},
				},
			},
		}

		result := formatCronJobList(cronJobList, false)
		assert.Contains(t, result, "CronJobs in namespace \"default\":")
		assert.Contains(t, result, "cronjob-1")
		assert.Contains(t, result, "cronjob-2")
		assert.Contains(t, result, "Schedule=*/5 * * * *")
		assert.Contains(t, result, "Schedule=0 0 * * *")
		assert.Contains(t, result, "Active=0")
		assert.Contains(t, result, "Active=1")
		assert.Contains(t, result, "Total: 2 CronJob(s)")
	})

	t.Run("Format cronjob list all namespaces", func(t *testing.T) {
		cronJobList := &batchv1.CronJobList{
			Items: []batchv1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "cronjob-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/10 * * * *",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "cronjob-2",
						Namespace:         "kube-system",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 * * * *",
					},
				},
			},
		}

		result := formatCronJobList(cronJobList, true)
		assert.Contains(t, result, "CronJobs across all namespaces:")
		assert.Contains(t, result, "default/cronjob-1")
		assert.Contains(t, result, "kube-system/cronjob-2")
	})

	t.Run("Format suspended cronjob in list", func(t *testing.T) {
		suspend := true
		cronJobList := &batchv1.CronJobList{
			Items: []batchv1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "suspended-cronjob",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
						Suspend:  &suspend,
					},
				},
			},
		}

		result := formatCronJobList(cronJobList, false)
		assert.Contains(t, result, "Suspended")
	})

	t.Run("Format cronjob list with last schedule", func(t *testing.T) {
		lastSchedule := metav1.Time{Time: time.Now().Add(-5 * time.Minute)}
		cronJobList := &batchv1.CronJobList{
			Items: []batchv1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "scheduled-cronjob",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/5 * * * *",
					},
					Status: batchv1.CronJobStatus{
						LastScheduleTime: &lastSchedule,
					},
				},
			},
		}

		result := formatCronJobList(cronJobList, false)
		assert.Contains(t, result, "LastSchedule=5m")
	})

	t.Run("Format cronjob list with labels", func(t *testing.T) {
		cronJobList := &batchv1.CronJobList{
			Items: []batchv1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "labeled-cronjob",
						Namespace:         "default",
						Labels:            map[string]string{"app": "batch", "env": "prod"},
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
					},
				},
			},
		}

		result := formatCronJobList(cronJobList, false)
		assert.Contains(t, result, "Labels: 2")
	})
}

func TestFormatIngress(t *testing.T) {
	t.Run("Format basic ingress", func(t *testing.T) {
		pathType := networkingv1.PathTypePrefix
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-ingress",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "backend",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "Ingress: test-ingress")
		assert.Contains(t, result, "Namespace: default")
		assert.Contains(t, result, "example.com")
		assert.Contains(t, result, "backend")
	})

	t.Run("Format ingress with TLS", func(t *testing.T) {
		pathType := networkingv1.PathTypePrefix
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tls-ingress",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: "secure.example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "secure-backend",
												Port: networkingv1.ServiceBackendPort{
													Name: "https",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				TLS: []networkingv1.IngressTLS{
					{
						Hosts:      []string{"secure.example.com"},
						SecretName: "tls-secret",
					},
				},
			},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "TLS:")
		assert.Contains(t, result, "secure.example.com")
		assert.Contains(t, result, "tls-secret")
	})

	t.Run("Format ingress with ingress class", func(t *testing.T) {
		ingressClass := "nginx"
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "class-ingress",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClass,
			},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "Ingress Class: nginx")
	})

	t.Run("Format ingress with default backend", func(t *testing.T) {
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "default-backend-ingress",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: networkingv1.IngressSpec{
				DefaultBackend: &networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: "default-service",
						Port: networkingv1.ServiceBackendPort{
							Number: 8080,
						},
					},
				},
			},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "Default Backend:")
		assert.Contains(t, result, "default-service")
		assert.Contains(t, result, "8080")
	})

	t.Run("Format ingress with load balancer", func(t *testing.T) {
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "lb-ingress",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: networkingv1.IngressSpec{},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{
							IP: "192.168.1.100",
						},
					},
				},
			},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "Load Balancer:")
		assert.Contains(t, result, "192.168.1.100")
	})

	t.Run("Format ingress with labels and annotations", func(t *testing.T) {
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "labeled-ingress",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web", "env": "prod"},
				Annotations:       map[string]string{"nginx.ingress.kubernetes.io/rewrite-target": "/"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: networkingv1.IngressSpec{},
		}

		result := formatIngress(ingress)
		assert.Contains(t, result, "Labels:")
		assert.Contains(t, result, "app")
		assert.Contains(t, result, "Annotations:")
		assert.Contains(t, result, "rewrite-target")
	})
}

func TestFormatIngressList(t *testing.T) {
	t.Run("Format ingress list single namespace", func(t *testing.T) {
		pathType := networkingv1.PathTypePrefix
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ingress-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "app1.example.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/",
												PathType: &pathType,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "service1",
														Port: networkingv1.ServiceBackendPort{Number: 80},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ingress-2",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "app2.example.com",
							},
						},
					},
				},
			},
		}

		result := formatIngressList(ingressList, false)
		assert.Contains(t, result, "Ingresses in namespace \"default\":")
		assert.Contains(t, result, "ingress-1")
		assert.Contains(t, result, "ingress-2")
		assert.Contains(t, result, "app1.example.com")
		assert.Contains(t, result, "Total: 2 Ingress(es)")
	})

	t.Run("Format ingress list all namespaces", func(t *testing.T) {
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ingress-1",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: "app1.example.com"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ingress-2",
						Namespace:         "kube-system",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: "app2.example.com"},
						},
					},
				},
			},
		}

		result := formatIngressList(ingressList, true)
		assert.Contains(t, result, "Ingresses across all namespaces:")
		assert.Contains(t, result, "default/ingress-1")
		assert.Contains(t, result, "kube-system/ingress-2")
	})

	t.Run("Format ingress list with TLS indicator", func(t *testing.T) {
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "tls-ingress",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{Host: "secure.example.com"},
						},
						TLS: []networkingv1.IngressTLS{
							{
								Hosts:      []string{"secure.example.com"},
								SecretName: "tls-secret",
							},
						},
					},
				},
			},
		}

		result := formatIngressList(ingressList, false)
		assert.Contains(t, result, "[TLS]")
	})

	t.Run("Format ingress list with ingress class", func(t *testing.T) {
		ingressClass := "nginx"
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "class-ingress",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{
						IngressClassName: &ingressClass,
					},
				},
			},
		}

		result := formatIngressList(ingressList, false)
		assert.Contains(t, result, "Class=nginx")
	})

	t.Run("Format ingress list with load balancer address", func(t *testing.T) {
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "lb-ingress",
						Namespace:         "default",
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{},
					Status: networkingv1.IngressStatus{
						LoadBalancer: networkingv1.IngressLoadBalancerStatus{
							Ingress: []networkingv1.IngressLoadBalancerIngress{
								{
									Hostname: "lb.example.com",
								},
							},
						},
					},
				},
			},
		}

		result := formatIngressList(ingressList, false)
		assert.Contains(t, result, "Address=lb.example.com")
	})

	t.Run("Format ingress list with labels", func(t *testing.T) {
		ingressList := &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "labeled-ingress",
						Namespace:         "default",
						Labels:            map[string]string{"app": "web", "env": "prod"},
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Spec: networkingv1.IngressSpec{},
				},
			},
		}

		result := formatIngressList(ingressList, false)
		assert.Contains(t, result, "Labels: 2")
	})
}
