package types

import "encoding/json"

// Request represents an MCP request
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     *int            `json:"id,omitempty"`
}

// Response represents an MCP response
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  *Error     `json:"error,omitempty"`
	ID     *int       `json:"id,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents an MCP notification
type Notification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError,omitempty"`
}

// InitializeRequest represents the initialize request
type InitializeRequest struct {
	ProtocolVersion string   `json:"protocolVersion"`
	Capabilities    []string `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResponse represents the initialize response
type InitializeResponse struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    []string      `json:"capabilities"`
	ServerInfo      ServerInfo    `json:"serverInfo"`
	Tools           []Tool        `json:"tools"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ListToolsResponse represents the listTools response
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest represents the callTool request
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// CallToolResponse represents the callTool response
type CallToolResponse struct {
	Result ToolResult `json:"result"`
}