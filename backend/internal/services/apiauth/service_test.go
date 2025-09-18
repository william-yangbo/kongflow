package apiauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindEnvironmentByAPIKey(ctx context.Context, apiKey string) (*RuntimeEnvironment, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RuntimeEnvironment), args.Error(1)
}

func (m *MockRepository) FindEnvironmentByPublicAPIKey(ctx context.Context, apiKey string, branch *string) (*RuntimeEnvironment, error) {
	args := m.Called(ctx, apiKey, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RuntimeEnvironment), args.Error(1)
}

func (m *MockRepository) GetEnvironmentWithProjectAndOrg(ctx context.Context, envID string) (*AuthenticatedEnvironment, error) {
	args := m.Called(ctx, envID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AuthenticatedEnvironment), args.Error(1)
}

func (m *MockRepository) AuthenticatePersonalAccessToken(ctx context.Context, token string) (*PersonalAccessTokenResult, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PersonalAccessTokenResult), args.Error(1)
}

func (m *MockRepository) AuthenticateOrganizationAccessToken(ctx context.Context, token string) (*OrganizationAccessTokenResult, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OrganizationAccessTokenResult), args.Error(1)
}

// Test helpers
func createTestEnvironment() *RuntimeEnvironment {
	uuid := pgtype.UUID{}
	uuid.Scan("550e8400-e29b-41d4-a716-446655440000")

	return &RuntimeEnvironment{
		ID:             uuid,
		Slug:           "test-env",
		APIKey:         "tr_test_key_123",
		Type:           EnvironmentTypeDevelopment,
		OrganizationID: uuid,
		ProjectID:      uuid,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// Test API Key Type Detection - 80/20 critical functionality
func TestGetAPIKeyType(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected APIKeyType
	}{
		{"Public Key", "pk_test_123", APIKeyTypePublic},
		{"Private Key", "tr_test_123", APIKeyTypePrivate},
		{"JWT Token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", APIKeyTypePublicJWT},
		{"Invalid Key", "invalid_key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAPIKeyType(tt.apiKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test API Key Format Validation
func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		shouldErr bool
	}{
		{"Valid Public Key", "pk_test_123456", false},
		{"Valid Private Key", "tr_test_123456", false},
		{"Too Short", "pk_123", true},
		{"Invalid Format", "invalid_key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKeyFormat(tt.apiKey)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test Private Key Authentication - Critical path
func TestAuthenticatePrivateKey(t *testing.T) {
	mockRepo := &MockRepository{}
	service := &service{
		repo:      mockRepo,
		jwtSecret: []byte("test-secret"),
	}

	ctx := context.Background()
	apiKey := "tr_test_private_key_123"
	testEnv := createTestEnvironment()

	mockRepo.On("FindEnvironmentByAPIKey", ctx, apiKey).Return(testEnv, nil)

	result, err := service.authenticatePrivateKey(ctx, apiKey, "")

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, APIKeyTypePrivate, result.Type)
	assert.Equal(t, apiKey, result.APIKey)
	assert.Equal(t, testEnv, result.Environment)

	mockRepo.AssertExpectations(t)
}

// Test Public Key Authentication - Critical path
func TestAuthenticatePublicKey(t *testing.T) {
	mockRepo := &MockRepository{}
	service := &service{
		repo:      mockRepo,
		jwtSecret: []byte("test-secret"),
	}

	ctx := context.Background()
	apiKey := "pk_test_public_key_123"
	testEnv := createTestEnvironment()
	branchName := "main"

	mockRepo.On("FindEnvironmentByPublicAPIKey", ctx, apiKey, &branchName).Return(testEnv, nil)

	result, err := service.authenticatePublicKey(ctx, apiKey, branchName)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, APIKeyTypePublic, result.Type)
	assert.Equal(t, apiKey, result.APIKey)
	assert.Equal(t, testEnv, result.Environment)

	mockRepo.AssertExpectations(t)
}

// Test Authorization Header Parsing - Critical path
func TestAuthenticateAuthorizationHeader(t *testing.T) {
	mockRepo := &MockRepository{}
	service := &service{
		repo:      mockRepo,
		jwtSecret: []byte("test-secret"),
	}

	ctx := context.Background()
	apiKey := "tr_test_key_123"
	testEnv := createTestEnvironment()
	authHeader := "Bearer " + apiKey

	mockRepo.On("FindEnvironmentByAPIKey", ctx, apiKey).Return(testEnv, nil)

	opts := &AuthOptions{
		AllowPublicKey: true,
		AllowJWT:       true,
	}

	result, err := service.AuthenticateAuthorizationHeader(ctx, authHeader, opts)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, testEnv, result.Environment)

	mockRepo.AssertExpectations(t)
}

// Test Invalid Authorization Header
func TestAuthenticateInvalidAuthorizationHeader(t *testing.T) {
	service := &service{
		repo:      &MockRepository{},
		jwtSecret: []byte("test-secret"),
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		authHeader string
		expectErr  string
	}{
		{"Missing Bearer", "InvalidHeader", "invalid authorization format"},
		{"Empty Token", "Bearer ", "empty token in authorization header"},
	}

	opts := &AuthOptions{
		AllowPublicKey: true,
		AllowJWT:       true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.AuthenticateAuthorizationHeader(ctx, tt.authHeader, opts)
			assert.NoError(t, err)
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, tt.expectErr)
		})
	}
}

// Test JWT Generation - Critical functionality
func TestGenerateJWTToken(t *testing.T) {
	service := &service{
		repo:      &MockRepository{},
		jwtSecret: []byte("test-secret-key"),
	}

	ctx := context.Background()
	testEnv := createTestEnvironment()

	payload := map[string]interface{}{
		"scopes":   []string{"read", "write"},
		"realtime": true,
	}

	opts := &JWTOptions{
		ExpirationTime: time.Hour,
		CustomClaims: map[string]interface{}{
			"otu": true,
		},
	}

	token, err := service.GenerateJWTToken(ctx, testEnv, payload, opts)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify it's a valid JWT format
	assert.Equal(t, 3, len(strings.Split(token, ".")))
}

// Test Secure Token Generation
func TestGenerateSecureToken(t *testing.T) {
	token1 := generateSecureToken()
	token2 := generateSecureToken()

	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should be unique
	assert.Equal(t, 32, len(token1))   // UUID without dashes
}

// Test Configuration Validation - Essential for security
func TestAuthOptionsValidation(t *testing.T) {
	mockRepo := &MockRepository{}
	service := &service{
		repo:      mockRepo,
		jwtSecret: []byte("test-secret"),
	}

	ctx := context.Background()

	// Test public key rejection when not allowed
	opts := &AuthOptions{
		AllowPublicKey: false,
		AllowJWT:       true,
	}

	result, err := service.authenticateAPIKey(ctx, "pk_test_123456", opts)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "public API keys are not allowed")

	// Test JWT rejection when not allowed
	opts = &AuthOptions{
		AllowPublicKey: true,
		AllowJWT:       false,
	}

	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	result, err = service.authenticateAPIKey(ctx, jwtToken, opts)
	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "public JWT API keys are not allowed")
}
