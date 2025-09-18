// Package email integration tests
package email

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResendProvider_Integration(t *testing.T) {
	// This is an integration test that requires a real Resend API key
	// Skip if API key is not provided
	apiKey := "your-test-api-key-here"
	if apiKey == "your-test-api-key-here" {
		t.Skip("Skipping Resend integration test - no API key provided")
	}

	// Create Resend provider
	provider := NewResendProvider(apiKey)

	// Test email
	email := Email{
		From:    "test@example.com",
		To:      "recipient@example.com",
		Subject: "Test Email from KongFlow",
		HTML:    "<html><body><h1>Test Email</h1><p>This is a test email from KongFlow.</p></body></html>",
		Text:    "Test Email\n\nThis is a test email from KongFlow.",
		ReplyTo: "noreply@example.com",
	}

	// Send email
	ctx := context.Background()
	err := provider.SendEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to send email through Resend: %v", err)
	}

	t.Log("Email sent successfully through Resend")
}

func TestEmailService_EndToEnd(t *testing.T) {
	// Create email service with all components
	provider := &MockEmailProvider{}
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	config := EmailConfig{
		FromEmail:     "noreply@kongflow.dev",
		ReplyToEmail:  "help@kongflow.dev",
		ImagesBaseURL: "https://kongflow.dev",
	}

	service := New(provider, templateEngine, config)
	ctx := context.Background()

	// Test all core email types
	t.Run("MagicLink", func(t *testing.T) {
		err := service.SendMagicLinkEmail(ctx, SendMagicLinkOptions{
			EmailAddress: "user@example.com",
			MagicLink:    "https://kongflow.dev/auth/verify?token=abc123",
		})
		if err != nil {
			t.Fatalf("Failed to send magic link email: %v", err)
		}

		if len(provider.SentEmails) == 0 {
			t.Fatal("No emails were sent")
		}

		email := provider.SentEmails[len(provider.SentEmails)-1]
		if !contains(email.HTML, "🔑") {
			t.Error("Magic link email should contain lock emoji")
		}
		if !contains(email.HTML, "Sign in to KongFlow") {
			t.Error("Magic link email should contain sign in text")
		}
	})

	t.Run("Welcome", func(t *testing.T) {
		name := "John Doe"
		err := service.ScheduleWelcomeEmail(ctx, User{
			ID:    "user123",
			Email: "john@example.com",
			Name:  &name,
		})
		if err != nil {
			t.Fatalf("Failed to schedule welcome email: %v", err)
		}

		email := provider.SentEmails[len(provider.SentEmails)-1]
		if !contains(email.HTML, "✨") {
			t.Error("Welcome email should contain sparkles emoji")
		}
		if !contains(email.HTML, "Welcome to KongFlow") {
			t.Error("Welcome email should contain welcome text")
		}
		if !contains(email.HTML, name) {
			t.Errorf("Welcome email should contain user name: %s", name)
		}
	})

	t.Run("Invite", func(t *testing.T) {
		inviteData := DeliverEmail{
			Email: string(EmailTypeInvite),
			To:    "invited@example.com",
		}

		// Marshal invite data
		inviteEmailData := InviteEmailData{
			OrgName:      "Acme Corp",
			InviterName:  stringPtr("Jane Smith"),
			InviterEmail: "jane@acme.com",
			InviteLink:   "https://kongflow.dev/invite?token=xyz789",
		}

		data, err := json.Marshal(inviteEmailData)
		if err != nil {
			t.Fatalf("Failed to marshal invite data: %v", err)
		}
		inviteData.Data = data

		err = service.SendEmail(ctx, inviteData)
		if err != nil {
			t.Fatalf("Failed to send invite email: %v", err)
		}

		email := provider.SentEmails[len(provider.SentEmails)-1]
		if !contains(email.HTML, "🎉") {
			t.Error("Invite email should contain celebration emoji")
		}
		if !contains(email.HTML, "Acme Corp") {
			t.Error("Invite email should contain organization name")
		}
		if !contains(email.HTML, "Jane Smith") {
			t.Error("Invite email should contain inviter name")
		}
	})
}

func TestTemplateRendering_Quality(t *testing.T) {
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	// Test magic link template with real data
	magicLinkData := MagicLinkEmailData{
		MagicLink: "https://kongflow.dev/auth/verify?token=test123",
	}

	html, err := templateEngine.RenderTemplate(string(EmailTypeMagicLink), magicLinkData)
	if err != nil {
		t.Fatalf("Failed to render magic link template: %v", err)
	}

	// Verify template quality
	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html lang=\"en\">",
		"🔑 Sign in to KongFlow",
		"Security Notice",
		"https://kongflow.dev/auth/verify?token=test123",
	}

	for _, element := range expectedElements {
		if !contains(html, element) {
			t.Errorf("Magic link template missing expected element: %s", element)
		}
	}

	// Test welcome template with user name
	name := "Alice Wonderland"
	welcomeData := WelcomeEmailData{
		Name: &name,
	}

	html, err = templateEngine.RenderTemplate(string(EmailTypeWelcome), welcomeData)
	if err != nil {
		t.Fatalf("Failed to render welcome template: %v", err)
	}

	expectedElements = []string{
		"Welcome to KongFlow!",
		"Hi Alice Wonderland,",
		"Create Your First Workflow",
		"Browse Templates",
	}

	for _, element := range expectedElements {
		if !contains(html, element) {
			t.Errorf("Welcome template missing expected element: %s", element)
		}
	}

	// Test welcome template without user name
	welcomeDataNoName := WelcomeEmailData{
		Name: nil,
	}

	html, err = templateEngine.RenderTemplate(string(EmailTypeWelcome), welcomeDataNoName)
	if err != nil {
		t.Fatalf("Failed to render welcome template without name: %v", err)
	}

	if !contains(html, "Hi there,") {
		t.Error("Welcome template should contain 'Hi there,' when no name provided")
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
