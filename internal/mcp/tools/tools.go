package tools

import (
	"context"
	"dev-mcp/entity"
	"dev-mcp/internal/provider/loki"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Helper function to create error result
func createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
		IsError: true,
	}
}

// Helper function to format JSON content
func formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}

// LokiQueryTool creates a tool definition for Loki log queries
func NewLokiQueryTool(client *loki.Client) entity.ToolDefinition {
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Query == "" {
			return createErrorResult(fmt.Errorf("query parameter is required")), nil
		}

		// For demonstration purposes, return a mock result
		// In a real implementation, you would call client.Query(args.Query)
		result := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result": []interface{}{
					map[string]interface{}{
						"stream": map[string]interface{}{
							"job":  "example",
							"host": "example-host",
						},
						"values": [][]interface{}{
							{
								"1234567890123456789",
								"Example log message for query: " + args.Query,
							},
						},
					},
				},
			},
		}

		return formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}
