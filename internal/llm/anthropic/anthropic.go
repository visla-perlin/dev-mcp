package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dev-mcp/internal/config"
	"dev-mcp/internal/llm/models"
)

// Client represents an Anthropic client
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// New creates a new Anthropic client
func New(cfg *config.ProviderConfig) *Client {
	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: "https://api.anthropic.com/v1",
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chat sends a chat request to the Anthropic API
func (c *Client) Chat(ctx context.Context, request *models.ChatRequest) (*models.ChatResponse, error) {
	// Use the model from the request or fallback to the default
	model := request.Model
	if model == "" {
		model = c.model
	}

	// Convert messages to Anthropic format
	// Anthropic uses a different format - we need to convert system messages and user/assistant messages
	var systemPrompt string
	var messages []anthropicMessage

	for _, msg := range request.Messages {
		if msg.Role == models.SystemRole {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	// Create Anthropic request
	anthropicReq := anthropicChatRequest{
		Model:         model,
		Messages:      messages,
		System:        systemPrompt,
		Temperature:   request.Temperature,
		MaxTokens:     request.MaxTokens,
		TopP:          request.TopP,
		Stream:        false,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var anthropicResp anthropicChatResponse
	err = json.Unmarshal(body, &anthropicResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to our response format
	choices := make([]models.Choice, 1)
	if len(anthropicResp.Content) > 0 {
		choices[0] = models.Choice{
			Index: 0,
			Message: models.Message{
				Role:    models.AssistantRole,
				Content: anthropicResp.Content[0].Text,
			},
			FinishReason: anthropicResp.StopReason,
		}
	}

	response := &models.ChatResponse{
		ID:      anthropicResp.ID,
		Model:   anthropicResp.Model,
		Choices: choices,
		Usage: models.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	return response, nil
}

// Complete is not directly supported by Anthropic, but we can simulate it using chat
func (c *Client) Complete(ctx context.Context, request *models.CompletionRequest) (*models.CompletionResponse, error) {
	// Convert completion request to chat request
	chatReq := &models.ChatRequest{
		Model:       request.Model,
		Messages:    []models.Message{{Role: models.UserRole, Content: request.Prompt}},
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
		TopP:        request.TopP,
	}

	chatResp, err := c.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	// Convert chat response to completion response
	choices := make([]models.CompletionChoice, len(chatResp.Choices))
	for i, choice := range chatResp.Choices {
		choices[i] = models.CompletionChoice{
			Index:        choice.Index,
			Text:         choice.Message.Content,
			FinishReason: choice.FinishReason,
		}
	}

	response := &models.CompletionResponse{
		ID:      chatResp.ID,
		Model:   chatResp.Model,
		Choices: choices,
		Usage:   chatResp.Usage,
	}

	return response, nil
}

// Embed is not supported by Anthropic
func (c *Client) Embed(ctx context.Context, request *models.EmbeddingRequest) (*models.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embedding is not supported by Anthropic")
}

// ListModels returns a list of available models
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	// Anthropic doesn't have a models endpoint, so we return a predefined list
	models := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	}
	return models, nil
}

// GetModelInfo returns information about a specific model
func (c *Client) GetModelInfo(ctx context.Context, model string) (*models.ModelInfo, error) {
	// Return basic info about the model
	info := &models.ModelInfo{
		ID:          model,
		Name:        model,
		Description: fmt.Sprintf("Anthropic %s model", model),
		Provider:    "anthropic",
		CreatedAt:   time.Now(),
		MaxTokens:   200000, // Most Claude models support up to 200K tokens
	}

	return info, nil
}

// Close closes the client
func (c *Client) Close() error {
	// Nothing to close for this client
	return nil
}

// Internal types for Anthropic API

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicChatRequest struct {
	Model         string              `json:"model"`
	Messages      []anthropicMessage  `json:"messages"`
	System        string              `json:"system,omitempty"`
	Temperature   float64             `json:"temperature,omitempty"`
	MaxTokens     int                 `json:"max_tokens,omitempty"`
	TopP          float64             `json:"top_p,omitempty"`
	Stream        bool                `json:"stream"`
}

type anthropicChatResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Model        string                 `json:"model"`
	Role         string                 `json:"role"`
	Content      []anthropicContent     `json:"content"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence interface{}            `json:"stop_sequence"`
	Usage        anthropicUsage         `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}