package sentry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"dev-mcp/internal/config"
)

// Issue represents a Sentry issue/group
type Issue struct {
	ID          string    `json:"id"`
	ShortID     string    `json:"shortId"`
	Title       string    `json:"title"`
	Culprit     string    `json:"culprit"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"project"`
	Count         string    `json:"count"`
	UserCount     int       `json:"userCount"`
	FirstSeen     time.Time `json:"firstSeen"`
	LastSeen      time.Time `json:"lastSeen"`
	Environment   *string   `json:"environment"`
}

// SentryClient provides enhanced Sentry operations
type SentryClient struct {
	client  *resty.Client
	config  *config.SentryConfig
	baseURL string
}

// IsAvailable checks if Sentry client is available
func (c *SentryClient) IsAvailable() bool {
	return c.client != nil
}

// NewSentryClient creates a new Sentry client wrapper from config
func NewSentryClient(cfg *config.SentryConfig) *SentryClient {
	if cfg == nil {
		return &SentryClient{
			client: nil,
		}
	}

	// Determine base URL
	baseURL := "https://sentry.io/api/0"
	if cfg.BaseURL != "" {
		baseURL = strings.TrimSuffix(cfg.BaseURL, "/") + "/api/0"
	}

	// Create resty client
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Authorization", "Bearer "+cfg.AuthToken).
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "dev-mcp/1.0")

	// Set timeout
	client.SetTimeout(30 * time.Second)

	return &SentryClient{
		client:  client,
		config:  cfg,
		baseURL: baseURL,
	}
}

// GetIssues retrieves Sentry issues with optional filtering
func (c *SentryClient) GetIssues(query string, limit int) (interface{}, error) {
	if c.client == nil || c.config == nil {
		return nil, fmt.Errorf("sentry client not initialized")
	}

	if limit == 0 {
		limit = 50
	}

	// Build URL for organization issues endpoint
	// Using organization endpoint instead of project endpoint for better flexibility
	url := fmt.Sprintf("/organizations/%s/issues/", c.config.Organization)

	// Prepare query parameters
	params := map[string]string{
		"query": query,
	}

	// Add project filter if configured
	if len(c.config.ProjectIDs) > 0 {
		// Join project IDs with comma
		params["project"] = strings.Join(c.config.ProjectIDs, ",")
	} else if c.config.Project != "" {
		// Use single project if no project IDs are specified
		params["project"] = c.config.Project
	}

	// Set limit
	params["limit"] = fmt.Sprintf("%d", limit)

	// Make API request
	resp, err := c.client.R().
		SetQueryParams(params).
		SetResult([]Issue{}).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch sentry issues: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("sentry API error: %s", resp.Status())
	}

	// Get the issues from response
	issues, ok := resp.Result().(*[]Issue)
	if !ok {
		return nil, fmt.Errorf("failed to parse sentry issues response")
	}

	// Convert to the expected format
	issuesData := make([]map[string]interface{}, len(*issues))
	for i, issue := range *issues {
		issuesData[i] = map[string]interface{}{
			"id":          issue.ID,
			"title":       issue.Title,
			"level":       issue.Level,
			"status":      issue.Status,
			"environment": issue.Environment,
			"firstSeen":   issue.FirstSeen.Format(time.RFC3339),
			"lastSeen":    issue.LastSeen.Format(time.RFC3339),
			"count":       issue.Count,
			"userCount":   issue.UserCount,
		}
	}

	result := map[string]interface{}{
		"issues": issuesData,
		"total":  len(issuesData),
		"query":  query,
		"limit":  limit,
	}

	return result, nil
}

// GetIssueDetails retrieves detailed information about a specific issue
func (c *SentryClient) GetIssueDetails(issueID string) (interface{}, error) {
	if c.client == nil || c.config == nil {
		return nil, fmt.Errorf("sentry client not initialized")
	}

	if issueID == "" {
		return nil, fmt.Errorf("issue ID is required")
	}

	// Build URL for getting issue details
	url := fmt.Sprintf("/issues/%s/", issueID)

	// Make API request
	resp, err := c.client.R().
		SetResult(Issue{}).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch sentry issue details: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 404 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		return nil, fmt.Errorf("sentry API error: %s", resp.Status())
	}

	// Get the issue from response
	issue, ok := resp.Result().(*Issue)
	if !ok {
		return nil, fmt.Errorf("failed to parse sentry issue response")
	}

	// Convert to the expected format
	result := map[string]interface{}{
		"id":          issue.ID,
		"title":       issue.Title,
		"level":       issue.Level,
		"status":      issue.Status,
		"environment": issue.Environment,
		"firstSeen":   issue.FirstSeen.Format(time.RFC3339),
		"lastSeen":    issue.LastSeen.Format(time.RFC3339),
		"count":       issue.Count,
		"userCount":   issue.UserCount,
	}

	return result, nil
}

// Close closes the Sentry client
func (c *SentryClient) Close() error {
	// Sentry client doesn't need explicit closing
	return nil
}

// FetchIssues fetches issues from Sentry with time-based filtering
func (c *SentryClient) FetchIssues(ctx context.Context, query string, minutesBack int) ([]Issue, error) {
	if c.client == nil || c.config == nil {
		return nil, fmt.Errorf("sentry client not initialized")
	}

	// Build URL for organization issues endpoint
	url := fmt.Sprintf("/organizations/%s/issues/", c.config.Organization)

	// Prepare query parameters
	params := map[string]string{}

	// Build the query string
	finalQuery := query

	// Add time-based filtering to the query
	if minutesBack > 0 {
		// Calculate start time
		startTime := time.Now().Add(-time.Duration(minutesBack) * time.Minute)
		timeFilter := fmt.Sprintf("lastSeen:>=%s", startTime.Format("2006-01-02T15:04:05"))

		if finalQuery != "" {
			finalQuery = fmt.Sprintf("%s %s", finalQuery, timeFilter)
		} else {
			finalQuery = timeFilter
		}
	}

	// Only add query parameter if it's not empty
	if finalQuery != "" {
		params["query"] = finalQuery
	}

	// Add project filter if configured
	if len(c.config.ProjectIDs) > 0 {
		params["project"] = strings.Join(c.config.ProjectIDs, ",")
	} else if c.config.Project != "" {
		params["project"] = c.config.Project
	}

	// Set reasonable limit
	params["limit"] = "100"

	// Make API request
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult([]Issue{}).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch sentry issues: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("sentry API error: %s (status: %d)", resp.Status(), resp.StatusCode())
	}

	// Get the issues from response
	issues, ok := resp.Result().(*[]Issue)
	if !ok {
		return nil, fmt.Errorf("failed to parse sentry issues response")
	}

	return *issues, nil
}

// GetQueryByName gets a predefined query by name from configuration
func (c *SentryClient) GetQueryByName(name string) (string, bool) {
	if c.config == nil || c.config.IssueQueries == nil {
		return "", false
	}

	query, exists := c.config.IssueQueries[name]
	return query, exists
}

// ListQueries lists all configured named queries
func (c *SentryClient) ListQueries() map[string]string {
	if c.config == nil || c.config.IssueQueries == nil {
		return map[string]string{}
	}

	// Return a copy of the queries map
	queries := make(map[string]string)
	for k, v := range c.config.IssueQueries {
		queries[k] = v
	}
	return queries
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
