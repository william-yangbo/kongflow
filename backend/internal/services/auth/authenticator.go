package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"kongflow/backend/internal/services/analytics"
	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/impersonation"
	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/sessionstorage"
	"kongflow/backend/internal/services/ulid"
	"kongflow/backend/internal/shared"
)

// AuthStrategy defines the interface for authentication strategies
// Aligned with trigger.dev's strategy pattern
type AuthStrategy interface {
	Name() string
	Authenticate(ctx context.Context, req *http.Request) (*AuthUser, error)
	HandleCallback(ctx context.Context, req *http.Request) (*AuthUser, error)
}

// Authenticator manages authentication strategies and session handling
// Enhanced with integrated services: logger, analytics, ulid
// Aligned with trigger.dev's Authenticator<AuthUser> pattern
type Authenticator struct {
	sessionStorage    SessionStorage
	strategies        map[string]AuthStrategy
	logger            *logger.Logger             // Enhanced: structured logging
	analytics         analytics.AnalyticsService // Enhanced: user behavior tracking
	ulid              *ulid.Service              // Enhanced: secure ID generation
	postAuthProcessor *PostAuthProcessor         // Enhanced: enterprise post-auth processing
}

// SessionStorage interface for session management
// Aligned with trigger.dev's sessionStorage pattern
type SessionStorage interface {
	GetSession(ctx context.Context, req *http.Request) (map[string]interface{}, error)
	CommitSession(ctx context.Context, w http.ResponseWriter, req *http.Request, session map[string]interface{}) error
	DestroySession(ctx context.Context, w http.ResponseWriter, req *http.Request) error
}

// CookieSessionStorage adapts the existing sessionstorage service to our interface
type CookieSessionStorage struct{}

// NewCookieSessionStorage creates a new cookie-based session storage
func NewCookieSessionStorage() *CookieSessionStorage {
	return &CookieSessionStorage{}
}

// GetSession gets session data from request
func (c *CookieSessionStorage) GetSession(ctx context.Context, req *http.Request) (map[string]interface{}, error) {
	session, err := sessionstorage.GetUserSession(req)
	if err != nil {
		return nil, err
	}

	// Convert map[interface{}]interface{} to map[string]interface{}
	result := make(map[string]interface{})
	for key, value := range session.Values {
		if keyStr, ok := key.(string); ok {
			result[keyStr] = value
		}
	}
	return result, nil
}

// CommitSession saves session data to response
func (c *CookieSessionStorage) CommitSession(ctx context.Context, w http.ResponseWriter, req *http.Request, sessionData map[string]interface{}) error {
	session, err := sessionstorage.GetUserSession(req)
	if err != nil {
		return err
	}

	// Copy data to session
	for key, value := range sessionData {
		session.Values[key] = value
	}

	return sessionstorage.CommitSession(req, w, session)
}

// DestroySession removes session from response
func (c *CookieSessionStorage) DestroySession(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	return sessionstorage.DestroySession(req, w)
}

// NewAuthenticator creates a new authenticator instance with enhanced services
// Aligned with trigger.dev's authenticator creation pattern
func NewAuthenticator(sessionStorage SessionStorage, postAuthProcessor *PostAuthProcessor) *Authenticator {
	return &Authenticator{
		sessionStorage:    sessionStorage,
		strategies:        make(map[string]AuthStrategy),
		logger:            logger.NewWebapp("auth"), // Enhanced: webapp-style logger
		analytics:         nil,                      // Enhanced: will be set via SetAnalytics()
		ulid:              ulid.New(),               // Enhanced: secure ID generation
		postAuthProcessor: postAuthProcessor,        // Enhanced: enterprise post-auth processing
	}
}

// SetAnalytics sets the analytics service for the authenticator
func (a *Authenticator) SetAnalytics(analytics analytics.AnalyticsService) {
	a.analytics = analytics
}

// RegisterStrategy adds an authentication strategy
// Aligned with trigger.dev's strategy registration pattern
func (a *Authenticator) RegisterStrategy(strategy AuthStrategy) {
	a.strategies[strategy.Name()] = strategy
}

// IsAuthenticated checks if user is authenticated from session
// Aligned with trigger.dev's authenticator.isAuthenticated method
func (a *Authenticator) IsAuthenticated(ctx context.Context, req *http.Request) (*AuthUser, error) {
	session, err := a.sessionStorage.GetSession(ctx, req)
	if err != nil {
		return nil, err
	}

	userIDInterface, exists := session["auth_user"]
	if !exists {
		return nil, nil // Not authenticated
	}

	userIDMap, ok := userIDInterface.(map[string]interface{})
	if !ok {
		return nil, ErrSessionInvalid
	}

	userID, ok := userIDMap["userId"].(string)
	if !ok || userID == "" {
		return nil, ErrSessionInvalid
	}

	return &AuthUser{UserID: userID}, nil
}

