package tools

import (
	"fmt"
	"strconv"
	"strings"
)

// validateContainerPort checks if the containerPort string has the correct format
// Returns true if valid, false if invalid
func validateContainerPort(port string) error {
	parts := strings.Split(port, "/")
	if len(parts) > 2 {
		return fmt.Errorf("invalid container port format: %s", port)
	}

	portNum, err := strconv.Atoi(parts[0])
	if err != nil || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("invalid port %v:%w, Port must be a number between 1 and 65535", parts[0], err)
	}

	if len(parts) == 2 {
		protocol := strings.ToUpper(parts[1])
		if protocol != "TCP" && protocol != "UDP" && protocol != "SCTP" {
			return fmt.Errorf("invalid protocol: %s. Must be TCP, UDP, or SCTP", parts[1])
		}
	}

	return nil
}

// validateImagePullPolicy checks if the image pull policy is one of "Always", "IfNotPresent", or "Never"
func validateImagePullPolicy(policy string) error {
	validPolicies := map[string]bool{
		"Always":       true,
		"IfNotPresent": true,
		"Never":        true,
	}
	if !validPolicies[policy] {
		return fmt.Errorf("invalid image_pull_policy: %s. Must be one of: Always, IfNotPresent, Never", policy)
	}
	return nil
}

// validateRestartPolicy checks if the restart policy is one of "Always", "OnFailure", or "Never"
func validateRestartPolicy(policy string) error {
	validPolicies := map[string]bool{
		"Always":    true,
		"OnFailure": true,
		"Never":     true,
	}
	if !validPolicies[policy] {
		return fmt.Errorf("invalid restart_policy: %s. Must be one of: Always, OnFailure, Never", policy)
	}
	return nil
}
