package kai

import "github.com/mark3labs/mcp-go/server"

type KaiServer struct {
	Name    string
	Version string
}

func (k *KaiServer) NewServer() *server.MCPServer {
	return server.NewMCPServer(
		k.Name,
		k.Version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)
}
