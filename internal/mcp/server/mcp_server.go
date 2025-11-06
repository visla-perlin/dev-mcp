package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp/transport"

	"dev-mcp/internal/database"
	"dev-mcp/internal/llm"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/mcp/tools"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// MCPServer represents an MCP server using the official Go SDK
type MCPServer struct {
	server       *mcp.Server
	tools        []tools.Tool
	toolsByName  map[string]tools.Tool
	db           *database.DB
	lokiClient   *loki.Client
	s3Client     *s3.Client
	sentryClient *sentry.Client
	swaggerClient *swagger.Client
	llmService   *llm.Service
	simulatorClient *simulator.Client
}

// NewMCPServer creates a new MCP server using the official SDK
func NewMCPServer(
	db *database.DB,
	lokiClient *loki.Client,
	s3Client *s3.Client,
	sentryClient *sentry.Client,
	swaggerClient *swagger.Client,
	llmService *llm.Service,
	simulatorClient *simulator.Client,
) *MCPServer {
	// Create MCP server
	server := mcp.NewServer(
		"dev-mcp-server",
		"1.0.0",
		mcp.WithServerInfo(mcp.ServerInfo{
			Name:    "Dev MCP Server",
			Version: "1.0.0",
		}),
	)

	mcpServer := &MCPServer{
		server:          server,
		tools:           make([]tools.Tool, 0),
		toolsByName:     make(map[string]tools.Tool),
		db:              db,
		lokiClient:      lokiClient,
		s3Client:        s3Client,
		sentryClient:    sentryClient,
		swaggerClient:   swaggerClient,
		llmService:      llmService,
		simulatorClient: simulatorClient,
	}

	// Initialize tools
	mcpServer.initializeTools()

	// Register tools with MCP server
	mcpServer.registerTools()

	return mcpServer
}

// initializeTools initializes all MCP tools
func (s *MCPServer) initializeTools() {
	// Add database query tool if database is available
	if s.db != nil {
		dbTool := tools.NewDatabaseQueryTool(s.db)
		s.tools = append(s.tools, dbTool)
		s.toolsByName[dbTool.Name()] = dbTool
	}

	// Add Loki query tool if Loki client is available
	if s.lokiClient != nil {
		lokiTool := tools.NewLokiQueryTool(s.lokiClient)
		s.tools = append(s.tools, lokiTool)
		s.toolsByName[lokiTool.Name()] = lokiTool
	}

	// Add S3 query tool if S3 client is available
	if s.s3Client != nil {
		s3Tool := tools.NewS3QueryTool(s.s3Client)
		s.tools = append(s.tools, s3Tool)
		s.toolsByName[s3Tool.Name()] = s3Tool
	}

	// Add Sentry query tool if Sentry client is available
	if s.sentryClient != nil {
		sentryTool := tools.NewSentryQueryTool(s.sentryClient)
		s.tools = append(s.tools, sentryTool)
		s.toolsByName[sentryTool.Name()] = sentryTool
	}

	// Add Swagger query tool if Swagger client is available
	if s.swaggerClient != nil {
		swaggerTool := tools.NewSwaggerQueryTool(s.swaggerClient)
		s.tools = append(s.tools, swaggerTool)
		s.toolsByName[swaggerTool.Name()] = swaggerTool
	}

	// Add LLM tool if LLM service is available
	if s.llmService != nil {
		llmTool := tools.NewLLMTool(s.llmService)
		s.tools = append(s.tools, llmTool)
		s.toolsByName[llmTool.Name()] = llmTool
	}

	// Add simulator tool if simulator client is available
	if s.simulatorClient != nil {
		simulatorTool := tools.NewSimulatorTool(s.simulatorClient)
		s.tools = append(s.tools, simulatorTool)
		s.toolsByName[simulatorTool.Name()] = simulatorTool
	}
}

// registerTools registers tools with the MCP server
func (s *MCPServer) registerTools() {
	for _, tool := range s.tools {
		s.server.AddTool(
			mcp.Tool{
				Name:        tool.Name(),
				Description: tool.Description(),
				InputSchema: mcp.ToolInputSchema{
					Type:       "object",
					Properties: tool.InputSchema(),
				},
			},
			s.handleToolCall(tool.Name(), tool),
		)
	}
}

// handleToolCall returns a handler function for tool calls
func (s *MCPServer) handleToolCall(toolName string, tool tools.Tool) mcp.ToolHandler {
	return func(ctx context.Context, arguments map[interface{}]interface{}) (*mcp.CallToolResult, error) {
		// Convert the map[interface{}]interface{} to map[string]interface{}
		args := make(map[string]interface{})
		for k, v := range arguments {
			if keyStr, ok := k.(string); ok {
				args[keyStr] = v
			}
		}

		// Execute the tool
		result, err := tool.Execute(ctx, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: fmt.Sprintf("Tool execution error: %v", err),
					},
				},
				IsError: true,
			}, err
		}

		// Convert result content
		content := make([]mcp.Content, 0)
		for _, c := range result.Content {
            if textContent, isText := c.(map[string]interface{})["text"].(string); isText {
                content = append(content, mcp.Content{
                    Type: "text",
                    Text: textContent,
                })
            }
		}

		return &mcp.CallToolResult{
			Content: content,
			IsError: result.IsError,
		}, nil
	}
}

// Start starts the MCP server
func (s *MCPServer) Start(ctx context.Context) error {
	// Create a new stdio transport for the server
    transport := transport.NewStdioTransport()

    // Start the server with the transport
    return s.server.Serve(ctx, transport)
}

// Close closes the MCP server
func (s *MCPServer) Close() {
	// The MCP server doesn't have an explicit Close method in the current SDK
	// Any cleanup can be done here if needed
}