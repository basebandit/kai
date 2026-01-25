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

func TestCronJobOperations(t *testing.T) {
	t.Run("CreateCronJob", testCreateCronJob)
	t.Run("GetCronJob", testGetCronJob)
	t.Run("ListCronJobs", testListCronJobs)
	t.Run("DeleteCronJob", testDeleteCronJob)
}

func testCreateCronJob(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		cronJob        *CronJob
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Create basic CronJob",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: testNamespace,
				Schedule:  "*/5 * * * *",
				Image:     "busybox:latest",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "CronJob \"test-cronjob\" created successfully",
			expectedError:  "",
		},
		{
			name: "Create CronJob with all options",
			cronJob: &CronJob{
				Name:              "full-cronjob",
				Namespace:         testNamespace,
				Schedule:          "0 0 * * *",
				Image:             "busybox:latest",
				Command:           []interface{}{"echo", "hello"},
				ConcurrencyPolicy: "Forbid",
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				}
				fakeClient := fake.NewSimpleClientset(ns)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "CronJob \"full-cronjob\" created successfully",
			expectedError:  "",
		},
		{
			name: "Missing CronJob name",
			cronJob: &CronJob{
				Schedule:  "*/5 * * * *",
				Image:     "busybox:latest",
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "CronJob name is required",
		},
		{
			name: "Missing namespace",
			cronJob: &CronJob{
				Name:     "test-cronjob",
				Schedule: "*/5 * * * *",
				Image:    "busybox:latest",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "namespace is required",
		},
		{
			name: "Missing schedule",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: testNamespace,
				Image:     "busybox:latest",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "schedule is required",
		},
		{
			name: "Missing image",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: testNamespace,
				Schedule:  "*/5 * * * *",
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "image is required",
		},
		{
			name: "Namespace not found",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: "nonexistent",
				Schedule:  "*/5 * * * *",
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

			result, err := tc.cronJob.Create(ctx, mockCM)

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

func testGetCronJob(t *testing.T) {
	ctx := context.Background()

	existingCronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cronjob", Namespace: testNamespace},
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
	}

	testCases := []struct {
		name           string
		cronJob        *CronJob
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
	}{
		{
			name: "Get existing CronJob",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingCronJob)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "test-cronjob",
			expectedError:  "",
		},
		{
			name: "CronJob not found",
			cronJob: &CronJob{
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

			result, err := tc.cronJob.Get(ctx, mockCM)

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

func testListCronJobs(t *testing.T) {
	ctx := context.Background()

	cronJob1 := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "cronjob1", Namespace: testNamespace},
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
	}

	cronJob2 := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "cronjob2", Namespace: testNamespace},
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

	testCases := []struct {
		name           string
		cronJob        *CronJob
		allNamespaces  bool
		labelSelector  string
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult []string
		expectedError  string
	}{
		{
			name: "List CronJobs in namespace",
			cronJob: &CronJob{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(cronJob1, cronJob2)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: []string{"cronjob1", "cronjob2"},
			expectedError:  "",
		},
		{
			name: "No CronJobs in empty namespace",
			cronJob: &CronJob{
				Namespace: testNamespace,
			},
			allNamespaces: false,
			labelSelector: "",
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset()
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedError: "no CronJobs found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.cronJob.List(ctx, mockCM, tc.allNamespaces, tc.labelSelector)

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

func testDeleteCronJob(t *testing.T) {
	ctx := context.Background()

	existingCronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cronjob", Namespace: testNamespace},
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
	}

	testCases := []struct {
		name           string
		cronJob        *CronJob
		setupMock      func(*testmocks.MockClusterManager)
		expectedResult string
		expectedError  string
		validateDelete func(*testing.T, kubernetes.Interface)
	}{
		{
			name: "Delete existing CronJob",
			cronJob: &CronJob{
				Name:      "test-cronjob",
				Namespace: testNamespace,
			},
			setupMock: func(mockCM *testmocks.MockClusterManager) {
				fakeClient := fake.NewSimpleClientset(existingCronJob)
				mockCM.On("GetCurrentClient").Return(fakeClient, nil)
			},
			expectedResult: "deleted successfully",
			expectedError:  "",
			validateDelete: func(t *testing.T, client kubernetes.Interface) {
				_, err := client.BatchV1().CronJobs(testNamespace).Get(ctx, "test-cronjob", metav1.GetOptions{})
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "CronJob not found",
			cronJob: &CronJob{
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
			name: "Missing CronJob name",
			cronJob: &CronJob{
				Namespace: testNamespace,
			},
			setupMock:     func(mockCM *testmocks.MockClusterManager) {},
			expectedError: "CronJob name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			tc.setupMock(mockCM)

			result, err := tc.cronJob.Delete(ctx, mockCM)

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
