package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

// AuthConfig represents the authentication configuration
type AuthConfig struct {
	Enabled bool     `yaml:"enabled"`
	APIKeys []APIKey `yaml:"api_keys"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	Name    string   `yaml:"name"`
	Key     string   `yaml:"key"`
	Roles   []string `yaml:"roles"`
	Enabled bool     `yaml:"enabled"`
}

// AuthResult represents authentication result
type AuthResult struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Method   string   `json:"method"`
}

// SimpleAuthenticator implements simple API key authentication
type SimpleAuthenticator struct {
	config *AuthConfig
}

// NewSimpleAuthenticator creates a new simple authenticator
func NewSimpleAuthenticator(config *AuthConfig) *SimpleAuthenticator {
	return &SimpleAuthenticator{
		config: config,
	}
}

// AuthenticateBearer validates a Bearer token (API key)
func (a *SimpleAuthenticator) AuthenticateBearer(token string) (*AuthResult, error) {
	if !a.config.Enabled {
		return nil, fmt.Errorf("authentication is disabled")
	}

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	// Find matching API key
	for _, apiKey := range a.config.APIKeys {
		if !apiKey.Enabled {
			continue
		}

		// Use constant-time comparison to prevent timing attacks
		if secureCompare(apiKey.Key, token) {
			return &AuthResult{
				UserID:   apiKey.Name,
				Username: apiKey.Name,
				Roles:    apiKey.Roles,
				Method:   "api_key",
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// HasPermission checks if the user has permission for a specific tool
func (a *SimpleAuthenticator) HasPermission(authResult *AuthResult, toolName string) bool {
	if authResult == nil {
		return false
	}

	// Define tool permissions
	toolPermissions := map[string][]string{
		"database_query": {"read", "write", "admin"},
		"loki_query":     {"read", "write", "admin", "monitor"},
		"s3_query":       {"read", "write", "admin"},
		"sentry_query":   {"monitor", "admin"},
		"swagger_query":  {"read", "write", "admin"},
		"llm_chat":       {"write", "admin"},
		"http_request":   {"write", "admin"},
	}

	requiredRoles, exists := toolPermissions[toolName]
	if !exists {
		// If tool is not defined, deny access
		return false
	}

	// Check if user has any of the required roles
	for _, userRole := range authResult.Roles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}

// IsEnabled returns whether authentication is enabled
func (a *SimpleAuthenticator) IsEnabled() bool {
	return a.config.Enabled
}

// GenerateAPIKey generates a secure random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return "mcp_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// secureCompare performs constant-time comparison of two strings
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// HasRole checks if the user has the specified role
func (ar *AuthResult) HasRole(role string) bool {
	for _, r := range ar.Roles {
		if r == role {
			return true
		}
	}
	return false
}
