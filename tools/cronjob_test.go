package tools

import (
	"context"
	"testing"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateCronJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockCronJobFactory, *testmocks.MockCronJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Create basic CronJob",
			args: map[string]any{
				"name":      "test-cronjob",
				"namespace": defaultNamespace,
				"schedule":  "*/5 * * * *",
				"image":     "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Name == "test-cronjob" &&
						params.Namespace == defaultNamespace &&
						params.Schedule == "*/5 * * * *" &&
						params.Image == "busybox:latest"
				})).Return(mockCronJob)
				mockCronJob.On("Create", mock.Anything, mockCM).Return("CronJob \"test-cronjob\" created successfully in namespace \"default\" with schedule \"*/5 * * * *\"", nil)
			},
			expectedOutput: "CronJob \"test-cronjob\" created successfully",
			expectedError:  false,
		},
		{
			name: "Create CronJob with all parameters",
			args: map[string]any{
				"name":                          "full-cronjob",
				"namespace":                     defaultNamespace,
				"schedule":                      "0 0 * * *",
				"image":                         nginxImage,
				"command":                       []any{"/bin/sh"},
				"args":                          []any{"-c", "echo hello"},
				"restart_policy":                "OnFailure",
				"concurrency_policy":            "Forbid",
				"suspend":                       true,
				"successful_jobs_history_limit": float64(3),
				"failed_jobs_history_limit":     float64(1),
				"starting_deadline_seconds":     float64(100),
				"backoff_limit":                 float64(3),
				"labels":                        map[string]any{"app": "test"},
				"env":                           map[string]any{"ENV": "test"},
				"image_pull_policy":             alwaysImagePullPolicy,
				"image_pull_secrets":            []any{registrySecretName},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Name == "full-cronjob" &&
						params.Schedule == "0 0 * * *" &&
						params.ConcurrencyPolicy == "Forbid" &&
						params.Suspend != nil && *params.Suspend == true &&
						*params.SuccessfulJobsHistoryLimit == int32(3) &&
						*params.FailedJobsHistoryLimit == int32(1) &&
						*params.StartingDeadlineSeconds == int64(100) &&
						*params.BackoffLimit == int32(3) &&
						params.ImagePullPolicy == alwaysImagePullPolicy
				})).Return(mockCronJob)
				mockCronJob.On("Create", mock.Anything, mockCM).Return("CronJob \"full-cronjob\" created successfully in namespace \"default\" with schedule \"0 0 * * *\"", nil)
			},
			expectedOutput: "CronJob \"full-cronjob\" created successfully",
			expectedError:  false,
		},
		{
			name: "Missing CronJob name",
			args: map[string]any{
				"schedule": "*/5 * * * *",
				"image":    "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty CronJob name",
			args: map[string]any{
				"name":     "",
				"schedule": "*/5 * * * *",
				"image":    "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
		{
			name: "Missing schedule",
			args: map[string]any{
				"name":  "test-cronjob",
				"image": "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: "schedule is required",
			expectedError:  false,
		},
		{
			name: "Empty schedule",
			args: map[string]any{
				"name":     "test-cronjob",
				"schedule": "",
				"image":    "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: "schedule cannot be empty",
			expectedError:  false,
		},
		{
			name: "Missing image",
			args: map[string]any{
				"name":     "test-cronjob",
				"schedule": "*/5 * * * *",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingImage,
			expectedError:  false,
		},
		{
			name: "Empty image",
			args: map[string]any{
				"name":     "test-cronjob",
				"schedule": "*/5 * * * *",
				"image":    "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyImage,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockCronJobFactory{}
			mockCronJob := &testmocks.MockCronJob{}
			tt.mockSetup(mockCM, mockFactory, mockCronJob)

			handler := createCronJobHandler(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockCronJob.AssertExpectations(t)
		})
	}
}

func TestGetCronJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockCronJobFactory, *testmocks.MockCronJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Get existing CronJob",
			args: map[string]any{
				"name":      "test-cronjob",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Name == "test-cronjob" && params.Namespace == defaultNamespace
				})).Return(mockCronJob)
				mockCronJob.On("Get", mock.Anything, mockCM).Return("CronJob: test-cronjob\nNamespace: default\nSchedule: */5 * * * *", nil)
			},
			expectedOutput: "CronJob: test-cronjob",
			expectedError:  false,
		},
		{
			name: "Missing CronJob name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty CronJob name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockCronJobFactory{}
			mockCronJob := &testmocks.MockCronJob{}
			tt.mockSetup(mockCM, mockFactory, mockCronJob)

			handler := getCronJobHandler(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockCronJob.AssertExpectations(t)
		})
	}
}

func TestListCronJobsHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockCronJobFactory, *testmocks.MockCronJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "List CronJobs in default namespace",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Namespace == defaultNamespace
				})).Return(mockCronJob)
				mockCronJob.On("List", mock.Anything, mockCM, false, "").Return("CronJobs in namespace default:\ncronjob1\ncronjob2", nil)
			},
			expectedOutput: "CronJobs in namespace default",
			expectedError:  false,
		},
		{
			name: "List CronJobs in specific namespace",
			args: map[string]any{
				"namespace": testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No GetCurrentNamespace call - namespace is provided in args
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Namespace == testNamespace
				})).Return(mockCronJob)
				mockCronJob.On("List", mock.Anything, mockCM, false, "").Return("CronJobs in namespace test-namespace:\ncronjob3", nil)
			},
			expectedOutput: "CronJobs in namespace test-namespace",
			expectedError:  false,
		},
		{
			name: "List CronJobs across all namespaces",
			args: map[string]any{
				"all_namespaces": true,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No GetCurrentNamespace call - all_namespaces=true
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Namespace == ""
				})).Return(mockCronJob)
				mockCronJob.On("List", mock.Anything, mockCM, true, "").Return("CronJobs across all namespaces:\ndefault/cronjob1\ntest-namespace/cronjob2", nil)
			},
			expectedOutput: "CronJobs across all namespaces",
			expectedError:  false,
		},
		{
			name: "List CronJobs with label selector",
			args: map[string]any{
				"label_selector": "app=nginx",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Namespace == defaultNamespace
				})).Return(mockCronJob)
				mockCronJob.On("List", mock.Anything, mockCM, false, "app=nginx").Return("CronJobs matching app=nginx:\ncronjob1", nil)
			},
			expectedOutput: "CronJobs matching app=nginx",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockCronJobFactory{}
			mockCronJob := &testmocks.MockCronJob{}
			tt.mockSetup(mockCM, mockFactory, mockCronJob)

			handler := listCronJobsHandler(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockCronJob.AssertExpectations(t)
		})
	}
}

func TestDeleteCronJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockCronJobFactory, *testmocks.MockCronJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Delete existing CronJob",
			args: map[string]any{
				"name":      "test-cronjob",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
					return params.Name == "test-cronjob" && params.Namespace == defaultNamespace
				})).Return(mockCronJob)
				mockCronJob.On("Delete", mock.Anything, mockCM).Return("CronJob \"test-cronjob\" deleted successfully from namespace \"default\"", nil)
			},
			expectedOutput: "CronJob \"test-cronjob\" deleted successfully",
			expectedError:  false,
		},
		{
			name: "Missing CronJob name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty CronJob name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockCronJobFactory, mockCronJob *testmocks.MockCronJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockCronJobFactory{}
			mockCronJob := &testmocks.MockCronJob{}
			tt.mockSetup(mockCM, mockFactory, mockCronJob)

			handler := deleteCronJobHandler(mockCM, mockFactory)
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
				}{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockCronJob.AssertExpectations(t)
		})
	}
}

func TestNewDefaultCronJobFactory(t *testing.T) {
	factory := NewDefaultCronJobFactory()
	assert.NotNil(t, factory)
}

