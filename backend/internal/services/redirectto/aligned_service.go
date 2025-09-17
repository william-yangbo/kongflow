package redirectto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AlignedService provides Remix-compatible cookie session behavior
// This closely mimics trigger.dev's createCookieSessionStorage approach
type AlignedService struct {
	config *Config
}

// Session represents a cookie session similar to Remix's session
type Session struct {
	data   map[string]interface{}
	id     string
	dirty  bool
	config *Config
}

// NewAlignedService creates a service that closely matches trigger.dev behavior
func NewAlignedService(config *Config) *AlignedService {
	if config == nil {
		config = DefaultConfig()
	}
	return &AlignedService{config: config}
}

// GetSession extracts session from request cookies (like trigger.dev's getSession)
func (s *AlignedService) GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(s.config.CookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			// Return empty session like Remix does
			return &Session{
				data:   make(map[string]interface{}),
				id:     generateSessionID(),
				dirty:  false,
				config: s.config,
			}, nil
		}
		return nil, err
	}

	// Decrypt and parse session data
	sessionData, err := s.unsignCookie(cookie.Value)
	if err != nil {
		// Return empty session on invalid cookie (like Remix)
		return &Session{
			data:   make(map[string]interface{}),
			id:     generateSessionID(),
			dirty:  false,
			config: s.config,
		}, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(sessionData), &data); err != nil {
		// Return empty session on parse error
		return &Session{
			data:   make(map[string]interface{}),
			id:     generateSessionID(),
			dirty:  false,
			config: s.config,
		}, nil
	}

	return &Session{
		data:   data,
		id:     fmt.Sprintf("%v", data["__id"]),
		dirty:  false,
		config: s.config,
	}, nil
}

// CommitSession serializes session and returns Set-Cookie header value
func (s *AlignedService) CommitSession(session *Session) (string, error) {
	if !session.dirty {
		return "", nil // No changes, no cookie needed
	}

	// Add session ID to data
	session.data["__id"] = session.id
	session.data["__expires"] = time.Now().Add(s.config.MaxAge).Unix()

	// Serialize session data
	jsonData, err := json.Marshal(session.data)
	if err != nil {
		return "", err
	}

	// Sign the data
	signedData, err := s.signCookie(string(jsonData))
	if err != nil {
		return "", err
	}

	// Create cookie
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    signedData,
		Path:     s.config.Path,
		MaxAge:   int(s.config.MaxAge.Seconds()),
		HttpOnly: s.config.HTTPOnly,
		Secure:   s.config.Secure,
		SameSite: s.config.SameSite,
	}

	return cookie.String(), nil
}

// Session methods that mirror Remix session API

// Set stores a value in the session
func (sess *Session) Set(key string, value interface{}) {
	sess.data[key] = value
	sess.dirty = true
}

// Get retrieves a value from the session
func (sess *Session) Get(key string) interface{} {
	return sess.data[key]
}

// Unset removes a value from the session
func (sess *Session) Unset(key string) {
	delete(sess.data, key)
	sess.dirty = true
}

// Has checks if a key exists in the session
func (sess *Session) Has(key string) bool {
	_, exists := sess.data[key]
	return exists
}

// trigger.dev compatible API functions

// GetRedirectSession mirrors trigger.dev's getRedirectSession function
func (s *AlignedService) GetRedirectSession(r *http.Request) (*Session, error) {
	return s.GetSession(r)
}

// SetRedirectTo mirrors trigger.dev's setRedirectTo function exactly
func (s *AlignedService) SetRedirectTo(r *http.Request, redirectTo string) (*Session, error) {
	session, err := s.GetRedirectSession(r)
	if err != nil {
		return nil, err
	}

	if session != nil {
		session.Set("redirectTo", redirectTo)
	}

	return session, nil
}

// ClearRedirectTo mirrors trigger.dev's clearRedirectTo function exactly
func (s *AlignedService) ClearRedirectTo(r *http.Request) (*Session, error) {
	session, err := s.GetRedirectSession(r)
	if err != nil {
		return nil, err
	}

	if session != nil {
		session.Unset("redirectTo")
	}

	return session, nil
}

// GetRedirectTo mirrors trigger.dev's getRedirectTo function exactly
func (s *AlignedService) GetRedirectTo(r *http.Request) (*string, error) {
	session, err := s.GetRedirectSession(r)
	if err != nil {
		return nil, err
	}

	if session != nil {
		value := session.Get("redirectTo")
		if value != nil {
			if str, ok := value.(string); ok && str != "" {
				return &str, nil
			}
		}
	}

	return nil, nil // Return nil for undefined, matching trigger.dev behavior
}

// Internal helper functions for cookie signing (similar to Remix's approach)

func (s *AlignedService) signCookie(value string) (string, error) {
	if len(s.config.SecretKey) == 0 {
		return "", errors.New("secret key required for signing")
	}

	// Create HMAC signature
	h := hmac.New(sha256.New, s.config.SecretKey)
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Return value.signature format (similar to Remix)
	return base64.URLEncoding.EncodeToString([]byte(value)) + "." + signature, nil
}

func (s *AlignedService) unsignCookie(signedValue string) (string, error) {
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid signed cookie format")
	}

	// Decode value
	valueBytes, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	value := string(valueBytes)

	// Verify signature
	expectedSigned, err := s.signCookie(value)
	if err != nil {
		return "", err
	}

	if !hmac.Equal([]byte(signedValue), []byte(expectedSigned)) {
		return "", errors.New("invalid signature")
	}

	return value, nil
}

func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}
