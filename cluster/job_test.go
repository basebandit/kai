package cluster

import (
	"context"
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestJobOperations(t *testing.T) {
	t.Run("CreateJob", testCreateJob)
	t.Run("GetJob", testGetJob)
	t.Run("ListJobs", testListJobs)
	t.Run("DeleteJob", testDeleteJob)
	t.Run("UpdateJob", testUpdateJob)
}

func testCreateJob(t *testing.T) {
	ctx := context.Background()
	completions := int32(1)

	testCases := []struct {
		name           string
		job            *Job
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Create basic Job",
			job: &Job{
				Name:        "test-job",
				Namespace:   testNamespace,
				Image:       "busybox:latest",
				Completions: &completions,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "Job \"test-job\" created successfully",
			expectedError:  "",
		},
		{
			name: "Missing Job name",
			job: &Job{
				Image:     "busybox:latest",
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Job name is required",
		},
		{
			name: "Missing namespace",
			job: &Job{
				Name:  "test-job",
				Image: "busybox:latest",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace is required",
		},
		{
			name: "Namespace not found",
			job: &Job{
				Name:      "test-job",
				Namespace: "nonexistent",
				Image:     "busybox:latest",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "namespace \"nonexistent\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.job.Create(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testGetJob(t *testing.T) {
	ctx := context.Background()
	completions := int32(1)

	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: testNamespace},
		Spec: batchv1.JobSpec{
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		job            *Job
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing Job",
			job: &Job{
				Name:      "test-job",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingJob)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "test-job",
			expectedError:  "",
		},
		{
			name: "Job not found",
			job: &Job{
				Name:      "nonexistent",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.job.Get(ctx, mockCM)

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

func testListJobs(t *testing.T) {
	ctx := context.Background()
	completions := int32(1)

	job1 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "job1", Namespace: testNamespace},
		Spec: batchv1.JobSpec{
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
				},
			},
		},
	}

	job2 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "job2", Namespace: testNamespace},
		Spec: batchv1.JobSpec{
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		job            *Job
		allNamespaces  bool
		labelSelector  string
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult []string
		expectedError  string
	}{
		{
			name: "List Jobs in namespace",
			job: &Job{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(job1, job2)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: []string{"job1", "job2"},
			expectedError:  "",
		},
		{
			name: "No Jobs in empty namespace",
			job: &Job{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no Jobs found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.job.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				for _, expected := range tc.expectedResult {
					assert.Contains(t, result, expected)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testDeleteJob(t *testing.T) {
	ctx := context.Background()
	completions := int32(1)

	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: testNamespace},
		Spec: batchv1.JobSpec{
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "test", Image: "busybox"}},
				},
			},
		},
	}

	testCases := []struct {
		name           string
		job            *Job
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing Job",
			job: &Job{
				Name:      "test-job",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingJob)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "deleted successfully",
			expectedError:  "",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.BatchV1().Jobs(testNamespace).Get(ctx, "test-job", metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Job not found",
			job: &Job{
				Name:      "nonexistent",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "not found",
		},
		{
			name: "Missing Job name",
			job: &Job{
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "Job name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.job.Delete(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				// Validate deletion if validator provided
				if tc.validateDelete != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateDelete(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}

func testUpdateJob(t *testing.T) {
	ctx := context.Background()
	parallelism := int32(2)
	completions := int32(1)

	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: testNamespace,
			Labels: map[string]string{
				"version": "v1",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: &parallelism,
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:1.19",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	testCases := []struct {
		name           string
		job            *Job
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateUpdate func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Update job labels",
			job: &Job{
				Name:      "test-job",
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"version": "v2",
					"env":     "prod",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(existingJob, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				job, err := client.BatchV1().Jobs(testNamespace).Get(ctx, "test-job", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "v2", job.Labels["version"])
				assert.Equal(t, "prod", job.Labels["env"])
			},
		},
		{
			name: "Update job parallelism",
			job: &Job{
				Name:        "test-job",
				Namespace:   testNamespace,
				Parallelism: func() *int32 { v := int32(5); return &v }(),
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(existingJob, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				job, err := client.BatchV1().Jobs(testNamespace).Get(ctx, "test-job", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, int32(5), *job.Spec.Parallelism)
			},
		},
		{
			name: "Update job with both labels and parallelism",
			job: &Job{
				Name:      "test-job",
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"updated": "true",
				},
				Parallelism: func() *int32 { v := int32(3); return &v }(),
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(existingJob, ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "updated successfully",
			validateUpdate: func(t *testing.T, client kubernetes.Interface) {
				job, err := client.BatchV1().Jobs(testNamespace).Get(ctx, "test-job", metav1.GetOptions{})
				assert.NoError(t, err)
				assert.Equal(t, "true", job.Labels["updated"])
				assert.Equal(t, int32(3), *job.Spec.Parallelism)
			},
		},
		{
			name: "Job not found",
			job: &Job{
				Name:      "nonexistent-job",
				Namespace: testNamespace,
				Labels: map[string]interface{}{
					"test": "true",
				},
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "failed to get Job",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.job.Update(ctx, mockCM)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expectedResult)

				if tc.validateUpdate != nil {
					client, _ := mockCM.GetCurrentClient()
					tc.validateUpdate(t, client)
				}
			}

			mockCM.AssertExpectations(t)
		})
	}
}
