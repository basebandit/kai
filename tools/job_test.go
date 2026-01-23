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

func TestCreateJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockJobFactory, *testmocks.MockJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Create basic Job",
			args: map[string]any{
				"name":      "test-job",
				"namespace": defaultNamespace,
				"image":     "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Name == "test-job" && params.Namespace == defaultNamespace && params.Image == "busybox:latest"
				})).Return(mockJob)
				mockJob.On("Create", mock.Anything, mockCM).Return("Job \"test-job\" created successfully in namespace \"default\"", nil)
			},
			expectedOutput: "Job \"test-job\" created successfully",
			expectedError:  false,
		},
		{
			name: "Create Job with all parameters",
			args: map[string]any{
				"name":               "full-job",
				"namespace":          defaultNamespace,
				"image":              nginxImage,
				"command":            []any{"/bin/sh"},
				"args":               []any{"-c", "echo hello"},
				"restart_policy":     "OnFailure",
				"backoff_limit":      float64(3),
				"completions":        float64(1),
				"parallelism":        float64(1),
				"labels":             map[string]any{"app": "test"},
				"env":                map[string]any{"ENV": "test"},
				"image_pull_policy":  alwaysImagePullPolicy,
				"image_pull_secrets": []any{registrySecretName},
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Name == "full-job" &&
						params.RestartPolicy == "OnFailure" &&
						*params.BackoffLimit == int32(3) &&
						*params.Completions == int32(1) &&
						*params.Parallelism == int32(1) &&
						params.ImagePullPolicy == alwaysImagePullPolicy
				})).Return(mockJob)
				mockJob.On("Create", mock.Anything, mockCM).Return("Job \"full-job\" created successfully in namespace \"default\"", nil)
			},
			expectedOutput: "Job \"full-job\" created successfully",
			expectedError:  false,
		},
		{
			name: "Missing Job name",
			args: map[string]any{
				"image": "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty Job name",
			args: map[string]any{
				"name":  "",
				"image": "busybox:latest",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
		{
			name: "Missing image",
			args: map[string]any{
				"name": "test-job",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingImage,
			expectedError:  false,
		},
		{
			name: "Empty image",
			args: map[string]any{
				"name":  "test-job",
				"image": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyImage,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockJobFactory{}
			mockJob := &testmocks.MockJob{}
			tt.mockSetup(mockCM, mockFactory, mockJob)

			handler := createJobHandler(mockCM, mockFactory)
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
			mockJob.AssertExpectations(t)
		})
	}
}

func TestGetJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockJobFactory, *testmocks.MockJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Get existing Job",
			args: map[string]any{
				"name":      "test-job",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Name == "test-job" && params.Namespace == defaultNamespace
				})).Return(mockJob)
				mockJob.On("Get", mock.Anything, mockCM).Return("Job: test-job\nNamespace: default\nStatus: Complete", nil)
			},
			expectedOutput: "Job: test-job",
			expectedError:  false,
		},
		{
			name: "Missing Job name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty Job name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockJobFactory{}
			mockJob := &testmocks.MockJob{}
			tt.mockSetup(mockCM, mockFactory, mockJob)

			handler := getJobHandler(mockCM, mockFactory)
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
			mockJob.AssertExpectations(t)
		})
	}
}

func TestListJobsHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockJobFactory, *testmocks.MockJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "List Jobs in default namespace",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Namespace == defaultNamespace
				})).Return(mockJob)
				mockJob.On("List", mock.Anything, mockCM, false, "").Return("Jobs in namespace default:\njob1\njob2", nil)
			},
			expectedOutput: "Jobs in namespace default",
			expectedError:  false,
		},
		{
			name: "List Jobs in specific namespace",
			args: map[string]any{
				"namespace": testNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No GetCurrentNamespace call - namespace is provided in args
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Namespace == testNamespace
				})).Return(mockJob)
				mockJob.On("List", mock.Anything, mockCM, false, "").Return("Jobs in namespace test-namespace:\njob3", nil)
			},
			expectedOutput: "Jobs in namespace test-namespace",
			expectedError:  false,
		},
		{
			name: "List Jobs across all namespaces",
			args: map[string]any{
				"all_namespaces": true,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No GetCurrentNamespace call - all_namespaces=true
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Namespace == ""
				})).Return(mockJob)
				mockJob.On("List", mock.Anything, mockCM, true, "").Return("Jobs across all namespaces:\ndefault/job1\ntest-namespace/job2", nil)
			},
			expectedOutput: "Jobs across all namespaces",
			expectedError:  false,
		},
		{
			name: "List Jobs with label selector",
			args: map[string]any{
				"label_selector": "app=nginx",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Namespace == defaultNamespace
				})).Return(mockJob)
				mockJob.On("List", mock.Anything, mockCM, false, "app=nginx").Return("Jobs matching app=nginx:\njob1", nil)
			},
			expectedOutput: "Jobs matching app=nginx",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockJobFactory{}
			mockJob := &testmocks.MockJob{}
			tt.mockSetup(mockCM, mockFactory, mockJob)

			handler := listJobsHandler(mockCM, mockFactory)
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
			mockJob.AssertExpectations(t)
		})
	}
}

func TestDeleteJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockJobFactory, *testmocks.MockJob)
		expectedOutput string
		expectedError  bool
	}{
		{
			name: "Delete existing Job",
			args: map[string]any{
				"name":      "test-job",
				"namespace": defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockFactory.On("NewJob", mock.MatchedBy(func(params kai.JobParams) bool {
					return params.Name == "test-job" && params.Namespace == defaultNamespace
				})).Return(mockJob)
				mockJob.On("Delete", mock.Anything, mockCM).Return("Job \"test-job\" deleted successfully from namespace \"default\"", nil)
			},
			expectedOutput: "Job \"test-job\" deleted successfully",
			expectedError:  false,
		},
		{
			name: "Missing Job name",
			args: map[string]any{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errMissingName,
			expectedError:  false,
		},
		{
			name: "Empty Job name",
			args: map[string]any{
				"name": "",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockJobFactory, mockJob *testmocks.MockJob) {
				// No mock setup - validation fails before any calls
			},
			expectedOutput: errEmptyName,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCM := &testmocks.MockClusterManager{}
			mockFactory := &testmocks.MockJobFactory{}
			mockJob := &testmocks.MockJob{}
			tt.mockSetup(mockCM, mockFactory, mockJob)

			handler := deleteJobHandler(mockCM, mockFactory)
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
			mockJob.AssertExpectations(t)
		})
	}
}
