package apiauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// validators.go implements API key validation logic aligned with trigger.dev

// getAPIKeyType determines the type of API key based on its prefix - trigger.dev alignment
func getAPIKeyType(apiKey string) APIKeyType {
	switch {
	case strings.HasPrefix(apiKey, "pk_"):
		return APIKeyTypePublic
	case strings.HasPrefix(apiKey, "tr_"):
		return APIKeyTypePrivate
	case isJWTToken(apiKey):
		return APIKeyTypePublicJWT
	default:
		return ""
	}
}

// isJWTToken checks if the string is a valid JWT format
func isJWTToken(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

// validateAPIKeyFormat validates API key format - trigger.dev alignment
func validateAPIKeyFormat(apiKey string) error {
	if len(apiKey) < 8 {
		return fmt.Errorf("API key too short")
	}

	keyType := getAPIKeyType(apiKey)
	if keyType == "" {
		return fmt.Errorf("invalid API key format")
	}

	return nil
}

// authenticatePublicKey validates public API key (pk_ prefix)
func (s *service) authenticatePublicKey(ctx context.Context, apiKey string, branchName string) (*AuthenticationResult, error) {
	// Validate format
	if err := validateAPIKeyFormat(apiKey); err != nil {
		return &AuthenticationResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Find environment (public keys can only access non-production)
	env, err := s.repo.FindEnvironmentByPublicAPIKey(ctx, apiKey, &branchName)
	if err != nil {
		return &AuthenticationResult{
			Success: false,
			Error:   "invalid public API key or environment not found",
		}, nil
	}

	return &AuthenticationResult{
		Success:     true,
		APIKey:      apiKey,
		Type:        APIKeyTypePublic,
		Environment: env,
	}, nil
}

// authenticatePrivateKey validates private API key (tr_ prefix)
func (s *service) authenticatePrivateKey(ctx context.Context, apiKey string, branchName string) (*AuthenticationResult, error) {
	// Validate format
	if err := validateAPIKeyFormat(apiKey); err != nil {
		return &AuthenticationResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Find environment (private keys can access any environment)
	env, err := s.repo.FindEnvironmentByAPIKey(ctx, apiKey)
	if err != nil {
		return &AuthenticationResult{
			Success: false,
			Error:   "invalid private API key or environment not found",
		}, nil
	}

	return &AuthenticationResult{
		Success:     true,
		APIKey:      apiKey,
		Type:        APIKeyTypePrivate,
		Environment: env,
	}, nil
}

// authenticateJWTKey validates JWT API key - trigger.dev alignment
func (s *service) authenticateJWTKey(ctx context.Context, jwtToken string) (*AuthenticationResult, error) {
	// Parse JWT token
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return &AuthenticationResult{
			Success: false,
			Error:   "invalid JWT token",
		}, nil
	}

	// Validate and extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Extract environment ID from subject
		envID, ok := claims["sub"].(string)
		if !ok {
			return &AuthenticationResult{
				Success: false,
				Error:   "invalid JWT claims: missing subject",
			}, nil
		}

		// Validate environment ID format
		if _, err := uuid.Parse(envID); err != nil {
			return &AuthenticationResult{
				Success: false,
				Error:   "invalid environment ID in JWT",
			}, nil
		}

		// Find environment
		env, err := s.repo.FindEnvironmentByAPIKey(ctx, envID)
		if err != nil {
			return &AuthenticationResult{
				Success: false,
				Error:   "environment not found for JWT",
			}, nil
		}

		// Build result with JWT-specific claims
		result := &AuthenticationResult{
			Success:     true,
			APIKey:      jwtToken,
			Type:        APIKeyTypePublicJWT,
			Environment: env,
		}

		// Extract optional JWT claims - trigger.dev alignment
		if scopes, ok := claims["scopes"].([]interface{}); ok {
			for _, scope := range scopes {
				if s, ok := scope.(string); ok {
					result.Scopes = append(result.Scopes, s)
				}
			}
		}

		if otu, ok := claims["otu"].(bool); ok {
			result.OneTimeUse = otu
		}

		if realtime, ok := claims["realtime"].(bool); ok {
			result.Realtime = realtime
		}

		return result, nil
	}

	return &AuthenticationResult{
		Success: false,
		Error:   "invalid JWT token claims",
	}, nil
}

// generateSecureToken generates a secure random token for PAT/OAT
func generateSecureToken() string {
	// Generate a UUID-based token
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
