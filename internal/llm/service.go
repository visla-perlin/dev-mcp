package llm

import (
	"context"
	"fmt"

	"dev-mcp/internal/config"
	"dev-mcp/internal/llm/models"
)

// Service represents the LLM service
type Service struct {
	router *Router
}

// NewService creates a new LLM service
func NewService(cfg *config.Config) (*Service, error) {
	// Create router
	router, err := NewRouter(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	return &Service{
		router: router,
	}, nil
}

// Chat sends a chat request to an appropriate model
func (s *Service) Chat(ctx context.Context, request *models.ChatRequest) (*models.ChatResponse, error) {
	return s.router.Chat(ctx, request)
}

// Complete sends a completion request to an appropriate model
func (s *Service) Complete(ctx context.Context, request *models.CompletionRequest) (*models.CompletionResponse, error) {
	return s.router.Complete(ctx, request)
}

// Embed generates embeddings for the given texts
func (s *Service) Embed(ctx context.Context, request *models.EmbeddingRequest) (*models.EmbeddingResponse, error) {
	return s.router.Embed(ctx, request)
}

// ListModels returns a list of available models
func (s *Service) ListModels(ctx context.Context) ([]string, error) {
	return s.router.ListModels(ctx)
}

// GetModelInfo returns information about a specific model
func (s *Service) GetModelInfo(ctx context.Context, model string) (*models.ModelInfo, error) {
	return s.router.GetModelInfo(ctx, model)
}

// GetProviders returns the list of available providers
func (s *Service) GetProviders() []models.ModelProvider {
	return s.router.GetProviders()
}

// Close closes the service and releases any resources
func (s *Service) Close() error {
	return s.router.Close()
}

// HealthCheck performs a health check on the LLM service
func (s *Service) HealthCheck(ctx context.Context) error {
	// Try to list models from each provider as a basic health check
	providers := s.GetProviders()

	if len(providers) == 0 {
		return fmt.Errorf("no LLM providers configured")
	}

	// Try to list models from each provider
	for _, provider := range providers {
		// We don't actually need the list, just checking if the provider is responsive
		_, err := s.ListModels(ctx)
		if err != nil {
			// Log the error but continue checking other providers
			fmt.Printf("Warning: Provider %s health check failed: %v\n", provider, err)
		}
	}

	return nil
}