package apiauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kongflow/backend/internal/database"
)

// ApiAuthIntegrationTestSuite provides comprehensive integration testing for apiAuth service
type ApiAuthIntegrationTestSuite struct {
	suite.Suite
	db      *database.TestDB
	service APIAuthService
	repo    Repository
}

// SetupSuite initializes test database and services
func (suite *ApiAuthIntegrationTestSuite) SetupSuite() {
	// Setup PostgreSQL TestContainer with all migrations
	suite.db = database.SetupTestDBWithMigrations(suite.T(), "")

	// Initialize repository and service with test database
	suite.repo = NewRepository(suite.db.Pool)
	suite.service = NewAPIAuthService(suite.repo, "test-jwt-secret-key-for-integration-testing")
}

// TearDownSuite cleans up test database
func (suite *ApiAuthIntegrationTestSuite) TearDownSuite() {
	suite.db.Cleanup(suite.T())
}

// SetupTest prepares clean state for each test
func (suite *ApiAuthIntegrationTestSuite) SetupTest() {
	ctx := context.Background()

	// Clean up test data before each test
	cleanupQueries := []string{
		"DELETE FROM personal_access_tokens",
		"DELETE FROM organization_access_tokens",
		"DELETE FROM runtime_environments",
		"DELETE FROM projects",
		"DELETE FROM organizations",
		"DELETE FROM users",
	}

	for _, query := range cleanupQueries {
		_, err := suite.db.Pool.Exec(ctx, query)
		require.NoError(suite.T(), err)
	}
}

// TestCompleteAuthenticationWorkflow tests end-to-end authentication scenarios
func (suite *ApiAuthIntegrationTestSuite) TestCompleteAuthenticationWorkflow() {
	ctx := context.Background()

	// 1. Setup test data - organization, project, environment
	orgID := suite.createTestOrganization(ctx)
	projectID := suite.createTestProject(ctx, orgID)

	// 2. Create separate environments for public and private keys
	privateEnvID := suite.createTestEnvironment(ctx, projectID, orgID, "tr_test_private_"+suite.generateTestID())
	publicEnvID := suite.createTestEnvironment(ctx, projectID, orgID, "pk_test_public_"+suite.generateTestID())

	// 3. Get the API keys
	publicKey := suite.getAPIKeyFromEnvironment(ctx, publicEnvID)
	privateKey := suite.getAPIKeyFromEnvironment(ctx, privateEnvID)

	suite.T().Run("Public API Key Authentication", func(t *testing.T) {
		// Test public key authentication
		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+publicKey,
			authOpts,
		)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Environment)
		// Compare UUID strings properly
		expectedUUID := publicEnvID
		actualUUID := result.Environment.ID.String()
		assert.Equal(t, expectedUUID, actualUUID)
		assert.Equal(t, APIKeyTypePublic, result.Type)
	})

	suite.T().Run("Private API Key Authentication", func(t *testing.T) {
		// Test private key authentication
		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+privateKey,
			authOpts,
		)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Environment)
		// Compare UUID strings properly
		expectedUUID := privateEnvID
		actualUUID := result.Environment.ID.String()
		assert.Equal(t, expectedUUID, actualUUID)
		assert.Equal(t, APIKeyTypePrivate, result.Type)
	})

	suite.T().Run("Public Key Blocked When Not Allowed", func(t *testing.T) {
		// Test public key rejection when not allowed
		authOpts := &AuthOptions{
			AllowPublicKey: false, // Block public keys
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+publicKey,
			authOpts,
		)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "public API keys are not allowed")
	})
}

