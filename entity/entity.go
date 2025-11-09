package entity

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolDefinition represents a simplified tool definition for providers
type ToolDefinition struct {
	Tool    *mcp.Tool
	Handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
