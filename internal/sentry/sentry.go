package sentry

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"dev-mcp/internal/config"
)

// Client represents a Sentry client
type Client struct {
	client *sentry.Client
	config *config.SentryConfig
}

// Issue represents a Sentry issue
type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	Environment string    `json:"environment"`
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	Count       int       `json:"count"`
	UserCount   int       `json:"userCount"`
}

// New creates a new Sentry client
func New(cfg *config.SentryConfig) (*Client, error) {
	// Initialize Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.DSN,
		Environment: cfg.Environment,
		Release:     cfg.Release,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	// Create client
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         cfg.DSN,
		Environment: cfg.Environment,
		Release:     cfg.Release,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Sentry client: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// CaptureException captures an exception in Sentry
func (c *Client) CaptureException(err error, tags map[string]string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Capture exception
		sentry.CaptureException(err)
	})
}

// CaptureMessage captures a message in Sentry
func (c *Client) CaptureMessage(message string, level sentry.Level, tags map[string]string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		// Set level
		scope.SetLevel(level)

		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Capture message
		sentry.CaptureMessage(message)
	})
}

// Flush waits for all events to be delivered
func (c *Client) Flush() {
	sentry.Flush(2 * time.Second)
}

// GetIssues retrieves issues from Sentry (simplified implementation)
func (c *Client) GetIssues() ([]Issue, error) {
	// In a real implementation, this would make API calls to Sentry
	// For now, we'll return an empty slice as a placeholder
	return []Issue{}, nil
}

// GetIssueByID retrieves a specific issue by ID (simplified implementation)
func (c *Client) GetIssueByID(id string) (*Issue, error) {
	// In a real implementation, this would make API calls to Sentry
	// For now, we'll return nil as a placeholder
	return nil, nil
}

// Close closes the Sentry client
func (c *Client) Close() {
	c.client.Flush(time.Second * 2)
}