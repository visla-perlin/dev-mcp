package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/loki"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// ToolDefinition represents a tool with its metadata and handler
type ToolDefinition struct {
	Tool    *mcp.Tool
	Handler mcp.ToolHandler
}

// ToolRegistrar defines the interface for registering tools.
type ToolRegistrar interface {
	RegisterTool(toolDef ToolDefinition)
}

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
func NewLokiQueryTool(client *loki.Client) ToolDefinition {
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

	return ToolDefinition{Tool: tool, Handler: handler}
}

// S3QueryTool creates a tool definition for S3 data retrieval
func NewS3QueryTool(client *s3.Client) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_query",
		Description: "Retrieve and parse JSON data from S3 URLs",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "S3 URL to retrieve data from"
				},
				"bucket": {
					"type": "string",
					"description": "S3 bucket name (alternative to URL)"
				},
				"key": {
					"type": "string",
					"description": "S3 object key (required if bucket is specified)"
				}
			},
			"oneOf": [
				{"required": ["url"]},
				{"required": ["bucket", "key"]}
			]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			URL    string `json:"url,omitempty"`
			Bucket string `json:"bucket,omitempty"`
			Key    string `json:"key,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		var jsonData map[string]interface{}
		var err error

		// Check if we have a URL
		if args.URL != "" {
			jsonData, err = client.GetJSONFromURL(args.URL)
		} else if args.Bucket != "" && args.Key != "" {
			jsonData, err = client.GetJSONFromBucketAndKey(args.Bucket, args.Key)
		} else {
			return createErrorResult(fmt.Errorf("either url or bucket/key parameters are required")), nil
		}

		if err != nil {
			return createErrorResult(fmt.Errorf("error retrieving S3 data: %w", err)), nil
		}

		return formatJSONResult(jsonData), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// SentryQueryTool creates a tool definition for Sentry issue queries
func NewSentryQueryTool(client *sentry.Client) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "sentry_query",
		Description: "Query Sentry issues and errors",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"issueId": {
					"type": "string",
					"description": "Specific issue ID to retrieve (optional)"
				}
			}
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			IssueID string `json:"issueId,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Check if we're getting a specific issue
		if args.IssueID != "" {
			issue, err := client.GetIssueByID(args.IssueID)
			if err != nil {
				return createErrorResult(fmt.Errorf("error getting issue: %w", err)), nil
			}

			if issue == nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "Issue not found"}},
				}, nil
			}

			return formatJSONResult(issue), nil
		}

		// Get all issues
		issues, err := client.GetIssues()
		if err != nil {
			return createErrorResult(fmt.Errorf("error getting issues: %w", err)), nil
		}

		return formatJSONResult(issues), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// SwaggerQueryTool creates a tool definition for Swagger API specification queries
func NewSwaggerQueryTool(client *swagger.Client) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "swagger_query",
		Description: "Parse and query Swagger/OpenAPI specifications",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operationId": {
					"type": "string",
					"description": "Specific operation ID to retrieve (optional)"
				},
				"path": {
					"type": "string",
					"description": "Specific path to retrieve (optional)"
				},
				"tag": {
					"type": "string",
					"description": "Filter operations by tag (optional)"
				}
			}
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			OperationID string `json:"operationId,omitempty"`
			Path        string `json:"path,omitempty"`
			Tag         string `json:"tag,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		spec := client.GetSpec()
		if spec == nil {
			return createErrorResult(fmt.Errorf("no Swagger specification loaded")), nil
		}

		// Check if we're getting a specific operation
		if args.OperationID != "" {
			// This would require implementing a method to find operations by ID
			result := map[string]interface{}{
				"operationId": args.OperationID,
				"message":     "Operation details would be returned here",
			}
			return formatJSONResult(result), nil
		}

		// Check if we're getting a specific path
		if args.Path != "" {
			pathItem := client.GetPath(args.Path)
			if pathItem == nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Path not found: %s", args.Path)}},
				}, nil
			}
			return formatJSONResult(pathItem), nil
		}

		// Check if we're filtering by tag
		if args.Tag != "" {
			operations := client.FindOperationsByTag(args.Tag)
			return formatJSONResult(operations), nil
		}

		// Return the entire spec
		return formatJSONResult(spec), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// SimulatorTool creates a tool definition for HTTP request simulation
func NewSimulatorTool(client *simulator.Client) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "http_request",
		Description: "Simulate HTTP requests for testing",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"method": {
					"type": "string",
					"description": "HTTP method",
					"default": "GET"
				},
				"url": {
					"type": "string",
					"description": "URL to send request to"
				},
				"headers": {
					"type": "object",
					"description": "HTTP headers"
				},
				"body": {
					"description": "Request body"
				},
				"timeout": {
					"type": "integer",
					"description": "Request timeout in seconds",
					"default": 30
				}
			},
			"required": ["url"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Method  string                 `json:"method,omitempty"`
			URL     string                 `json:"url"`
			Headers map[string]interface{} `json:"headers,omitempty"`
			Body    interface{}            `json:"body,omitempty"`
			Timeout int                    `json:"timeout,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.URL == "" {
			return createErrorResult(fmt.Errorf("url parameter is required")), nil
		}

		method := "GET"
		if args.Method != "" {
			method = args.Method
		}

		var headers map[string]string
		if args.Headers != nil {
			headers = make(map[string]string)
			for k, v := range args.Headers {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}

		timeout := 30
		if args.Timeout > 0 {
			timeout = args.Timeout
		}

		reqObj := &simulator.Request{
			Method:  method,
			URL:     args.URL,
			Headers: headers,
			Body:    args.Body,
			Timeout: timeout,
		}

		resp, err := client.Simulate(reqObj)
		if err != nil {
			return createErrorResult(fmt.Errorf("error simulating request: %w", err)), nil
		}

		result := map[string]interface{}{
			"statusCode": resp.StatusCode,
			"headers":    resp.Headers,
			"body":       resp.Body,
			"timeTaken":  resp.TimeTaken.String(),
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}