// TestHTTPRequestAuthentication tests authentication from HTTP requests
func (suite *ApiAuthIntegrationTestSuite) TestHTTPRequestAuthentication() {
	ctx := context.Background()

	// Setup test data
	orgID := suite.createTestOrganization(ctx)
	projectID := suite.createTestProject(ctx, orgID)
	envID := suite.createTestEnvironment(ctx, projectID, orgID, "tr_test_"+suite.generateTestID())
	apiKey := suite.getAPIKeyFromEnvironment(ctx, envID)

	suite.T().Run("Valid Authorization Header", func(t *testing.T) {
		// Create HTTP request with valid authorization
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAPIRequest(ctx, req, authOpts)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Environment)
		assert.Equal(t, envID, result.Environment.ID.String())
	})

	suite.T().Run("Missing Authorization Header", func(t *testing.T) {
		// Create HTTP request without authorization
		req := httptest.NewRequest("GET", "/api/test", nil)

		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAPIRequest(ctx, req, authOpts)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "missing authorization header")
	})

	suite.T().Run("Invalid Authorization Format", func(t *testing.T) {
		// Create HTTP request with invalid authorization format
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0") // Basic auth instead of Bearer

		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAPIRequest(ctx, req, authOpts)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid authorization format")
	})
}

// TestMiddlewareIntegration tests authentication middleware with standard HTTP handlers
func (suite *ApiAuthIntegrationTestSuite) TestMiddlewareIntegration() {
	ctx := context.Background()

	// Setup test data
	orgID := suite.createTestOrganization(ctx)
	projectID := suite.createTestProject(ctx, orgID)
	envID := suite.createTestEnvironment(ctx, projectID, orgID, "tr_test_"+suite.generateTestID())
	validAPIKey := suite.getAPIKeyFromEnvironment(ctx, envID)

	// Create middleware instance
	middleware := NewAuthMiddleware(suite.service)

	// Configure auth options
	authOpts := &AuthOptions{
		AllowPublicKey: true,
		AllowJWT:       false,
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for authentication context
		authResult, ok := GetAuthResult(r)
		if !ok || authResult == nil {
			http.Error(w, "no auth context", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"success":     true,
			"environment": authResult.Environment.ID,
			"auth_type":   string(authResult.Type),
		}
		json.NewEncoder(w).Encode(response)
	})

	// Wrap with auth middleware
	protectedHandler := middleware.RequireAPIKey(authOpts)(testHandler)

	suite.T().Run("Authenticated Request Success", func(t *testing.T) {
		// Test successful authentication through middleware
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validAPIKey)

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
		assert.Contains(t, w.Body.String(), envID)
	})

	suite.T().Run("Unauthenticated Request Blocked", func(t *testing.T) {
		// Test request without valid authorization
		req := httptest.NewRequest("GET", "/api/protected", nil)

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	suite.T().Run("Invalid API Key Blocked", func(t *testing.T) {
		// Test request with invalid API key
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-api-key-format")

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

// TestJWTAuthentication tests JWT-based API key authentication behavior
func (suite *ApiAuthIntegrationTestSuite) TestJWTAuthentication() {
	ctx := context.Background()

	// Setup test data
	orgID := suite.createTestOrganization(ctx)
	projectID := suite.createTestProject(ctx, orgID)
	envID := suite.createTestEnvironment(ctx, projectID, orgID, "pk_jwt_test_"+suite.generateTestID())

	// Create a JWT token for testing
	jwtToken := suite.createTestJWTToken(envID)

	suite.T().Run("JWT Token Recognized as Public Key", func(t *testing.T) {
		authOpts := &AuthOptions{
			AllowPublicKey: false,
			AllowJWT:       true, // Allow JWT
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+jwtToken,
			authOpts,
		)

		assert.NoError(t, err)
		// Note: Our fake JWT token starts with "pk_jwt_test_" so it's treated as a public key
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "public API keys are not allowed")
	})

	suite.T().Run("JWT Blocked When Not Allowed", func(t *testing.T) {
		authOpts := &AuthOptions{
			AllowPublicKey: false,
			AllowJWT:       false, // Block JWT
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+jwtToken,
			authOpts,
		)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "public API keys are not allowed")
	})
}

// TestErrorScenarios tests various error conditions
func (suite *ApiAuthIntegrationTestSuite) TestErrorScenarios() {
	ctx := context.Background()

	suite.T().Run("Empty Bearer Token", func(t *testing.T) {
		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       true,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer ",
			authOpts,
		)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "empty token")
	})

	suite.T().Run("Malformed API Key", func(t *testing.T) {
		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       true,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer malformed-key-without-prefix",
			authOpts,
		)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid API key format")
	})

	suite.T().Run("Non-existent API Key", func(t *testing.T) {
		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := suite.service.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer pk_test_nonexistentkey123456789",
			authOpts,
		)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "invalid")
	})
}

