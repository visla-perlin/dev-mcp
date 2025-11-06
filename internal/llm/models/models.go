package models

import (
	"context"
	"time"
)

// ModelProvider represents a provider of large language models
type ModelProvider string

const (
	OpenAIProvider     ModelProvider = "openai"
	AnthropicProvider  ModelProvider = "anthropic"
	HuggingFaceProvider ModelProvider = "huggingface"
	LocalProvider      ModelProvider = "local"
)

// MessageRole represents the role of a message in a conversation
type MessageRole string

const (
	SystemRole    MessageRole = "system"
	UserRole      MessageRole = "user"
	AssistantRole MessageRole = "assistant"
)

// Message represents a single message in a conversation
type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// ChatRequest represents a request to chat with a model
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
}

// ChatResponse represents a response from a chat model
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a single choice in a chat response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents the token usage in a response
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionRequest represents a request for text completion
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// CompletionResponse represents a response from a completion model
type CompletionResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// CompletionChoice represents a single choice in a completion response
type CompletionChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

// EmbeddingRequest represents a request for embeddings
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents a response containing embeddings
type EmbeddingResponse struct {
	Model  string     `json:"model"`
	Data   []Embedding `json:"data"`
	Usage  Usage      `json:"usage"`
}

// Embedding represents a single embedding
type Embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ModelService defines the interface for large language model services
type ModelService interface {
	// Chat sends a chat request to the model
	Chat(ctx context.Context, request *ChatRequest) (*ChatResponse, error)

	// Complete sends a completion request to the model
	Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error)

	// Embed generates embeddings for the given texts
	Embed(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error)

	// ListModels returns a list of available models
	ListModels(ctx context.Context) ([]string, error)

	// GetModelInfo returns information about a specific model
	GetModelInfo(ctx context.Context, model string) (*ModelInfo, error)

	// Close closes the service and releases any resources
	Close() error
}

// ModelInfo contains information about a model
type ModelInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
	MaxTokens   int       `json:"max_tokens"`
}