package tools

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/testmocks"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testPodName      = "test-pod"
	defaultNamespace = "default"
	nginxLatestImage = "nginx:latest"
)

// Test case structs for table-driven tests
type createPodTestCase struct {
	name              string
	args              map[string]interface{}
	expectedParams    kai.PodParams
	mockSetup         func(*testmocks.MockClusterManager, *testmocks.MockPodFactory, *testmocks.MockPod)
	expectedOutput    string
	expectPodCreation bool
}

type listPodsTestCase struct {
	name           string
	args           map[string]interface{}
	expectedParams kai.PodParams
	mockSetup      func(*testmocks.MockClusterManager, *testmocks.MockPodFactory, *testmocks.MockPod)
	expectedOutput string
}

type getPodTestCase struct {
	name              string
	args              map[string]interface{}
	expectedParams    kai.PodParams
	mockSetup         func(*testmocks.MockClusterManager, *testmocks.MockPodFactory, *testmocks.MockPod)
	expectedOutput    string
	expectPodCreation bool
}

type deletePodTestCase struct {
	name              string
	args              map[string]interface{}
	expectedParams    kai.PodParams
	mockSetup         func(*testmocks.MockClusterManager, *testmocks.MockPodFactory, *testmocks.MockPod)
	expectedOutput    string
	expectPodCreation bool
}

type logsTestCase struct {
	name              string
	args              map[string]interface{}
	expectedParams    kai.PodParams
	mockSetup         func(*testmocks.MockClusterManager, *testmocks.MockPodFactory, *testmocks.MockPod)
	expectedOutput    string
	expectPodCreation bool
}

