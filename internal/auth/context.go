package auth

import (
	"context"
)

// contextKey is used for context values
type contextKey string

const authResultKey contextKey = "auth_result"

// WithAuthResult adds auth result to context
func WithAuthResult(ctx context.Context, authResult *AuthResult) context.Context {
	return context.WithValue(ctx, authResultKey, authResult)
}

// GetAuthResult retrieves auth result from context
func GetAuthResult(ctx context.Context) (*AuthResult, bool) {
	authResult, ok := ctx.Value(authResultKey).(*AuthResult)
	return authResult, ok
}
