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
				"image": nginxImage,
			},
			expectedParams: kai.PodParams{
				Name:          testPodName,
				Namespace:     defaultNamespace,
				Image:         nginxImage,
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
				"image":              nginxImage,
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
				Image:              nginxImage,
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
				mockPod.On("Create", mock.Anything, mockCM).Return(fmt.Sprintf("Pod %q created successfully in namespace %q", podName, testNamespace), nil)
			},
			expectedOutput:    fmt.Sprintf("Pod %q created successfully", podName),
			expectPodCreation: true,
		},
		{
			name: "MissingName",
			args: map[string]interface{}{
				"image": nginxImage,
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
				"image":             nginxImage,
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
				"image":          nginxImage,
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
				"image":          nginxImage,
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
				"image": nginxImage,
			},
			expectedParams: kai.PodParams{
				Name:          "error-pod",
				Namespace:     defaultNamespace,
				Image:         nginxImage,
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
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
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
	labelSelector := "app=nginx"

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
					Return(fmt.Sprintf("Pods in namespace %q:\n- pod1\n- pod2", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Pods in namespace %q:", defaultNamespace),
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
				"label_selector": labelSelector,
			},
			expectedParams: kai.PodParams{
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("List", mock.Anything, mockCM, int64(0), labelSelector, "").
					Return(fmt.Sprintf("Pods in namespace %q with label %q:\n- nginx-pod-1\n- nginx-pod-2", defaultNamespace, labelSelector), nil)
			},
			expectedOutput: fmt.Sprintf("Pods in namespace %q with label %q:", defaultNamespace, labelSelector),
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
					Return(fmt.Sprintf("Pods in namespace %q (limited to 5):\n- pod1\n- pod2\n- pod3\n- pod4\n- pod5", defaultNamespace), nil)
			},
			expectedOutput: fmt.Sprintf("Pods in namespace %q (limited to 5):", defaultNamespace),
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
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
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
				"name": nginxPodName,
			},
			expectedParams: kai.PodParams{
				Name:      nginxPodName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Get", mock.Anything, mockCM).
					Return(fmt.Sprintf("Pod %q in namespace %q:\nStatus: Running\nNode: worker-1\nIP: 192.168.1.10", nginxPodName, defaultNamespace), nil)
			},
			expectedOutput:    fmt.Sprintf("Pod %q in namespace %q:", nginxPodName, defaultNamespace),
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
				"name": nonexistentPodName,
			},
			expectedParams: kai.PodParams{
				Name:      nonexistentPodName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Get", mock.Anything, mockCM).
					Return("", fmt.Errorf("pod %q not found in namespace %q", nonexistentPodName, defaultNamespace))
			},
			expectedOutput:    fmt.Sprintf("pod %q not found in namespace %q", nonexistentPodName, defaultNamespace),
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
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
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
				"name": nginxPodName,
			},
			expectedParams: kai.PodParams{
				Name:      nginxPodName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Delete", mock.Anything, mockCM, false).
					Return(fmt.Sprintf(deleteSuccessMsgFmt, nginxPodName, defaultNamespace), nil)
			},
			expectedOutput:    fmt.Sprintf(deleteSuccessMsgFmt, nginxPodName, defaultNamespace),
			expectPodCreation: true,
		},
		{
			name: "WithForce",
			args: map[string]interface{}{
				"name":  nginxPodName,
				"force": true,
			},
			expectedParams: kai.PodParams{
				Name:      nginxPodName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("Delete", mock.Anything, mockCM, true).
					Return(fmt.Sprintf(deleteSuccessMsgFmt, nginxPodName, defaultNamespace), nil)
			},
			expectedOutput:    fmt.Sprintf(deleteSuccessMsgFmt, nginxPodName, defaultNamespace),
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
				"name": nonexistentPodName,
			},
			expectedParams: kai.PodParams{
				Name:      nonexistentPodName,
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
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
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
				"pod": nginxPodName,
			},
			expectedParams: kai.PodParams{
				Name:      nginxPodName,
				Namespace: defaultNamespace,
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
					Return(fmt.Sprintf("Logs from container 'nginx' in pod '%s/%s':\n2023-05-01T12:00:00Z INFO starting nginx\n2023-05-01T12:00:01Z INFO nginx started", defaultNamespace, nginxPodName), nil)
			},
			expectedOutput:    fmt.Sprintf("Logs from container 'nginx' in pod '%s/%s':", defaultNamespace, nginxPodName),
			expectPodCreation: true,
		},
		{
			name: "WithContainer",
			args: map[string]interface{}{
				"pod":       nginxPodName,
				"container": "sidecar",
			},
			expectedParams: kai.PodParams{
				Name:          nginxPodName,
				Namespace:     defaultNamespace,
				ContainerName: "sidecar",
			},
			mockSetup: func(mockCM *testmocks.MockClusterManager, mockFactory *testmocks.MockPodFactory, mockPod *testmocks.MockPod) {
				mockCM.On("GetCurrentNamespace").Return(defaultNamespace)
				mockPod.On("StreamLogs", mock.Anything, mockCM, int64(0), false, (*time.Duration)(nil)).
					Return(fmt.Sprintf("Logs from container 'sidecar' in pod '%s/%s':\n2023-05-01T12:00:00Z INFO starting sidecar\n2023-05-01T12:00:01Z INFO sidecar started", defaultNamespace, nginxPodName), nil)
			},
			expectedOutput:    fmt.Sprintf("Logs from container 'sidecar' in pod '%s/%s':", defaultNamespace, nginxPodName),
			expectPodCreation: true,
		},
		{
			name: "InvalidSince",
			args: map[string]interface{}{
				"pod":   nginxPodName,
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
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *mcp.Meta      `json:"_meta,omitempty"`
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
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	RegisterPodTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

func TestRegisterPodToolsWithFactory(t *testing.T) {
	mockServer := new(testmocks.MockServer)
	mockCM := testmocks.NewMockClusterManager()
	mockFactory := new(testmocks.MockPodFactory)

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	RegisterPodToolsWithFactory(mockServer, mockCM, mockFactory)

	mockServer.AssertExpectations(t)
}
