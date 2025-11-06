package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"dev-mcp/internal/database"
	"dev-mcp/internal/llm"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/mcp/tools"
	"dev-mcp/internal/mcp/types"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// Server represents an MCP server
type Server struct {
	tools       []tools.Tool
	toolsByName map[string]tools.Tool
	db          *database.DB
	lokiClient  *loki.Client
	s3Client    *s3.Client
	sentryClient *sentry.Client
	swaggerClient *swagger.Client
	llmService   *llm.Service
	simulatorClient *simulator.Client
	mu          sync.RWMutex
}

// NewServer creates a new MCP server
func NewServer(
	db *database.DB,
	lokiClient *loki.Client,
	s3Client *s3.Client,
	sentryClient *sentry.Client,
	swaggerClient *swagger.Client,
	llmService *llm.Service,
	simulatorClient *simulator.Client,
) *Server {
	server := &Server{
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
	server.initializeTools()

	return server
}

// initializeTools initializes all MCP tools
func (s *Server) initializeTools() {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// Start starts the MCP server
func (s *Server) Start() error {
	fmt.Println("MCP Server started")

	// Create a scanner to read from stdin
	scanner := bufio.NewScanner(os.Stdin)

	// Process requests
	for scanner.Scan() {
		line := scanner.Text()

		// Parse the request
		var request types.Request
		err := json.Unmarshal([]byte(line), &request)
		if err != nil {
			s.sendError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
			continue
		}

		// Handle the request
		s.handleRequest(&request)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	return nil
}

// handleRequest handles an MCP request
func (s *Server) handleRequest(request *types.Request) {
	switch request.Method {
	case "initialize":
		s.handleInitialize(request)
	case "tools/list":
		s.handleListTools(request)
	case "tools/call":
		s.handleCallTool(request)
	default:
		s.sendError(request.ID, -32601, fmt.Sprintf("Method not found: %s", request.Method))
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(request *types.Request) {
	var initReq types.InitializeRequest
	err := json.Unmarshal(request.Params, &initReq)
	if err != nil {
		s.sendError(request.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	// Create the response
	response := types.InitializeResponse{
		ProtocolVersion: "1.0.0",
		Capabilities:    []string{"tools"},
		ServerInfo: types.ServerInfo{
			Name:    "Dev MCP Server",
			Version: "1.0.0",
		},
		Tools: s.getToolsForResponse(),
	}

	s.sendResponse(request.ID, response)
}

// handleListTools handles the tools/list request
func (s *Server) handleListTools(request *types.Request) {
	response := types.ListToolsResponse{
		Tools: s.getToolsForResponse(),
	}

	s.sendResponse(request.ID, response)
}

// handleCallTool handles the tools/call request
func (s *Server) handleCallTool(request *types.Request) {
	var callReq types.CallToolRequest
	err := json.Unmarshal(request.Params, &callReq)
	if err != nil {
		s.sendError(request.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	// Find the tool
	s.mu.RLock()
	tool, exists := s.toolsByName[callReq.Name]
	s.mu.RUnlock()

	if !exists {
		s.sendError(request.ID, -32601, fmt.Sprintf("Tool not found: %s", callReq.Name))
		return
	}

	// Execute the tool
	result, err := tool.Execute(context.Background(), callReq.Arguments)
	if err != nil {
		s.sendError(request.ID, -32603, fmt.Sprintf("Tool execution error: %v", err))
		return
	}

	response := types.CallToolResponse{
		Result: types.ToolResult{
			Content: result.Content,
			IsError: result.IsError,
		},
	}

	s.sendResponse(request.ID, response)
}

// getToolsForResponse returns the tools in the format expected by the MCP protocol
func (s *Server) getToolsForResponse() []types.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]types.Tool, len(s.tools))
	for i, tool := range s.tools {
		tools[i] = types.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		}
	}

	return tools
}

// sendResponse sends a response to stdout
func (s *Server) sendResponse(id *int, result interface{}) {
	response := types.Response{
		Result: result,
		ID:     id,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

// sendError sends an error response to stdout
func (s *Server) sendError(id *int, code int, message string) {
	response := types.Response{
		Error: &types.Error{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling error response: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

// sendNotification sends a notification to stdout
func (s *Server) sendNotification(method string, params interface{}) {
	notification := types.Notification{
		Method: method,
	}

	if params != nil {
		jsonData, err := json.Marshal(params)
		if err != nil {
			log.Printf("Error marshaling notification params: %v", err)
			return
		}
		notification.Params = json.RawMessage(jsonData)
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling notification: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}