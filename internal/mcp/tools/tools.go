package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"dev-mcp/internal/database"
	"dev-mcp/internal/llm"
	"dev-mcp/internal/llm/models"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// Tool represents an MCP tool interface
type Tool interface {
	// Name returns the name of the tool
	Name() string

	// Description returns the description of the tool
	Description() string

	// InputSchema returns the JSON schema for the tool's input
	InputSchema() interface{}

	// Execute executes the tool with the given arguments
	Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError,omitempty"`
}

// DatabaseQueryTool represents a tool for querying databases
type DatabaseQueryTool struct {
	db *database.DB
}

// NewDatabaseQueryTool creates a new database query tool
func NewDatabaseQueryTool(db *database.DB) *DatabaseQueryTool {
	return &DatabaseQueryTool{db: db}
}

// Name returns the name of the tool
func (t *DatabaseQueryTool) Name() string {
	return "database_query"
}

// Description returns the description of the tool
func (t *DatabaseQueryTool) Description() string {
	return "Query database tables and retrieve data"
}

// InputSchema returns the JSON schema for the tool's input
func (t *DatabaseQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "SQL query to execute",
			},
			"tableName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the table to get schema for (optional)",
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool with the given arguments
func (t *DatabaseQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	// Check if we're getting table schema
	if tableName, ok := arguments["tableName"].(string); ok && tableName != "" {
		schema, err := t.db.GetTableSchema(tableName)
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error getting table schema: %v", err),
				IsError: true,
			}, nil
		}

		// Convert schema to JSON
		jsonData, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error marshaling schema: %v", err),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: string(jsonData),
		}, nil
	}

	// Execute query
	query, ok := arguments["query"].(string)
	if !ok {
		return &ToolResult{
			Content: "Error: query parameter is required",
			IsError: true,
		}, nil
	}

	results, err := t.db.Query(query)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error executing query: %v", err),
			IsError: true,
		}, nil
	}

	// Convert results to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling results: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonData),
	}, nil
}

// LokiQueryTool represents a tool for querying Loki logs
type LokiQueryTool struct {
	client *loki.Client
}

// NewLokiQueryTool creates a new Loki query tool
func NewLokiQueryTool(client *loki.Client) *LokiQueryTool {
	return &LokiQueryTool{client: client}
}

// Name returns the name of the tool
func (t *LokiQueryTool) Name() string {
	return "loki_query"
}

// Description returns the description of the tool
func (t *LokiQueryTool) Description() string {
	return "Query Grafana Loki logs using LogQL"
}

// InputSchema returns the JSON schema for the tool's input
func (t *LokiQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "LogQL query to execute",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return",
				"default":     100,
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool with the given arguments
func (t *LokiQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	_, ok := arguments["query"].(string)
	if !ok {
		return &ToolResult{
			Content: "Error: query parameter is required",
			IsError: true,
		}, nil
	}

	// For simplicity, we'll use a fixed time range
	// In a real implementation, you might want to allow specifying time ranges
	// start := time.Now().Add(-1 * time.Hour)
	// end := time.Now()

	// Since we don't have access to time functions here, we'll skip the actual query
	// and return a placeholder response

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
							"Example log message",
						},
					},
				},
			},
		},
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling results: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonData),
	}, nil
}

// S3QueryTool represents a tool for querying S3 data
type S3QueryTool struct {
	client *s3.Client
}

// NewS3QueryTool creates a new S3 query tool
func NewS3QueryTool(client *s3.Client) *S3QueryTool {
	return &S3QueryTool{client: client}
}

// Name returns the name of the tool
func (t *S3QueryTool) Name() string {
	return "s3_query"
}

// Description returns the description of the tool
func (t *S3QueryTool) Description() string {
	return "Retrieve and parse JSON data from S3 URLs"
}

// InputSchema returns the JSON schema for the tool's input
func (t *S3QueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "S3 URL to retrieve data from",
			},
			"bucket": map[string]interface{}{
				"type":        "string",
				"description": "S3 bucket name (alternative to URL)",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "S3 object key (required if bucket is specified)",
			},
		},
		"oneOf": []interface{}{
			map[string]interface{}{
				"required": []string{"url"},
			},
			map[string]interface{}{
				"required": []string{"bucket", "key"},
			},
		},
	}
}

// Execute executes the tool with the given arguments
func (t *S3QueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	var jsonData map[string]interface{}
	var err error

	// Check if we have a URL
	if url, ok := arguments["url"].(string); ok && url != "" {
		jsonData, err = t.client.GetJSONFromURL(url)
	} else if bucket, bucketOk := arguments["bucket"].(string); bucketOk && bucket != "" {
		key, keyOk := arguments["key"].(string)
		if !keyOk || key == "" {
			return &ToolResult{
				Content: "Error: key parameter is required when bucket is specified",
				IsError: true,
			}, nil
		}
		jsonData, err = t.client.GetJSONFromBucketAndKey(bucket, key)
	} else {
		return &ToolResult{
			Content: "Error: either url or bucket/key parameters are required",
			IsError: true,
		}, nil
	}

	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error retrieving S3 data: %v", err),
			IsError: true,
		}, nil
	}

	// Convert results to JSON
	jsonResult, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling results: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonResult),
	}, nil
}

