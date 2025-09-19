package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/ulid"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
)

// EmailStrategy implements Magic Link authentication
// Enhanced with logger and ulid services
// Aligned with trigger.dev's EmailLinkStrategy
type EmailStrategy struct {
	emailService      email.EmailService
	queries           *shared.Queries
	secret            string
	callbackURL       string
	logger            *logger.Logger     // Enhanced: structured logging
	ulid              *ulid.Service      // Enhanced: secure ID generation
	postAuthProcessor *PostAuthProcessor // Enhanced: enterprise post-auth processing
}

// NewEmailStrategy creates a new email strategy
// Enhanced with logger and ulid services
// Aligned with trigger.dev's EmailLinkStrategy constructor
func NewEmailStrategy(emailService email.EmailService, queries *shared.Queries, postAuthProcessor *PostAuthProcessor) *EmailStrategy {
	secret := os.Getenv("MAGIC_LINK_SECRET")
	if secret == "" {
		panic("MAGIC_LINK_SECRET environment variable is required")
	}

	return &EmailStrategy{
		emailService:      emailService,
		queries:           queries,
		secret:            secret,
		callbackURL:       "/magic",                       // Same as trigger.dev
		logger:            logger.NewWebapp("auth.email"), // Enhanced: structured logging
		ulid:              ulid.New(),                     // Enhanced: secure ID generation
		postAuthProcessor: postAuthProcessor,              // Enhanced: enterprise post-auth processing
	}
}

// Name returns the strategy name
func (e *EmailStrategy) Name() string {
	return "email"
}

