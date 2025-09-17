package impersonation

import (
	"net/http"
	"strings"
)

// SetImpersonation sets the impersonated user ID in a secure cookie
// This method aligns with trigger.dev's setImpersonationId functionality
func (s *Service) SetImpersonation(w http.ResponseWriter, r *http.Request, userID string) error {
	// Validate input
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidUserID
	}

	if len(s.config.SecretKey) == 0 {
		return ErrInvalidSecretKey
	}

	// Encode the user ID
	encoded := encodeUserID(userID)

	// Sign the encoded value
	signed, err := s.signValue(encoded)
	if err != nil {
		return err
	}

	// Set the cookie
	s.setCookie(w, signed)

	return nil
}

// GetImpersonation retrieves the impersonated user ID from the request
// Returns empty string if no impersonation is active or cookie is invalid
// This aligns with trigger.dev's getImpersonationId functionality
func (s *Service) GetImpersonation(r *http.Request) (string, error) {
	// Get the cookie
	cookie, err := s.getCookie(r)
	if err != nil {
		if err == ErrCookieNotFound {
			return "", nil // No impersonation active, return empty string
		}
		return "", err
	}

	// Verify and unsign the cookie value
	unsigned, err := s.unsignValue(cookie.Value)
	if err != nil {
		// Invalid signature or format, treat as no impersonation
		return "", nil
	}

	// Decode the user ID
	userID, err := decodeUserID(unsigned)
	if err != nil {
		// Invalid encoding, treat as no impersonation
		return "", nil
	}

	return userID, nil
}

// ClearImpersonation removes the impersonation cookie
// This aligns with trigger.dev's clearImpersonationId functionality
func (s *Service) ClearImpersonation(w http.ResponseWriter, r *http.Request) error {
	s.clearCookie(w)
	return nil
}

// IsImpersonating checks if the request has an active impersonation session
// This is a convenience method not present in trigger.dev but useful for middleware
func (s *Service) IsImpersonating(r *http.Request) bool {
	userID, err := s.GetImpersonation(r)
	if err != nil {
		return false
	}
	return userID != ""
}

// GetImpersonationWithFallback returns the impersonated user ID if active,
// otherwise returns the provided fallback user ID
// This is useful for implementing the same pattern as trigger.dev's session.server.ts
func (s *Service) GetImpersonationWithFallback(r *http.Request, fallbackUserID string) (string, error) {
	impersonatedID, err := s.GetImpersonation(r)
	if err != nil {
		return "", err
	}

	if impersonatedID != "" {
		return impersonatedID, nil
	}

	return fallbackUserID, nil
}
