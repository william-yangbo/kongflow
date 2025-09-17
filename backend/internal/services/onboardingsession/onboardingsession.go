// Package onboardingsession provides cookie-based onboarding session management
// that strictly aligns with trigger.dev's onboardingSession implementation.
//
// This package replicates the exact behavior of trigger.dev's onboarding session:
// - Same cookie configuration (__onboarding, 24h expiration, security settings)
// - Compatible API methods (GetWorkflowDate, SetWorkflowDate, ClearWorkflowDate)
// - Identical workflow date handling (ISO string format, undefined handling)
package onboardingsession

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

const (
	// Cookie name matches trigger.dev exactly
	cookieName = "__onboarding"

	// Session key for workflow date storage
	workflowDateKey = "workflowDate"
)

// Initialize onboarding session store with trigger.dev compatible configuration
func init() {
	// Register types for gob encoding (required for gorilla/sessions)
	gob.Register(map[string]interface{}{})
	gob.Register(time.Time{})

	once.Do(func() {
		secret := os.Getenv("SESSION_SECRET")
		if secret == "" {
			panic("SESSION_SECRET environment variable is required")
		}

		// Create store with secret (same as trigger.dev's secrets[0])
		store = sessions.NewCookieStore([]byte(secret))

		// Configure exactly like trigger.dev's onboardingSessionStorage
		store.Options = &sessions.Options{
			Path:     "/",                                   // cookie.path: "/"
			MaxAge:   int((24 * time.Hour).Seconds()),       // cookie.maxAge: 60 * 60 * 24 (1 day)
			HttpOnly: true,                                  // cookie.httpOnly: true
			Secure:   os.Getenv("NODE_ENV") == "production", // cookie.secure: env.NODE_ENV === "production"
			SameSite: http.SameSiteLaxMode,                  // cookie.sameSite: "lax"
		}
	})
}

// GetWorkflowDate retrieves the workflow date from the onboarding session.
// Returns nil if no date is set, matching trigger.dev's undefined behavior.
//
// Equivalent to trigger.dev's:
//
//	const rawWorkflowDate = session.get("workflowDate");
//	if (rawWorkflowDate) { return new Date(rawWorkflowDate); }
func GetWorkflowDate(r *http.Request) (*time.Time, error) {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return nil, err
	}

	rawDate, exists := session.Values[workflowDateKey]
	if !exists {
		return nil, nil // Return nil like trigger.dev's undefined
	}

	// Handle both string (from trigger.dev format) and time.Time (from Go)
	switch v := rawDate.(type) {
	case string:
		// Parse ISO string format from trigger.dev
		date, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, err
		}
		return &date, nil
	case time.Time:
		// Direct time.Time value
		return &v, nil
	default:
		// Invalid format, treat as not set
		return nil, nil
	}
}

// SetWorkflowDate sets the workflow date in the onboarding session.
// Automatically commits the session to the response.
//
// Equivalent to trigger.dev's:
//
//	const session = await getOnboardingSession(request);
//	session.set("workflowDate", date.toISOString());
//	return session;
func SetWorkflowDate(w http.ResponseWriter, r *http.Request, date time.Time) error {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return err
	}

	// Store as ISO string to match trigger.dev format
	session.Values[workflowDateKey] = date.Format(time.RFC3339)

	// Automatically commit session (simplified API)
	return session.Save(r, w)
}

// ClearWorkflowDate removes the workflow date from the onboarding session.
// Automatically commits the session to the response.
//
// Equivalent to trigger.dev's:
//
//	const session = await getOnboardingSession(request);
//	session.unset("workflowDate");
//	return session;
func ClearWorkflowDate(w http.ResponseWriter, r *http.Request) error {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return err
	}

	// Remove the workflow date
	delete(session.Values, workflowDateKey)

	// Automatically commit session (simplified API)
	return session.Save(r, w)
}

// GetSession retrieves the raw onboarding session (advanced API).
// This provides lower-level access for custom operations.
func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, cookieName)
}

// CommitSession commits the session to the response (advanced API).
// Use this with GetSession for manual session management.
func CommitSession(w http.ResponseWriter, r *http.Request, session *sessions.Session) error {
	return session.Save(r, w)
}
