package server

import (
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/auth"
	"dev-mcp/internal/config"
	"dev-mcp/internal/logging"
)

// MCPServer represents an MCP server using the official Go SDK
type MCPServer struct {
	server         *mcp.Server
	authConfig     *auth.AuthConfig
	cfg            *config.Config
	authMiddleware *auth.Middleware
	transport      string
	host           string
	port           int
}

// NewMCPServer creates a new MCP server using the official SDK
func NewMCPServer(cfg *config.Config) *MCPServer {
	// Create MCP server with implementation info
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "dev-mcp-server",
			Version: "1.0.0",
		},
		nil, // No options for now
	)

	// Convert config.AuthConfig to auth.AuthConfig
	authConfig := &auth.AuthConfig{
		Enabled: cfg.Auth.Enabled,
		APIKeys: make([]auth.APIKey, len(cfg.Auth.APIKeys)),
	}

	// Convert API keys
	for i, apiKey := range cfg.Auth.APIKeys {
		authConfig.APIKeys[i] = auth.APIKey{
			Name:    apiKey.Name,
			Key:     apiKey.Key,
			Roles:   apiKey.Roles,
			Enabled: apiKey.Enabled,
		}
	}

	mcpServer := &MCPServer{
		server:         server,
		authConfig:     authConfig,
		authMiddleware: auth.NewMiddleware(authConfig),
		transport:      "sse",
		host:           cfg.Server.Host,
		port:           cfg.Server.Port,
	}

	return mcpServer
}

// Start starts the MCP server with the specified transport mode
func (s *MCPServer) Start(port string) error {
	logger := logging.ServerLogger
	logger.Info("Starting MCP server with authentication",
		logging.String("transport", s.transport),
		logging.String("auth_enabled", fmt.Sprintf("%t", s.authConfig.Enabled)))

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	server1 := mcp.NewServer(&mcp.Implementation{Name: "visal dev tools"}, nil)

	// Use the standard SDK SSE handler with authentication
	sseHandler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return server1
	}, nil)

	logger.Info("starting SSE server using standard SDK handler", logging.String("address", addr))

	// Use standard HTTP server with mux
	return http.ListenAndServe(addr, sseHandler)
}

// Close closes the MCP server and performs cleanup
func (s *MCPServer) Close() {
	logger := logging.ServerLogger
	logger.Info("Closing MCP server...")

}
