// Package email provides email template rendering functionality
package email

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed templates/*.html
var templatesFS embed.FS

// templateEngine implements the TemplateEngine interface
type templateEngine struct {
	templates map[string]*template.Template
}

// NewTemplateEngine creates a new template engine instance
func NewTemplateEngine() (TemplateEngine, error) {
	engine := &templateEngine{
		templates: make(map[string]*template.Template),
	}

	// Load all templates
	if err := engine.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return engine, nil
}

// RenderTemplate renders a template with the given data
func (t *templateEngine) RenderTemplate(templateName string, data interface{}) (string, error) {
	tmpl, exists := t.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// loadTemplates loads all email templates from the embedded filesystem
func (t *templateEngine) loadTemplates() error {
	// Template mapping from email type to template file
	templateFiles := map[string]string{
		string(EmailTypeMagicLink): "templates/magic_link.html",
		string(EmailTypeWelcome):   "templates/welcome.html",
		string(EmailTypeInvite):    "templates/invite.html",
		// Placeholder templates for remaining types (will be implemented later)
		string(EmailTypeConnectIntegration):  "",
		string(EmailTypeWorkflowFailed):      "",
		string(EmailTypeWorkflowIntegration): "",
	}

	// Load templates from embedded filesystem
	for emailType, filename := range templateFiles {
		if filename != "" {
			// Load from embedded file
			content, err := templatesFS.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read template file %s: %w", filename, err)
			}

			tmpl, err := template.New(emailType).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", emailType, err)
			}
			t.templates[emailType] = tmpl
		} else {
			// Use placeholder template for unimplemented types
			placeholderContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>KongFlow Notification</title></head>
<body>
    <h1>%s</h1>
    <p>This email template is not yet implemented.</p>
</body>
</html>`, emailType)

			tmpl, err := template.New(emailType).Parse(placeholderContent)
			if err != nil {
				return fmt.Errorf("failed to parse placeholder template %s: %w", emailType, err)
			}
			t.templates[emailType] = tmpl
		}
	}

	return nil
}
