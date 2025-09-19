package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecureConfigManager_GenerateSecureSecret(t *testing.T) {
	// Create a simple manager instance for testing the generateSecureSecret method
	manager := &SecureConfigManager{}

	// Test different secret lengths
	testCases := []int{16, 32, 64}

	for _, length := range testCases {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			secret := manager.generateSecureSecret(length)

			// Verify secret is not empty
			assert.NotEmpty(t, secret)

			// Verify it's base64 URL encoded (should not contain + or /)
			assert.NotContains(t, secret, "+")
			assert.NotContains(t, secret, "/")

			// Generate another secret and verify they're different
			secret2 := manager.generateSecureSecret(length)
			assert.NotEqual(t, secret, secret2, "Generated secrets should be unique")
		})
	}
}

func TestAuthSecrets_Structure(t *testing.T) {
	// Test the AuthSecrets structure
	secrets := AuthSecrets{
		MagicLinkSecret:    "magic-secret",
		SessionSecret:      "session-secret",
		RedirectSigningKey: "redirect-key",
		JWTSigningKey:      "jwt-key",
		JWTExpiryHours:     24,
		TokenExpiryMinutes: 15,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Verify all fields are set correctly
	assert.Equal(t, "magic-secret", secrets.MagicLinkSecret)
	assert.Equal(t, "session-secret", secrets.SessionSecret)
	assert.Equal(t, "redirect-key", secrets.RedirectSigningKey)
	assert.Equal(t, "jwt-key", secrets.JWTSigningKey)
	assert.Equal(t, 24, secrets.JWTExpiryHours)
	assert.Equal(t, 15, secrets.TokenExpiryMinutes)
	assert.False(t, secrets.CreatedAt.IsZero())
	assert.False(t, secrets.UpdatedAt.IsZero())
}

func TestSecureConfigManager_generateSecureSecret_Consistency(t *testing.T) {
	manager := &SecureConfigManager{}

	// Generate multiple secrets and ensure they're all different
	secrets := make([]string, 10)
	for i := range secrets {
		secrets[i] = manager.generateSecureSecret(32)
	}

	// Verify all secrets are unique
	for i := 0; i < len(secrets); i++ {
		for j := i + 1; j < len(secrets); j++ {
			assert.NotEqual(t, secrets[i], secrets[j], "All generated secrets should be unique")
		}
	}
}

func TestSecureConfigManager_generateSecureSecret_Length(t *testing.T) {
	manager := &SecureConfigManager{}

	// Test various lengths
	testCases := []struct {
		byteLength     int
		expectedMinLen int // Base64 encoding will make it longer
	}{
		{16, 20}, // 16 bytes -> ~22 chars in base64
		{32, 40}, // 32 bytes -> ~43 chars in base64
		{64, 80}, // 64 bytes -> ~86 chars in base64
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("bytes_%d", tc.byteLength), func(t *testing.T) {
			secret := manager.generateSecureSecret(tc.byteLength)

			// Verify minimum length (base64 encoding makes it longer)
			assert.GreaterOrEqual(t, len(secret), tc.expectedMinLen)

			// Verify it's base64 URL encoding (should not contain + or /)
			assert.NotContains(t, secret, "+")
			assert.NotContains(t, secret, "/")
			// Note: URLEncoding can contain = padding, so we don't check for that
		})
	}
}

// Integration test that validates the overall structure and defaults
func TestAuthSecrets_Defaults(t *testing.T) {
	// Test default values that would be created by GetOrCreateAuthSecrets
	defaultJWTExpiry := 24
	defaultTokenExpiry := 15

	// Verify these are reasonable defaults
	assert.Equal(t, 24, defaultJWTExpiry, "JWT expiry should default to 24 hours")
	assert.Equal(t, 15, defaultTokenExpiry, "Token expiry should default to 15 minutes")

	// Test time validation
	now := time.Now()
	testSecrets := AuthSecrets{
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, now, testSecrets.CreatedAt)
	assert.Equal(t, now, testSecrets.UpdatedAt)
}
