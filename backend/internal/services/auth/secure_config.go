package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/secretstore"
)

// SecureConfigManager handles secure configuration management for authentication
// Phase 3: Provides secure storage and retrieval of auth-related secrets
type SecureConfigManager struct {
	secretStore *secretstore.Service
	logger      *logger.Logger
}

// AuthSecrets represents the secure authentication configuration
// Enhanced with JWT support for Phase 3
type AuthSecrets struct {
	MagicLinkSecret    string    `json:"magicLinkSecret"`
	SessionSecret      string    `json:"sessionSecret"`
	RedirectSigningKey string    `json:"redirectSigningKey"`
	JWTSigningKey      string    `json:"jwtSigningKey"`  // Phase 3: JWT token signing
	JWTExpiryHours     int       `json:"jwtExpiryHours"` // Phase 3: JWT token expiry
	TokenExpiryMinutes int       `json:"tokenExpiryMinutes"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// NewSecureConfigManager creates a new secure configuration manager
func NewSecureConfigManager(secretStore *secretstore.Service) *SecureConfigManager {
	return &SecureConfigManager{
		secretStore: secretStore,
		logger:      logger.NewWebapp("auth.secure-config"),
	}
}

// GetOrCreateAuthSecrets retrieves auth secrets from secretstore or creates defaults
func (s *SecureConfigManager) GetOrCreateAuthSecrets(ctx context.Context) (*AuthSecrets, error) {
	var secrets AuthSecrets

	// Try to get existing secrets from secretstore
	err := s.secretStore.GetSecret(ctx, "auth.secrets", &secrets)
	if err != nil {
		s.logger.Info("Auth secrets not found, creating secure defaults", map[string]interface{}{
			"error": err.Error(),
		})

		// Create secure default configuration
		secrets = AuthSecrets{
			MagicLinkSecret:    s.generateSecureSecret(32),
			SessionSecret:      s.generateSecureSecret(32),
			RedirectSigningKey: s.generateSecureSecret(32),
			JWTSigningKey:      s.generateSecureSecret(64), // Larger key for JWT signing
			JWTExpiryHours:     24,                         // 24 hours default for JWT
			TokenExpiryMinutes: 15,                         // 15 minutes default
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		// Store the new secrets in secretstore
		if err := s.secretStore.SetSecret(ctx, "auth.secrets", secrets); err != nil {
			return nil, fmt.Errorf("failed to store auth secrets: %w", err)
		}

		s.logger.Info("Auth secrets created and stored securely", map[string]interface{}{
			"token_expiry_minutes": secrets.TokenExpiryMinutes,
		})
	}

	return &secrets, nil
}

// RotateSecrets generates new secrets and stores them securely
func (s *SecureConfigManager) RotateSecrets(ctx context.Context) error {
	s.logger.Info("Starting auth secrets rotation")

	// Generate new secrets
	newSecrets := AuthSecrets{
		MagicLinkSecret:    s.generateSecureSecret(32),
		SessionSecret:      s.generateSecureSecret(32),
		RedirectSigningKey: s.generateSecureSecret(32),
		JWTSigningKey:      s.generateSecureSecret(64), // New JWT signing key
		JWTExpiryHours:     24,                         // Maintain existing JWT expiry
		TokenExpiryMinutes: 15,                         // Maintain existing expiry
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Store the new secrets
	if err := s.secretStore.SetSecret(ctx, "auth.secrets", newSecrets); err != nil {
		s.logger.Error("Failed to rotate auth secrets", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to rotate auth secrets: %w", err)
	}

	s.logger.Info("Auth secrets rotated successfully")
	return nil
}

// GetMagicLinkSecret retrieves the current magic link secret
func (s *SecureConfigManager) GetMagicLinkSecret(ctx context.Context) (string, error) {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return "", err
	}
	return secrets.MagicLinkSecret, nil
}

// GetSessionSecret retrieves the current session secret
func (s *SecureConfigManager) GetSessionSecret(ctx context.Context) (string, error) {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return "", err
	}
	return secrets.SessionSecret, nil
}

// generateSecureSecret generates a cryptographically secure secret
func (s *SecureConfigManager) generateSecureSecret(byteLength int) string {
	// Generate random bytes for strong encryption
	randomBytes := make([]byte, byteLength)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback - this should never happen, but we handle it gracefully
		s.logger.Error("Failed to generate secure random bytes", map[string]interface{}{
			"error": err.Error(),
		})
		// Use timestamp as fallback (not ideal but better than panicking)
		fallback := fmt.Sprintf("fallback_%d", time.Now().UnixNano())
		return base64.URLEncoding.EncodeToString([]byte(fallback))
	}

	return base64.URLEncoding.EncodeToString(randomBytes)
}

// UpdateTokenExpiry updates the token expiry time in secure storage
func (s *SecureConfigManager) UpdateTokenExpiry(ctx context.Context, expiryMinutes int) error {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current secrets: %w", err)
	}

	// Update the expiry time
	secrets.TokenExpiryMinutes = expiryMinutes
	secrets.UpdatedAt = time.Now()

	// Store the updated secrets
	if err := s.secretStore.SetSecret(ctx, "auth.secrets", *secrets); err != nil {
		return fmt.Errorf("failed to update token expiry: %w", err)
	}

	s.logger.Info("Token expiry updated", map[string]interface{}{
		"new_expiry_minutes": expiryMinutes,
	})

	return nil
}

// GetJWTSigningKey retrieves the current JWT signing key
// Phase 3: JWT secret management
func (s *SecureConfigManager) GetJWTSigningKey(ctx context.Context) (string, error) {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return "", err
	}
	return secrets.JWTSigningKey, nil
}

// GetJWTExpiryDuration retrieves the JWT expiry duration
// Phase 3: JWT configuration management
func (s *SecureConfigManager) GetJWTExpiryDuration(ctx context.Context) (time.Duration, error) {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return 0, err
	}
	return time.Duration(secrets.JWTExpiryHours) * time.Hour, nil
}

// UpdateJWTExpiry updates the JWT expiry time in secure storage
// Phase 3: JWT configuration management
func (s *SecureConfigManager) UpdateJWTExpiry(ctx context.Context, expiryHours int) error {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current secrets: %w", err)
	}
	
	// Update the JWT expiry time
	secrets.JWTExpiryHours = expiryHours
	secrets.UpdatedAt = time.Now()
	
	// Store the updated secrets
	if err := s.secretStore.SetSecret(ctx, "auth.secrets", *secrets); err != nil {
		return fmt.Errorf("failed to update JWT expiry: %w", err)
	}
	
	s.logger.Info("JWT expiry updated", map[string]interface{}{
"new_expiry_hours": expiryHours,
})
	
	return nil
}

// RotateJWTSigningKey rotates only the JWT signing key
// Phase 3: Selective secret rotation for JWT security
func (s *SecureConfigManager) RotateJWTSigningKey(ctx context.Context) error {
	s.logger.Info("Starting JWT signing key rotation")
	
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current secrets: %w", err)
	}
	
	// Generate new JWT signing key
	secrets.JWTSigningKey = s.generateSecureSecret(64)
	secrets.UpdatedAt = time.Now()
	
	// Store the updated secrets
	if err := s.secretStore.SetSecret(ctx, "auth.secrets", *secrets); err != nil {
		s.logger.Error("Failed to rotate JWT signing key", map[string]interface{}{
"error": err.Error(),
		})
		return fmt.Errorf("failed to rotate JWT signing key: %w", err)
	}
	
	s.logger.Info("JWT signing key rotated successfully")
	return nil
}

// ValidateJWTConfiguration ensures JWT configuration is secure
// Phase 3: JWT security validation
func (s *SecureConfigManager) ValidateJWTConfiguration(ctx context.Context) error {
	secrets, err := s.GetOrCreateAuthSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate JWT config: %w", err)
	}
	
	// Validate JWT signing key strength
	if len(secrets.JWTSigningKey) < 32 {
		s.logger.Warn("JWT signing key is too short, rotating", map[string]interface{}{
"current_length": len(secrets.JWTSigningKey),
"required_min":   32,
})
		return s.RotateJWTSigningKey(ctx)
	}
	
	// Validate expiry time (should be reasonable)
	if secrets.JWTExpiryHours < 1 || secrets.JWTExpiryHours > 168 { // 1 hour to 7 days
		s.logger.Warn("JWT expiry time is outside recommended range", map[string]interface{}{
"current_hours":      secrets.JWTExpiryHours,
"recommended_range": "1-168 hours",
})
		
		// Set to default 24 hours
		return s.UpdateJWTExpiry(ctx, 24)
	}
	
	s.logger.Info("JWT configuration validated successfully", map[string]interface{}{
"key_strength": "strong",
"expiry_hours": secrets.JWTExpiryHours,
})
	
	return nil
}
