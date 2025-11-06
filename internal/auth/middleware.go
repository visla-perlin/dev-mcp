package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// Middleware provides HTTP authentication middleware
type Middleware struct {
	authenticator *SimpleAuthenticator
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(config *AuthConfig) *Middleware {
	return &Middleware{
		authenticator: NewSimpleAuthenticator(config),
	}
}

// AuthorizeRequest checks if the HTTP request is authorized
func (m *Middleware) AuthorizeRequest(r *http.Request) (*AuthResult, error) {
	// Skip authentication if disabled
	if !m.authenticator.IsEnabled() {
		return &AuthResult{
			UserID:   "anonymous",
			Username: "anonymous",
			Roles:    []string{"admin"}, // Full access when auth is disabled
			Method:   "disabled",
		}, nil
	}

	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	// Check for Bearer token
	if strings.HasPrefix(authHeader, "Bearer ") {
		return m.authenticator.AuthenticateBearer(authHeader)
	}

	return nil, fmt.Errorf("unsupported authorization method")
}

// CheckToolPermission checks if the authenticated user can access a specific tool
func (m *Middleware) CheckToolPermission(authResult *AuthResult, toolName string) error {
	if !m.authenticator.HasPermission(authResult, toolName) {
		return fmt.Errorf("insufficient permissions for tool: %s", toolName)
	}
	return nil
}

// IsEnabled returns whether authentication is enabled
func (m *Middleware) IsEnabled() bool {
	return m.authenticator.IsEnabled()
}

// HTTPMiddleware returns an HTTP middleware function
func (m *Middleware) HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Perform authentication
		authResult, err := m.AuthorizeRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
			return
		}

		// Add auth result to request context
		if authResult != nil {
			// Store auth result in request context for later use
			r = r.WithContext(WithAuthResult(r.Context(), authResult))
		}

		// Continue to next handler
		next(w, r)
	}
}

// GetRolesList returns a list of available roles
func (m *Middleware) GetRolesList() []string {
	return []string{"admin", "read", "write", "monitor"}
}

// GetToolsList returns a list of available tools and their required permissions
func (m *Middleware) GetToolsList() map[string][]string {
	return map[string][]string{
		"database_query": {"read", "write", "admin"},
		"loki_query":     {"read", "write", "admin", "monitor"},
		"s3_query":       {"read", "write", "admin"},
		"sentry_query":   {"monitor", "admin"},
		"swagger_query":  {"read", "write", "admin"},
		"llm_chat":       {"write", "admin"},
		"http_request":   {"write", "admin"},
	}
}
