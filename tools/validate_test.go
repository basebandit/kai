package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateContainerPort(t *testing.T) {
	testCases := []struct {
		name        string
		port        string
		expectError bool
		errContains string
	}{
		{"Valid port number", "8080", false, ""},
		{"Valid port with TCP protocol", "80/TCP", false, ""},
		{"Valid port with tcp lowercase", "443/tcp", false, ""},
		{"Valid port with UDP protocol", "53/UDP", false, ""},
		{"Valid port with udp lowercase", "53/udp", false, ""},
		{"Valid port with SCTP protocol", "9999/SCTP", false, ""},
		{"Valid port with sctp lowercase", "9999/sctp", false, ""},
		{"Valid min port", "1", false, ""},
		{"Valid max port", "65535", false, ""},
		{"Invalid port zero", "0", true, "invalid port"},
		{"Invalid port negative", "-1", true, "invalid port"},
		{"Invalid port exceeds max", "65536", true, "invalid port"},
		{"Invalid port non-numeric", "abc", true, "invalid port"},
		{"Invalid port empty string", "", true, "invalid port"},
		{"Invalid protocol", "8080/HTTP", true, "invalid protocol"},
		{"Invalid protocol unknown", "8080/QUIC", true, "invalid protocol"},
		{"Too many parts", "8080/TCP/extra", true, "invalid container port format"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateContainerPort(tc.port)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateImagePullPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		policy      string
		expectError bool
	}{
		{"Valid Always", "Always", false},
		{"Valid IfNotPresent", "IfNotPresent", false},
		{"Valid Never", "Never", false},
		{"Invalid lowercase always", "always", true},
		{"Invalid uppercase ALWAYS", "ALWAYS", true},
		{"Invalid empty string", "", true},
		{"Invalid random string", "sometimes", true},
		{"Invalid ifnotpresent lowercase", "ifnotpresent", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateImagePullPolicy(tc.policy)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image_pull_policy")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRestartPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		policy      string
		expectError bool
	}{
		{"Valid Always", "Always", false},
		{"Valid OnFailure", "OnFailure", false},
		{"Valid Never", "Never", false},
		{"Invalid lowercase always", "always", true},
		{"Invalid onfailure lowercase", "onfailure", true},
		{"Invalid empty string", "", true},
		{"Invalid random string", "Restart", true},
		{"Invalid uppercase NEVER", "NEVER", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRestartPolicy(tc.policy)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid restart_policy")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
