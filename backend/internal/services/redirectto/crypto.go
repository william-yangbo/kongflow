package redirectto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	// ErrInvalidCookie indicates the cookie format is invalid
	ErrInvalidCookie = errors.New("invalid cookie format")

	// ErrDecryptionFailed indicates decryption failed
	ErrDecryptionFailed = errors.New("failed to decrypt cookie")

	// ErrCookieNotFound indicates the redirect cookie was not found
	ErrCookieNotFound = errors.New("redirect cookie not found")

	// ErrInvalidRedirectURL indicates the redirect URL is invalid
	ErrInvalidRedirectURL = errors.New("invalid redirect URL")

	// ErrInvalidSecretKey indicates the secret key is invalid
	ErrInvalidSecretKey = errors.New("invalid secret key: must be 16, 24, or 32 bytes")
)

// encrypt encrypts plaintext using AES-GCM and returns base64-encoded ciphertext
// This ensures both confidentiality and authenticity of the data
func (s *Service) encrypt(plaintext string) (string, error) {
	if len(s.config.SecretKey) == 0 {
		return "", ErrInvalidSecretKey
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.config.SecretKey)
	if err != nil {
		return "", err
	}

	// Create GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for cookie storage
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts base64-encoded ciphertext using AES-GCM
func (s *Service) decrypt(ciphertext string) (string, error) {
	if len(s.config.SecretKey) == 0 {
		return "", ErrInvalidSecretKey
	}

	// Decode from base64
	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCookie
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.config.SecretKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	// Create GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	// Check minimum size (nonce + at least some data)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCookie
	}

	// Extract nonce and ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// validateSecretKey validates that the secret key has the correct length for AES
func validateSecretKey(key []byte) error {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return ErrInvalidSecretKey
	}
	return nil
}
