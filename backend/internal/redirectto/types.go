package redirectto

import (
	"net/http"
	"time"
)

// RedirectToService defines the interface for redirect management
type RedirectToService interface {
	// SetRedirectTo sets the redirect URL in an encrypted cookie
	SetRedirectTo(w http.ResponseWriter, r *http.Request, redirectTo string) error

	// GetRedirectTo retrieves the redirect URL from the cookie
	GetRedirectTo(r *http.Request) (string, error)

	// ClearRedirectTo removes the redirect cookie
	ClearRedirectTo(w http.ResponseWriter, r *http.Request) error
}

// Config holds the configuration for the RedirectTo service
type Config struct {
	// CookieName is the name of the cookie (default: "__redirectTo")
	CookieName string

	// SecretKey is used for AES encryption (must be 16, 24, or 32 bytes)
	SecretKey []byte

	// MaxAge is the cookie expiration time (default: 24 hours)
	MaxAge time.Duration

	// Secure indicates if the cookie should only be sent over HTTPS
	Secure bool

	// HTTPOnly indicates if the cookie should be HTTP only
	HTTPOnly bool

	// SameSite controls the SameSite attribute
	SameSite http.SameSite

	// Path is the cookie path
	Path string
}

// DefaultConfig returns the default configuration aligned with trigger.dev
func DefaultConfig() *Config {
	return &Config{
		CookieName: "__redirectTo",
		MaxAge:     24 * time.Hour, // ONE_DAY = 60 * 60 * 24
		HTTPOnly:   true,
		SameSite:   http.SameSiteLaxMode, // "lax"
		Path:       "/",
		Secure:     false, // Will be set dynamically based on environment
	}
}

// Service represents the concrete implementation of RedirectToService
type Service struct {
	config *Config
}

// NewService creates a new RedirectTo service with the given configuration
func NewService(config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		config: config,
	}
}
