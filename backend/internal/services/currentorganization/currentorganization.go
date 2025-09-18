// Package currentorganization provides cookie-based current organization session management
// that strictly aligns with trigger.dev's currentOrganization implementation.
//
// This package replicates the exact behavior of trigger.dev's organization session:
// - Same cookie configuration (__organization, 24h expiration, security settings)
// - Compatible API methods (GetCurrentOrg, SetCurrentOrg, ClearCurrentOrg)
// - Identical organization slug handling (string storage, undefined handling)
package currentorganization

import (
	"encoding/gob"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/sessions"
)

var (
	store *sessions.CookieStore
	once  sync.Once
)

const (
	// Cookie name matches trigger.dev exactly
	cookieName = "__organization"

	// Session key for organization slug storage
	orgSlugKey = "currentOrg"
)

// Initialize current organization session store with trigger.dev compatible configuration
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

		// Configure exactly like trigger.dev's currentOrgSessionStorage
		store.Options = &sessions.Options{
			Path:     "/",                                   // cookie.path: "/"
			MaxAge:   int((24 * 60 * 60)),                   // cookie.maxAge: 60 * 60 * 24 (1 day)
			HttpOnly: true,                                  // cookie.httpOnly: true
			Secure:   os.Getenv("NODE_ENV") == "production", // cookie.secure: env.NODE_ENV === "production"
			SameSite: http.SameSiteLaxMode,                  // cookie.sameSite: "lax"
		}
	})
}

// GetCurrentOrg retrieves the current organization slug from the session.
// Returns nil if no organization is set, matching trigger.dev's undefined behavior.
//
// Equivalent to trigger.dev's:
//
//	const session = await getCurrentOrgSession(request);
//	return session.get("currentOrg");
func GetCurrentOrg(r *http.Request) (*string, error) {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return nil, err
	}

	orgSlug, exists := session.Values[orgSlugKey]
	if !exists {
		return nil, nil // Return nil like trigger.dev's undefined
	}

	// Handle string organization slug
	if slug, ok := orgSlug.(string); ok {
		if slug == "" {
			return nil, nil // Empty string treated as not set
		}
		return &slug, nil
	}

	// Invalid format, treat as not set
	return nil, nil
}

// SetCurrentOrg sets the current organization slug in the session.
// Automatically commits the session to the response.
//
// Equivalent to trigger.dev's:
//
//	const session = await getCurrentOrgSession(request);
//	session.set("currentOrg", slug);
//	return session;
func SetCurrentOrg(w http.ResponseWriter, r *http.Request, slug string) error {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return err
	}

	// Store organization slug
	session.Values[orgSlugKey] = slug

	// Automatically commit session (simplified API)
	return session.Save(r, w)
}

// ClearCurrentOrg removes the current organization from the session.
// Automatically commits the session to the response.
//
// Equivalent to trigger.dev's:
//
//	const session = await getCurrentOrgSession(request);
//	session.unset("currentOrg");
//	return session;
func ClearCurrentOrg(w http.ResponseWriter, r *http.Request) error {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return err
	}

	// Remove the organization slug
	delete(session.Values, orgSlugKey)

	// Automatically commit session (simplified API)
	return session.Save(r, w)
}

// GetSession retrieves the raw organization session (advanced API).
// This provides lower-level access for custom operations.
//
// Equivalent to trigger.dev's getCurrentOrgSession(request)
func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, cookieName)
}

// CommitSession commits the session to the response (advanced API).
// Use this with GetSession for manual session management.
//
// Equivalent to trigger.dev's commitCurrentOrgSession(session)
func CommitSession(w http.ResponseWriter, r *http.Request, session *sessions.Session) error {
	return session.Save(r, w)
}
