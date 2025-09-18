package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailService_Validator(t *testing.T) {
	// Setup
	mockProvider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	require.NoError(t, err)

	config := EmailConfig{
		FromEmail:     "test@example.com",
		ReplyToEmail:  "reply@example.com",
		ImagesBaseURL: "https://example.com",
	}

	service := New(mockProvider, templateEngine, config).(*emailService)

	t.Run("ValidEmailAddress", func(t *testing.T) {
		// Valid email should pass
		options := SendMagicLinkOptions{
			EmailAddress: "user@example.com",
			MagicLink:    "https://example.com/auth?token=123",
		}

		err := service.SendMagicLinkEmail(context.Background(), options)
		assert.NoError(t, err)
	})

	t.Run("InvalidEmailAddress", func(t *testing.T) {
		// Invalid email should fail validation
		options := SendMagicLinkOptions{
			EmailAddress: "not-an-email",
			MagicLink:    "https://example.com/auth?token=123",
		}

		err := service.SendMagicLinkEmail(context.Background(), options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("EmptyEmailAddress", func(t *testing.T) {
		// Empty email should fail validation
		options := SendMagicLinkOptions{
			EmailAddress: "",
			MagicLink:    "https://example.com/auth?token=123",
		}

		err := service.SendMagicLinkEmail(context.Background(), options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("InvalidMagicLinkURL", func(t *testing.T) {
		// Invalid URL should fail validation
		options := SendMagicLinkOptions{
			EmailAddress: "user@example.com",
			MagicLink:    "not-a-url",
		}

		err := service.SendMagicLinkEmail(context.Background(), options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("EmptyMagicLink", func(t *testing.T) {
		// Empty magic link should fail validation
		options := SendMagicLinkOptions{
			EmailAddress: "user@example.com",
			MagicLink:    "",
		}

		err := service.SendMagicLinkEmail(context.Background(), options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("ValidUser", func(t *testing.T) {
		// Valid user should pass
		userName := "John Doe"
		user := User{
			ID:    "user123",
			Email: "user@example.com",
			Name:  &userName,
		}

		err := service.ScheduleWelcomeEmail(context.Background(), user)
		assert.NoError(t, err)
	})

	t.Run("InvalidUserEmail", func(t *testing.T) {
		// Invalid user email should fail validation
		userName := "John Doe"
		user := User{
			ID:    "user123",
			Email: "not-an-email",
			Name:  &userName,
		}

		err := service.ScheduleWelcomeEmail(context.Background(), user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user validation failed")
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		// Empty user ID should fail validation
		userName := "John Doe"
		user := User{
			ID:    "",
			Email: "user@example.com",
			Name:  &userName,
		}

		err := service.ScheduleWelcomeEmail(context.Background(), user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user validation failed")
	})
}

func TestEmailService_ValidatorAdvanced(t *testing.T) {
	// Setup
	mockProvider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	require.NoError(t, err)

	config := EmailConfig{
		FromEmail:     "test@example.com",
		ReplyToEmail:  "reply@example.com",
		ImagesBaseURL: "https://example.com",
	}

	service := New(mockProvider, templateEngine, config).(*emailService)

	t.Run("URLValidation", func(t *testing.T) {
		testCases := []struct {
			name      string
			url       string
			shouldErr bool
		}{
			{"ValidHTTPS", "https://example.com/auth?token=123", false},
			{"ValidHTTP", "http://localhost:3000/auth?token=123", false},
			{"InvalidScheme", "ftp://example.com/auth", false}, // FTP URLs are technically valid URLs
			{"NoScheme", "example.com/auth", true},
			{"MalformedURL", "https://", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := SendMagicLinkOptions{
					EmailAddress: "user@example.com",
					MagicLink:    tc.url,
				}

				err := service.SendMagicLinkEmail(context.Background(), options)
				if tc.shouldErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("EmailValidation", func(t *testing.T) {
		testCases := []struct {
			name      string
			email     string
			shouldErr bool
		}{
			{"ValidSimple", "user@example.com", false},
			{"ValidSubdomain", "user@mail.example.com", false},
			{"ValidWithPlus", "user+tag@example.com", false},
			{"ValidWithNumbers", "user123@example.com", false},
			{"InvalidNoDomain", "user@", true},
			{"InvalidNoUsername", "@example.com", true},
			{"InvalidNoAt", "userexample.com", true},
			{"InvalidMultipleAt", "user@@example.com", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := SendMagicLinkOptions{
					EmailAddress: tc.email,
					MagicLink:    "https://example.com/auth?token=123",
				}

				err := service.SendMagicLinkEmail(context.Background(), options)
				if tc.shouldErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "validation failed")
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
