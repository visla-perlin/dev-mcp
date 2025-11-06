package server

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/auth"
	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
	"dev-mcp/internal/errors"
	"dev-mcp/internal/logging"
	"dev-mcp/internal/mcp/resources"
	"dev-mcp/internal/mcp/tools"
)

// MCPServer represents an MCP server using the official Go SDK
type MCPServer struct {
	server         *mcp.Server
	services       *ServiceContainer
	authConfig     *auth.AuthConfig
	authMiddleware *auth.Middleware
	serviceManager *config.ServiceManager
}

// NewMCPServer creates a new MCP server using the official SDK
func NewMCPServer(cfg *config.Config, services *ServiceContainer) *MCPServer {
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
		services:       services,
		authConfig:     authConfig,
		authMiddleware: auth.NewMiddleware(authConfig),
		serviceManager: config.NewServiceManager(cfg),
	}

	// Initialize and register tools
	mcpServer.registerTools()

	// Register resources
	mcpServer.registerResources()

	return mcpServer
}

// registerTools registers all available tools with the MCP server
func (s *MCPServer) registerTools() {
	logger := logging.ServerLogger
	logger.Info("Registering MCP tools...")

	// Create tool registry
	registry := tools.NewToolRegistry(s, s.serviceManager)

	// Create tool context with all available services
	toolContext := &tools.ToolContext{
		DatabaseManager: s.services.DatabaseManager,
		LokiClient:      s.services.LokiClient,
		S3Client:        s.services.S3Client,
		SentryClient:    s.services.SentryClient,
		SwaggerClient:   s.services.SwaggerClient,
		SimulatorClient: s.services.SimulatorClient,
		ServiceManager:  s.serviceManager,
	}

	// Register all tools
	registry.RegisterAll(toolContext)

	logger.Info("All available tools registered successfully")
}

// registerResources registers all available resources with the MCP server
func (s *MCPServer) registerResources() {
	logger := logging.ServerLogger
	logger.Info("Registering MCP resources...")

	ctx := context.Background()

	// Get all resources from different resource managers
	var db *database.DB
	if s.services.DatabaseManager != nil && s.services.DatabaseManager.GetDatabase() != nil {
		// Create a wrapper to convert DatabaseInterface to DB for resource queries
		if enhancedDB, ok := s.services.DatabaseManager.GetDatabase().(*database.EnhancedDB); ok {
			db = &database.DB{DB: enhancedDB.GetDB()}
		}
	}
	allResources := resources.GetAllResources(ctx, db, s.services.LokiClient, s.services.S3Client, s.services.SwaggerClient)

	// Register each resource with the server
	for _, resDef := range allResources {
		s.server.AddResource(resDef.Resource, resDef.Handler)
		logger.Info("Registered resource", logging.String("uri", resDef.Resource.URI))
	}

	logger.Info("All available resources registered successfully", logging.Int("count", len(allResources)))
}

// RegisterTool adds a tool to the MCP server and logs it
func (s *MCPServer) RegisterTool(toolDef tools.ToolDefinition) {
	s.server.AddTool(toolDef.Tool, toolDef.Handler)
	logging.ServerLogger.Info("Registered tool", logging.String("name", toolDef.Tool.Name))
}

// AddResource adds a resource to the MCP server
func (s *MCPServer) AddResource(resource *mcp.Resource, handler mcp.ResourceHandler) {
	s.server.AddResource(resource, handler)
	log.Printf("Registered resource: %s", resource.URI)
}

// Start starts the MCP server with the specified transport mode
func (s *MCPServer) Start(transportMode string) error {
	logger := logging.ServerLogger
	logger.Info("Starting MCP server with authentication",
		logging.String("transport", transportMode),
		logging.String("auth_enabled", fmt.Sprintf("%t", s.authConfig.Enabled)))

	// Create context for the server
	ctx := context.Background()

	// Force SSE mode and use authenticated transport
	if transportMode != "sse" {
		logger.Info("forcing SSE mode for authentication support",
			logging.String("original_mode", transportMode))
		transportMode = "sse"
	}

	// Create authenticated SSE transport
	authTransport := NewAuthenticatedSSETransport(s.authConfig, 8080)

	// Start the authenticated transport
	logger.Info("starting authenticated SSE transport on port 8080")
	if err := authTransport.Start(ctx, s.server); err != nil {
		return errors.ServerWrap(err, "start", "failed to start authenticated SSE transport")
	}

	return nil
}

// Close closes the MCP server and performs cleanup
func (s *MCPServer) Close() {
	logger := logging.ServerLogger
	logger.Info("Closing MCP server...")
	// Any cleanup can be done here if needed
	// The official SDK handles the server lifecycle
}
