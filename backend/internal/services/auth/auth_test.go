package auth

import (
	"os"
	"testing"

	"kongflow/backend/internal/services/logger"
)

// init sets up test environment
func init() {
	// Set required environment variables for testing
	os.Setenv("SESSION_SECRET", "test-session-secret-for-testing")
}

// createTestPostAuthProcessor creates a test PostAuthProcessor
func createTestPostAuthProcessor() *PostAuthProcessor {
	// Create a nil analytics service for testing
	testLogger := logger.NewWebapp("test.auth")
	return NewPostAuthProcessor(nil, testLogger, nil)
} // TestAuthenticatorCreation tests that authenticator can be created successfully
func TestAuthenticatorCreation(t *testing.T) {
	authenticator := NewAuthenticator(NewCookieSessionStorage(), createTestPostAuthProcessor())

	if authenticator == nil {
		t.Error("Expected authenticator to be created")
	}

	if authenticator.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if authenticator.ulid == nil {
		t.Error("Expected ULID service to be initialized")
	}

	if authenticator.strategies == nil {
		t.Error("Expected strategies map to be initialized")
	}
}

// TestULIDService tests ULID generation
func TestULIDService(t *testing.T) {
	authenticator := NewAuthenticator(NewCookieSessionStorage(), createTestPostAuthProcessor())

	// Generate two ULIDs
	id1 := authenticator.ulid.Generate()
	id2 := authenticator.ulid.Generate()

	// They should be different
	if id1 == id2 {
		t.Errorf("Expected different ULIDs, got: %s and %s", id1, id2)
	}

	// They should be lowercase (matching trigger.dev behavior)
	if id1 != toLower(id1) || id2 != toLower(id2) {
		t.Errorf("Expected lowercase ULIDs, got: %s and %s", id1, id2)
	}
}

// Helper function to check lowercase
func toLower(s string) string {
	result := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else {
			result += string(r)
		}
	}
	return result
}

// TestAnalyticsServiceIntegration tests analytics integration
func TestAnalyticsServiceIntegration(t *testing.T) {
	authenticator := NewAuthenticator(NewCookieSessionStorage(), createTestPostAuthProcessor())

	// Initially analytics should be nil
	if authenticator.analytics != nil {
		t.Error("Expected analytics to be nil initially")
	}

	// Set analytics service (nil is valid for testing)
	authenticator.SetAnalytics(nil)

	// Should not panic
	if authenticator.analytics != nil {
		t.Error("Expected analytics to remain nil")
	}
}
