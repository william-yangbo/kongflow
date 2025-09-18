// Package email debug tests
package email

import (
	"fmt"
	"testing"
)

func TestTemplateRendering_Debug(t *testing.T) {
	templateEngine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	// Debug magic link template
	magicLinkData := MagicLinkEmailData{
		MagicLink: "https://kongflow.dev/auth/verify?token=test123",
	}

	html, err := templateEngine.RenderTemplate(string(EmailTypeMagicLink), magicLinkData)
	if err != nil {
		t.Fatalf("Failed to render magic link template: %v", err)
	}

	fmt.Printf("Magic Link Template Output:\n%s\n\n", html)

	// Debug welcome template
	name := "Alice"
	welcomeData := WelcomeEmailData{
		Name: &name,
	}

	html, err = templateEngine.RenderTemplate(string(EmailTypeWelcome), welcomeData)
	if err != nil {
		t.Fatalf("Failed to render welcome template: %v", err)
	}

	fmt.Printf("Welcome Template Output:\n%s\n\n", html)
}