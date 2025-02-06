package kai

import "github.com/mark3labs/mcp-go/server"

type KaiServer struct {
	Name    string
	Version string
}

func (k *KaiServer) InitServer() *server.MCPServer {
	return server.NewMCPServer(
		"Kubernetes MCP Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)
}
