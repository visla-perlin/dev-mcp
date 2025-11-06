package openai

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

// Client represents an OpenAI client
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// New creates a new OpenAI client
func New(cfg *config.ProviderConfig) *Client {
	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: "https://api.openai.com/v1",
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chat sends a chat request to the OpenAI API
func (c *Client) Chat(ctx context.Context, request *models.ChatRequest) (*models.ChatResponse, error) {
	// Use the model from the request or fallback to the default
	model := request.Model
	if model == "" {
		model = c.model
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openaiMessage, len(request.Messages))
	for i, msg := range request.Messages {
		openaiMessages[i] = openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Create OpenAI request
	openaiReq := openaiChatRequest{
		Model:       model,
		Messages:    openaiMessages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
		TopP:        request.TopP,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openaiResp openaiChatResponse
	err = json.Unmarshal(body, &openaiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to our response format
	choices := make([]models.Choice, len(openaiResp.Choices))
	for i, choice := range openaiResp.Choices {
		choices[i] = models.Choice{
			Index: choice.Index,
			Message: models.Message{
				Role:    models.MessageRole(choice.Message.Role),
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		}
	}

	response := &models.ChatResponse{
		ID:      openaiResp.ID,
		Model:   openaiResp.Model,
		Choices: choices,
		Usage: models.Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// Complete sends a completion request to the OpenAI API
func (c *Client) Complete(ctx context.Context, request *models.CompletionRequest) (*models.CompletionResponse, error) {
	// Use the model from the request or fallback to the default
	model := request.Model
	if model == "" {
		model = c.model
	}

	// Create OpenAI request
	openaiReq := openaiCompletionRequest{
		Model:       model,
		Prompt:      request.Prompt,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
		TopP:        request.TopP,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openaiResp openaiCompletionResponse
	err = json.Unmarshal(body, &openaiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to our response format
	choices := make([]models.CompletionChoice, len(openaiResp.Choices))
	for i, choice := range openaiResp.Choices {
		choices[i] = models.CompletionChoice{
			Index:        choice.Index,
			Text:         choice.Text,
			FinishReason: choice.FinishReason,
		}
	}

	response := &models.CompletionResponse{
		ID:      openaiResp.ID,
		Model:   openaiResp.Model,
		Choices: choices,
		Usage: models.Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// Embed generates embeddings using the OpenAI API
func (c *Client) Embed(ctx context.Context, request *models.EmbeddingRequest) (*models.EmbeddingResponse, error) {
	// Use the model from the request or fallback to the default
	model := request.Model
	if model == "" {
		model = "text-embedding-ada-002" // Default embedding model
	}

	// Create OpenAI request
	openaiReq := openaiEmbeddingRequest{
		Model: model,
		Input: request.Input,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openaiResp openaiEmbeddingResponse
	err = json.Unmarshal(body, &openaiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to our response format
	embeddings := make([]models.Embedding, len(openaiResp.Data))
	for i, emb := range openaiResp.Data {
		embeddings[i] = models.Embedding{
			Object:    emb.Object,
			Index:     emb.Index,
			Embedding: emb.Embedding,
		}
	}

	response := &models.EmbeddingResponse{
		Model: openaiResp.Model,
		Data:  embeddings,
		Usage: models.Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// ListModels returns a list of available models
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openaiResp openaiModelsResponse
	err = json.Unmarshal(body, &openaiResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract model IDs
	models := make([]string, len(openaiResp.Data))
	for i, model := range openaiResp.Data {
		models[i] = model.ID
	}

	return models, nil
}

// GetModelInfo returns information about a specific model
func (c *Client) GetModelInfo(ctx context.Context, model string) (*models.ModelInfo, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models/"+model, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return nil, fmt.Errorf("OpenAI API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var openaiModel openaiModel
	err = json.Unmarshal(body, &openaiModel)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to our response format
	info := &models.ModelInfo{
		ID:          openaiModel.ID,
		Name:        openaiModel.ID,
		Description: "",
		Provider:    "openai",
		CreatedAt:   time.Unix(openaiModel.Created, 0),
		MaxTokens:   4096, // Default value, would need to look up actual model info
	}

	return info, nil
}

// Close closes the client
func (c *Client) Close() error {
	// Nothing to close for this client
	return nil
}

// Internal types for OpenAI API

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
}

type openaiChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openaiChoice     `json:"choices"`
	Usage   openaiUsage        `json:"usage"`
}

type openaiChoice struct {
	Index        int          `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiCompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

type openaiCompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []openaiCompletionChoice `json:"choices"`
	Usage   openaiUsage              `json:"usage"`
}

type openaiCompletionChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type openaiEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openaiEmbeddingResponse struct {
	Object string           `json:"object"`
	Data   []openaiEmbedding `json:"data"`
	Model  string           `json:"model"`
	Usage  openaiUsage      `json:"usage"`
}

type openaiEmbedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type openaiModelsResponse struct {
	Object string       `json:"object"`
	Data   []openaiModel `json:"data"`
}

type openaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}