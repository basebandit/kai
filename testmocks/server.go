package testmocks

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/mock"
)

// MockServer to check that each respective tool is registered correctly
type MockServer struct {
	mock.Mock
}

func (m *MockServer) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	m.Called(tool, handler)
}

func (m *MockServer) Serve() error {
	args := m.Called()
	return args.Error(0)
}