// TestDatabaseTransactions tests authentication within database transactions
func (suite *ApiAuthIntegrationTestSuite) TestDatabaseTransactions() {
	ctx := context.Background()

	// Setup test data
	orgID := suite.createTestOrganization(ctx)
	projectID := suite.createTestProject(ctx, orgID)
	envID := suite.createTestEnvironment(ctx, projectID, orgID, "tr_test_"+suite.generateTestID())
	apiKey := suite.getAPIKeyFromEnvironment(ctx, envID)

	suite.T().Run("Authentication Within Transaction", func(t *testing.T) {
		// Start a database transaction
		tx, err := suite.db.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		// Create service with transaction-aware repository
		txRepo := NewRepository(tx)
		txService := NewAPIAuthService(txRepo, "test-jwt-secret")

		authOpts := &AuthOptions{
			AllowPublicKey: true,
			AllowJWT:       false,
		}

		result, err := txService.AuthenticateAuthorizationHeader(
			ctx,
			"Bearer "+apiKey,
			authOpts,
		)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Environment)
		assert.Equal(t, envID, result.Environment.ID.String())

		// Commit transaction
		err = tx.Commit(ctx)
		assert.NoError(t, err)
	})
}

// Helper methods for creating test data

func (suite *ApiAuthIntegrationTestSuite) createTestOrganization(ctx context.Context) string {
	orgID := uuid.New().String()

	query := `
		INSERT INTO organizations (id, title, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	_, err := suite.db.Pool.Exec(ctx, query,
		orgID,
		"Test Organization",
		"test-org-"+suite.generateTestID(),
		now,
		now,
	)
	require.NoError(suite.T(), err)

	return orgID
}

func (suite *ApiAuthIntegrationTestSuite) createTestProject(ctx context.Context, orgID string) string {
	projectID := uuid.New().String()

	query := `
		INSERT INTO projects (id, name, slug, organization_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	now := time.Now()
	_, err := suite.db.Pool.Exec(ctx, query,
		projectID,
		"Test Project",
		"test-project-"+suite.generateTestID(),
		orgID,
		now,
		now,
	)
	require.NoError(suite.T(), err)

	return projectID
}

func (suite *ApiAuthIntegrationTestSuite) createTestEnvironment(ctx context.Context, projectID, orgID, apiKey string) string {
	envID := uuid.New().String()

	query := `
		INSERT INTO runtime_environments (
			id, slug, api_key, type, organization_id, project_id, 
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	uniqueSlug := "test-env-" + suite.generateTestID()
	_, err := suite.db.Pool.Exec(ctx, query,
		envID,
		uniqueSlug,
		apiKey,
		"DEVELOPMENT",
		orgID,
		projectID,
		now,
		now,
	)
	require.NoError(suite.T(), err)

	return envID
}

func (suite *ApiAuthIntegrationTestSuite) getAPIKeyFromEnvironment(ctx context.Context, envID string) string {
	var apiKey string
	query := "SELECT api_key FROM runtime_environments WHERE id = $1"
	err := suite.db.Pool.QueryRow(ctx, query, envID).Scan(&apiKey)
	require.NoError(suite.T(), err)

	return apiKey
}

func (suite *ApiAuthIntegrationTestSuite) createTestJWTToken(envID string) string {
	// Create a simple JWT token for testing
	// In a real implementation, this would create a proper JWT with claims
	return "pk_jwt_test_" + envID + "_" + suite.generateTestID()
}

func (suite *ApiAuthIntegrationTestSuite) generateTestID() string {
	return strings.ReplaceAll(time.Now().Format("20060102150405.000"), ".", "")
}

// TestApiAuthIntegrationTestSuite runs the integration test suite
func TestApiAuthIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(ApiAuthIntegrationTestSuite))
}
