package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kongflow/backend/internal/services/analytics"
	"kongflow/backend/internal/services/apiauth"
	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/redirectto"
	"kongflow/backend/internal/services/secretstore"
	"kongflow/backend/internal/services/workerqueue"
	"kongflow/backend/internal/shared"

	"github.com/riverqueue/river/rivertype"
) // WorkerQueueInterface defines the interface needed for post-auth processing
type WorkerQueueInterface interface {
	EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
}

// PostAuthProcessor handles post-authentication activities
// Aligned with trigger.dev postAuth.server.ts functionality
// Enhanced with security features: redirectto, secretstore, and JWT (Phase 3)
type PostAuthProcessor struct {
	analytics     analytics.AnalyticsService
	logger        *logger.Logger
	workerQueue   WorkerQueueInterface
	redirectTo    redirectto.RedirectToService // Phase 3: Secure redirect management
	secretStore   *secretstore.Service         // Phase 3: Secure secret management
	configManager *SecureConfigManager         // Phase 3: Secure config management
	apiAuthSvc    apiauth.APIAuthService       // Phase 3: JWT token generation
}

// NewPostAuthProcessor creates a new PostAuthProcessor (legacy constructor)
func NewPostAuthProcessor(
	analyticsService analytics.AnalyticsService,
	loggerService *logger.Logger,
	workerQueue WorkerQueueInterface,
) *PostAuthProcessor {
	return &PostAuthProcessor{
		analytics:   analyticsService,
		logger:      loggerService,
		workerQueue: workerQueue,
	}
}

