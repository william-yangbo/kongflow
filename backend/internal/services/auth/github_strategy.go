package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"kongflow/backend/internal/services/analytics"
	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/ulid"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
)

// GitHubStrategy implements OAuth authentication with GitHub
// Aligned with trigger.dev's GitHubStrategy pattern from remix-auth-github
type GitHubStrategy struct {
	clientID     string
	clientSecret string
	callbackURL  string
	logger       *logger.Logger
	ulid         *ulid.Service
	queries      shared.Querier // For user operations
	analytics    analytics.AnalyticsService
}

// GitHubProfile represents the GitHub user profile from API
// Aligned with trigger.dev's profile structure
type GitHubProfile struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubTokenResponse represents OAuth token response
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// NewGitHubStrategy creates a new GitHub OAuth strategy
// Follows trigger.dev's addGitHubStrategy pattern
func NewGitHubStrategy(clientID, clientSecret, callbackURL string, queries shared.Querier, analytics analytics.AnalyticsService) *GitHubStrategy {
	return &GitHubStrategy{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		logger:       logger.NewWebapp("auth.github"),
		ulid:         ulid.New(),
		queries:      queries,
		analytics:    analytics,
	}
}

// Name returns the strategy name
func (g *GitHubStrategy) Name() string {
	return "github"
}

// Authenticate initiates the GitHub OAuth flow
// Aligned with trigger.dev's OAuth redirect pattern
func (g *GitHubStrategy) Authenticate(ctx context.Context, req *http.Request) (*AuthUser, error) {
	// Generate state parameter for CSRF protection
	state, err := g.generateState()
	if err != nil {
		g.logger.Error("Failed to generate OAuth state", "error", err)
		return nil, fmt.Errorf("failed to generate OAuth state: %w", err)
	}

	// Store state in session for verification
	session, err := getSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	session["github_oauth_state"] = state

	// Build GitHub OAuth URL
	authURL := g.buildAuthURL(state)

	g.logger.Info("Redirecting to GitHub OAuth",
		"url", authURL,
		"client_id", g.clientID,
		"callback_url", g.callbackURL)

	// Return redirect instruction (will be handled by caller)
	return nil, &OAuthRedirectError{
		RedirectURL: authURL,
		State:       state,
	}
}

