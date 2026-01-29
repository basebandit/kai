package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterOperationsTools registers all cluster operation tools with the server
func RegisterOperationsTools(s kai.ServerInterface, cm kai.ClusterManager) {
	manager, ok := cm.(*cluster.Manager)
	if !ok {
		return
	}

	registerPortForwardTools(s, manager)
}

// registerPortForwardTools registers port-forward-related tools
func registerPortForwardTools(s kai.ServerInterface, manager *cluster.Manager) {
	startPortForwardTool := mcp.NewTool("start_port_forward",
		mcp.WithDescription("Start port forwarding to a pod or service. Similar to 'kubectl port-forward'"),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Target to forward to. Use 'pod/name' or 'service/name' or 'svc/name' format"),
		),
		mcp.WithString("ports",
			mcp.Required(),
			mcp.Description("Port mapping in format 'LOCAL:REMOTE' (e.g., '8080:80') or just 'PORT' for same local and remote"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the target (defaults to current namespace)"),
		),
	)

	s.AddTool(startPortForwardTool, startPortForwardHandler(manager))

	stopPortForwardTool := mcp.NewTool("stop_port_forward",
		mcp.WithDescription("Stop an active port forwarding session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the port forward session to stop (e.g., 'pf-1')"),
		),
	)

	s.AddTool(stopPortForwardTool, stopPortForwardHandler(manager))

	listPortForwardsTool := mcp.NewTool("list_port_forwards",
		mcp.WithDescription("List all active port forwarding sessions"),
	)

	s.AddTool(listPortForwardsTool, listPortForwardsHandler(manager))
}

// startPortForwardHandler handles the start_port_forward tool
func startPortForwardHandler(manager *cluster.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		target, ok := request.Params.Arguments["target"].(string)
		if !ok || target == "" {
			return mcp.NewToolResultError("target is required"), nil
		}

		portsStr, ok := request.Params.Arguments["ports"].(string)
		if !ok || portsStr == "" {
			return mcp.NewToolResultError("ports is required"), nil
		}

		namespace := ""
		if ns, ok := request.Params.Arguments["namespace"].(string); ok {
			namespace = ns
		}

		targetType, targetName, err := parseTarget(target)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		localPort, remotePort, err := parsePortMapping(portsStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		session, err := manager.StartPortForward(ctx, namespace, targetType, targetName, localPort, remotePort)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start port forward: %s", err.Error())), nil
		}

		result := formatPortForwardSession(session)
		return mcp.NewToolResultText(result), nil
	}
}

// stopPortForwardHandler handles the stop_port_forward tool
func stopPortForwardHandler(manager *cluster.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, ok := request.Params.Arguments["session_id"].(string)
		if !ok || sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		err := manager.StopPortForward(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to stop port forward: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Port forward session %q stopped successfully", sessionID)), nil
	}
}

// listPortForwardsHandler handles the list_port_forwards tool
func listPortForwardsHandler(manager *cluster.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessions := manager.ListPortForwards()
		result := formatPortForwardList(sessions)
		return mcp.NewToolResultText(result), nil
	}
}

// parseTarget parses a target string like "pod/nginx" or "service/my-svc" or "svc/my-svc"
func parseTarget(target string) (targetType, targetName string, err error) {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid target format: %q (expected 'pod/name' or 'service/name' or 'svc/name')", target)
	}

	targetType = strings.ToLower(parts[0])
	targetName = parts[1]

	if targetName == "" {
		return "", "", fmt.Errorf("target name cannot be empty")
	}

	switch targetType {
	case "pod":
		return "pod", targetName, nil
	case "service", "svc":
		return "service", targetName, nil
	default:
		return "", "", fmt.Errorf("invalid target type: %q (expected 'pod', 'service', or 'svc')", targetType)
	}
}

// parsePortMapping parses a port mapping string like "8080:80" or "8080"
func parsePortMapping(portStr string) (localPort, remotePort int, err error) {
	parts := strings.Split(portStr, ":")
	switch len(parts) {
	case 1:
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid port: %s", parts[0])
		}
		if port < 1 || port > 65535 {
			return 0, 0, fmt.Errorf("port must be between 1 and 65535")
		}
		return port, port, nil
	case 2:
		local, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid local port: %s", parts[0])
		}
		remote, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid remote port: %s", parts[1])
		}
		if local < 1 || local > 65535 || remote < 1 || remote > 65535 {
			return 0, 0, fmt.Errorf("ports must be between 1 and 65535")
		}
		return local, remote, nil
	default:
		return 0, 0, fmt.Errorf("invalid port mapping format: %s (expected LOCAL:REMOTE or PORT)", portStr)
	}
}

// formatPortForwardSession formats a port forward session for display
func formatPortForwardSession(session *cluster.PortForwardSession) string {
	var sb strings.Builder
	sb.WriteString("Port forward started successfully\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("Session ID: %s\n", session.ID))
	sb.WriteString(fmt.Sprintf("Namespace:  %s\n", session.Namespace))
	sb.WriteString(fmt.Sprintf("Target:     %s/%s\n", session.TargetType, session.Target))
	if session.TargetType == "service" {
		sb.WriteString(fmt.Sprintf("Pod:        %s\n", session.PodName))
	}
	sb.WriteString(fmt.Sprintf("Forwarding: localhost:%d -> %d\n", session.LocalPort, session.RemotePort))
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("Access via: http://localhost:%d\n", session.LocalPort))
	return sb.String()
}

// formatPortForwardList formats a list of port forward sessions
func formatPortForwardList(sessions []*cluster.PortForwardSession) string {
	if len(sessions) == 0 {
		return "No active port forwards"
	}

	var sb strings.Builder
	sb.WriteString("Active Port Forwards:\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("%-10s %-15s %-25s %-15s %s\n",
		"ID", "NAMESPACE", "TARGET", "POD", "PORTS"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, session := range sessions {
		podDisplay := session.PodName
		if session.TargetType == "pod" {
			podDisplay = "-"
		}
		targetDisplay := fmt.Sprintf("%s/%s", session.TargetType, session.Target)
		if len(targetDisplay) > 25 {
			targetDisplay = targetDisplay[:22] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-10s %-15s %-25s %-15s %d:%d\n",
			session.ID,
			session.Namespace,
			targetDisplay,
			podDisplay,
			session.LocalPort,
			session.RemotePort,
		))
	}

	return sb.String()
}
