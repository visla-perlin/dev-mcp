package provider

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolDefinition represents a tool with its metadata and handler
type ToolDefinition struct {
	Tool    *mcp.Tool
	Handler mcp.ToolHandler
}

// ResourceDefinition represents a resource with its metadata and handler
type ResourceDefinition struct {
	Resource *mcp.Resource
	Handler  mcp.ResourceHandler
}

// ToolRegistrar defines the interface for registering tools
type ToolRegistrar interface {
	RegisterTool(toolDef ToolDefinition)
}

// ResourceRegistrar defines the interface for registering resources
type ResourceRegistrar interface {
	RegisterResource(resDef ResourceDefinition)
}

// Provider defines the interface for all MCP providers
type Provider interface {
	// Name returns the provider name
	Name() string
	
	// IsAvailable returns whether the provider is available and ready
	IsAvailable() bool
	
	// RegisterTools registers all tools provided by this provider
	RegisterTools(registrar ToolRegistrar) error
	
	// RegisterResources registers all resources provided by this provider
	RegisterResources(registrar ResourceRegistrar) error
	
	// Close cleans up the provider
	Close() error
	
	// HealthCheck performs health check
	HealthCheck() error
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name      string
	available bool
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string) *BaseProvider {
	return &BaseProvider{
		name:      name,
		available: false,
	}
}

// Name returns the provider name
func (bp *BaseProvider) Name() string {
	return bp.name
}

// IsAvailable returns whether the provider is available
func (bp *BaseProvider) IsAvailable() bool {
	return bp.available
}

// SetAvailable sets the availability status
func (bp *BaseProvider) SetAvailable(available bool) {
	bp.available = available
}

// Close provides default close implementation (can be overridden)
func (bp *BaseProvider) Close() error {
	return nil
}

// HealthCheck provides default health check implementation (can be overridden)
func (bp *BaseProvider) HealthCheck() error {
	if !bp.available {
		return nil // No error if not available - just not ready
	}
	return nil
}