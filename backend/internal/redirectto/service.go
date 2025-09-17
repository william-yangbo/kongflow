package redirectto

import (
	"net/http"
)

// SetRedirectTo implements the RedirectToService interface
// It validates and stores the redirect URL in an encrypted cookie
func (s *Service) SetRedirectTo(w http.ResponseWriter, r *http.Request, redirectTo string) error {
	// Validate the redirect URL
	if !isValidRedirectURL(redirectTo) {
		return ErrInvalidRedirectURL
	}

	// Validate secret key
	if err := validateSecretKey(s.config.SecretKey); err != nil {
		return err
	}

	// Set the encrypted cookie
	return s.setCookie(w, redirectTo)
}

// GetRedirectTo implements the RedirectToService interface
// It retrieves and decrypts the redirect URL from the cookie
func (s *Service) GetRedirectTo(r *http.Request) (string, error) {
	// Validate secret key
	if err := validateSecretKey(s.config.SecretKey); err != nil {
		return "", err
	}

	// Get and decrypt the cookie value
	redirectURL, err := s.getCookie(r)
	if err != nil {
		return "", err
	}

	// Validate the decrypted URL (additional safety check)
	if !isValidRedirectURL(redirectURL) {
		return "", ErrInvalidRedirectURL
	}

	return redirectURL, nil
}

// ClearRedirectTo implements the RedirectToService interface
// It removes the redirect cookie
func (s *Service) ClearRedirectTo(w http.ResponseWriter, r *http.Request) error {
	// Clear the cookie
	s.clearCookie(w)
	return nil
}

// NewServiceWithSecretKey creates a new service with a custom secret key
// This is a convenience function for quick setup
func NewServiceWithSecretKey(secretKey []byte) (*Service, error) {
	if err := validateSecretKey(secretKey); err != nil {
		return nil, err
	}

	config := DefaultConfig()
	config.SecretKey = secretKey

	return NewService(config), nil
}

// SetSecure sets the secure flag for cookies based on the environment
// This should typically be called during service initialization
func (s *Service) SetSecure(secure bool) {
	s.config.Secure = secure
}
