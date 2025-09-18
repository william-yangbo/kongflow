package apiauth

import (
	"context"
	"encoding/json"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys for authentication results
const (
	authResultKey        contextKey = "auth_result"
	unifiedAuthResultKey contextKey = "unified_auth_result"
)

// AuthMiddleware provides HTTP middleware for API authentication
type AuthMiddleware struct {
	service APIAuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(service APIAuthService) *AuthMiddleware {
	return &AuthMiddleware{
		service: service,
	}
}

// RequireAPIKey middleware that requires valid API key authentication
func (m *AuthMiddleware) RequireAPIKey(opts *AuthOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := m.service.AuthenticateAPIRequest(r.Context(), r, opts)
			if err != nil {
				http.Error(w, "Authentication error", http.StatusInternalServerError)
				return
			}

			if !result.Success {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": result.Error,
				})
				return
			}

			// Add authentication result to request context
			ctx := context.WithValue(r.Context(), authResultKey, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth middleware that supports multiple authentication methods
func (m *AuthMiddleware) RequireAuth(config *AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := m.service.AuthenticateRequest(r.Context(), r, config)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}

			// Add authentication result to request context
			ctx := context.WithValue(r.Context(), unifiedAuthResultKey, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthResult extracts authentication result from request context
func GetAuthResult(r *http.Request) (*AuthenticationResult, bool) {
	result, ok := r.Context().Value(authResultKey).(*AuthenticationResult)
	return result, ok
}

// GetUnifiedAuthResult extracts unified authentication result from request context
func GetUnifiedAuthResult(r *http.Request) (*UnifiedAuthResult, bool) {
	result, ok := r.Context().Value(unifiedAuthResultKey).(*UnifiedAuthResult)
	return result, ok
}
