package impersonation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// signValue creates an HMAC signature for the given value
// This implementation is compatible with Remix's cookie signing
func (s *Service) signValue(value string) (string, error) {
	if len(s.config.SecretKey) == 0 {
		return "", ErrInvalidSecretKey
	}

	h := hmac.New(sha256.New, s.config.SecretKey)
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Format: value.signature (compatible with Remix)
	return fmt.Sprintf("%s.%s", value, signature), nil
}

// unsignValue verifies and extracts the original value from a signed value
func (s *Service) unsignValue(signedValue string) (string, error) {
	if len(s.config.SecretKey) == 0 {
		return "", ErrInvalidSecretKey
	}

	// Split value and signature
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", ErrInvalidCookieFormat
	}

	value, signature := parts[0], parts[1]

	// Verify signature
	h := hmac.New(sha256.New, s.config.SecretKey)
	h.Write([]byte(value))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", ErrInvalidSignature
	}

	return value, nil
}

// encodeUserID encodes the user ID as base64 for cookie storage
func encodeUserID(userID string) string {
	return base64.URLEncoding.EncodeToString([]byte(userID))
}

// decodeUserID decodes the base64 encoded user ID from cookie
func decodeUserID(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCookieFormat
	}
	return string(decoded), nil
}
