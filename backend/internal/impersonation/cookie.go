package impersonation

import (
	"net/http"
)

// setCookie sets a secure cookie with the impersonation data
func (s *Service) setCookie(w http.ResponseWriter, value string) {
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    value,
		Path:     s.config.Path,
		Domain:   s.config.Domain,
		MaxAge:   int(s.config.MaxAge.Seconds()),
		HttpOnly: s.config.HttpOnly,
		Secure:   s.config.Secure,
		SameSite: s.config.SameSite,
	}

	http.SetCookie(w, cookie)
}

// getCookie retrieves the impersonation cookie from the request
func (s *Service) getCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(s.config.CookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, ErrCookieNotFound
		}
		return nil, err
	}
	return cookie, nil
}

// clearCookie removes the impersonation cookie by setting it to expire
func (s *Service) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    "",
		Path:     s.config.Path,
		Domain:   s.config.Domain,
		MaxAge:   -1, // Expire immediately
		HttpOnly: s.config.HttpOnly,
		Secure:   s.config.Secure,
		SameSite: s.config.SameSite,
	}

	http.SetCookie(w, cookie)
}

// SetSecure configures the secure flag based on the environment
// This is a utility function for production vs development environments
func (s *Service) SetSecure(secure bool) {
	s.config.Secure = secure
}
