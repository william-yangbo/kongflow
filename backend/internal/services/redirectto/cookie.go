package redirectto

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// setCookie sets an encrypted cookie with the redirect URL
func (s *Service) setCookie(w http.ResponseWriter, value string) error {
	// Encrypt the value
	encryptedValue, err := s.encrypt(value)
	if err != nil {
		return err
	}

	// Create cookie with configuration matching trigger.dev
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    encryptedValue,
		Path:     s.config.Path,
		MaxAge:   int(s.config.MaxAge.Seconds()),
		HttpOnly: s.config.HTTPOnly,
		Secure:   s.config.Secure,
		SameSite: s.config.SameSite,
	}

	// Set the cookie
	http.SetCookie(w, cookie)
	return nil
}

// getCookie retrieves and decrypts the redirect URL from the cookie
func (s *Service) getCookie(r *http.Request) (string, error) {
	// Get the cookie
	cookie, err := r.Cookie(s.config.CookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", ErrCookieNotFound
		}
		return "", err
	}

	// Decrypt the value
	decryptedValue, err := s.decrypt(cookie.Value)
	if err != nil {
		return "", err
	}

	return decryptedValue, nil
}

// clearCookie removes the redirect cookie by setting it with a past expiration
func (s *Service) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    "",
		Path:     s.config.Path,
		MaxAge:   -1, // Negative MaxAge means delete immediately
		HttpOnly: s.config.HTTPOnly,
		Secure:   s.config.Secure,
		SameSite: s.config.SameSite,
		Expires:  time.Unix(0, 0), // Set to epoch for compatibility
	}

	http.SetCookie(w, cookie)
}

// isValidRedirectURL performs basic validation on redirect URLs
// This is a simple validation to prevent obvious security issues
func isValidRedirectURL(redirectURL string) bool {
	if redirectURL == "" {
		return false
	}

	// Prevent obviously malicious URLs
	if strings.Contains(redirectURL, "\n") || strings.Contains(redirectURL, "\r") {
		return false
	}

	// Allow relative URLs and same-origin URLs
	// This is a basic check - more sophisticated validation might be needed
	// depending on security requirements
	if strings.HasPrefix(redirectURL, "/") {
		return true
	}

	// Parse and validate absolute URLs
	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}

	// Allow only HTTP/HTTPS schemes
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}