func TestDefaultCronJobFactoryNewCronJob(t *testing.T) {
	factory := NewDefaultCronJobFactory()

	suspend := true
	successfulJobsHistoryLimit := int32(3)
	failedJobsHistoryLimit := int32(1)
	startingDeadlineSeconds := int64(100)
	backoffLimit := int32(6)

	params := kai.CronJobParams{
		Name:                       "test-cronjob",
		Namespace:                  "default",
		Schedule:                   "*/5 * * * *",
		Image:                      "busybox:latest",
		Command:                    []interface{}{"echo", "hello"},
		Args:                       []interface{}{"world"},
		RestartPolicy:              "OnFailure",
		ConcurrencyPolicy:          "Forbid",
		Suspend:                    &suspend,
		SuccessfulJobsHistoryLimit: &successfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     &failedJobsHistoryLimit,
		StartingDeadlineSeconds:    &startingDeadlineSeconds,
		BackoffLimit:               &backoffLimit,
		Labels:                     map[string]interface{}{"app": "test"},
		Env:                        map[string]interface{}{"ENV": "test"},
		ImagePullPolicy:            "Always",
		ImagePullSecrets:           []interface{}{"registry-secret"},
	}

	cronJob := factory.NewCronJob(params)
	assert.NotNil(t, cronJob)
}

func TestRegisterCronJobTools(t *testing.T) {
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(7)

	RegisterCronJobTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

func TestRegisterCronJobToolsWithFactory(t *testing.T) {
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()
	mockFactory := new(testmocks.MockCronJobFactory)

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(7)

	RegisterCronJobToolsWithFactory(mockServer, mockCM, mockFactory)

	mockServer.AssertExpectations(t)
}

func TestCreateCronJobHandlerError(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.Anything).Return(mockCronJob)
	mockCronJob.On("Create", mock.Anything, mockCM).Return("", assert.AnError)

	handler := createCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name":     "test-cronjob",
				"schedule": "*/5 * * * *",
				"image":    "busybox:latest",
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to create CronJob")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestGetCronJobHandlerError(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.Anything).Return(mockCronJob)
	mockCronJob.On("Get", mock.Anything, mockCM).Return("", assert.AnError)

	handler := getCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name": "test-cronjob",
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to get CronJob")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestListCronJobsHandlerError(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.Anything).Return(mockCronJob)
	mockCronJob.On("List", mock.Anything, mockCM, false, "").Return("", assert.AnError)

	handler := listCronJobsHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to list CronJobs")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestDeleteCronJobHandlerError(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.Anything).Return(mockCronJob)
	mockCronJob.On("Delete", mock.Anything, mockCM).Return("", assert.AnError)

	handler := deleteCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name": "test-cronjob",
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Failed to delete CronJob")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestCreateCronJobHandlerDefaultNamespace(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
		return params.Name == "test-cronjob" &&
			params.Namespace == defaultNamespace &&
			params.Schedule == "*/5 * * * *" &&
			params.Image == "busybox:latest"
	})).Return(mockCronJob)
	mockCronJob.On("Create", mock.Anything, mockCM).Return("CronJob \"test-cronjob\" created successfully", nil)

	handler := createCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name":     "test-cronjob",
				"schedule": "*/5 * * * *",
				"image":    "busybox:latest",
				// No namespace provided - should use default
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "CronJob \"test-cronjob\" created successfully")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestGetCronJobHandlerDefaultNamespace(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
		return params.Name == "test-cronjob" && params.Namespace == defaultNamespace
	})).Return(mockCronJob)
	mockCronJob.On("Get", mock.Anything, mockCM).Return("CronJob: test-cronjob", nil)

	handler := getCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name": "test-cronjob",
				// No namespace provided - should use default
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "CronJob: test-cronjob")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}

func TestDeleteCronJobHandlerDefaultNamespace(t *testing.T) {
	mockCM := &testmocks.MockClusterManager{}
	mockFactory := &testmocks.MockCronJobFactory{}
	mockCronJob := &testmocks.MockCronJob{}

	mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
	mockFactory.On("NewCronJob", mock.MatchedBy(func(params kai.CronJobParams) bool {
		return params.Name == "test-cronjob" && params.Namespace == defaultNamespace
	})).Return(mockCronJob)
	mockCronJob.On("Delete", mock.Anything, mockCM).Return("CronJob \"test-cronjob\" deleted successfully", nil)

	handler := deleteCronJobHandler(mockCM, mockFactory)
	request := mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *mcp.Meta      `json:"_meta,omitempty"`
		}{
			Arguments: map[string]any{
				"name": "test-cronjob",
				// No namespace provided - should use default
			},
		},
	}

	result, err := handler(context.Background(), request)
	assert.NoError(t, err)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "CronJob \"test-cronjob\" deleted successfully")

	mockCM.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockCronJob.AssertExpectations(t)
}
