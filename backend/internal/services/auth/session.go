package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"kongflow/backend/internal/services/impersonation"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
)

// SessionService implementation
// Strictly aligned with trigger.dev's session.server.ts
type sessionService struct {
	authenticator        *Authenticator
	queries              *shared.Queries
	impersonationService impersonation.ImpersonationService
}

// NewSessionService creates a new session service
func NewSessionService(authenticator *Authenticator, queries *shared.Queries, impersonationService impersonation.ImpersonationService) SessionService {
	return &sessionService{
		authenticator:        authenticator,
		queries:              queries,
		impersonationService: impersonationService,
	}
}

// GetUserID returns the current user ID, considering impersonation
// Aligned with trigger.dev's getUserId function
func (s *sessionService) GetUserID(ctx context.Context, request *http.Request) (string, error) {
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

// GetUser returns the current user, considering impersonation
// Aligned with trigger.dev's getUser function
func (s *sessionService) GetUser(ctx context.Context, request *http.Request) (*shared.Users, error) {
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

	// Get user from database
	user, err := s.queries.GetUser(ctx, userUUID)
	if err != nil {
		// If user not found, trigger logout (like trigger.dev)
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// RequireUserID ensures user is authenticated, redirects if not
// Aligned with trigger.dev's requireUserId function
func (s *sessionService) RequireUserID(ctx context.Context, request *http.Request, redirectTo string) (string, error) {
	userID, err := s.GetUserID(ctx, request)
	if err != nil {
		return "", err
	}
	if userID == "" {
		// Build redirect URL with redirectTo parameter, like trigger.dev
		if redirectTo == "" {
			redirectTo = request.URL.Path
			if request.URL.RawQuery != "" {
				redirectTo += "?" + request.URL.RawQuery
			}
		}

		redirectURL := fmt.Sprintf("/login?redirectTo=%s", url.QueryEscape(redirectTo))
		return "", &RedirectError{URL: redirectURL}
	}
	return userID, nil
}

// RequireUser ensures user is authenticated and returns user data
// Aligned with trigger.dev's requireUser function
func (s *sessionService) RequireUser(ctx context.Context, request *http.Request) (*shared.Users, error) {
	userID, err := s.RequireUserID(ctx, request, "")
	if err != nil {
		return nil, err
	}

	// Convert string userID to pgtype.UUID
	var userUUID pgtype.UUID
	err = userUUID.Scan(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Get user from database
	user, err := s.queries.GetUser(ctx, userUUID)
	if err != nil {
		// If user not found, trigger logout (like trigger.dev)
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// Logout handles user logout
// Aligned with trigger.dev's logout function
func (s *sessionService) Logout(ctx context.Context, request *http.Request) error {
	// Return redirect to logout endpoint (like trigger.dev)
	return &RedirectError{URL: "/logout"}
}

// IsAuthenticated checks if user is authenticated
func (s *sessionService) IsAuthenticated(ctx context.Context, request *http.Request) (bool, error) {
	userID, err := s.GetUserID(ctx, request)
	if err != nil {
		return false, err
	}
	return userID != "", nil
}

// GetAuthUser returns the authenticated user data from session
func (s *sessionService) GetAuthUser(ctx context.Context, request *http.Request) (*AuthUser, error) {
	return s.authenticator.IsAuthenticated(ctx, request)
}

// RedirectError represents a redirect response
type RedirectError struct {
	URL string
}

func (e *RedirectError) Error() string {
	return "redirect to " + e.URL
}
