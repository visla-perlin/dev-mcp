package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/entity"
	"dev-mcp/internal/config"
	"dev-mcp/internal/provider"
)

// LokiProvider provides Loki log query functionality
type LokiProvider struct {
	*provider.BaseProvider
	client *Client
}

// NewLokiProvider creates a new Loki provider with config and server
func NewLokiProvider(cfg *config.LokiConfig, server *mcp.Server) *LokiProvider {
	p := &LokiProvider{
		BaseProvider: provider.NewBaseProvider("loki"),
	}

	// Initialize Loki client from config
	p.client = NewClient(cfg)

	if p.client.IsAvailable() {
		p.SetAvailable(true)
		// Add tools to server immediately
		p.addToolsToServer(server)
		log.Printf("✓ Loki provider initialized successfully")
	} else {
		p.SetStatus(false, "Loki client initialization failed", nil)
	}

	return p
}

// Close closes the Loki provider
func (p *LokiProvider) Close() error {
	return p.client.Close()
}

// Test tests the Loki configuration and connection (for ProviderClient interface compatibility)
func (p *LokiProvider) Test(config interface{}) error {
	// Since client is already initialized in constructor, just check availability
	if !p.IsAvailable() {
		return fmt.Errorf("loki provider not available")
	}
	return nil
}

// AddTools adds Loki tools to the MCP server (for ProviderClient interface compatibility)
func (p *LokiProvider) AddTools(server *mcp.Server, config interface{}) error {
	// Tools are already added in constructor, but we can call addToolsToServer again if needed
	p.addToolsToServer(server)
	return nil
}

// addToolsToServer adds Loki tools to the MCP server
func (p *LokiProvider) addToolsToServer(server *mcp.Server) {
	if !p.IsAvailable() {
		log.Printf("⚠ Loki provider not available, tools not added")
		return
	}

	// Add tools to server
	tools := []struct {
		tool    *mcp.Tool
		handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{p.createLokiQueryTool().Tool, p.createLokiQueryTool().Handler},
		{p.createLokiPresetQueryTool().Tool, p.createLokiPresetQueryTool().Handler},
		{p.createLokiListPresetsTool().Tool, p.createLokiListPresetsTool().Handler},
	}

	for _, tool := range tools {
		server.AddTool(tool.tool, tool.handler)
		log.Printf("✓ Registered Loki tool: %s", tool.tool.Name)
	}

	log.Printf("✓ All Loki tools registered successfully")
}

// createLokiQueryTool creates the Loki query tool
func (p *LokiProvider) createLokiQueryTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "loki_query",
		Description: "Query Grafana Loki logs using LogQL",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "LogQL query to execute"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of results to return",
					"default": 100
				}
			},
			"required": ["query"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Query == "" {
			return p.createErrorResult(fmt.Errorf("query parameter is required")), nil
		}

		// Set default limit
		if args.Limit == 0 {
			args.Limit = 100
		}

		// For demonstration purposes, return a mock result
		// In a real implementation, you would call p.client.Query(args.Query)
		result := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result": []interface{}{
					map[string]interface{}{
						"stream": map[string]interface{}{
							"job":      "api-server",
							"instance": "localhost:8080",
						},
						"values": [][]string{
							{"1640995200000000000", "INFO: API request received"},
							{"1640995201000000000", "INFO: Processing request"},
							{"1640995202000000000", "INFO: Request completed successfully"},
						},
					},
				},
			},
			"stats": map[string]interface{}{
				"summary": map[string]interface{}{
					"bytesTotal": 1024,
					"linesTotal": 3,
					"execTime":   0.1,
					"queueTime":  0.01,
				},
			},
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createLokiPresetQueryTool creates a tool to run predefined / parameterized queries.
func (p *LokiProvider) createLokiPresetQueryTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "loki_preset_query",
		Description: "Execute a predefined Loki query (use loki_list_presets to discover).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string", "description": "Preset query name"},
				"params": {"type": "object", "description": "Parameter key/value overrides"},
				"limit": {"type": "integer", "description": "Maximum number of results (for raw queries)", "default": 100}
			},
			"required": ["name"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name   string            `json:"name"`
			Params map[string]string `json:"params,omitempty"`
			Limit  int               `json:"limit,omitempty"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}
		if args.Name == "" {
			return p.createErrorResult(fmt.Errorf("name parameter is required")), nil
		}
		if args.Params == nil {
			args.Params = map[string]string{}
		}
		q, err := BuildPresetQuery(args.Name, args.Params)
		if err != nil {
			return p.createErrorResult(err), nil
		}
		// Simulate execution using mock client
		result, err := p.client.QueryLogs(q, args.Limit)
		if err != nil {
			return p.createErrorResult(err), nil
		}
		// Annotate result with preset metadata
		out := map[string]interface{}{
			"preset": args.Name,
			"query":  q,
			"result": result,
		}
		return p.formatJSONResult(out), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createLokiListPresetsTool lists available preset queries and their parameters.
func (p *LokiProvider) createLokiListPresetsTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "loki_list_presets",
		Description: "List available Loki preset queries and parameter metadata.",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		presets := ListPresetMetadata()
		// Build a compact textual table for readability in plain clients.
		var b strings.Builder
		b.WriteString("Available Loki Preset Queries (mock environment)\n\n")
		for _, pset := range presets {
			b.WriteString(pset.Name + ": " + pset.Description + "\n")
			if len(pset.Params) > 0 {
				b.WriteString("  Params:\n")
				// stable order
				keys := make([]string, 0, len(pset.Params))
				for k := range pset.Params {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					meta := pset.Params[k]
					def := meta.Default
					if def == "" {
						def = "(none)"
					}
					reqFlag := ""
					if meta.Required {
						reqFlag = " required"
					}
					b.WriteString(fmt.Sprintf("    - %s: %s (default=%s%s)\n", k, meta.Description, def, reqFlag))
				}
			}
			if pset.Example != "" {
				b.WriteString("  Example: " + pset.Example + "\n")
			}
			b.WriteString("\n")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: b.String()}}}, nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// Helper functions
func (p *LokiProvider) createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Loki Error: %v", err)}},
		IsError: true,
	}
}

func (p *LokiProvider) formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return p.createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}

// Verify that LokiProvider implements ProviderClient interface
var _ provider.ProviderClient = (*LokiProvider)(nil)
