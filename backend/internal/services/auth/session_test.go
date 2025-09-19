package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"kongflow/backend/internal/services/auth/testutil"
	"kongflow/backend/internal/services/impersonation"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockImpersonationService is a mock for impersonation functionality
type MockImpersonationService struct {
	mock.Mock
}

func (m *MockImpersonationService) GetImpersonation(req *http.Request) (string, error) {
	args := m.Called(req)
	return args.String(0), args.Error(1)
}

func (m *MockImpersonationService) SetImpersonation(w http.ResponseWriter, req *http.Request, userID string) error {
	args := m.Called(w, req, userID)
	return args.Error(0)
}

func (m *MockImpersonationService) ClearImpersonation(w http.ResponseWriter, req *http.Request) error {
	args := m.Called(w, req)
	return args.Error(0)
}

func (m *MockImpersonationService) IsImpersonating(req *http.Request) bool {
	args := m.Called(req)
	return args.Bool(0)
}

// createTestSessionService creates a session service for testing with all required mocks
func createTestSessionService() (SessionService, *testutil.MockSessionStorage, *testutil.MockQueries, *MockImpersonationService) {
	mockSessionStorage := testutil.NewMockSessionStorage()
	mockAuthenticator := NewAuthenticator(mockSessionStorage, createTestPostAuthProcessor())
	mockQueries := &testutil.MockQueries{}
	mockImpersonation := &MockImpersonationService{}

	// Create a test sessionService that uses the mockQueries
	sessionService := &testSessionServiceWithMocks{
		authenticator:        mockAuthenticator,
		mockQueries:          mockQueries,
		impersonationService: impersonation.ImpersonationService(mockImpersonation),
	}

	return sessionService, mockSessionStorage, mockQueries, mockImpersonation
}

// testSessionServiceWithMocks is a test version of sessionService that uses mocks
type testSessionServiceWithMocks struct {
	authenticator        *Authenticator
	mockQueries          *testutil.MockQueries
	impersonationService impersonation.ImpersonationService
}

func (s *testSessionServiceWithMocks) GetUserID(ctx context.Context, request *http.Request) (string, error) {
	// Check for impersonation first, exactly like trigger.dev
	impersonatedUserID, err := s.impersonationService.GetImpersonation(request)
	if err == nil && impersonatedUserID != "" {
		return impersonatedUserID, nil
	}

	// Get authenticated user from session
	authUser, err := s.authenticator.IsAuthenticated(ctx, request)
	if err != nil {
		return "", err
	}
	if authUser == nil {
		return "", nil // Not authenticated
	}

	return authUser.UserID, nil
}

func (s *testSessionServiceWithMocks) GetUser(ctx context.Context, request *http.Request) (*shared.Users, error) {
	userID, err := s.GetUserID(ctx, request)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, nil // Not authenticated
	}

	// Convert string userID to pgtype.UUID
	var userUUID pgtype.UUID
	err = userUUID.Scan(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Use mock GetUser
	user, err := s.mockQueries.GetUser(ctx, userUUID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (s *testSessionServiceWithMocks) RequireUserID(ctx context.Context, request *http.Request, redirectTo string) (string, error) {
	userID, err := s.GetUserID(ctx, request)
	if err != nil {
		return "", err
	}
	if userID == "" {
		// Redirect to login
		loginURL := "/login"
		if redirectTo != "" {
			loginURL += "?redirectTo=" + url.QueryEscape(redirectTo)
		}
		return "", &RedirectError{URL: loginURL}
	}
	return userID, nil
}

func (s *testSessionServiceWithMocks) RequireUser(ctx context.Context, request *http.Request) (*shared.Users, error) {
	user, err := s.GetUser(ctx, request)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Redirect to login
		return nil, &RedirectError{URL: "/login"}
	}
	return user, nil
}

func (s *testSessionServiceWithMocks) IsAuthenticated(ctx context.Context, request *http.Request) (bool, error) {
	userID, err := s.GetUserID(ctx, request)
	if err != nil {
		return false, err
	}
	return userID != "", nil
}

func (s *testSessionServiceWithMocks) GetAuthUser(ctx context.Context, request *http.Request) (*AuthUser, error) {
	return s.authenticator.IsAuthenticated(ctx, request)
}

func (s *testSessionServiceWithMocks) Logout(ctx context.Context, request *http.Request) error {
	// Return redirect error to logout endpoint
	return &RedirectError{URL: "/logout"}
}

func TestNewSessionServiceNew(t *testing.T) {
	sessionService, _, _, _ := createTestSessionService()
	assert.NotNil(t, sessionService)
}

func TestSessionService_GetUserID_NotAuthenticatedNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError)                      // No impersonation
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(map[string]interface{}{}, nil) // Empty session

	// Execute
	ctx := context.Background()
	userID, err := sessionService.GetUserID(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.Empty(t, userID)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_GetUserID_WithImpersonationNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	impersonatedUserID := "impersonated-user-123"

	// Set up mock expectations - impersonation found
	mockImpersonation.On("GetImpersonation", req).Return(impersonatedUserID, nil)

	// Execute
	ctx := context.Background()
	userID, err := sessionService.GetUserID(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, impersonatedUserID, userID)
	mockImpersonation.AssertExpectations(t)
	// Session storage should not be called when impersonation is found
	mockSessionStorage.AssertNotCalled(t, "GetSession")
}

