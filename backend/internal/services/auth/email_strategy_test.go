package auth

import (
	"context"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"kongflow/backend/internal/services/auth/testutil"
	"kongflow/backend/internal/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEmailStrategy_Name(t *testing.T) {
	strategy := &EmailStrategy{}
	assert.Equal(t, "email", strategy.Name())
}

func TestNewEmailStrategy_WithValidSecret(t *testing.T) {
	// Set required environment variable
	originalSecret := os.Getenv("MAGIC_LINK_SECRET")
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer func() {
		if originalSecret == "" {
			os.Unsetenv("MAGIC_LINK_SECRET")
		} else {
			os.Setenv("MAGIC_LINK_SECRET", originalSecret)
		}
	}()

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	assert.NotNil(t, strategy)
	assert.Equal(t, "email", strategy.Name())
	assert.Equal(t, "/magic", strategy.callbackURL)
	assert.Equal(t, mockEmailService, strategy.emailService)
	assert.NotNil(t, strategy.logger)
	assert.NotNil(t, strategy.ulid)
}

func TestNewEmailStrategy_MissingSecret(t *testing.T) {
	// Ensure MAGIC_LINK_SECRET is not set
	originalSecret := os.Getenv("MAGIC_LINK_SECRET")
	os.Unsetenv("MAGIC_LINK_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("MAGIC_LINK_SECRET", originalSecret)
		}
	}()

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	// Should panic when MAGIC_LINK_SECRET is not set
	assert.Panics(t, func() {
		NewEmailStrategy(mockEmailService, queries, postAuthProcessor)
	})
}

func TestEmailStrategy_Authenticate_MissingEmail(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Create test request without email
	req := httptest.NewRequest("POST", "/auth/email", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute
	ctx := context.Background()
	result, err := strategy.Authenticate(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
	assert.Nil(t, result)
}

func TestEmailStrategy_Authenticate_Success(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Create test request
	form := url.Values{}
	form.Add("email", "test@example.com")

	req := httptest.NewRequest("POST", "/auth/email", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set up mock expectations
	mockEmailService.On("SendMagicLinkEmail", mock.Anything, mock.Anything).Return(nil)

	// Execute
	ctx := context.Background()
	result, err := strategy.Authenticate(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.Nil(t, result) // Should return nil for email strategy (email sent, not authenticated yet)
	mockEmailService.AssertExpectations(t)
}

func TestEmailStrategy_Authenticate_EmailServiceError(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Create test request
	form := url.Values{}
	form.Add("email", "test@example.com")

	req := httptest.NewRequest("POST", "/auth/email", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set up mock expectations - email service fails
	mockEmailService.On("SendMagicLinkEmail", mock.Anything, mock.Anything).Return(assert.AnError)

	// Execute
	ctx := context.Background()
	result, err := strategy.Authenticate(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send magic link email")
	assert.Nil(t, result)
	mockEmailService.AssertExpectations(t)
}

func TestEmailStrategy_HandleCallback_MissingToken(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Create test request without token
	req := httptest.NewRequest("GET", "/magic", nil)

	// Execute
	ctx := context.Background()
	result, err := strategy.HandleCallback(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "magic link token is required")
	assert.Nil(t, result)
}

func TestEmailStrategy_HandleCallback_InvalidToken(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Create test request with invalid token
	req := httptest.NewRequest("GET", "/magic?token=invalid-token", nil)

	// Execute
	ctx := context.Background()
	result, err := strategy.HandleCallback(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid magic link token")
	assert.Nil(t, result)
}

func TestEmailStrategy_generateMagicLinkToken(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Generate tokens
	token1, err1 := strategy.generateMagicLinkToken("test@example.com")
	token2, err2 := strategy.generateMagicLinkToken("test@example.com")

	// Verify
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should generate different tokens each time
}

func TestEmailStrategy_verifyMagicLinkToken(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	testEmail := "verify@example.com"

	// Generate and verify token
	token, err := strategy.generateMagicLinkToken(testEmail)
	require.NoError(t, err)

	email, err := strategy.verifyMagicLinkToken(token)
	assert.NoError(t, err)
	assert.Equal(t, testEmail, email)
}

func TestEmailStrategy_verifyMagicLinkToken_InvalidEncoding(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Test with invalid base64 encoding
	invalidToken := "invalid-base64!"
	email, err := strategy.verifyMagicLinkToken(invalidToken)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token encoding")
	assert.Empty(t, email)
}

func TestEmailStrategy_getBaseURL(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	// Test HTTP request
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	baseURL := strategy.getBaseURL(req)
	assert.Equal(t, "http://example.com", baseURL)

	// Test with X-Forwarded-Host header
	req.Header.Set("X-Forwarded-Host", "forwarded.example.com")
	baseURL = strategy.getBaseURL(req)
	assert.Equal(t, "http://forwarded.example.com", baseURL)
}

func TestEmailStrategy_Integration_TokenLifecycle(t *testing.T) {
	// Setup
	os.Setenv("MAGIC_LINK_SECRET", "test-secret-for-testing")
	defer os.Unsetenv("MAGIC_LINK_SECRET")

	mockEmailService := &testutil.MockEmailService{}
	queries := (*shared.Queries)(nil)
	postAuthProcessor := createTestPostAuthProcessor()

	strategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	testEmail := "integration@example.com"

	// Step 1: Generate token (simulating Authenticate call)
	token, err := strategy.generateMagicLinkToken(testEmail)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Step 2: Verify token can be decoded (simulating HandleCallback call)
	decodedEmail, err := strategy.verifyMagicLinkToken(token)
	require.NoError(t, err)
	assert.Equal(t, testEmail, decodedEmail)

	// Step 3: Test token with wrong secret (simulate token tampering)
	os.Setenv("MAGIC_LINK_SECRET", "different-secret")
	invalidStrategy := NewEmailStrategy(mockEmailService, queries, postAuthProcessor)

	_, err = invalidStrategy.verifyMagicLinkToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token signature")
}