func TestCreatePodHandler(t *testing.T) {
	podName := "full-pod"
	testNamespace := "test-namespace"
	containerName := "custom-container"
	containerPort := "8080/TCP"
	defaultRestartPolicy := "Always"

	testCases := []createPodTestCase{
		{
			name: "RequiredParams",
			args: map[string]interface{}{
				"name":  testPodName,
				"image": nginxLatestImage,
			},
			expectedParams: kai.PodParams{
				Name:          testPodName,
				Namespace:     defaultNamespace,
				Image:         nginxLatestImage,
				ContainerName: testPodName,          // Default to pod name
				RestartPolicy: defaultRestartPolicy, // Default
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Create", mock.Anything, mockCM).Return(fmt.Sprintf("Pod %q created successfully in namespace %q, ", testPodName, defaultNamespace), nil)
			},
			expectedOutput:    fmt.Sprintf("Pod %q created successfully", testPodName),
			expectPodCreation: true,
		},
		{
			name: "AllParams",
			args: map[string]interface{}{
				"name":               podName,
				"image":              nginxLatestImage,
				"namespace":          testNamespace,
				"command":            []interface{}{"/bin/sh", "-c"},
				"args":               []interface{}{"echo hello; sleep 3600"},
				"container_name":     containerName,
				"container_port":     containerPort,
				"labels":             map[string]interface{}{"app": "web", "env": "test"},
				"env":                map[string]interface{}{"DEBUG": "true"},
				"image_pull_policy":  "Always",
				"image_pull_secrets": []interface{}{"registry-secret"},
				"restart_policy":     "OnFailure",
				"node_selector":      map[string]interface{}{"disktype": "ssd"},
				"service_account":    "custom-sa",
			},
			expectedParams: kai.PodParams{
				Name:               podName,
				Image:              nginxLatestImage,
				Namespace:          testNamespace,
				Command:            []interface{}{"/bin/sh", "-c"},
				Args:               []interface{}{"echo hello; sleep 3600"},
				ContainerName:      containerName,
				ContainerPort:      containerPort,
				Labels:             map[string]interface{}{"app": "web", "env": "test"},
				Env:                map[string]interface{}{"DEBUG": "true"},
				ImagePullPolicy:    "Always",
				ImagePullSecrets:   []interface{}{"registry-secret"},
				RestartPolicy:      "OnFailure",
				NodeSelector:       map[string]interface{}{"disktype": "ssd"},
				ServiceAccountName: "custom-sa",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Create", mock.Anything, mockCM).Return("Pod \"full-pod\" created successfully in namespace \"test-namespace\"", nil)
			},
			expectedOutput:    "Pod \"full-pod\" created successfully",
			expectPodCreation: true,
		},
		{
			name: "MissingName",
			args: map[string]interface{}{
				"image": nginxLatestImage,
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Required parameter 'name' is missing",
			expectPodCreation: false,
		},
		{
			name: "MissingImage",
			args: map[string]interface{}{
				"name": testPodName,
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Required parameter 'image' is missing",
			expectPodCreation: false,
		},
		{
			name: "InvalidImagePullPolicy",
			args: map[string]interface{}{
				"name":              testPodName,
				"image":             nginxLatestImage,
				"image_pull_policy": "InvalidPolicy",
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:    "Invalid image_pull_policy",
			expectPodCreation: false,
		},
		{
			name: "InvalidRestartPolicy",
			args: map[string]interface{}{
				"name":           testPodName,
				"image":          nginxLatestImage,
				"restart_policy": "InvalidPolicy",
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:    "Invalid restart_policy",
			expectPodCreation: false,
		},
		{
			name: "InvalidContainerPort",
			args: map[string]interface{}{
				"name":           testPodName,
				"image":          nginxLatestImage,
				"container_port": "invalid-port",
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:    "Port must be a number",
			expectPodCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name":  "error-pod",
				"image": nginxLatestImage,
			},
			expectedParams: kai.PodParams{
				Name:          "error-pod",
				Namespace:     defaultNamespace,
				Image:         nginxLatestImage,
				ContainerName: "error-pod",          // Default to pod name
				RestartPolicy: defaultRestartPolicy, // Default
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Create", mock.Anything, mockCM).Return("", errors.New("failed to create pod: resource quota exceeded"))
			},
			expectedOutput:    "failed to create pod: resource quota exceeded",
			expectPodCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := new(testmocks.MockPodFactory)

			var mockPod *testmocks.MockPod
			if tc.expectPodCreation {
				mockPod = testmocks.NewMockPod(tc.expectedParams)
				mockFactory.On("NewPod", tc.expectedParams).Return(mockPod)
			}

			tc.mockSetup(mockCM, mockFactory, mockPod)

			handler := createPodHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockPod != nil {
				mockPod.AssertExpectations(t)
			}
		})
	}
}

func TestListPodsHandler(t *testing.T) {
	testCases := []listPodsTestCase{
		{
			name: "DefaultNamespace",
			args: map[string]interface{}{},
			expectedParams: kai.PodParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("List", mock.Anything, mockCM, int64(0), "", "").
					Return("Pods in namespace 'default':\n- pod1\n- pod2", nil)
			},
			expectedOutput: "Pods in namespace 'default':",
		},
		{
			name: "AllNamespaces",
			args: map[string]interface{}{
				"all_namespaces": true,
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockPod.On("List", mock.Anything, mockCM, int64(0), "", "").
					Return("Pods across all namespaces:\n- namespace1/pod1\n- namespace2/pod2", nil)
			},
			expectedOutput: "Pods across all namespaces:",
		},
		{
			name: "WithLabelSelector",
			args: map[string]interface{}{
				"label_selector": "app=nginx",
			},
			expectedParams: kai.PodParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("List", mock.Anything, mockCM, int64(0), "app=nginx", "").
					Return("Pods in namespace 'default' with label 'app=nginx':\n- nginx-pod-1\n- nginx-pod-2", nil)
			},
			expectedOutput: "Pods in namespace 'default' with label 'app=nginx':",
		},
		{
			name: "WithLimit",
			args: map[string]interface{}{
				"limit": float64(5),
			},
			expectedParams: kai.PodParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("List", mock.Anything, mockCM, int64(5), "", "").
					Return("Pods in namespace 'default' (limited to 5):\n- pod1\n- pod2\n- pod3\n- pod4\n- pod5", nil)
			},
			expectedOutput: "Pods in namespace 'default' (limited to 5):",
		},
		{
			name: "Error",
			args: map[string]interface{}{},
			expectedParams: kai.PodParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("List", mock.Anything, mockCM, int64(0), "", "").
					Return("", errors.New("failed to list pods: connection error"))
			},
			expectedOutput: "failed to list pods: connection error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := new(testmocks.MockPodFactory)
			mockPod := testmocks.NewMockPod(tc.expectedParams)

			mockFactory.On("NewPod", tc.expectedParams).Return(mockPod)
			tc.mockSetup(mockCM, mockFactory, mockPod)

			handler := listPodsHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			mockPod.AssertExpectations(t)
		})
	}
}

func TestGetPodHandler(t *testing.T) {
	testCases := []getPodTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "nginx-pod",
			},
			expectedParams: kai.PodParams{
				Name:      "nginx-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Get", mock.Anything, mockCM).
					Return("Pod 'nginx-pod' in namespace 'default':\nStatus: Running\nNode: worker-1\nIP: 192.168.1.10", nil)
			},
			expectedOutput:    "Pod 'nginx-pod' in namespace 'default':",
			expectPodCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Required parameter 'name' is missing",
			expectPodCreation: false,
		},
		{
			name: "EmptyName",
			args: map[string]interface{}{
				"name": "",
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Parameter 'name' must be a non-empty string",
			expectPodCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "non-existent-pod",
			},
			expectedParams: kai.PodParams{
				Name:      "non-existent-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Get", mock.Anything, mockCM).
					Return("", errors.New("pod 'non-existent-pod' not found in namespace 'default'"))
			},
			expectedOutput:    "pod 'non-existent-pod' not found in namespace 'default'",
			expectPodCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := new(testmocks.MockPodFactory)

			var mockPod *testmocks.MockPod
			if tc.expectPodCreation {
				mockPod = testmocks.NewMockPod(tc.expectedParams)
				mockFactory.On("NewPod", tc.expectedParams).Return(mockPod)
			}

			tc.mockSetup(mockCM, mockFactory, mockPod)

			handler := getPodHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockPod != nil {
				mockPod.AssertExpectations(t)
			}
		})
	}
}

