package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/entity"
	appcfg "dev-mcp/internal/config"
	"dev-mcp/internal/provider"
)

// S3Provider provides S3 storage functionality
type S3Provider struct {
	*provider.BaseProvider
	client *S3Client
}

// NewS3Provider creates a new S3 provider with config and server
func NewS3Provider(cfg *appcfg.S3Config, server *mcp.Server) *S3Provider {
	p := &S3Provider{
		BaseProvider: provider.NewBaseProvider("s3"),
	}

	// Initialize S3 client from config
	p.client = NewS3Client(cfg)

	if p.client.IsAvailable() {
		p.SetAvailable(true)
		// Add tools to server immediately
		p.addToolsToServer(server)
		log.Printf("✓ S3 provider initialized successfully")
	} else {
		p.SetStatus(false, "S3 client initialization failed", nil)
	}

	return p
}

// Test tests the S3 configuration and connection (for ProviderClient interface compatibility)
func (p *S3Provider) Test(config interface{}) error {
	// Since client is already initialized in constructor, just check availability
	if !p.IsAvailable() {
		return fmt.Errorf("s3 provider not available")
	}
	return nil
}

// AddTools adds S3 tools to the MCP server (for ProviderClient interface compatibility)
func (p *S3Provider) AddTools(server *mcp.Server, config interface{}) error {
	// Tools are already added in constructor, but we can call addToolsToServer again if needed
	p.addToolsToServer(server)
	return nil
}

// addToolsToServer adds S3 tools to the MCP server
func (p *S3Provider) addToolsToServer(server *mcp.Server) {
	if !p.IsAvailable() {
		log.Printf("⚠ S3 provider not available, tools not added")
		return
	}

	// Add tools to server
	tools := []struct {
		tool    *mcp.Tool
		handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{p.createS3GetContentTool().Tool, p.createS3GetContentTool().Handler},
		{p.createS3ListObjectsTool().Tool, p.createS3ListObjectsTool().Handler},
		{p.createS3GetObjectSizeTool().Tool, p.createS3GetObjectSizeTool().Handler},
		{p.createS3GetBucketSizeTool().Tool, p.createS3GetBucketSizeTool().Handler},
		{p.createS3GetSizeStatisticsTool().Tool, p.createS3GetSizeStatisticsTool().Handler},
	}

	for _, tool := range tools {
		server.AddTool(tool.tool, tool.handler)
		log.Printf("✓ Registered S3 tool: %s", tool.tool.Name)
	}

	log.Printf("✓ All S3 tools registered successfully")
}

// createS3GetContentTool creates the S3 get content tool (文本文件+自动签名)
func (p *S3Provider) createS3GetContentTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_get_content",
		Description: "Get the content of a text file (json, txt, csv, xml) from S3, returns signed URL if needed",
		InputSchema: json.RawMessage(`{
		       "type": "object",
		       "properties": {
			       "bucket": {
				       "type": "string",
				       "description": "S3 bucket name"
			       },
			       "key": {
				       "type": "string",
				       "description": "Object key (must be .json, .txt, .csv, .xml)"
			       }
		       },
		       "required": ["bucket", "key"]
	       }`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket string `json:"bucket"`
			Key    string `json:"key"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" || args.Key == "" {
			return p.createErrorResult(fmt.Errorf("bucket and key parameters are required")), nil
		}

		result, err := p.client.GetContent(args.Bucket, args.Key)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createS3SignUrlTool creates the S3 sign url tool
func (p *S3Provider) createS3SignUrlTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_sign_url",
		Description: "Generate a signed URL for an S3 object",
		InputSchema: json.RawMessage(`{
		       "type": "object",
		       "properties": {
			       "bucket": {
				       "type": "string",
				       "description": "S3 bucket name"
			       },
			       "key": {
				       "type": "string",
				       "description": "Object key"
			       },
			       "expireSeconds": {
				       "type": "integer",
				       "description": "Expiration time in seconds",
				       "default": 600
			       }
		       },
		       "required": ["bucket", "key"]
	       }`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket        string `json:"bucket"`
			Key           string `json:"key"`
			ExpireSeconds int32  `json:"expireSeconds"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" || args.Key == "" {
			return p.createErrorResult(fmt.Errorf("bucket and key parameters are required")), nil
		}

		url, err := p.client.GetSignedURL(args.Bucket, args.Key, args.ExpireSeconds)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		result := map[string]interface{}{
			"bucket":    args.Bucket,
			"key":       args.Key,
			"signedUrl": url,
		}
		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// Close closes the S3 provider
