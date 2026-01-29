package tools

import (
	"testing"

	"github.com/basebandit/kai/cluster"
	"github.com/stretchr/testify/assert"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedType   string
		expectedName   string
		expectedError  string
	}{
		{
			name:          "valid pod target",
			input:         "pod/nginx",
			expectedType:  "pod",
			expectedName:  "nginx",
			expectedError: "",
		},
		{
			name:          "valid service target",
			input:         "service/my-service",
			expectedType:  "service",
			expectedName:  "my-service",
			expectedError: "",
		},
		{
			name:          "valid svc shorthand",
			input:         "svc/my-service",
			expectedType:  "service",
			expectedName:  "my-service",
			expectedError: "",
		},
		{
			name:          "uppercase pod",
			input:         "POD/nginx",
			expectedType:  "pod",
			expectedName:  "nginx",
			expectedError: "",
		},
		{
			name:          "uppercase service",
			input:         "SERVICE/my-service",
			expectedType:  "service",
			expectedName:  "my-service",
			expectedError: "",
		},
		{
			name:          "invalid format - no slash",
			input:         "nginx",
			expectedType:  "",
			expectedName:  "",
			expectedError: "invalid target format",
		},
		{
			name:          "invalid type",
			input:         "deployment/nginx",
			expectedType:  "",
			expectedName:  "",
			expectedError: "invalid target type",
		},
		{
			name:          "empty name",
			input:         "pod/",
			expectedType:  "",
			expectedName:  "",
			expectedError: "target name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetType, targetName, err := parseTarget(tt.input)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedType, targetType)
				assert.Equal(t, tt.expectedName, targetName)
			}
		})
	}
}

func TestParsePortMapping(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLocal int
		expectedRemote int
		expectedError string
	}{
		{
			name:           "single port",
			input:          "8080",
			expectedLocal:  8080,
			expectedRemote: 8080,
			expectedError:  "",
		},
		{
			name:           "local:remote mapping",
			input:          "8080:80",
			expectedLocal:  8080,
			expectedRemote: 80,
			expectedError:  "",
		},
		{
			name:           "same local and remote",
			input:          "3000:3000",
			expectedLocal:  3000,
			expectedRemote: 3000,
			expectedError:  "",
		},
		{
			name:           "high port numbers",
			input:          "65535:65534",
			expectedLocal:  65535,
			expectedRemote: 65534,
			expectedError:  "",
		},
		{
			name:          "invalid single port - not a number",
			input:         "abc",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "invalid port",
		},
		{
			name:          "invalid local port - not a number",
			input:         "abc:80",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "invalid local port",
		},
		{
			name:          "invalid remote port - not a number",
			input:         "8080:abc",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "invalid remote port",
		},
		{
			name:          "port too low",
			input:         "0",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "port must be between 1 and 65535",
		},
		{
			name:          "port too high",
			input:         "65536",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "port must be between 1 and 65535",
		},
		{
			name:          "local port too high",
			input:         "65536:80",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "ports must be between 1 and 65535",
		},
		{
			name:          "remote port too high",
			input:         "8080:65536",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "ports must be between 1 and 65535",
		},
		{
			name:          "too many colons",
			input:         "8080:80:90",
			expectedLocal: 0,
			expectedRemote: 0,
			expectedError: "invalid port mapping format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localPort, remotePort, err := parsePortMapping(tt.input)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLocal, localPort)
				assert.Equal(t, tt.expectedRemote, remotePort)
			}
		})
	}
}

func TestFormatPortForwardList_Empty(t *testing.T) {
	result := formatPortForwardList(nil)
	assert.Equal(t, "No active port forwards", result)

	result = formatPortForwardList([]*cluster.PortForwardSession{})
	assert.Equal(t, "No active port forwards", result)
}

func TestFormatPortForwardSession(t *testing.T) {
	session := &cluster.PortForwardSession{
		ID:         "pf-1",
		Namespace:  "default",
		Target:     "nginx",
		TargetType: "pod",
		LocalPort:  8080,
		RemotePort: 80,
		PodName:    "nginx",
	}

	result := formatPortForwardSession(session)

	assert.Contains(t, result, "Port forward started successfully")
	assert.Contains(t, result, "Session ID: pf-1")
	assert.Contains(t, result, "Namespace:  default")
	assert.Contains(t, result, "Target:     pod/nginx")
	assert.Contains(t, result, "localhost:8080 -> 80")
	assert.Contains(t, result, "http://localhost:8080")
}

func TestFormatPortForwardSession_Service(t *testing.T) {
	session := &cluster.PortForwardSession{
		ID:         "pf-2",
		Namespace:  "web",
		Target:     "my-service",
		TargetType: "service",
		LocalPort:  3000,
		RemotePort: 80,
		PodName:    "my-service-pod-abc123",
	}

	result := formatPortForwardSession(session)

	assert.Contains(t, result, "Session ID: pf-2")
	assert.Contains(t, result, "Namespace:  web")
	assert.Contains(t, result, "Target:     service/my-service")
	assert.Contains(t, result, "Pod:        my-service-pod-abc123")
	assert.Contains(t, result, "localhost:3000 -> 80")
}

func TestFormatPortForwardList(t *testing.T) {
	sessions := []*cluster.PortForwardSession{
		{
			ID:         "pf-1",
			Namespace:  "default",
			Target:     "nginx",
			TargetType: "pod",
			LocalPort:  8080,
			RemotePort: 80,
			PodName:    "nginx",
		},
		{
			ID:         "pf-2",
			Namespace:  "web",
			Target:     "my-service",
			TargetType: "service",
			LocalPort:  3000,
			RemotePort: 80,
			PodName:    "my-service-pod-abc123",
		},
	}

	result := formatPortForwardList(sessions)

	assert.Contains(t, result, "Active Port Forwards:")
	assert.Contains(t, result, "pf-1")
	assert.Contains(t, result, "pf-2")
	assert.Contains(t, result, "default")
	assert.Contains(t, result, "web")
	assert.Contains(t, result, "pod/nginx")
	assert.Contains(t, result, "service/my-service")
	assert.Contains(t, result, "8080:80")
	assert.Contains(t, result, "3000:80")
}