func TestDeletePodHandler(t *testing.T) {
	testCases := []deletePodTestCase{
		{
			name: "Success",
			args: map[string]interface{}{
				"name": "nginx-pod",
			},
			expectedParams: kai.PodParams{
				Name:      "nginx-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Delete", mock.Anything, mockCM, false).
					Return("Successfully delete pod \"nginx-pod\" in namespace \"default\"", nil)
			},
			expectedOutput:    "Successfully delete pod \"nginx-pod\" in namespace \"default\"",
			expectPodCreation: true,
		},
		{
			name: "WithForce",
			args: map[string]interface{}{
				"name":  "nginx-pod",
				"force": true,
			},
			expectedParams: kai.PodParams{
				Name:      "nginx-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Delete", mock.Anything, mockCM, true).
					Return("Successfully delete pod \"nginx-pod\" in namespace \"default\"", nil)
			},
			expectedOutput:    "Successfully delete pod \"nginx-pod\" in namespace \"default\"",
			expectPodCreation: true,
		},
		{
			name:           "MissingName",
			args:           map[string]interface{}{},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Required parameter 'name' is missing",
			expectPodCreation: false,
		},
		{
			name: "Error",
			args: map[string]interface{}{
				"name": "non-existent-pod",
			},
			expectedParams: kai.PodParams{
				Name:      "non-existent-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Delete", mock.Anything, mockCM, false).
					Return("", errors.New("failed to delete pod: not found"))
			},
			expectedOutput:    "failed to delete pod: not found",
			expectPodCreation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := new(testmocks.MockPodFactory)

			var mockPod *testmocks.MockPod
			if tc.expectPodCreation {
				mockPod = testmocks.NewMockPod(tc.expectedParams)
				mockFactory.On("NewPod", tc.expectedParams).Return(mockPod)
			}

			tc.mockSetup(mockCM, mockFactory, mockPod)

			handler := deletePodHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockPod != nil {
				mockPod.AssertExpectations(t)
			}
		})
	}
}

func TestStreamLogsHandler(t *testing.T) {
	testCases := []logsTestCase{
		{
			name: "BasicLogs",
			args: map[string]interface{}{
				"pod": "nginx-pod",
			},
			expectedParams: kai.PodParams{
				Name:      "nginx-pod",
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
					Return("Logs from container 'nginx' in pod 'default/nginx-pod':\n2023-05-01T12:00:00Z INFO starting nginx\n2023-05-01T12:00:01Z INFO nginx started", nil)
			},
			expectedOutput:    "Logs from container 'nginx' in pod 'default/nginx-pod':",
			expectPodCreation: true,
		},
		{
			name: "WithContainer",
			args: map[string]interface{}{
				"pod":       "nginx-pod",
				"container": "sidecar",
			},
			expectedParams: kai.PodParams{
				Name:          "nginx-pod",
				Namespace:     defaultNamespace,
				ContainerName: "sidecar",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
					Return("Logs from container 'sidecar' in pod 'default/nginx-pod':\n2023-05-01T12:00:00Z INFO starting sidecar\n2023-05-01T12:00:01Z INFO sidecar started", nil)
			},
			expectedOutput:    "Logs from container 'sidecar' in pod 'default/nginx-pod':",
			expectPodCreation: true,
		},
		{
			name: "InvalidSince",
			args: map[string]interface{}{
				"pod":   "nginx-pod",
				"since": "invalid",
			},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
			},
			expectedOutput:    "Failed to parse 'since' parameter",
			expectPodCreation: false,
		},
		{
			name:           "MissingPod",
			args:           map[string]interface{}{},
			expectedParams: kai.PodParams{},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				// No setup needed
			},
			expectedOutput:    "Required parameter 'pod' is missing",
			expectPodCreation: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCM := testmocks.NewMockClusterManager()
			mockFactory := new(testmocks.MockPodFactory)

			var mockPod *testmocks.MockPod
			if tc.expectPodCreation {
				mockPod = testmocks.NewMockPod(tc.expectedParams)
				mockFactory.On("NewPod", tc.expectedParams).Return(mockPod)
			}

			tc.mockSetup(mockCM, mockFactory, mockPod)

			handler := streamLogsHandler(mockCM, mockFactory)

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.args,
				},
			}

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.expectedOutput)

			mockCM.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockPod != nil {
				mockPod.AssertExpectations(t)
			}
		})
	}
}

func TestRegisterPodTools(t *testing.T) {
	// Setup mock server and cluster manager
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()

	// Expect tool registrations - use the correct type match
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	// Call the function
	RegisterPodTools(mockServer, mockCM)

	// Verify that the expected methods were called
	mockServer.AssertExpectations(t)
}

func TestRegisterPodToolsWithFactory(t *testing.T) {
	// Setup mock server and cluster manager
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()
	mockFactory := new(testmocks.MockPodFactory)

	// Expect tool registrations - use the correct type match
	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	// Call the function
	RegisterPodToolsWithFactory(mockServer, mockCM, mockFactory)

	// Verify that the expected methods were called
	mockServer.AssertExpectations(t)
}
