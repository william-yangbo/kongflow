package auth

// Package auth provides user authentication and session management services
// Strictly aligned with trigger.dev's auth system architecture

import (
	"context"
	"errors"
	"net/http"

	"kongflow/backend/internal/shared"
)

// AuthUser represents the authenticated user data stored in session
// Strictly aligned with trigger.dev's AuthUser type
type AuthUser struct {
	UserID string `json:"userId" validate:"required"`
}

// SessionService defines the interface for session management
// Aligned with trigger.dev's session.server.ts functionality
type SessionService interface {
	// GetUserID returns the current user ID, considering impersonation
	// Aligned with trigger.dev's getUserId function
	GetUserID(ctx context.Context, request *http.Request) (string, error)

	// GetUser returns the current user, considering impersonation
	// Aligned with trigger.dev's getUser function
	GetUser(ctx context.Context, request *http.Request) (*shared.Users, error)

	// RequireUserID ensures user is authenticated, redirects if not
	// Aligned with trigger.dev's requireUserId function
	RequireUserID(ctx context.Context, request *http.Request, redirectTo string) (string, error)

	// RequireUser ensures user is authenticated and returns user data
	// Aligned with trigger.dev's requireUser function
	RequireUser(ctx context.Context, request *http.Request) (*shared.Users, error)

	// Logout handles user logout
	// Aligned with trigger.dev's logout function
	Logout(ctx context.Context, request *http.Request) error

	// Additional utility methods
	IsAuthenticated(ctx context.Context, request *http.Request) (bool, error)
	GetAuthUser(ctx context.Context, request *http.Request) (*AuthUser, error)
}

// AuthMiddleware defines authentication middleware interface
type AuthMiddleware interface {
	// RequireAuth middleware that ensures user is authenticated
	RequireAuth(redirectPath string) func(http.Handler) http.Handler

	// OptionalAuth middleware that sets user context if authenticated
	OptionalAuth() func(http.Handler) http.Handler

	// SetUserContext sets user information in request context
	SetUserContext(userID string, user *shared.Users) func(http.Handler) http.Handler
}

// Configuration for the auth service
type Config struct {
	LoginPath    string `env:"AUTH_LOGIN_PATH" default:"/login"`
	LogoutPath   string `env:"AUTH_LOGOUT_PATH" default:"/logout"`
	SessionKey   string `env:"AUTH_SESSION_KEY" default:"auth_user"`
	CookieDomain string `env:"AUTH_COOKIE_DOMAIN"`
	CookieSecure bool   `env:"AUTH_COOKIE_SECURE" default:"true"`
}

// Standard authentication errors
var (
	ErrUnauthenticated = errors.New("user not authenticated")
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionInvalid  = errors.New("session is invalid")
	ErrSessionExpired  = errors.New("session has expired")
	ErrInvalidRequest  = errors.New("invalid request")
)
