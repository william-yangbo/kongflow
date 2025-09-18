// Package email provides email sending functionality that strictly aligns with trigger.dev's email service
// Replicates trigger.dev's email.server.ts functionality for perfect compatibility with Go best practices.
package email

import (
	"context"
	"encoding/json"
	"time"

	"kongflow/backend/internal/services/workerqueue"
)

// DeliverEmailType represents the discriminated union type from trigger.dev
// Strictly aligned with trigger.dev's DeliverEmail email field values
type DeliverEmailType string

const (
	EmailTypeMagicLink           DeliverEmailType = "magic_link"
	EmailTypeWelcome             DeliverEmailType = "welcome"
	EmailTypeInvite              DeliverEmailType = "invite"
	EmailTypeConnectIntegration  DeliverEmailType = "connect_integration"
	EmailTypeWorkflowFailed      DeliverEmailType = "workflow_failed"
	EmailTypeWorkflowIntegration DeliverEmailType = "workflow_integration"
)

// DeliverEmail represents the email data structure from trigger.dev
// Strictly aligned with trigger.dev's DeliverEmail discriminated union type
type DeliverEmail struct {
	Email string          `json:"email" validate:"required,oneof=magic_link welcome invite connect_integration workflow_failed workflow_integration"`
	To    string          `json:"to" validate:"required,email"`
	Data  json.RawMessage `json:"data"` // Stores the specific email type data
}

// MagicLinkEmailData represents data for magic link emails
// Aligned with trigger.dev's magic link email implementation
type MagicLinkEmailData struct {
	MagicLink string `json:"magicLink" validate:"required,url"`
}

// WelcomeEmailData represents data for welcome emails
// Aligned with trigger.dev's welcome email implementation
type WelcomeEmailData struct {
	Name *string `json:"name,omitempty"`
}

// InviteEmailData represents data for invite emails
// Aligned with trigger.dev's invite email implementation
type InviteEmailData struct {
	OrgName      string  `json:"orgName" validate:"required"`
	InviterName  *string `json:"inviterName,omitempty"`
	InviterEmail string  `json:"inviterEmail" validate:"required,email"`
	InviteLink   string  `json:"inviteLink" validate:"required,url"`
}

// ConnectIntegrationEmailData represents data for integration connection emails
type ConnectIntegrationEmailData struct {
	IntegrationName string `json:"integrationName" validate:"required"`
	ConnectLink     string `json:"connectLink" validate:"required,url"`
}

// WorkflowFailedEmailData represents data for workflow failure notifications
type WorkflowFailedEmailData struct {
	WorkflowName  string `json:"workflowName" validate:"required"`
	FailureReason string `json:"failureReason" validate:"required"`
	WorkflowLink  string `json:"workflowLink" validate:"required,url"`
}

// WorkflowIntegrationEmailData represents data for workflow integration emails
type WorkflowIntegrationEmailData struct {
	WorkflowName    string `json:"workflowName" validate:"required"`
	IntegrationName string `json:"integrationName" validate:"required"`
}

// SendMagicLinkOptions represents options for sending magic link emails
// Aligned with trigger.dev's SendEmailOptions<AuthUser> interface
type SendMagicLinkOptions struct {
	EmailAddress string `json:"emailAddress" validate:"required,email"`
	MagicLink    string `json:"magicLink" validate:"required,url"`
}

// User represents user data for email operations
// Aligned with trigger.dev's User type
type User struct {
	ID    string  `json:"id" validate:"required"`
	Email string  `json:"email" validate:"required,email"`
	Name  *string `json:"name,omitempty"`
}

// EmailConfig represents configuration for the email service
type EmailConfig struct {
	ResendAPIKey  string `env:"RESEND_API_KEY" validate:"required"`
	FromEmail     string `env:"FROM_EMAIL" validate:"required,email"`
	ReplyToEmail  string `env:"REPLY_TO_EMAIL" validate:"required,email"`
	ImagesBaseURL string `env:"APP_ORIGIN" validate:"required,url"`
}

// Email represents an email to be sent through a provider
type Email struct {
	From    string            `json:"from"`
	To      string            `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html"`
	Text    string            `json:"text,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	ReplyTo string            `json:"reply_to,omitempty"`
}

// EmailService defines the main email service interface
// Strictly aligned with trigger.dev's email.server.ts exported functions
type EmailService interface {
	// SendMagicLinkEmail sends a magic link authentication email
	// Aligned with trigger.dev's sendMagicLinkEmail function
	SendMagicLinkEmail(ctx context.Context, options SendMagicLinkOptions) error

	// ScheduleWelcomeEmail schedules a welcome email with environment-based delay
	// Aligned with trigger.dev's scheduleWelcomeEmail function
	ScheduleWelcomeEmail(ctx context.Context, user User) error

	// ScheduleEmail schedules an email with optional delay
	// Aligned with trigger.dev's scheduleEmail function
	ScheduleEmail(ctx context.Context, data DeliverEmail, delay *time.Duration) error

	// SendEmail sends an email immediately
	// Aligned with trigger.dev's sendEmail function
	SendEmail(ctx context.Context, data DeliverEmail) error

	// SetWorkerQueue configures the worker queue client for delayed email processing
	SetWorkerQueue(workerQueue *workerqueue.Client)
}

// EmailProvider defines the interface for email sending providers
type EmailProvider interface {
	SendEmail(ctx context.Context, email Email) error
}

// TemplateEngine defines the interface for email template rendering
type TemplateEngine interface {
	RenderTemplate(templateName string, data interface{}) (string, error)
}
