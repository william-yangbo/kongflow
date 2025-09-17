// Package sessionstorage provides cookie-based session management
// that strictly aligns with trigger.dev's sessionStorage implementation.
//
// This package replicates the exact behavior of trigger.dev's cookie session storage:
// - Same cookie configuration (name, security flags, expiration)
// - Compatible API methods (GetUserSession, CommitSession, DestroySession)
// - Identical security settings (HttpOnly, Secure, SameSite)
package sessionstorage

import (
	"encoding/gob"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/sessions"
)

var (
	store *sessions.CookieStore
	once  sync.Once
)

// Initialize session store with trigger.dev compatible configuration
func init() {
	// Register types for gob encoding (required for gorilla/sessions)
	gob.Register(map[string]interface{}{})

	once.Do(func() {
		secret := os.Getenv("SESSION_SECRET")
		if secret == "" {
			panic("SESSION_SECRET environment variable is required")
		}

		// Create store with secret (same as trigger.dev's secrets[0])
		store = sessions.NewCookieStore([]byte(secret))

		// Configure exactly like trigger.dev's sessionStorage
		store.Options = &sessions.Options{
			Path:     "/",                                   // cookie.path: "/"
			MaxAge:   int((365 * 24 * time.Hour).Seconds()), // cookie.maxAge: 60 * 60 * 24 * 365 (1 year)
			HttpOnly: true,                                  // cookie.httpOnly: true
			Secure:   os.Getenv("NODE_ENV") == "production", // cookie.secure: env.NODE_ENV === "production"
			SameSite: http.SameSiteLaxMode,                  // cookie.sameSite: "lax"
		}
	})
}

// GetUserSession retrieves the user session from the request.
// This function directly aligns with trigger.dev's getUserSession(request).
//
// Example usage (matching trigger.dev pattern):
//
//	session, err := GetUserSession(r)
//	if err != nil {
//	    // handle error
//	}
//	// Use session.Values["key"] to get/set data
func GetUserSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, "__session") // Uses same cookie name as trigger.dev
}

// GetSession retrieves a named session from the request.
// This aligns with trigger.dev's sessionStorage.getSession() method.
func GetSession(r *http.Request, name string) (*sessions.Session, error) {
	return store.Get(r, name)
}

// CommitSession saves the session data to the response.
// This function aligns with trigger.dev's commitSession(session).
//
// Example usage (matching trigger.dev pattern):
//
//	session.Values["user_id"] = "12345"
//	err := CommitSession(r, w, session)
//	if err != nil {
//	    // handle error
//	}
func CommitSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	return session.Save(r, w)
}

// DestroySession removes the session cookie by setting MaxAge to -1.
// This function aligns with trigger.dev's destroySession().
//
// Example usage (matching trigger.dev pattern):
//
//	err := DestroySession(r, w)
//	if err != nil {
//	    // handle error
//	}
func DestroySession(r *http.Request, w http.ResponseWriter) error {
	session, err := GetUserSession(r)
	if err != nil {
		return err
	}

	// Set MaxAge to -1 to delete the cookie (same behavior as trigger.dev)
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// GetSessionStore returns the underlying session store for advanced usage.
// This is provided for cases where direct store access is needed.
func GetSessionStore() *sessions.CookieStore {
	return store
}
