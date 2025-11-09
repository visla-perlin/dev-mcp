package loki

// This file defines a collection of commonly used (mock) Loki LogQL queries.
// They are exposed via preset tools so that a caller can quickly run standard
// queries without remembering exact LogQL syntax. Since the current Loki
// client implementation returns mock data, these queries only construct the
// query strings; real backend integration can later plug into the same API.

import (
	"fmt"
	"sort"
	"strings"
)

// ParamMeta describes a parameter used inside a preset template.
type ParamMeta struct {
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PresetQuery describes a reusable Loki query template.
type PresetQuery struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Template    string               `json:"template"`
	Params      map[string]ParamMeta `json:"params,omitempty"`
	Example     string               `json:"example,omitempty"`
}

// PresetQueries holds all available presets keyed by name.
var PresetQueries = map[string]PresetQuery{
	"error_logs": {
		Name:        "error_logs",
		Description: "Raw error level log lines over a time window (client decides the range).",
		Template:    `{level="error"}`,
		Params:      map[string]ParamMeta{},
		Example:     `{level="error"}`,
	},
	"error_rate": {
		Name:        "error_rate",
		Description: "Per-second rate of error logs over a sliding window (LogQL rate()).",
		Template:    `sum(rate({level="error"}[${window}]))`,
		Params: map[string]ParamMeta{
			"window": {Description: "Range / window duration, e.g. 5m, 1h", Default: "5m"},
		},
		Example: `sum(rate({level="error"}[5m]))`,
	},
	"warn_vs_error_ratio": {
		Name:        "warn_vs_error_ratio",
		Description: "Ratio of error log rate to warning log rate for the given window.",
		Template:    `sum(rate({level="error"}[${window}])) / ignoring(level) sum(rate({level="warn"}[${window}]))`,
		Params: map[string]ParamMeta{
			"window": {Description: "Range / window duration", Default: "5m"},
		},
		Example: `sum(rate({level="error"}[5m])) / ignoring(level) sum(rate({level="warn"}[5m]))`,
	},
	"count_by_level": {
		Name:        "count_by_level",
		Description: "Count of log lines grouped by level in a window.",
		Template:    `sum by (level) (count_over_time({level=~".+"}[${window}]))`,
		Params: map[string]ParamMeta{
			"window": {Description: "Range / window duration", Default: "5m"},
		},
		Example: `sum by (level) (count_over_time({level=~".+"}[5m]))`,
	},
	"top_services_errors": {
		Name:        "top_services_errors",
		Description: "Top-K services (label app) by error log rate.",
		Template:    `topk(${k}, sum by (app) (rate({level="error"}[${window}])))`,
		Params: map[string]ParamMeta{
			"k":      {Description: "Number of services to return", Default: "5"},
			"window": {Description: "Range / window duration", Default: "15m"},
		},
		Example: `topk(5, sum by (app) (rate({level="error"}[15m])))`,
	},
	"p95_latency": {
		Name:        "p95_latency",
		Description: "Estimate p95 latency from logs with parsed JSON field 'latency_seconds' (example template).",
		Template:    `histogram_quantile(0.95, sum by (le) (rate({app="${app}"} | json | unwrap latency_seconds | histogram_over_time(${bucket} [${window}]))) )`,
		Params: map[string]ParamMeta{
			"app":    {Description: "Application / service name", Required: true},
			"window": {Description: "Range / window duration", Default: "5m"},
			"bucket": {Description: "Bucket width, e.g. 1m", Default: "1m"},
		},
		Example: `histogram_quantile(0.95, sum by (le) (rate({app="payments"} | json | unwrap latency_seconds | histogram_over_time(1m [5m]))) )`,
	},
}

// ListPresetMetadata returns a slice of presets sorted by name for display.
func ListPresetMetadata() []PresetQuery {
	keys := make([]string, 0, len(PresetQueries))
	for k := range PresetQueries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]PresetQuery, 0, len(keys))
	for _, k := range keys {
		out = append(out, PresetQueries[k])
	}
	return out
}

// BuildPresetQuery builds the final LogQL query string for a preset using provided params.
func BuildPresetQuery(name string, params map[string]string) (string, error) {
	preset, ok := PresetQueries[name]
	if !ok {
		return "", fmt.Errorf("unknown preset: %s", name)
	}

	// Start with template
	query := preset.Template

	// Fill parameters: use provided, else default (if any), else error if required.
	for pname, meta := range preset.Params {
		val, provided := params[pname]
		if !provided || val == "" {
			if meta.Default != "" {
				val = meta.Default
			} else if meta.Required {
				return "", fmt.Errorf("missing required parameter '%s' for preset '%s'", pname, name)
			}
		}
		placeholder := "${" + pname + "}"
		query = strings.ReplaceAll(query, placeholder, val)
	}

	// If any unreplaced placeholders remain, surface an error (helps catch typos)
	if strings.Contains(query, "${") {
		return "", fmt.Errorf("unresolved placeholders remain in query: %s", query)
	}

	return query, nil
}
