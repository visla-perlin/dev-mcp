package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/auth"
	"dev-mcp/internal/logging"
)

// AuthenticatedSSETransport wraps SSE transport with authentication
type AuthenticatedSSETransport struct {
	authMiddleware *auth.Middleware
	port           int
}

// NewAuthenticatedSSETransport creates a new authenticated SSE transport
func NewAuthenticatedSSETransport(authConfig *auth.AuthConfig, port int) *AuthenticatedSSETransport {
	return &AuthenticatedSSETransport{
		authMiddleware: auth.NewMiddleware(authConfig),
		port:           port,
	}
}

// Start starts the authenticated SSE server
func (t *AuthenticatedSSETransport) Start(ctx context.Context, server *mcp.Server) error {
	logger := logging.New("SSE")
	logger.Info("starting authenticated SSE transport", logging.String("port", fmt.Sprintf("%d", t.port)))

	// Create HTTP server with authentication
	mux := http.NewServeMux()

	// Add the MCP endpoint with authentication
	mux.HandleFunc("/mcp", t.authMiddleware.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Get auth result from context
		authResult, ok := auth.GetAuthResult(r.Context())
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		logger.Info("authenticated request",
			logging.String("user", authResult.Username),
			logging.String("roles", strings.Join(authResult.Roles, ",")))

		// Handle SSE connection here
		// For now, we'll create a basic SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")

		// Send initial connection message
		fmt.Fprintf(w, "data: {\"type\":\"connection\",\"status\":\"connected\",\"user\":\"%s\"}\n\n", authResult.Username)

		// Flush to ensure the data is sent immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Keep connection alive
		select {
		case <-ctx.Done():
			return
		case <-r.Context().Done():
			return
		}
	}))

	// Add health check endpoint (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","auth_enabled":%t}`, t.authMiddleware.IsEnabled())
	})

	// Add authentication info endpoint
	mux.HandleFunc("/auth/info", t.authMiddleware.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authResult, _ := auth.GetAuthResult(r.Context())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"user":"%s","roles":["%s"],"method":"%s"}`,
			authResult.Username,
			strings.Join(authResult.Roles, `","`),
			authResult.Method)
	}))

	// Create and start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", t.port),
		Handler: mux,
	}

	logger.Info("SSE server started", logging.String("address", httpServer.Addr))

	// Start server in background
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("SSE server error", logging.String("error", err.Error()))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	logger.Info("shutting down SSE server")
	return httpServer.Shutdown(context.Background())
}

// CheckToolAccess validates if the current user can access a specific tool
func (t *AuthenticatedSSETransport) CheckToolAccess(ctx context.Context, toolName string) error {
	authResult, ok := auth.GetAuthResult(ctx)
	if !ok {
		return fmt.Errorf("no authentication context")
	}

	return t.authMiddleware.CheckToolPermission(authResult, toolName)
}
