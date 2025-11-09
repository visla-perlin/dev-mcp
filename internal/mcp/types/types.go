package types

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolDefinition represents a simplified tool definition for providers
type ToolDefinition struct {
	Tool    *mcp.Tool
	Handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// ToolRegistrar defines the interface for registering tools
type ToolRegistrar interface {
	RegisterTool(toolDef ToolDefinition)
}

// ResourceRegistrar defines the interface for registering resources (simplified for now)
type ResourceRegistrar interface {
	RegisterResource(resDef ResourceDefinition)
}

// ResourceDefinition represents a resource with its metadata and handler
// Simplified to generic interface for now - can be expanded later
type ResourceDefinition struct {
	Resource interface{}
	Handler  interface{}
}