// Authenticate authenticates user using specified strategy
// Enhanced with logging and analytics
// Aligned with trigger.dev's authenticator.authenticate method
func (a *Authenticator) Authenticate(ctx context.Context, strategyName string, req *http.Request) (*AuthUser, error) {
	// Enhanced: log authentication attempt
	a.logger.Info("Authentication attempt", map[string]interface{}{
		"strategy":  strategyName,
		"userAgent": req.UserAgent(),
		"ip":        getClientIP(req),
	})

	strategy, exists := a.strategies[strategyName]
	if !exists {
		a.logger.Error("Strategy not found", map[string]interface{}{
			"strategy": strategyName,
		})
		return nil, fmt.Errorf("strategy %s not found", strategyName)
	}

	authUser, err := strategy.Authenticate(ctx, req)
	if err != nil {
		a.logger.Error("Authentication failed", map[string]interface{}{
			"strategy": strategyName,
			"error":    err.Error(),
		})
		return nil, err
	}

	// Enhanced: track authentication event if analytics available
	if a.analytics != nil && authUser != nil {
		a.analytics.Capture(ctx, &analytics.TelemetryEvent{
			UserID: authUser.UserID,
			Event:  "Authentication Attempted",
			Properties: map[string]interface{}{
				"strategy": strategyName,
			},
		})
	}

	a.logger.Info("Authentication successful", map[string]interface{}{
		"strategy": strategyName,
		"userID":   authUser.UserID,
	})

	return authUser, nil
}

// HandleCallback handles authentication callback for specified strategy
func (a *Authenticator) HandleCallback(ctx context.Context, strategyName string, req *http.Request) (*AuthUser, error) {
	strategy, exists := a.strategies[strategyName]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategyName)
	}

	return strategy.HandleCallback(ctx, req)
}

// StoreUserInSession stores authenticated user in session
func (a *Authenticator) StoreUserInSession(ctx context.Context, w http.ResponseWriter, req *http.Request, user *AuthUser) error {
	session, err := a.sessionStorage.GetSession(ctx, req)
	if err != nil {
		return err
	}

	session["auth_user"] = map[string]interface{}{
		"userId": user.UserID,
	}

	return a.sessionStorage.CommitSession(ctx, w, req, session)
}

// Logout removes user from session
func (a *Authenticator) Logout(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	return a.sessionStorage.DestroySession(ctx, w, req)
}

// IsGitHubAuthSupported checks if GitHub OAuth is configured
// Aligned with trigger.dev's isGithubAuthSupported
func IsGitHubAuthSupported() bool {
	clientID := os.Getenv("AUTH_GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("AUTH_GITHUB_CLIENT_SECRET")
	return clientID != "" && clientSecret != ""
}

// Default authenticator instance
var (
	DefaultAuthenticator  *Authenticator
	DefaultSessionService SessionService
)

// InitializeAuthenticator sets up the default authenticator with strategies
// Aligned with trigger.dev's auth.server.ts setup pattern
func InitializeAuthenticator(sessionStorage SessionStorage, queries *shared.Queries, impersonationService impersonation.ImpersonationService, emailService email.EmailService, analyticsService analytics.AnalyticsService) {
	// Create post-auth processor with required dependencies
	postAuthProcessor := NewPostAuthProcessor(
		analyticsService,
		logger.NewWebapp("auth.postauth"),
		nil, // WorkerQueue can be nil for basic usage
	)

	DefaultAuthenticator = NewAuthenticator(sessionStorage, postAuthProcessor)
	DefaultSessionService = NewSessionService(DefaultAuthenticator, queries, impersonationService)

	// Add email strategy (always available)
	emailStrategy := NewEmailStrategy(emailService, queries, postAuthProcessor)
	DefaultAuthenticator.RegisterStrategy(emailStrategy)

	// Add GitHub strategy if configured (aligned with trigger.dev's conditional setup)
	if IsGitHubAuthSupported() {
		clientID := os.Getenv("AUTH_GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("AUTH_GITHUB_CLIENT_SECRET")
		AddGitHubStrategy(DefaultAuthenticator, clientID, clientSecret, queries, analyticsService)
	}
}

// getClientIP extracts the client IP address from the request
// Handles various proxy headers for accurate IP detection
func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header first (most common)
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return req.RemoteAddr
}