// HandleCallback processes the OAuth callback
// Aligned with trigger.dev's callback handling pattern
func (g *GitHubStrategy) HandleCallback(ctx context.Context, req *http.Request) (*AuthUser, error) {
	// Verify state parameter
	if err := g.verifyState(ctx, req); err != nil {
		g.logger.Error("OAuth state verification failed", "error", err)
		return nil, fmt.Errorf("state verification failed: %w", err)
	}

	// Get authorization code
	code := req.URL.Query().Get("code")
	if code == "" {
		g.logger.Error("No authorization code in callback")
		return nil, fmt.Errorf("no authorization code provided")
	}

	// Exchange code for access token
	accessToken, err := g.exchangeCodeForToken(code)
	if err != nil {
		g.logger.Error("Failed to exchange code for token", "error", err)
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user profile from GitHub
	profile, err := g.getUserProfile(accessToken)
	if err != nil {
		g.logger.Error("Failed to get user profile", "error", err)
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Validate required fields
	if profile.Email == "" {
		g.logger.Error("GitHub login requires an email address", "github_id", profile.ID)
		return nil, fmt.Errorf("GitHub login requires an email address")
	}

	// Find or create user (aligned with trigger.dev's findOrCreateUser)
	user, isNewUser, err := g.findOrCreateUser(ctx, profile, accessToken)
	if err != nil {
		g.logger.Error("Failed to find or create user", "error", err, "email", profile.Email)
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Post authentication processing (aligned with trigger.dev's postAuthentication)
	if err := g.postAuthentication(ctx, user, isNewUser); err != nil {
		g.logger.Error("Post authentication failed", "error", err, "user_id", user.UserID)
		// Don't fail login for analytics issues
	}

	g.logger.Info("GitHub authentication successful",
		"user_id", user.UserID,
		"is_new_user", isNewUser)

	return user, nil
}

// generateState creates a secure random state for CSRF protection
func (g *GitHubStrategy) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// buildAuthURL constructs the GitHub OAuth authorization URL
func (g *GitHubStrategy) buildAuthURL(state string) string {
	baseURL := "https://github.com/login/oauth/authorize"
	params := url.Values{
		"client_id":    {g.clientID},
		"redirect_uri": {g.callbackURL},
		"scope":        {"user:email"},
		"state":        {state},
	}
	return baseURL + "?" + params.Encode()
}

// verifyState validates the OAuth state parameter
func (g *GitHubStrategy) verifyState(ctx context.Context, req *http.Request) error {
	receivedState := req.URL.Query().Get("state")
	if receivedState == "" {
		return fmt.Errorf("no state parameter provided")
	}

	session, err := getSession(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	expectedState, ok := session["github_oauth_state"].(string)
	if !ok || expectedState == "" {
		return fmt.Errorf("no state in session")
	}

	if receivedState != expectedState {
		return fmt.Errorf("state mismatch")
	}

	// Clear state from session
	delete(session, "github_oauth_state")
	return nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (g *GitHubStrategy) exchangeCodeForToken(code string) (string, error) {
	tokenURL := "https://github.com/login/oauth/access_token"

	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub token exchange failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp GitHubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// getUserProfile fetches user profile from GitHub API
func (g *GitHubStrategy) getUserProfile(accessToken string) (*GitHubProfile, error) {
	userURL := "https://api.github.com/user"

	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profile GitHubProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	// If profile doesn't include email, fetch it separately
	if profile.Email == "" {
		email, err := g.getUserEmail(accessToken)
		if err == nil {
			profile.Email = email
		}
	}

	return &profile, nil
}

// getUserEmail fetches primary email from GitHub API
func (g *GitHubStrategy) getUserEmail(accessToken string) (string, error) {
	emailURL := "https://api.github.com/user/emails"

	req, err := http.NewRequest("GET", emailURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub email API failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	// Find primary email
	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	// Fallback to first email
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

// findOrCreateUser finds existing user or creates new one
// Aligned with trigger.dev's findOrCreateUser function
func (g *GitHubStrategy) findOrCreateUser(ctx context.Context, profile *GitHubProfile, accessToken string) (*AuthUser, bool, error) {
	// Try to find existing user by email
	existingUser, err := g.queries.FindUserByEmail(ctx, profile.Email)
	if err == nil {
		// User exists, return it
		return &AuthUser{
			UserID: existingUser.ID.String(),
		}, false, nil
	}

	// User doesn't exist, create new one
	newUser := shared.CreateUserParams{
		Email: profile.Email,
		Name: pgtype.Text{
			String: profile.Name,
			Valid:  profile.Name != "",
		},
		AvatarUrl: pgtype.Text{
			String: profile.AvatarURL,
			Valid:  profile.AvatarURL != "",
		},
	}

	createdUser, err := g.queries.CreateUser(ctx, newUser)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return &AuthUser{
		UserID: createdUser.ID.String(),
	}, true, nil
}

// postAuthentication handles post-authentication logic
// Aligned with trigger.dev's postAuthentication function
func (g *GitHubStrategy) postAuthentication(ctx context.Context, user *AuthUser, isNewUser bool) error {
	if g.analytics == nil {
		return nil // Analytics service not configured
	}

	// First, get the full user data from database
	fullUser, err := g.queries.FindUserByEmail(ctx, "")
	if err != nil {
		// If we can't get full user data, we'll skip analytics
		g.logger.Error("Failed to get full user data for analytics", "error", err, "user_id", user.UserID)
		return nil // Don't fail login for analytics issues
	}

	// User identification (matches trigger.dev's analytics.user.identify())
	if err := g.analytics.UserIdentify(ctx, &fullUser, isNewUser); err != nil {
		g.logger.Error("Failed to identify user in analytics", "error", err, "user_id", user.UserID)
	}

	// Track sign-in event using Capture method
	signInEvent := &analytics.TelemetryEvent{
		UserID: user.UserID,
		Event:  "Signed In",
		Properties: map[string]interface{}{
			"loginMethod": "GITHUB",
			"isNewUser":   isNewUser,
		},
	}
	if err := g.analytics.Capture(ctx, signInEvent); err != nil {
		g.logger.Error("Failed to track sign-in event", "error", err, "user_id", user.UserID)
	}

	// Track sign-up event for new users
	if isNewUser {
		signUpEvent := &analytics.TelemetryEvent{
			UserID: user.UserID,
			Event:  "Signed Up",
			Properties: map[string]interface{}{
				"authenticationMethod": "GITHUB",
			},
		}
		if err := g.analytics.Capture(ctx, signUpEvent); err != nil {
			g.logger.Error("Failed to track sign-up event", "error", err, "user_id", user.UserID)
		}
	}

	return nil
}

// OAuthRedirectError indicates that a redirect is needed for OAuth flow
type OAuthRedirectError struct {
	RedirectURL string
	State       string
}

func (e *OAuthRedirectError) Error() string {
	return fmt.Sprintf("OAuth redirect required to: %s", e.RedirectURL)
}

// IsOAuthRedirect checks if error is an OAuth redirect instruction
func IsOAuthRedirect(err error) (*OAuthRedirectError, bool) {
	if redirectErr, ok := err.(*OAuthRedirectError); ok {
		return redirectErr, true
	}
	return nil, false
}

// Helper function to get session (simplified for this context)
func getSession(ctx context.Context, req *http.Request) (map[string]interface{}, error) {
	// This would use the actual session storage implementation
	// For now, return a mock session
	return make(map[string]interface{}), nil
}

// AddGitHubStrategy adds GitHub OAuth strategy to authenticator
// Aligned with trigger.dev's addGitHubStrategy function
func AddGitHubStrategy(authenticator *Authenticator, clientID, clientSecret string, queries shared.Querier, analytics analytics.AnalyticsService) {
	// Build callback URL from environment
	loginOrigin := os.Getenv("LOGIN_ORIGIN")
	if loginOrigin == "" {
		loginOrigin = "http://localhost:8080" // Default for development
	}
	callbackURL := loginOrigin + "/auth/github/callback"

	githubStrategy := NewGitHubStrategy(clientID, clientSecret, callbackURL, queries, analytics)
	authenticator.RegisterStrategy(githubStrategy)
}
