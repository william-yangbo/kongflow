package apiauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// service implements APIAuthService interface - trigger.dev apiAuth.server.ts alignment
type service struct {
	repo      Repository
	jwtSecret []byte
}

// NewAPIAuthService creates a new API authentication service
func NewAPIAuthService(repo Repository, jwtSecret string) APIAuthService {
	return &service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// AuthenticateAPIRequest authenticates API request from HTTP request - trigger.dev alignment
func (s *service) AuthenticateAPIRequest(ctx context.Context, req *http.Request, opts *AuthOptions) (*AuthenticationResult, error) {
	// Extract Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return &AuthenticationResult{
			Success: false,
			Error:   "missing authorization header",
		}, nil
	}

	return s.AuthenticateAuthorizationHeader(ctx, authHeader, opts)
}

// AuthenticateAuthorizationHeader authenticates from Authorization header directly
func (s *service) AuthenticateAuthorizationHeader(ctx context.Context, authorization string, opts *AuthOptions) (*AuthenticationResult, error) {
	// Validate Bearer format
	if !strings.HasPrefix(authorization, "Bearer ") {
		return &AuthenticationResult{
			Success: false,
			Error:   "invalid authorization format, expected 'Bearer {token}'",
		}, nil
	}

	// Extract API key/token
	apiKey := strings.TrimPrefix(authorization, "Bearer ")
	if apiKey == "" {
		return &AuthenticationResult{
			Success: false,
			Error:   "empty token in authorization header",
		}, nil
	}

	return s.authenticateAPIKey(ctx, apiKey, opts)
}

// authenticateAPIKey performs API key authentication based on type - trigger.dev alignment
func (s *service) authenticateAPIKey(ctx context.Context, apiKey string, opts *AuthOptions) (*AuthenticationResult, error) {
	// Determine API key type
	keyType := getAPIKeyType(apiKey)
	if keyType == "" {
		return &AuthenticationResult{
			Success: false,
			Error:   "invalid API key format",
		}, nil
	}

	// Check if key type is allowed
	if !opts.AllowPublicKey && keyType == APIKeyTypePublic {
		return &AuthenticationResult{
			Success: false,
			Error:   "public API keys are not allowed for this request",
		}, nil
	}

	if !opts.AllowJWT && keyType == APIKeyTypePublicJWT {
		return &AuthenticationResult{
			Success: false,
			Error:   "public JWT API keys are not allowed for this request",
		}, nil
	}

	// Route to appropriate authentication method
	switch keyType {
	case APIKeyTypePublic:
		return s.authenticatePublicKey(ctx, apiKey, opts.BranchName)
	case APIKeyTypePrivate:
		return s.authenticatePrivateKey(ctx, apiKey, opts.BranchName)
	case APIKeyTypePublicJWT:
		return s.authenticateJWTKey(ctx, apiKey)
	default:
		return &AuthenticationResult{
			Success: false,
			Error:   "unsupported API key type",
		}, nil
	}
}

// AuthenticateRequest supports multiple authentication methods - trigger.dev alignment
func (s *service) AuthenticateRequest(ctx context.Context, req *http.Request, config *AuthConfig) (*UnifiedAuthResult, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Try API key authentication if enabled
	if config.APIKey {
		keyType := getAPIKeyType(token)
		if keyType != "" {
			opts := &AuthOptions{
				AllowPublicKey: true,
				AllowJWT:       true,
			}
			result, err := s.authenticateAPIKey(ctx, token, opts)
			if err != nil {
				return nil, err
			}
			if result.Success {
				return &UnifiedAuthResult{
					Type:      AuthTypeAPIKey,
					APIResult: result,
				}, nil
			}
		}
	}

	// Try personal access token if enabled
	if config.PersonalAccessToken {
		patResult, err := s.repo.AuthenticatePersonalAccessToken(ctx, token)
		if err == nil && patResult != nil {
			return &UnifiedAuthResult{
				Type:   AuthTypePersonalAccessToken,
				UserID: patResult.UserID,
			}, nil
		}
	}

	// Try organization access token if enabled
	if config.OrganizationAccessToken {
		oatResult, err := s.repo.AuthenticateOrganizationAccessToken(ctx, token)
		if err == nil && oatResult != nil {
			return &UnifiedAuthResult{
				Type:  AuthTypeOrganizationAccessToken,
				OrgID: oatResult.OrgID,
			}, nil
		}
	}

	return nil, fmt.Errorf("authentication failed with all configured methods")
}

// GenerateJWTToken generates JWT token for environment - trigger.dev alignment
func (s *service) GenerateJWTToken(ctx context.Context, env *RuntimeEnvironment, payload map[string]interface{}, opts *JWTOptions) (string, error) {
	// Build JWT claims
	claims := jwt.MapClaims{
		"sub": env.ID.Bytes,
		"pub": true,
		"iat": time.Now().Unix(),
	}

	// Add custom payload
	for k, v := range payload {
		claims[k] = v
	}

	// Set expiration time
	if opts != nil && opts.ExpirationTime != nil {
		switch exp := opts.ExpirationTime.(type) {
		case time.Duration:
			claims["exp"] = time.Now().Add(exp).Unix()
		case int64:
			claims["exp"] = exp
		case string:
			duration, err := time.ParseDuration(exp)
			if err != nil {
				return "", fmt.Errorf("invalid expiration duration: %w", err)
			}
			claims["exp"] = time.Now().Add(duration).Unix()
		default:
			return "", fmt.Errorf("invalid expiration time type")
		}
	} else {
		// Default 1 hour expiration
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}

	// Add custom claims if provided
	if opts != nil && opts.CustomClaims != nil {
		for k, v := range opts.CustomClaims {
			claims[k] = v
		}
	}

	// Generate and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GetAuthenticatedEnvironment gets environment with full project and organization context
func (s *service) GetAuthenticatedEnvironment(ctx context.Context, authResult *AuthenticationResult, projectRef, envSlug string) (*AuthenticatedEnvironment, error) {
	if authResult == nil || !authResult.Success || authResult.Environment == nil {
		return nil, fmt.Errorf("invalid authentication result")
	}

	if !authResult.Environment.ID.Valid {
		return nil, fmt.Errorf("invalid environment ID")
	}

	envID := fmt.Sprintf("%x-%x-%x-%x-%x",
		authResult.Environment.ID.Bytes[0:4],
		authResult.Environment.ID.Bytes[4:6],
		authResult.Environment.ID.Bytes[6:8],
		authResult.Environment.ID.Bytes[8:10],
		authResult.Environment.ID.Bytes[10:16])

	return s.repo.GetEnvironmentWithProjectAndOrg(ctx, envID)
}