// Authenticate sends a magic link email
// Enhanced with structured logging
// Aligned with trigger.dev's email authentication flow
func (e *EmailStrategy) Authenticate(ctx context.Context, req *http.Request) (*AuthUser, error) {
	// Parse email from form data
	err := req.ParseForm()
	if err != nil {
		e.logger.Error("Failed to parse form", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	emailAddr := req.FormValue("email")
	if emailAddr == "" {
		e.logger.Warn("Missing email in authentication request", map[string]interface{}{})
		return nil, fmt.Errorf("email is required")
	}

	e.logger.Info("Magic link authentication started", map[string]interface{}{
		"email":     emailAddr,
		"userAgent": req.UserAgent(),
	})

	// Generate magic link token
	token, err := e.generateMagicLinkToken(emailAddr)
	if err != nil {
		e.logger.Error("Failed to generate magic link token", map[string]interface{}{
			"email": emailAddr,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to generate magic link token: %w", err)
	}

	// Build magic link URL
	magicLink := fmt.Sprintf("%s%s?token=%s", e.getBaseURL(req), e.callbackURL, token)

	// Send magic link email
	err = e.emailService.SendMagicLinkEmail(ctx, email.SendMagicLinkOptions{
		EmailAddress: emailAddr,
		MagicLink:    magicLink,
	})
	if err != nil {
		e.logger.Error("Failed to send magic link email", map[string]interface{}{
			"email": emailAddr,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to send magic link email: %w", err)
	}

	e.logger.Info("Magic link email sent successfully", map[string]interface{}{
		"email": emailAddr,
	})

	// Return nil to indicate email was sent (not authenticated yet)
	return nil, nil
}

// HandleCallback verifies magic link and authenticates user
// Enhanced with structured logging
// Aligned with trigger.dev's magic link verification flow
func (e *EmailStrategy) HandleCallback(ctx context.Context, req *http.Request) (*AuthUser, error) {
	// Parse token from query parameters
	token := req.URL.Query().Get("token")
	if token == "" {
		e.logger.Warn("Magic link callback missing token", map[string]interface{}{})
		return nil, fmt.Errorf("magic link token is required")
	}

	e.logger.Debug("Magic link callback started", map[string]interface{}{
		"hasToken": true,
	})

	// Verify and extract email from token
	emailAddr, err := e.verifyMagicLinkToken(token)
	if err != nil {
		e.logger.Error("Magic link token verification failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("invalid magic link token: %w", err)
	}

	e.logger.Info("Magic link token verified successfully", map[string]interface{}{
		"email": emailAddr,
	})

	// Find or create user
	user, isNewUser, err := e.findOrCreateUser(ctx, emailAddr)
	if err != nil {
		e.logger.Error("Failed to find or create user", map[string]interface{}{
			"email": emailAddr,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	e.logger.Info("User authentication completed", map[string]interface{}{
		"email":     emailAddr,
		"userID":    user.ID.String(),
		"isNewUser": isNewUser,
	})

	// Enhanced: Post authentication processing (like trigger.dev's postAuth.server.ts)
	err = e.postAuthProcessor.PostAuthentication(ctx, user, isNewUser)
	if err != nil {
		e.logger.Error("Post authentication processing failed", map[string]interface{}{
			"userID": user.ID.String(),
			"error":  err.Error(),
		})
		// Don't fail the authentication for post-auth errors
	}

	return &AuthUser{UserID: user.ID.String()}, nil
}

// generateMagicLinkToken creates a secure token for magic link
// Enhanced with ULID for better security and traceability
func (e *EmailStrategy) generateMagicLinkToken(email string) (string, error) {
	// Enhanced: use ULID instead of timestamp for better security
	tokenID := e.ulid.Generate()
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%s:%s:%d", email, tokenID, timestamp)

	// Enhanced: log token generation
	e.logger.Debug("Magic link token generated", map[string]interface{}{
		"email":   email,
		"tokenID": tokenID,
	})

	// Generate random salt
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		e.logger.Error("Failed to generate salt", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}

	// Create HMAC signature
	h := sha256.New()
	h.Write([]byte(e.secret))
	h.Write([]byte(payload))
	h.Write(salt)
	signature := h.Sum(nil)

	// Encode token as base64
	tokenData := fmt.Sprintf("%s:%s:%s", payload, base64.URLEncoding.EncodeToString(salt), base64.URLEncoding.EncodeToString(signature))
	return base64.URLEncoding.EncodeToString([]byte(tokenData)), nil
}

// verifyMagicLinkToken verifies and extracts email from token
func (e *EmailStrategy) verifyMagicLinkToken(token string) (string, error) {
	// Decode token
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid token encoding: %w", err)
	}

	// Parse token components - need to find last two colons for salt:signature
	tokenStr := string(tokenBytes)

	// Find the last two colons to separate payload:salt:signature
	lastColonIndex := strings.LastIndex(tokenStr, ":")
	if lastColonIndex == -1 {
		return "", fmt.Errorf("invalid token format")
	}

	secondLastColonIndex := strings.LastIndex(tokenStr[:lastColonIndex], ":")
	if secondLastColonIndex == -1 {
		return "", fmt.Errorf("invalid token format")
	}

	payload := tokenStr[:secondLastColonIndex]
	saltStr := tokenStr[secondLastColonIndex+1 : lastColonIndex]
	signatureStr := tokenStr[lastColonIndex+1:]

	// Decode salt and signature
	salt, err := base64.URLEncoding.DecodeString(saltStr)
	if err != nil {
		return "", fmt.Errorf("invalid salt encoding: %w", err)
	}

	expectedSignature, err := base64.URLEncoding.DecodeString(signatureStr)
	if err != nil {
		return "", fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Verify signature
	h := sha256.New()
	h.Write([]byte(e.secret))
	h.Write([]byte(payload))
	h.Write(salt)
	actualSignature := h.Sum(nil)

	if !equalBytes(expectedSignature, actualSignature) {
		return "", fmt.Errorf("invalid token signature")
	}

	// Parse payload to get email, tokenID, and timestamp
	payloadParts := parseToken(payload)
	if len(payloadParts) != 3 { // Enhanced: now expects email:tokenID:timestamp
		return "", fmt.Errorf("invalid payload format")
	}

	emailAddr := payloadParts[0]
	tokenID := payloadParts[1]
	// timestamp := payloadParts[2] // Could be used for expiration validation

	// Enhanced: log token verification
	e.logger.Debug("Magic link token verified", map[string]interface{}{
		"email":   emailAddr,
		"tokenID": tokenID,
	})

	return emailAddr, nil
}

// findOrCreateUser finds existing user or creates new one
// Aligned with trigger.dev's findOrCreateUser function
func (e *EmailStrategy) findOrCreateUser(ctx context.Context, emailAddr string) (*shared.Users, bool, error) {
	// Try to find existing user
	user, err := e.queries.FindUserByEmail(ctx, emailAddr)
	if err == nil {
		// User exists
		return &user, false, nil
	}

	// User doesn't exist, create new one
	createParams := shared.CreateUserParams{
		Email:     emailAddr,
		Name:      pgtype.Text{String: "", Valid: false}, // Empty name initially
		AvatarUrl: pgtype.Text{String: "", Valid: false}, // Empty avatar initially
	}

	newUser, err := e.queries.CreateUser(ctx, createParams)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return &newUser, true, nil
}

// getBaseURL extracts base URL from request
func (e *EmailStrategy) getBaseURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

// Helper functions
func parseToken(s string) []string {
	var result []string
	var current string

	for i, char := range s {
		if char == ':' && (i == 0 || s[i-1] != '\\') {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	result = append(result, current)
	return result
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