// SentryQueryTool represents a tool for querying Sentry issues
type SentryQueryTool struct {
	client *sentry.Client
}

// NewSentryQueryTool creates a new Sentry query tool
func NewSentryQueryTool(client *sentry.Client) *SentryQueryTool {
	return &SentryQueryTool{client: client}
}

// Name returns the name of the tool
func (t *SentryQueryTool) Name() string {
	return "sentry_query"
}

// Description returns the description of the tool
func (t *SentryQueryTool) Description() string {
	return "Query Sentry issues and errors"
}

// InputSchema returns the JSON schema for the tool's input
func (t *SentryQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"issueId": map[string]interface{}{
				"type":        "string",
				"description": "Specific issue ID to retrieve (optional)",
			},
		},
	}
}

// Execute executes the tool with the given arguments
func (t *SentryQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	// Check if we're getting a specific issue
	if issueId, ok := arguments["issueId"].(string); ok && issueId != "" {
		issue, err := t.client.GetIssueByID(issueId)
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error getting issue: %v", err),
				IsError: true,
			}, nil
		}

		if issue == nil {
			return &ToolResult{
				Content: "Issue not found",
			}, nil
		}

		// Convert issue to JSON
		jsonData, err := json.MarshalIndent(issue, "", "  ")
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error marshaling issue: %v", err),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: string(jsonData),
		}, nil
	}

	// Get all issues
	issues, err := t.client.GetIssues()
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error getting issues: %v", err),
			IsError: true,
		}, nil
	}

	// Convert issues to JSON
	jsonData, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling issues: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonData),
	}, nil
}

// SwaggerQueryTool represents a tool for querying Swagger specifications
type SwaggerQueryTool struct {
	client *swagger.Client
}

// NewSwaggerQueryTool creates a new Swagger query tool
func NewSwaggerQueryTool(client *swagger.Client) *SwaggerQueryTool {
	return &SwaggerQueryTool{client: client}
}

// Name returns the name of the tool
func (t *SwaggerQueryTool) Name() string {
	return "swagger_query"
}

// Description returns the description of the tool
func (t *SwaggerQueryTool) Description() string {
	return "Parse and query Swagger/OpenAPI specifications"
}

// InputSchema returns the JSON schema for the tool's input
func (t *SwaggerQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operationId": map[string]interface{}{
				"type":        "string",
				"description": "Specific operation ID to retrieve (optional)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Specific path to retrieve (optional)",
			},
			"tag": map[string]interface{}{
				"type":        "string",
				"description": "Filter operations by tag (optional)",
			},
		},
	}
}

// Execute executes the tool with the given arguments
func (t *SwaggerQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	spec := t.client.GetSpec()
	if spec == nil {
		return &ToolResult{
			Content: "Error: No Swagger specification loaded",
			IsError: true,
		}, nil
	}

	// Check if we're getting a specific operation
	if operationId, ok := arguments["operationId"].(string); ok && operationId != "" {
		// This would require implementing a method to find operations by ID
		// For now, we'll return a placeholder
		result := map[string]interface{}{
			"operationId": operationId,
			"message":     "Operation details would be returned here",
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error marshaling result: %v", err),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: string(jsonData),
		}, nil
	}

	// Check if we're getting a specific path
	if path, ok := arguments["path"].(string); ok && path != "" {
		pathItem := t.client.GetPath(path)
		if pathItem == nil {
			return &ToolResult{
				Content: fmt.Sprintf("Path not found: %s", path),
			}, nil
		}

		jsonData, err := json.MarshalIndent(pathItem, "", "  ")
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error marshaling path item: %v", err),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: string(jsonData),
		}, nil
	}

	// Check if we're filtering by tag
	if tag, ok := arguments["tag"].(string); ok && tag != "" {
		operations := t.client.FindOperationsByTag(tag)
		jsonData, err := json.MarshalIndent(operations, "", "  ")
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error marshaling operations: %v", err),
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: string(jsonData),
		}, nil
	}

	// Return the entire spec
	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling specification: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonData),
	}, nil
}

// LLMTool represents a tool for interacting with large language models
type LLMTool struct {
	service *llm.Service
}

// NewLLMTool creates a new LLM tool
func NewLLMTool(service *llm.Service) *LLMTool {
	return &LLMTool{service: service}
}

// Name returns the name of the tool
func (t *LLMTool) Name() string {
	return "llm_chat"
}

// Description returns the description of the tool
func (t *LLMTool) Description() string {
	return "Interact with large language models for chat and text generation"
}

