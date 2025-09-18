// Package email tests
package email

import (
	"context"
	"testing"
)

// MockEmailProvider implements EmailProvider for testing
type MockEmailProvider struct {
	SentEmails []Email
	SendError  error
}

func (m *MockEmailProvider) SendEmail(ctx context.Context, email Email) error {
	if m.SendError != nil {
		return m.SendError
	}
	m.SentEmails = append(m.SentEmails, email)
	return nil
}

func TestEmailService_SendMagicLinkEmail(t *testing.T) {
	// Create mock provider and template engine
	provider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	config := EmailConfig{
		FromEmail:    "noreply@kongflow.dev",
		ReplyToEmail: "help@kongflow.dev",
	}

	// Create email service
	service := New(provider, templateEngine, config)

	// Test data
	options := SendMagicLinkOptions{
		EmailAddress: "user@example.com",
		MagicLink:    "https://kongflow.dev/auth/verify?token=abc123",
	}

	// Send magic link email
	ctx := context.Background()
	err = service.SendMagicLinkEmail(ctx, options)
	if err != nil {
		t.Fatalf("Failed to send magic link email: %v", err)
	}

	// Verify email was sent
	if len(provider.SentEmails) != 1 {
		t.Fatalf("Expected 1 email to be sent, got %d", len(provider.SentEmails))
	}

	email := provider.SentEmails[0]
	if email.To != options.EmailAddress {
		t.Errorf("Expected email to %s, got %s", options.EmailAddress, email.To)
	}

	if email.From != config.FromEmail {
		t.Errorf("Expected email from %s, got %s", config.FromEmail, email.From)
	}

	if email.Subject != "Sign in to KongFlow" {
		t.Errorf("Expected subject 'Sign in to KongFlow', got %s", email.Subject)
	}

	// Verify magic link is in the email content
	if !contains(email.HTML, options.MagicLink) {
		t.Errorf("Expected magic link %s to be in email HTML", options.MagicLink)
	}
}

func TestEmailService_ScheduleWelcomeEmail(t *testing.T) {
	// Create mock provider and template engine
	provider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	config := EmailConfig{
		FromEmail:    "noreply@kongflow.dev",
		ReplyToEmail: "help@kongflow.dev",
	}

	// Create email service
	service := New(provider, templateEngine, config)

	// Test data
	name := "John Doe"
	user := User{
		ID:    "user123",
		Email: "john@example.com",
		Name:  &name,
	}

	// Schedule welcome email
	ctx := context.Background()
	err = service.ScheduleWelcomeEmail(ctx, user)

	// Since we don't have worker queue integration yet, this should send immediately
	if err != nil {
		t.Fatalf("Failed to schedule welcome email: %v", err)
	}

	// Verify email was sent
	if len(provider.SentEmails) != 1 {
		t.Fatalf("Expected 1 email to be sent, got %d", len(provider.SentEmails))
	}

	email := provider.SentEmails[0]
	if email.To != user.Email {
		t.Errorf("Expected email to %s, got %s", user.Email, email.To)
	}

	if email.Subject != "Welcome to KongFlow" {
		t.Errorf("Expected subject 'Welcome to KongFlow', got %s", email.Subject)
	}

	// Verify user name is in the email content
	if !contains(email.HTML, name) {
		t.Errorf("Expected user name %s to be in email HTML", name)
	}
}

func TestEmailService_ValidationErrors(t *testing.T) {
	provider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	config := EmailConfig{
		FromEmail:    "noreply@kongflow.dev",
		ReplyToEmail: "help@kongflow.dev",
	}

	service := New(provider, templateEngine, config)
	ctx := context.Background()

	// Test empty email address
	err = service.SendMagicLinkEmail(ctx, SendMagicLinkOptions{
		EmailAddress: "",
		MagicLink:    "https://example.com/link",
	})
	if err == nil {
		t.Error("Expected error for empty email address")
	}

	// Test empty magic link
	err = service.SendMagicLinkEmail(ctx, SendMagicLinkOptions{
		EmailAddress: "user@example.com",
		MagicLink:    "",
	})
	if err == nil {
		t.Error("Expected error for empty magic link")
	}

	// Test empty user ID
	err = service.ScheduleWelcomeEmail(ctx, User{
		ID:    "",
		Email: "user@example.com",
	})
	if err == nil {
		t.Error("Expected error for empty user ID")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		})())
}
