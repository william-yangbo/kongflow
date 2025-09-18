// Package apiauth provides authentication services for KongFlow APIs
// This package is aligned with trigger.dev's apiAuth.server.ts functionality
package apiauth

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// APIKeyType represents the type of API key - aligned with trigger.dev
type APIKeyType string

const (
	APIKeyTypePublic    APIKeyType = "PUBLIC"     // pk_ prefix - public key, limited permissions
	APIKeyTypePrivate   APIKeyType = "PRIVATE"    // tr_ prefix - private key, full permissions
	APIKeyTypePublicJWT APIKeyType = "PUBLIC_JWT" // JWT format public key
)

// EnvironmentType represents runtime environment types - trigger.dev alignment
type EnvironmentType string

const (
	EnvironmentTypeProduction  EnvironmentType = "PRODUCTION"
	EnvironmentTypeStaging     EnvironmentType = "STAGING"
	EnvironmentTypeDevelopment EnvironmentType = "DEVELOPMENT"
	EnvironmentTypePreview     EnvironmentType = "PREVIEW"
)

// AuthenticationType represents different authentication methods
type AuthenticationType string

const (
	AuthTypePersonalAccessToken     AuthenticationType = "personalAccessToken"
	AuthTypeOrganizationAccessToken AuthenticationType = "organizationAccessToken"
	AuthTypeAPIKey                  AuthenticationType = "apiKey"
)

// AuthOptions configures API key authentication behavior - trigger.dev alignment
type AuthOptions struct {
	AllowPublicKey bool   `json:"allowPublicKey"`
	AllowJWT       bool   `json:"allowJWT"`
	BranchName     string `json:"branchName,omitempty"`
}

// AuthConfig configures which authentication methods are allowed
type AuthConfig struct {
	PersonalAccessToken     bool `json:"personalAccessToken"`
	OrganizationAccessToken bool `json:"organizationAccessToken"`
	APIKey                  bool `json:"apiKey"`
}

// RuntimeEnvironment represents trigger.dev RuntimeEnvironment entity
type RuntimeEnvironment struct {
	ID             pgtype.UUID     `json:"id"`
	Slug           string          `json:"slug"`
	APIKey         string          `json:"apiKey"`
	Type           EnvironmentType `json:"type"`
	OrganizationID pgtype.UUID     `json:"organizationId"`
	ProjectID      pgtype.UUID     `json:"projectId"`
	OrgMemberID    pgtype.UUID     `json:"orgMemberId,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// AuthenticationResult represents API key authentication outcome - trigger.dev alignment
type AuthenticationResult struct {
	Success     bool                `json:"success"`
	Error       string              `json:"error,omitempty"`
	APIKey      string              `json:"apiKey"`
	Type        APIKeyType          `json:"type"`
	Environment *RuntimeEnvironment `json:"environment"`
	Scopes      []string            `json:"scopes,omitempty"`
	OneTimeUse  bool                `json:"oneTimeUse,omitempty"`
	Realtime    bool                `json:"realtime,omitempty"`
}

// UnifiedAuthResult represents unified authentication result supporting multiple auth types
type UnifiedAuthResult struct {
	Type      AuthenticationType    `json:"type"`
	UserID    string                `json:"userId,omitempty"`
	OrgID     string                `json:"organizationId,omitempty"`
	APIResult *AuthenticationResult `json:"apiResult,omitempty"`
}

// PersonalAccessTokenResult represents PAT authentication result
type PersonalAccessTokenResult struct {
	Token  *PersonalAccessTokens `json:"token"`
	UserID string                `json:"userId"`
}

// OrganizationAccessTokenResult represents OAT authentication result
type OrganizationAccessTokenResult struct {
	Token *OrganizationAccessTokens `json:"token"`
	OrgID string                    `json:"organizationId"`
}

// AuthenticatedEnvironment represents environment with full project and organization info
type AuthenticatedEnvironment struct {
	Environment RuntimeEnvironment `json:"environment"`
	ProjectID   pgtype.UUID        `json:"projectId"`
	ProjectSlug string             `json:"projectSlug"`
	ProjectName string             `json:"projectName"`
	OrgID       pgtype.UUID        `json:"orgId"`
	OrgSlug     string             `json:"orgSlug"`
	OrgTitle    string             `json:"orgTitle"`
}

// JWTOptions configures JWT token generation
type JWTOptions struct {
	ExpirationTime interface{}            `json:"expirationTime,omitempty"` // time.Duration or int64 (unix timestamp)
	CustomClaims   map[string]interface{} `json:"customClaims,omitempty"`
}

// APIAuthService defines the main authentication service interface - trigger.dev alignment
type APIAuthService interface {
	// Core authentication methods - matching trigger.dev apiAuth.server.ts
	AuthenticateAPIRequest(ctx context.Context, req *http.Request, opts *AuthOptions) (*AuthenticationResult, error)
	AuthenticateAuthorizationHeader(ctx context.Context, authorization string, opts *AuthOptions) (*AuthenticationResult, error)
	AuthenticateRequest(ctx context.Context, req *http.Request, config *AuthConfig) (*UnifiedAuthResult, error)

	// JWT token generation - trigger.dev generateJWTTokenForEnvironment alignment
	GenerateJWTToken(ctx context.Context, env *RuntimeEnvironment, payload map[string]interface{}, opts *JWTOptions) (string, error)

	// Environment lookup with full context
	GetAuthenticatedEnvironment(ctx context.Context, authResult *AuthenticationResult, projectRef, envSlug string) (*AuthenticatedEnvironment, error)
}

// Repository interface defines data access layer combining shared + apiauth queries
type Repository interface {
	// Shared queries - environment lookup
	FindEnvironmentByAPIKey(ctx context.Context, apiKey string) (*RuntimeEnvironment, error)
	FindEnvironmentByPublicAPIKey(ctx context.Context, apiKey string, branch *string) (*RuntimeEnvironment, error)
	GetEnvironmentWithProjectAndOrg(ctx context.Context, envID string) (*AuthenticatedEnvironment, error)

	// ApiAuth specific queries - token authentication
	AuthenticatePersonalAccessToken(ctx context.Context, token string) (*PersonalAccessTokenResult, error)
	AuthenticateOrganizationAccessToken(ctx context.Context, token string) (*OrganizationAccessTokenResult, error)
}
