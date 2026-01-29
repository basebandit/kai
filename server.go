package kai

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server to provide additional behavior
type Server struct {
	mcpServer *server.MCPServer
}

// ServerOption configures the server
type ServerOption func(*serverConfig)

type serverConfig struct {
	version string
}

// WithVersion sets the server version
func WithVersion(version string) ServerOption {
	return func(c *serverConfig) {
		c.version = version
	}
}

// NewServer creates a new MCP server for Kubernetes
func NewServer(opts ...ServerOption) *Server {
	cfg := &serverConfig{
		version: "0.0.1",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"Kubernetes MCP Server",
		cfg.version,
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

// Serve starts the server using stdio transport (for Claude Desktop, etc.)
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// ServeSSE starts the server using SSE transport (for web clients)
func (s *Server) ServeSSE(addr string) error {
	sseServer := server.NewSSEServer(s.mcpServer)
	return sseServer.Start(addr)
}
