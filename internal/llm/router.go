package llm

import (
	"context"
	"fmt"
	"sync"

	"dev-mcp/internal/config"
	"dev-mcp/internal/llm/anthropic"
	"dev-mcp/internal/llm/models"
	"dev-mcp/internal/llm/openai"
)

// Router manages multiple LLM providers and routes requests to them
type Router struct {
	providers map[models.ModelProvider]models.ModelService
	mu        sync.RWMutex
	config    *config.LLMConfig
}

// NewRouter creates a new LLM router
func NewRouter(cfg *config.LLMConfig) (*Router, error) {
	router := &Router{
		providers: make(map[models.ModelProvider]models.ModelService),
		config:    cfg,
	}

	// Initialize providers based on configuration
	err := router.initializeProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return router, nil
}

// initializeProviders initializes all enabled providers
func (r *Router) initializeProviders() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, providerConfig := range r.config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		var provider models.ModelService
		var providerType models.ModelProvider

		switch providerConfig.Type {
		case "openai":
			provider = openai.New(&providerConfig)
			providerType = models.OpenAIProvider
		case "anthropic":
			provider = anthropic.New(&providerConfig)
			providerType = models.AnthropicProvider
		case "local":
			// For local models, we can use the OpenAI compatible API
			providerConfig.APIKey = "dummy" // Local models don't need API keys
			provider = openai.New(&providerConfig)
			providerType = models.LocalProvider
		default:
			// Skip unknown provider types
			continue
		}

		r.providers[providerType] = provider
	}

	return nil
}

// Chat sends a chat request to an appropriate provider
func (r *Router) Chat(ctx context.Context, request *models.ChatRequest) (*models.ChatResponse, error) {
	// For now, we'll use a simple routing strategy
	// In a more advanced implementation, we could implement load balancing, failover, etc.

	// Try to determine the provider from the model name
	provider := r.determineProvider(request.Model)

	r.mu.RLock()
	service, exists := r.providers[provider]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no provider available for model: %s", request.Model)
	}

	return service.Chat(ctx, request)
}

// Complete sends a completion request to an appropriate provider
func (r *Router) Complete(ctx context.Context, request *models.CompletionRequest) (*models.CompletionResponse, error) {
	// Try to determine the provider from the model name
	provider := r.determineProvider(request.Model)

	r.mu.RLock()
	service, exists := r.providers[provider]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no provider available for model: %s", request.Model)
	}

	return service.Complete(ctx, request)
}

// Embed generates embeddings using an appropriate provider
func (r *Router) Embed(ctx context.Context, request *models.EmbeddingRequest) (*models.EmbeddingResponse, error) {
	// Try to determine the provider from the model name
	provider := r.determineProvider(request.Model)

	r.mu.RLock()
	service, exists := r.providers[provider]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no provider available for model: %s", request.Model)
	}

	// Check if the provider supports embeddings
	resp, err := service.Embed(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("provider does not support embeddings: %w", err)
	}

	return resp, nil
}

// ListModels returns a list of available models from all providers
func (r *Router) ListModels(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allModels []string
	for _, service := range r.providers {
		models, err := service.ListModels(ctx)
		if err != nil {
			// Continue with other providers even if one fails
			continue
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// GetModelInfo returns information about a specific model
func (r *Router) GetModelInfo(ctx context.Context, model string) (*models.ModelInfo, error) {
	// Try each provider to find the model
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, service := range r.providers {
		info, err := service.GetModelInfo(ctx, model)
		if err == nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", model)
}

// determineProvider determines the provider based on the model name
func (r *Router) determineProvider(model string) models.ModelProvider {
	if model == "" {
		// Return the first available provider
		r.mu.RLock()
		defer r.mu.RUnlock()

		for provider := range r.providers {
			return provider
		}
		return models.OpenAIProvider // Default fallback
	}

	// Simple heuristic to determine provider from model name
	switch {
	case contains(model, "gpt"):
		return models.OpenAIProvider
	case contains(model, "claude"):
		return models.AnthropicProvider
	case contains(model, "llama"):
		return models.LocalProvider
	case contains(model, "mistral"):
		return models.LocalProvider
	default:
		// Return the first available provider
		r.mu.RLock()
		defer r.mu.RUnlock()

		for provider := range r.providers {
			return provider
		}
		return models.OpenAIProvider // Default fallback
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) < len(s) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

// indexOf returns the index of the first instance of substr in s, or -1 if substr is not present in s.
func indexOf(s, substr string) int {
	n := len(substr)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return indexByte(s, substr[0])
	case n == len(s):
		if substr == s {
			return 0
		}
		return -1
	case n > len(s):
		return -1
	}

	// Rabin-Karp search
	hashss, pow := hashStr(substr)
	var h uint32
	for i := 0; i < n; i++ {
		h = h*primeRK + uint32(s[i])
	}
	if h == hashss && s[:n] == substr {
		return 0
	}
	for i := n; i < len(s); {
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		if h == hashss && s[i-n:i] == substr {
			return i - n
		}
	}
	return -1
}

// indexByte returns the index of the first instance of c in s, or -1 if c is not present in s.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

const primeRK = 16777619

// hashStr returns the hash and the appropriate multiplicative factor for use in Rabin-Karp algorithm.
func hashStr(sep string) (uint32, uint32) {
	hash := uint32(0)
	for i := 0; i < len(sep); i++ {
		hash = hash*primeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// Close closes all providers
func (r *Router) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for _, service := range r.providers {
		if err := service.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// GetProviders returns the list of available providers
func (r *Router) GetProviders() []models.ModelProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]models.ModelProvider, 0, len(r.providers))
	for provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}