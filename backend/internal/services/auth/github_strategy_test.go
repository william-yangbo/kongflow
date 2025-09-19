package auth

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"kongflow/backend/internal/services/auth/testutil"
)

// TestGitHubStrategy_Integration covers the main GitHub authentication flow
func TestGitHubStrategy_Integration(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("SESSION_SECRET", "test-session-secret")
	defer os.Unsetenv("SESSION_SECRET")

	// Create test dependencies
	mockQueries := &testutil.MockQueries{}
	mockAnalytics := &testutil.MockAnalyticsService{}

	strategy := NewGitHubStrategy(
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/auth/github/callback",
		mockQueries,
		mockAnalytics,
	)

	// Test 1: Strategy creation and basic properties
	assert.NotNil(t, strategy)
	assert.Equal(t, "github", strategy.Name())

	// Test 2: Authentication request initiation (should return redirect error)
	req := httptest.NewRequest("GET", "/auth/github", nil)

	authUser, err := strategy.Authenticate(context.Background(), req)

	// Should redirect and not return authUser yet
	assert.Nil(t, authUser)

	// Should return OAuthRedirectError
	var redirectErr *OAuthRedirectError
	assert.ErrorAs(t, err, &redirectErr)
	assert.NotEmpty(t, redirectErr.RedirectURL)
	assert.Contains(t, redirectErr.RedirectURL, "github.com/login/oauth/authorize")
	assert.Contains(t, redirectErr.RedirectURL, "client_id=test-client-id")
}

// TestGitHubAuthSupported_Environment tests environment variable detection
func TestGitHubAuthSupported_Environment(t *testing.T) {
	// Save original values
	originalClientID := os.Getenv("AUTH_GITHUB_CLIENT_ID")
	originalClientSecret := os.Getenv("AUTH_GITHUB_CLIENT_SECRET")
	defer func() {
		os.Setenv("AUTH_GITHUB_CLIENT_ID", originalClientID)
		os.Setenv("AUTH_GITHUB_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name              string
		clientID          string
		clientSecret      string
		expectedSupported bool
	}{
		{
			name:              "Both env vars set",
			clientID:          "test-client-id",
			clientSecret:      "test-client-secret",
			expectedSupported: true,
		},
		{
			name:              "Missing client ID",
			clientID:          "",
			clientSecret:      "test-client-secret",
			expectedSupported: false,
		},
		{
			name:              "Missing client secret",
			clientID:          "test-client-id",
			clientSecret:      "",
			expectedSupported: false,
		},
		{
			name:              "Both missing",
			clientID:          "",
			clientSecret:      "",
			expectedSupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AUTH_GITHUB_CLIENT_ID", tt.clientID)
			os.Setenv("AUTH_GITHUB_CLIENT_SECRET", tt.clientSecret)

			supported := IsGitHubAuthSupported()
			assert.Equal(t, tt.expectedSupported, supported)
		})
	}
}

// TestGitHubStrategy_AddToAuthenticator tests adding GitHub strategy to authenticator
func TestGitHubStrategy_AddToAuthenticator(t *testing.T) {
	// Set required environment variables
	os.Setenv("AUTH_GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("AUTH_GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("SESSION_SECRET", "test-session-secret")
	defer func() {
		os.Unsetenv("AUTH_GITHUB_CLIENT_ID")
		os.Unsetenv("AUTH_GITHUB_CLIENT_SECRET")
		os.Unsetenv("SESSION_SECRET")
	}()

	// Create test authenticator
	mockSessionStorage := testutil.NewMockSessionStorage()
	postAuthProcessor := createTestPostAuthProcessor()
	authenticator := NewAuthenticator(mockSessionStorage, postAuthProcessor)

	// Create test dependencies
	mockQueries := &testutil.MockQueries{}
	mockAnalytics := &testutil.MockAnalyticsService{}

	// Add GitHub strategy
	AddGitHubStrategy(authenticator, "test-client-id", "test-client-secret", mockQueries, mockAnalytics)

	// Verify strategy was added by trying to access it
	assert.NotNil(t, authenticator.strategies["github"])
	assert.Equal(t, "github", authenticator.strategies["github"].Name())
}
