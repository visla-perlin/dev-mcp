package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/entity"
	"dev-mcp/internal/config"
	"dev-mcp/internal/provider"
)

// SentryProvider provides Sentry error tracking functionality
type SentryProvider struct {
	*provider.BaseProvider
	client *SentryClient
}

// NewSentryProvider creates a new Sentry provider with config and server
func NewSentryProvider(cfg *config.SentryConfig, server *mcp.Server) *SentryProvider {
	p := &SentryProvider{
		BaseProvider: provider.NewBaseProvider("sentry"),
	}

	// Initialize Sentry client from config
	p.client = NewSentryClient(cfg)

	if p.client != nil && p.client.IsAvailable() {
		p.SetAvailable(true)
		// Add tools to server immediately
		p.addToolsToServer(server)
		log.Printf("✓ Sentry provider initialized successfully")
	} else {
		p.SetStatus(false, "Sentry client initialization failed", nil)
	}

	return p
}

// Test tests the Sentry configuration and connection (for ProviderClient interface compatibility)
func (p *SentryProvider) Test(config interface{}) error {
	// Since client is already initialized in constructor, just check availability
	if !p.IsAvailable() {
		return fmt.Errorf("sentry provider not available")
	}
	return nil
}

// AddTools adds Sentry tools to the MCP server (for ProviderClient interface compatibility)
func (p *SentryProvider) AddTools(server *mcp.Server, config interface{}) error {
	// Tools are already added in constructor, but we can call addToolsToServer again if needed
	p.addToolsToServer(server)
	return nil
}

// addToolsToServer adds Sentry tools to the MCP server
func (p *SentryProvider) addToolsToServer(server *mcp.Server) {
	if !p.IsAvailable() {
		log.Printf("⚠ Sentry provider not available, tools not added")
		return
	}

	// Add tools to server
	tools := []struct {
		tool    *mcp.Tool
		handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{p.createGetIssuesTools().Tool, p.createGetIssuesTools().Handler},
		{p.createGetIssueDetailsTool().Tool, p.createGetIssueDetailsTool().Handler},
	}

	for _, tool := range tools {
		server.AddTool(tool.tool, tool.handler)
		log.Printf("✓ Registered Sentry tool: %s", tool.tool.Name)
	}

	log.Printf("✓ All Sentry tools registered successfully")
}

// Close closes the Sentry provider
func (p *SentryProvider) Close() error {
	return p.client.Close()
}

// createGetIssuesTools creates the get issues tool
func (p *SentryProvider) createGetIssuesTools() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "sentry_get_issues",
		Description: "Get Sentry issues with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Search query to filter issues",
					"default": ""
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of issues to return",
					"default": 50
				}
			}
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Query string `json:"query,omitempty"`
			Limit int    `json:"limit,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		result, err := p.client.GetIssues(args.Query, args.Limit)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createGetIssueDetailsTool creates the get issue details tool
func (p *SentryProvider) createGetIssueDetailsTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "sentry_get_issue_details",
		Description: "Get detailed information about a specific Sentry issue",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"issue_id": {
					"type": "string",
					"description": "The ID of the issue to retrieve details for"
				}
			},
			"required": ["issue_id"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			IssueID string `json:"issue_id"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.IssueID == "" {
			return p.createErrorResult(fmt.Errorf("issue_id is required")), nil
		}

		result, err := p.client.GetIssueDetails(args.IssueID)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// Helper functions
func (p *SentryProvider) createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Sentry Error: %v", err)}},
		IsError: true,
	}
}

func (p *SentryProvider) formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return p.createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}

// Verify that SentryProvider implements ProviderClient interface
var _ provider.ProviderClient = (*SentryProvider)(nil)
