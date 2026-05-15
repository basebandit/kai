package tools

import (
	"testing"

	"github.com/basebandit/kai/testmocks"
	"github.com/stretchr/testify/mock"
)

func TestRegisterEventTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(1)

	RegisterEventTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

func TestRegisterNodeTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(5)

	RegisterNodeTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}

func TestRegisterHealthTools(t *testing.T) {
	mockServer := &testmocks.MockServer{}
	mockCM := testmocks.NewMockClusterManager()

	mockServer.On("AddTool", mock.AnythingOfType("mcp.Tool"), mock.AnythingOfType("server.ToolHandlerFunc")).Return().Times(3)

	RegisterHealthTools(mockServer, mockCM)

	mockServer.AssertExpectations(t)
}