// PostAuthentication handles all post-authentication activities for successful auth
// This includes analytics tracking, user identification, and async welcome emails
// Aligned with trigger.dev postAuth.server.ts implementation
func (p *PostAuthProcessor) PostAuthentication(ctx context.Context, user *shared.Users, isNewUser bool) error {
	p.logger.Info("Starting post-authentication processing", map[string]interface{}{
		"user_id":     user.ID,
		"is_new_user": isNewUser,
		"email":       user.Email,
	})

	// Track sign-in event (aligned with trigger.dev analytics)
	if err := p.trackSignInEvent(ctx, user, isNewUser); err != nil {
		p.logger.Error("Failed to track sign-in event", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		// Continue processing even if analytics fails
	}

	// Identify user for analytics (aligned with trigger.dev postAuth)
	if err := p.identifyUser(ctx, user); err != nil {
		p.logger.Error("Failed to identify user for analytics", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		// Continue processing even if analytics fails
	}

	// Schedule welcome email for new users (aligned with trigger.dev scheduleWelcomeEmail)
	if isNewUser {
		if err := p.scheduleWelcomeEmail(ctx, user); err != nil {
			p.logger.Error("Failed to schedule welcome email", map[string]interface{}{
				"error":   err.Error(),
				"user_id": user.ID,
				"email":   user.Email,
			})
			// Continue processing even if email scheduling fails
		}
	}

	p.logger.Info("Post-authentication processing completed", map[string]interface{}{
		"user_id":     user.ID,
		"is_new_user": isNewUser,
	})

	return nil
}

// trackSignInEvent tracks user sign-in events with proper event names
// Aligned with trigger.dev analytics tracking
func (p *PostAuthProcessor) trackSignInEvent(ctx context.Context, user *shared.Users, isNewUser bool) error {
	// Use analytics service with proper user identification
	// analytics.User is an alias for shared.Users, so we can cast directly
	analyticsUser := (*analytics.User)(user)
	return p.analytics.UserIdentify(ctx, analyticsUser, isNewUser)
}

// identifyUser identifies the user for analytics tracking
// Aligned with trigger.dev postAuth user identification
func (p *PostAuthProcessor) identifyUser(ctx context.Context, user *shared.Users) error {
	// For user identification, we call UserIdentify with isNewUser = false
	// since this is separate from the sign-in event tracking
	analyticsUser := (*analytics.User)(user)
	return p.analytics.UserIdentify(ctx, analyticsUser, false)
}

// scheduleWelcomeEmail schedules a welcome email to be sent after a 2-minute delay
// Aligned with trigger.dev scheduleWelcomeEmail implementation
func (p *PostAuthProcessor) scheduleWelcomeEmail(ctx context.Context, user *shared.Users) error {
	// Schedule welcome email with 2-minute delay (matching trigger.dev)
	delay := 2 * time.Minute
	runAt := time.Now().Add(delay)

	// Prepare email payload
	emailPayload := map[string]interface{}{
		"to":      user.Email,
		"subject": "Welcome to KongFlow!",
		"body":    fmt.Sprintf("Welcome %s! Thank you for joining KongFlow. We're excited to have you on board.", user.Email),
		"user_id": user.ID,
	}

	// Schedule the email job
	jobOptions := &workerqueue.JobOptions{
		RunAt:       &runAt,
		QueueName:   "email",
		Priority:    2, // Normal priority
		MaxAttempts: 3,
		Tags:        []string{"welcome", "new_user", "email"},
		JobKey:      fmt.Sprintf("welcome_email_%s", user.ID.Bytes),
		JobKeyMode:  "preserve_run_at", // Don't replace if already scheduled
	}

	result, err := p.workerQueue.EnqueueJob(ctx, "scheduleEmail", emailPayload, jobOptions)
	if err != nil {
		return fmt.Errorf("failed to schedule welcome email: %w", err)
	}

	p.logger.Info("Welcome email scheduled successfully", map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"job_id":     result.Job.ID,
		"run_at":     runAt,
		"delay_mins": 2,
	})

	return nil
}

// PostAuthenticationWithRedirect handles post-auth processing with secure redirect management
// Phase 3 enhancement: Includes secure redirect validation and handling
func (p *PostAuthProcessor) PostAuthenticationWithRedirect(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	user *shared.Users,
	isNewUser bool,
) (redirectURL string, err error) {
	// Perform standard post-authentication processing
	if err := p.PostAuthentication(ctx, user, isNewUser); err != nil {
		return "", fmt.Errorf("post-authentication processing failed: %w", err)
	}

	// Handle secure redirect only if redirectTo service is available
	if p.redirectTo != nil {
		redirectURL, err = p.handleSecureRedirect(w, r)
		if err != nil {
			p.logger.Error("Failed to handle secure redirect", map[string]interface{}{
				"error":   err.Error(),
				"user_id": user.ID,
			})
			// Return default redirect on error
			redirectURL = "/"
		}
	} else {
		// Default redirect if no redirect service
		redirectURL = "/"
	}

	p.logger.Info("Post-authentication completed with redirect", map[string]interface{}{
		"user_id":      user.ID,
		"redirect_url": redirectURL,
	})

	return redirectURL, nil
}

// handleSecureRedirect manages secure post-authentication redirects
// Phase 3: Prevents open redirect vulnerabilities
func (p *PostAuthProcessor) handleSecureRedirect(w http.ResponseWriter, r *http.Request) (string, error) {
	if p.redirectTo == nil {
		return "/", fmt.Errorf("redirect service not available")
	}

	// Try to get the stored redirect URL
	redirectURL, err := p.redirectTo.GetRedirectTo(r)
	if err != nil {
		p.logger.Debug("No redirect URL found or invalid", map[string]interface{}{
			"error": err.Error(),
		})
		// Return default redirect if no valid redirect is found
		return "/", nil
	}

	// Clear the redirect cookie after successful retrieval
	if err := p.redirectTo.ClearRedirectTo(w, r); err != nil {
		p.logger.Error("Failed to clear redirect cookie", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue anyway, the redirect URL is valid
	}

	p.logger.Info("Secure redirect retrieved", map[string]interface{}{
		"redirect_url": redirectURL,
	})

	return redirectURL, nil
}

// SetRedirectURL sets a secure redirect URL for post-authentication
// Phase 3: Secure redirect management to prevent open redirect attacks
func (p *PostAuthProcessor) SetRedirectURL(w http.ResponseWriter, r *http.Request, redirectURL string) error {
	if p.redirectTo == nil {
		return fmt.Errorf("redirect service not available")
	}

	if err := p.redirectTo.SetRedirectTo(w, r, redirectURL); err != nil {
		p.logger.Error("Failed to set redirect URL", map[string]interface{}{
			"redirect_url": redirectURL,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to set redirect URL: %w", err)
	}

	p.logger.Info("Redirect URL set securely", map[string]interface{}{
		"redirect_url": redirectURL,
	})

	return nil
}

// NewPostAuthProcessorWithSecurity creates a new PostAuthProcessor with full security features
// Phase 3: Constructor for security-enhanced PostAuthProcessor
func NewPostAuthProcessorWithSecurity(
	analyticsService analytics.AnalyticsService,
	loggerService *logger.Logger,
	workerQueue WorkerQueueInterface,
	redirectService redirectto.RedirectToService,
	secretService *secretstore.Service,
	apiAuthService apiauth.APIAuthService,
) *PostAuthProcessor {
	return &PostAuthProcessor{
		analytics:     analyticsService,
		logger:        loggerService,
		workerQueue:   workerQueue,
		redirectTo:    redirectService,
		secretStore:   secretService,
		configManager: NewSecureConfigManager(secretService),
		apiAuthSvc:    apiAuthService,
	}
}

// InitializeSecureAuth initializes and validates secure authentication configuration
// Phase 3: Ensures all auth secrets are properly configured and stored securely
func (p *PostAuthProcessor) InitializeSecureAuth(ctx context.Context) error {
	if p.configManager == nil {
		return fmt.Errorf("secure config manager not available")
	}

	p.logger.Info("Initializing secure authentication configuration")

	// Get or create secure auth configuration
	secrets, err := p.configManager.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		p.logger.Error("Failed to initialize secure auth config", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to initialize secure auth: %w", err)
	}

	p.logger.Info("Secure authentication configuration initialized", map[string]interface{}{
		"token_expiry_minutes": secrets.TokenExpiryMinutes,
		"created_at":           secrets.CreatedAt,
	})

	return nil
}

// RotateAuthSecrets rotates all authentication secrets for enhanced security
// Phase 3: Security operation for periodic secret rotation
func (p *PostAuthProcessor) RotateAuthSecrets(ctx context.Context) error {
	if p.configManager == nil {
		return fmt.Errorf("secure config manager not available")
	}

	p.logger.Info("Starting auth secrets rotation")

	if err := p.configManager.RotateSecrets(ctx); err != nil {
		p.logger.Error("Failed to rotate auth secrets", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("secret rotation failed: %w", err)
	}

	p.logger.Info("Auth secrets rotation completed successfully")
	return nil
}

// GetSecureAuthSecrets retrieves current authentication secrets
// Phase 3: Secure secret access for auth operations
func (p *PostAuthProcessor) GetSecureAuthSecrets(ctx context.Context) (*AuthSecrets, error) {
	if p.configManager == nil {
		return nil, fmt.Errorf("secure config manager not available")
	}

	secrets, err := p.configManager.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		p.logger.Error("Failed to retrieve auth secrets", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get auth secrets: %w", err)
	}

	return secrets, nil
}

// GeneratePostAuthJWT generates a JWT token for the user after successful authentication
// Phase 3: JWT integration for enhanced post-authentication security
func (p *PostAuthProcessor) GeneratePostAuthJWT(ctx context.Context, user *shared.Users, environment *apiauth.RuntimeEnvironment, opts *JWTGenerationOptions) (string, error) {
	if p.apiAuthSvc == nil {
		return "", fmt.Errorf("apiauth service not available")
	}

	// Build JWT payload based on user and authentication context
	payload := map[string]interface{}{
		"user_id": user.ID.String(),
		"email":   user.Email,
	}

	// Add scopes based on user role and context
	if opts != nil && len(opts.Scopes) > 0 {
		payload["scopes"] = opts.Scopes
	} else {
		// Default scopes for authenticated users
		payload["scopes"] = []string{"read:user", "write:user", "read:projects"}
	}

	// Add optional claims
	if opts != nil {
		if opts.Realtime {
			payload["realtime"] = true
		}
		if opts.OneTimeUse {
			payload["otu"] = true
		}
		if opts.CustomClaims != nil {
			for k, v := range opts.CustomClaims {
				payload[k] = v
			}
		}
	}

	// Configure JWT options
	jwtOpts := &apiauth.JWTOptions{
		ExpirationTime: time.Hour, // Default 1 hour
		CustomClaims:   make(map[string]interface{}),
	}

	if opts != nil && opts.ExpirationTime != nil {
		jwtOpts.ExpirationTime = opts.ExpirationTime
	}

	// Add authentication context claims
	jwtOpts.CustomClaims["auth_method"] = "magic_link"
	jwtOpts.CustomClaims["issued_for"] = "post_authentication"

	// Generate JWT token
	token, err := p.apiAuthSvc.GenerateJWTToken(ctx, environment, payload, jwtOpts)
	if err != nil {
		p.logger.Error("Failed to generate post-auth JWT", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	p.logger.Info("Post-authentication JWT generated successfully", map[string]interface{}{
		"user_id":    user.ID,
		"expires_in": "1h",
		"scopes":     payload["scopes"],
	})

	return token, nil
}

// JWTGenerationOptions configures JWT generation for post-authentication
type JWTGenerationOptions struct {
	Scopes         []string               `json:"scopes,omitempty"`
	Realtime       bool                   `json:"realtime,omitempty"`
	OneTimeUse     bool                   `json:"oneTimeUse,omitempty"`
	ExpirationTime interface{}            `json:"expirationTime,omitempty"`
	CustomClaims   map[string]interface{} `json:"customClaims,omitempty"`
}

// PostAuthenticationWithJWT handles post-auth processing with JWT generation and secure redirect
// Phase 3: Enhanced authentication flow with JWT token generation
func (p *PostAuthProcessor) PostAuthenticationWithJWT(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	user *shared.Users,
	environment *apiauth.RuntimeEnvironment,
	isNewUser bool,
	jwtOpts *JWTGenerationOptions,
) (*PostAuthJWTResult, error) {
	// Perform standard post-authentication processing
	if err := p.PostAuthentication(ctx, user, isNewUser); err != nil {
		return nil, fmt.Errorf("post-authentication processing failed: %w", err)
	}

	// Generate JWT token for the authenticated user
	jwtToken, err := p.GeneratePostAuthJWT(ctx, user, environment, jwtOpts)
	if err != nil {
		p.logger.Error("Failed to generate JWT in post-auth flow", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		// Continue with redirect even if JWT generation fails
		jwtToken = ""
	}

	// Handle secure redirect
	var redirectURL string
	if p.redirectTo != nil {
		redirectURL, err = p.handleSecureRedirect(w, r)
		if err != nil {
			p.logger.Error("Failed to handle secure redirect", map[string]interface{}{
				"error":   err.Error(),
				"user_id": user.ID,
			})
			// Return default redirect on error
			redirectURL = "/"
		}

		// Append JWT token to redirect URL if available
		if jwtToken != "" {
			redirectURL = p.appendJWTToRedirectURL(redirectURL, jwtToken)
		}
	} else {
		// Default redirect if no redirect service
		redirectURL = "/"
		if jwtToken != "" {
			redirectURL = p.appendJWTToRedirectURL(redirectURL, jwtToken)
		}
	}

	result := &PostAuthJWTResult{
		RedirectURL: redirectURL,
		JWTToken:    jwtToken,
		User:        user,
		IsNewUser:   isNewUser,
	}

	p.logger.Info("Post-authentication with JWT completed", map[string]interface{}{
		"user_id":      user.ID,
		"redirect_url": redirectURL,
		"jwt_provided": jwtToken != "",
		"is_new_user":  isNewUser,
	})

	return result, nil
}

// appendJWTToRedirectURL safely appends JWT token to redirect URL
// Phase 3: Secure JWT token transmission via URL parameters
func (p *PostAuthProcessor) appendJWTToRedirectURL(redirectURL, jwtToken string) string {
	if jwtToken == "" {
		return redirectURL
	}

	// Add JWT as URL parameter
	separator := "?"
	if strings.Contains(redirectURL, "?") {
		separator = "&"
	}

	return fmt.Sprintf("%s%stoken=%s", redirectURL, separator, jwtToken)
}

// PostAuthJWTResult represents the result of post-authentication with JWT
type PostAuthJWTResult struct {
	RedirectURL string        `json:"redirectUrl"`
	JWTToken    string        `json:"jwtToken,omitempty"`
	User        *shared.Users `json:"user"`
	IsNewUser   bool          `json:"isNewUser"`
}

// GenerateUserScopes generates JWT scopes based on user role and context
// Phase 3: Role-based access control for JWT tokens
func (p *PostAuthProcessor) GenerateUserScopes(ctx context.Context, user *shared.Users, environment *apiauth.RuntimeEnvironment) []string {
	baseScopes := []string{
		"read:user",
		"read:profile",
	}

	// Add scopes based on user's role in the organization
// Note: This would be enhanced with actual role checking once user roles are implemented
userScopes := append(baseScopes, []string{
"write:user",
"read:projects",
"read:environments", 
}...)

// Add project-specific scopes if environment is provided
if environment != nil {
projectScopes := []string{
"read:project:" + environment.ProjectID.String(),
"read:environment:" + environment.ID.String(),
}
userScopes = append(userScopes, projectScopes...)
}

p.logger.Debug("Generated JWT scopes for user", map[string]interface{}{
"user_id": user.ID,
"scopes":  userScopes,
"count":   len(userScopes),
})

return userScopes
}

// GenerateRoleBasedJWTOptions creates JWT options based on user role and context
// Phase 3: Enhanced JWT configuration based on user permissions
func (p *PostAuthProcessor) GenerateRoleBasedJWTOptions(ctx context.Context, user *shared.Users, environment *apiauth.RuntimeEnvironment, requestOpts *JWTGenerationOptions) (*JWTGenerationOptions, error) {
// Generate base scopes for the user
userScopes := p.GenerateUserScopes(ctx, user, environment)

// Merge with requested scopes (if any)
if requestOpts != nil && len(requestOpts.Scopes) > 0 {
// Validate that requested scopes are subset of user's allowed scopes
		validatedScopes := p.validateRequestedScopes(userScopes, requestOpts.Scopes)
		userScopes = validatedScopes
	}

	// Get JWT expiry from secure config
	var expirationTime interface{} = time.Hour * 1 // Default 1 hour
	if p.configManager != nil {
		if duration, err := p.configManager.GetJWTExpiryDuration(ctx); err == nil {
			expirationTime = duration
		}
	}

	// Build JWT options
	jwtOpts := &JWTGenerationOptions{
		Scopes:         userScopes,
		ExpirationTime: expirationTime,
		CustomClaims: map[string]interface{}{
			"user_email": user.Email,
			"auth_time":  time.Now().Unix(),
		},
	}

	// Apply request-specific options
	if requestOpts != nil {
		if requestOpts.Realtime {
			jwtOpts.Realtime = true
			jwtOpts.CustomClaims["realtime"] = true
		}
		if requestOpts.OneTimeUse {
			jwtOpts.OneTimeUse = true
			jwtOpts.CustomClaims["otu"] = true
		}
		// Merge custom claims
		if requestOpts.CustomClaims != nil {
			for k, v := range requestOpts.CustomClaims {
				jwtOpts.CustomClaims[k] = v
			}
		}
		// Override expiration if specified
		if requestOpts.ExpirationTime != nil {
			jwtOpts.ExpirationTime = requestOpts.ExpirationTime
		}
	}

	p.logger.Info("Generated role-based JWT options", map[string]interface{}{
"user_id":       user.ID,
"scopes_count":  len(jwtOpts.Scopes),
"realtime":      jwtOpts.Realtime,
"one_time_use":  jwtOpts.OneTimeUse,
"custom_claims": len(jwtOpts.CustomClaims),
})

	return jwtOpts, nil
}

// validateRequestedScopes ensures requested scopes are within user's permissions
// Phase 3: Scope validation for security
func (p *PostAuthProcessor) validateRequestedScopes(allowedScopes, requestedScopes []string) []string {
allowedMap := make(map[string]bool)
for _, scope := range allowedScopes {
allowedMap[scope] = true
}

var validScopes []string
for _, requested := range requestedScopes {
if allowedMap[requested] {
validScopes = append(validScopes, requested)
} else {
p.logger.Warn("Requested scope not allowed for user", map[string]interface{}{
"requested_scope": requested,
"allowed_scopes":  allowedScopes,
})
}
}

return validScopes
}

// PostAuthenticationWithEnhancedJWT provides the complete JWT authentication flow
// Phase 3: Complete JWT integration with role-based access control
func (p *PostAuthProcessor) PostAuthenticationWithEnhancedJWT(
ctx context.Context,
w http.ResponseWriter,
r *http.Request,
user *shared.Users,
environment *apiauth.RuntimeEnvironment,
isNewUser bool,
requestOpts *JWTGenerationOptions,
) (*PostAuthJWTResult, error) {
// Perform standard post-authentication processing
if err := p.PostAuthentication(ctx, user, isNewUser); err != nil {
return nil, fmt.Errorf("post-authentication processing failed: %w", err)
}

// Generate role-based JWT options
jwtOpts, err := p.GenerateRoleBasedJWTOptions(ctx, user, environment, requestOpts)
if err != nil {
p.logger.Error("Failed to generate role-based JWT options", map[string]interface{}{
"error":   err.Error(),
"user_id": user.ID,
})
// Continue with basic options
jwtOpts = &JWTGenerationOptions{
Scopes:         []string{"read:user"},
ExpirationTime: time.Hour,
}
}

// Generate JWT token with enhanced options
jwtToken, err := p.GeneratePostAuthJWT(ctx, user, environment, jwtOpts)
if err != nil {
p.logger.Error("Failed to generate JWT in enhanced flow", map[string]interface{}{
"error":   err.Error(),
"user_id": user.ID,
})
// Continue with redirect even if JWT generation fails
jwtToken = ""
}

// Handle secure redirect with JWT
var redirectURL string
if p.redirectTo != nil {
redirectURL, err = p.handleSecureRedirect(w, r)
if err != nil {
p.logger.Error("Failed to handle secure redirect", map[string]interface{}{
"error":   err.Error(),
"user_id": user.ID,
})
redirectURL = "/"
}

// Append JWT token to redirect URL if available
if jwtToken != "" {
redirectURL = p.appendJWTToRedirectURL(redirectURL, jwtToken)
}
} else {
redirectURL = "/"
if jwtToken != "" {
redirectURL = p.appendJWTToRedirectURL(redirectURL, jwtToken)
}
}

result := &PostAuthJWTResult{
RedirectURL: redirectURL,
JWTToken:    jwtToken,
User:        user,
IsNewUser:   isNewUser,
}

p.logger.Info("Enhanced JWT post-authentication completed", map[string]interface{}{
"user_id":       user.ID,
"redirect_url":  redirectURL,
"jwt_provided":  jwtToken != "",
"scopes_count":  len(jwtOpts.Scopes),
"is_new_user":   isNewUser,
})

return result, nil
}
