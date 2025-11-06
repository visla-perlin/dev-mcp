package loki

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"dev-mcp/internal/config"
)

// Client represents a Grafana Loki client
type Client struct {
	client *resty.Client
	config *config.LokiConfig
}

// New creates a new Loki client
func New(cfg *config.LokiConfig) *Client {
	client := resty.New().
		SetBaseURL(cfg.Host).
		SetTimeout(30 * time.Second)

	if cfg.Username != "" && cfg.Password != "" {
		client.SetBasicAuth(cfg.Username, cfg.Password)
	}

	return &Client{
		client: client,
		config: cfg,
	}
}

// QueryResponse represents the response from a Loki query
type QueryResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

// Data represents the data part of the Loki query response
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

// Result represents a single result from a Loki query
type Result struct {
	Stream map[string]string `json:"stream"`
	Values [][]interface{}   `json:"values"`
}

// Query executes a LogQL query against Grafana Loki
func (c *Client) Query(query string, limit int, start, end time.Time) (*QueryResponse, error) {
	resp, err := c.client.R().
		SetQueryParam("query", query).
		SetQueryParam("limit", fmt.Sprintf("%d", limit)).
		SetQueryParam("start", fmt.Sprintf("%d", start.UnixNano())).
		SetQueryParam("end", fmt.Sprintf("%d", end.UnixNano())).
		SetResult(&QueryResponse{}).
		Get("/loki/api/v1/query_range")

	if err != nil {
		return nil, fmt.Errorf("failed to execute Loki query: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Loki query failed with status code: %d", resp.StatusCode())
	}

	return resp.Result().(*QueryResponse), nil
}

// GetLogLabels retrieves the list of labels from Loki
func (c *Client) GetLogLabels() ([]string, error) {
	resp, err := c.client.R().
		SetResult(map[string]interface{}{}).
		Get("/loki/api/v1/labels")

	if err != nil {
		return nil, fmt.Errorf("failed to get labels from Loki: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Loki labels request failed with status code: %d", resp.StatusCode())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels response: %w", err)
	}

	// Extract labels from the response
	labels, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format for labels")
	}

	labelStrings := make([]string, len(labels))
	for i, label := range labels {
		if labelStr, ok := label.(string); ok {
			labelStrings[i] = labelStr
		}
	}

	return labelStrings, nil
}