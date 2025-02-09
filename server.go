package kai

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server to provide additional functionality
type Server struct {
	mcpServer *server.MCPServer
}

// NewServer creates a new MCP server for Kubernetes
func NewServer() *Server {
	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"Kubernetes MCP Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	return &Server{
		mcpServer: mcpServer,
	}
}

// AddTool adds a tool to the MCP server
func (s *Server) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.mcpServer.AddTool(tool, handler)
}

// Serve starts the server
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}