func (p *S3Provider) Close() error {
	return p.client.Close()
}

// createS3GetObjectTool creates the S3 get object tool
func (p *S3Provider) createS3GetObjectTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_get_object",
		Description: "Retrieve objects from S3",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"bucket": {
					"type": "string",
					"description": "S3 bucket name"
				},
				"key": {
					"type": "string",
					"description": "Object key"
				}
			},
			"required": ["bucket", "key"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket string `json:"bucket"`
			Key    string `json:"key"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" || args.Key == "" {
			return p.createErrorResult(fmt.Errorf("bucket and key parameters are required")), nil
		}

		// Use the S3 client to get object
		result, err := p.client.GetContent(args.Bucket, args.Key)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createS3ListObjectsTool creates the S3 list objects tool
func (p *S3Provider) createS3ListObjectsTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_list_objects",
		Description: "List objects in S3 bucket",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"bucket": {
					"type": "string",
					"description": "S3 bucket name"
				},
				"prefix": {
					"type": "string",
					"description": "Object key prefix (optional)"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of objects to return",
					"default": 100
				}
			},
			"required": ["bucket"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket string `json:"bucket"`
			Prefix string `json:"prefix,omitempty"`
			Limit  int    `json:"limit,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" {
			return p.createErrorResult(fmt.Errorf("bucket parameter is required")), nil
		}

		if args.Limit == 0 {
			args.Limit = 100
		}

		// Use the S3 client to list objects
		result, err := p.client.ListObjects(args.Bucket, args.Prefix, args.Limit)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// Helper functions
func (p *S3Provider) createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("S3 Error: %v", err)}},
		IsError: true,
	}
}

func (p *S3Provider) formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return p.createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}

// createS3GetObjectSizeTool creates the S3 get object size tool
func (p *S3Provider) createS3GetObjectSizeTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_get_object_size",
		Description: "Get the size of a specific S3 object",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"bucket": {
					"type": "string",
					"description": "S3 bucket name"
				},
				"key": {
					"type": "string", 
					"description": "Object key"
				},
				"detailed": {
					"type": "boolean",
					"description": "Return detailed size information",
					"default": false
				}
			},
			"required": ["bucket", "key"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket   string `json:"bucket"`
			Key      string `json:"key"`
			Detailed bool   `json:"detailed,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" || args.Key == "" {
			return p.createErrorResult(fmt.Errorf("bucket and key parameters are required")), nil
		}

		if args.Detailed {
			// Return detailed size information
			result, err := p.client.GetObjectSizeInfo(args.Bucket, args.Key)
			if err != nil {
				return p.createErrorResult(err), nil
			}
			return p.formatJSONResult(result), nil
		} else {
			// Return simple size in bytes
			size, err := p.client.GetObjectSize(args.Bucket, args.Key)
			if err != nil {
				return p.createErrorResult(err), nil
			}
			result := map[string]interface{}{
				"bucket":    args.Bucket,
				"key":       args.Key,
				"sizeBytes": size,
			}
			return p.formatJSONResult(result), nil
		}
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createS3GetBucketSizeTool creates the S3 get bucket size tool
func (p *S3Provider) createS3GetBucketSizeTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_get_bucket_size",
		Description: "Get the total size and statistics of an S3 bucket",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"bucket": {
					"type": "string",
					"description": "S3 bucket name"
				}
			},
			"required": ["bucket"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket string `json:"bucket"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" {
			return p.createErrorResult(fmt.Errorf("bucket parameter is required")), nil
		}

		result, err := p.client.GetBucketSize(args.Bucket)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createS3GetSizeStatisticsTool creates the S3 get size statistics tool
func (p *S3Provider) createS3GetSizeStatisticsTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "s3_get_size_statistics",
		Description: "Get comprehensive size statistics for objects with a specific prefix",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"bucket": {
					"type": "string",
					"description": "S3 bucket name"
				},
				"prefix": {
					"type": "string",
					"description": "Object key prefix to filter statistics",
					"default": ""
				}
			},
			"required": ["bucket"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Bucket string `json:"bucket"`
			Prefix string `json:"prefix,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Bucket == "" {
			return p.createErrorResult(fmt.Errorf("bucket parameter is required")), nil
		}

		result, err := p.client.GetSizeStatistics(args.Bucket, args.Prefix)
		if err != nil {
			return p.createErrorResult(err), nil
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}
