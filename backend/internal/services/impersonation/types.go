package impersonation

import (
	"errors"
	"net/http"
	"time"
)

// Common errors
var (
	ErrInvalidSecretKey     = errors.New("secret key must be at least 16 bytes")
	ErrInvalidUserID        = errors.New("user ID cannot be empty")
	ErrCookieNotFound       = errors.New("impersonation cookie not found")
	ErrInvalidCookieFormat  = errors.New("invalid cookie format")
	ErrInvalidSignature     = errors.New("invalid cookie signature")
)

// ImpersonationService defines the interface for user impersonation management
type ImpersonationService interface {
	// SetImpersonation sets the impersonated user ID in a secure cookie
	SetImpersonation(w http.ResponseWriter, r *http.Request, userID string) error
	
	// GetImpersonation retrieves the impersonated user ID from the request
	// Returns empty string if no impersonation is active
	GetImpersonation(r *http.Request) (string, error)
	
	// ClearImpersonation removes the impersonation cookie
	ClearImpersonation(w http.ResponseWriter, r *http.Request) error
	
	// IsImpersonating checks if the request has an active impersonation session
	IsImpersonating(r *http.Request) bool
}

// Config holds the configuration for the impersonation service
type Config struct {
	// SecretKey is used for HMAC signing of cookies (required)
	SecretKey []byte
	
	// CookieName is the name of the cookie (default: "__impersonate")
	CookieName string
	
	// Domain sets the cookie domain (optional)
	Domain string
	
	// Path sets the cookie path (default: "/")
	Path string
	
	// MaxAge sets the cookie expiration time (default: 24 hours)
	MaxAge time.Duration
	
	// Secure enables secure flag for HTTPS environments
	Secure bool
	
	// HttpOnly enables HttpOnly flag for security (default: true)
	HttpOnly bool
	
	// SameSite sets the SameSite cookie attribute (default: Lax)
	SameSite http.SameSite
}

// DefaultConfig returns a configuration with sensible defaults
// aligned with trigger.dev's impersonation service settings
func DefaultConfig() *Config {
	return &Config{
		CookieName: "__impersonate", // Exact match with trigger.dev
		Path:       "/",
		MaxAge:     24 * time.Hour, // 1 day, same as trigger.dev
		HttpOnly:   true,           // Security best practice
		SameSite:   http.SameSiteLaxMode, // CSRF protection
		Secure:     false,          // Will be set based on environment
	}
}

// Service implements the ImpersonationService interface
type Service struct {
	config *Config
}

// NewService creates a new impersonation service with the given configuration
func NewService(config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Ensure required fields have defaults
	if config.CookieName == "" {
		config.CookieName = "__impersonate"
	}
	if config.Path == "" {
		config.Path = "/"
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}
	
	return &Service{config: config}
}

// NewServiceWithSecretKey creates a new service with a custom secret key
// This is a convenience function for quick setup
func NewServiceWithSecretKey(secretKey []byte) (*Service, error) {
	if len(secretKey) < 16 {
		return nil, ErrInvalidSecretKey
	}
	
	config := DefaultConfig()
	config.SecretKey = secretKey
	
	return NewService(config), nil
}