func TestSessionService_GetUserID_WithAuthenticationNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	authenticatedUserID := "auth-user-123"

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError) // No impersonation
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": authenticatedUserID,
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)

	// Execute
	ctx := context.Background()
	userID, err := sessionService.GetUserID(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, authenticatedUserID, userID)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_GetUser_NotAuthenticatedNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError)                      // No impersonation
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(map[string]interface{}{}, nil) // Empty session

	// Execute
	ctx := context.Background()
	user, err := sessionService.GetUser(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.Nil(t, user)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_GetUser_WithAuthenticationNew(t *testing.T) {
	sessionService, mockSessionStorage, mockQueries, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	userUUID := testutil.TestUUID("user-123")
	expectedUser := shared.Users{
		ID:    userUUID,
		Email: "test@example.com",
		Name:  pgtype.Text{String: "Test User", Valid: true},
	}

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError) // No impersonation
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": userUUID.String(),
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)
	mockQueries.On("GetUser", mock.Anything, userUUID).Return(expectedUser, nil)

	// Execute
	ctx := context.Background()
	user, err := sessionService.GetUser(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
	mockQueries.AssertExpectations(t)
}

func TestSessionService_RequireUserID_SuccessNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	authenticatedUserID := "auth-user-123"

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError) // No impersonation
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": authenticatedUserID,
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)

	// Execute
	ctx := context.Background()
	userID, err := sessionService.RequireUserID(ctx, req, "")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, authenticatedUserID, userID)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_RequireUserID_NotAuthenticatedNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/protected", nil)

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError)                      // No impersonation
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(map[string]interface{}{}, nil) // Empty session

	// Execute
	ctx := context.Background()
	userID, err := sessionService.RequireUserID(ctx, req, "")

	// Verify
	assert.Error(t, err)
	assert.Empty(t, userID)

	// Check that it's a redirect error
	redirectErr, ok := err.(*RedirectError)
	assert.True(t, ok)
	assert.Contains(t, redirectErr.URL, "/login")
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_RequireUser_SuccessNew(t *testing.T) {
	sessionService, mockSessionStorage, mockQueries, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	userUUID := testutil.TestUUID("user-123")
	expectedUser := shared.Users{
		ID:    userUUID,
		Email: "test@example.com",
		Name:  pgtype.Text{String: "Test User", Valid: true},
	}

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError) // No impersonation
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": userUUID.String(),
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)
	mockQueries.On("GetUser", mock.Anything, userUUID).Return(expectedUser, nil)

	// Execute
	ctx := context.Background()
	user, err := sessionService.RequireUser(ctx, req)

	// Verify
	assert.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
	mockQueries.AssertExpectations(t)
}

func TestSessionService_IsAuthenticated_TrueNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	authenticatedUserID := "auth-user-123"

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError) // No impersonation
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": authenticatedUserID,
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)

	// Execute
	ctx := context.Background()
	isAuth, err := sessionService.IsAuthenticated(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.True(t, isAuth)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_IsAuthenticated_FalseNew(t *testing.T) {
	sessionService, mockSessionStorage, _, mockImpersonation := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Set up mock expectations
	mockImpersonation.On("GetImpersonation", req).Return("", assert.AnError)                      // No impersonation
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(map[string]interface{}{}, nil) // Empty session

	// Execute
	ctx := context.Background()
	isAuth, err := sessionService.IsAuthenticated(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.False(t, isAuth)
	mockImpersonation.AssertExpectations(t)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_GetAuthUserNew(t *testing.T) {
	sessionService, mockSessionStorage, _, _ := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	authenticatedUserID := "auth-user-123"

	// Set up mock expectations
	sessionData := map[string]interface{}{
		"auth_user": map[string]interface{}{
			"userId": authenticatedUserID,
		},
	}
	mockSessionStorage.On("GetSession", mock.Anything, req).Return(sessionData, nil)

	// Execute
	ctx := context.Background()
	authUser, err := sessionService.GetAuthUser(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, authUser)
	assert.Equal(t, authenticatedUserID, authUser.UserID)
	mockSessionStorage.AssertExpectations(t)
}

func TestSessionService_LogoutNew(t *testing.T) {
	sessionService, _, _, _ := createTestSessionService()

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Execute
	ctx := context.Background()
	err := sessionService.Logout(ctx, req)

	// Verify - should return redirect error
	assert.Error(t, err)
	redirectErr, ok := err.(*RedirectError)
	assert.True(t, ok)
	assert.Equal(t, "/logout", redirectErr.URL)
}