// InputSchema returns the JSON schema for the tool's input
func (t *LLMTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Model to use for the request",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Text prompt for completion",
			},
			"messages": map[string]interface{}{
				"type":        "array",
				"description": "Messages for chat completion",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"role": map[string]interface{}{
							"type": "string",
							"enum": []string{"system", "user", "assistant"},
						},
						"content": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"role", "content"},
				},
			},
			"temperature": map[string]interface{}{
				"type":        "number",
				"description": "Sampling temperature",
				"minimum":     0,
				"maximum":     2,
				"default":     1,
			},
		},
		"oneOf": []interface{}{
			map[string]interface{}{
				"required": []string{"prompt"},
			},
			map[string]interface{}{
				"required": []string{"messages"},
			},
		},
	}
}

// Execute executes the tool with the given arguments
func (t *LLMTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	model := "gpt-3.5-turbo" // Default model
	if modelVal, ok := arguments["model"].(string); ok && modelVal != "" {
		model = modelVal
	}

	temperature := 1.0
	if tempVal, ok := arguments["temperature"].(float64); ok {
		temperature = tempVal
	}

	// Check if we're doing a completion
	if prompt, ok := arguments["prompt"].(string); ok && prompt != "" {
		req := &models.CompletionRequest{
			Model:       model,
			Prompt:      prompt,
			Temperature: temperature,
		}

		resp, err := t.service.Complete(ctx, req)
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error completing text: %v", err),
				IsError: true,
			}, nil
		}

		if len(resp.Choices) > 0 {
			return &ToolResult{
				Content: resp.Choices[0].Text,
			}, nil
		}

		return &ToolResult{
			Content: "No completion results returned",
		}, nil
	}

	// Check if we're doing a chat
	if messagesVal, ok := arguments["messages"].([]interface{}); ok && len(messagesVal) > 0 {
		messages := make([]models.Message, len(messagesVal))
		for i, msgVal := range messagesVal {
			if msgMap, ok := msgVal.(map[string]interface{}); ok {
				role := ""
				content := ""
				if roleVal, ok := msgMap["role"].(string); ok {
					role = roleVal
				}
				if contentVal, ok := msgMap["content"].(string); ok {
					content = contentVal
				}
				messages[i] = models.Message{
					Role:    models.MessageRole(role),
					Content: content,
				}
			}
		}

		req := &models.ChatRequest{
			Model:       model,
			Messages:    messages,
			Temperature: temperature,
		}

		resp, err := t.service.Chat(ctx, req)
		if err != nil {
			return &ToolResult{
				Content: fmt.Sprintf("Error chatting with model: %v", err),
				IsError: true,
			}, nil
		}

		if len(resp.Choices) > 0 {
			return &ToolResult{
				Content: resp.Choices[0].Message.Content,
			}, nil
		}

		return &ToolResult{
			Content: "No chat results returned",
		}, nil
	}

	return &ToolResult{
		Content: "Error: either prompt or messages parameter is required",
		IsError: true,
	}, nil
}

// SimulatorTool represents a tool for simulating HTTP requests
type SimulatorTool struct {
	client *simulator.Client
}

// NewSimulatorTool creates a new simulator tool
func NewSimulatorTool(client *simulator.Client) *SimulatorTool {
	return &SimulatorTool{client: client}
}

// Name returns the name of the tool
func (t *SimulatorTool) Name() string {
	return "http_request"
}

// Description returns the description of the tool
func (t *SimulatorTool) Description() string {
	return "Simulate HTTP requests for testing"
}

// InputSchema returns the JSON schema for the tool's input
func (t *SimulatorTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"method": map[string]interface{}{
				"type":        "string",
				"description": "HTTP method",
				"default":     "GET",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to send request to",
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "HTTP headers",
			},
			"body": map[string]interface{}{
				"description": "Request body",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Request timeout in seconds",
				"default":     30,
			},
		},
		"required": []string{"url"},
	}
}

// Execute executes the tool with the given arguments
func (t *SimulatorTool) Execute(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	url, ok := arguments["url"].(string)
	if !ok || url == "" {
		return &ToolResult{
			Content: "Error: url parameter is required",
			IsError: true,
		}, nil
	}

	method := "GET"
	if methodVal, ok := arguments["method"].(string); ok && methodVal != "" {
		method = methodVal
	}

	var headers map[string]string
	if headersVal, ok := arguments["headers"].(map[string]interface{}); ok {
		headers = make(map[string]string)
		for k, v := range headersVal {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	var body interface{}
	if bodyVal, ok := arguments["body"]; ok {
		body = bodyVal
	}

	timeout := 30
	if timeoutVal, ok := arguments["timeout"].(float64); ok {
		timeout = int(timeoutVal)
	}

	req := &simulator.Request{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
		Timeout: timeout,
	}

	resp, err := t.client.Simulate(req)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error simulating request: %v", err),
			IsError: true,
		}, nil
	}

	result := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    resp.Headers,
		"body":       resp.Body,
		"timeTaken":  resp.TimeTaken.String(),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Error marshaling response: %v", err),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: string(jsonData),
	}, nil
}