// Package email provides the main email service implementation
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/go-playground/validator/v10"
)

// emailService implements the EmailService interface
// Strictly aligned with trigger.dev's email.server.ts implementation
type emailService struct {
	provider       EmailProvider
	templateEngine TemplateEngine
	config         EmailConfig
	validator      *validator.Validate
	workerQueue    *workerqueue.Client
}

// New creates a new email service instance
func New(provider EmailProvider, templateEngine TemplateEngine, config EmailConfig) EmailService {
	return &emailService{
		provider:       provider,
		templateEngine: templateEngine,
		config:         config,
		validator:      validator.New(),
		workerQueue:    nil, // Will be set via SetWorkerQueue
	}
}

// NewWithWorkerQueue creates a new email service instance with worker queue support
func NewWithWorkerQueue(provider EmailProvider, templateEngine TemplateEngine, config EmailConfig, workerQueue *workerqueue.Client) EmailService {
	return &emailService{
		provider:       provider,
		templateEngine: templateEngine,
		config:         config,
		validator:      validator.New(),
		workerQueue:    workerQueue,
	}
}

// SetWorkerQueue sets the worker queue client for delayed email processing
func (s *emailService) SetWorkerQueue(workerQueue *workerqueue.Client) {
	s.workerQueue = workerQueue
}

// SendMagicLinkEmail sends a magic link authentication email
// Aligned with trigger.dev's sendMagicLinkEmail function
func (s *emailService) SendMagicLinkEmail(ctx context.Context, options SendMagicLinkOptions) error {
	// Validate options using validator
	if err := s.validator.Struct(options); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create magic link email data
	emailData := DeliverEmail{
		Email: string(EmailTypeMagicLink),
		To:    options.EmailAddress,
	}

	// Marshal magic link data
	magicLinkData := MagicLinkEmailData{
		MagicLink: options.MagicLink,
	}

	// Validate magic link data
	if err := s.validator.Struct(magicLinkData); err != nil {
		return fmt.Errorf("magic link data validation failed: %w", err)
	}

	data, err := json.Marshal(magicLinkData)
	if err != nil {
		return fmt.Errorf("failed to marshal magic link data: %w", err)
	}
	emailData.Data = data

	// Send email immediately (aligned with trigger.dev behavior)
	return s.SendEmail(ctx, emailData)
}

// ScheduleWelcomeEmail schedules a welcome email with environment-based delay
// Aligned with trigger.dev's scheduleWelcomeEmail function
func (s *emailService) ScheduleWelcomeEmail(ctx context.Context, user User) error {
	// Validate user using validator
	if err := s.validator.Struct(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Create welcome email data
	emailData := DeliverEmail{
		Email: string(EmailTypeWelcome),
		To:    user.Email,
	}

	// Marshal welcome data
	welcomeData := WelcomeEmailData{
		Name: user.Name,
	}

	// Validate welcome data
	if err := s.validator.Struct(welcomeData); err != nil {
		return fmt.Errorf("welcome data validation failed: %w", err)
	}

	data, err := json.Marshal(welcomeData)
	if err != nil {
		return fmt.Errorf("failed to marshal welcome data: %w", err)
	}
	emailData.Data = data

	// Calculate delay based on environment (aligned with trigger.dev logic)
	// Development: 1 minute, Production: 22 minutes
	var delay time.Duration
	if isDevelopment() {
		delay = 1 * time.Minute
	} else {
		delay = 22 * time.Minute
	}

	return s.ScheduleEmail(ctx, emailData, &delay)
}

// ScheduleEmail schedules an email with optional delay
// Aligned with trigger.dev's scheduleEmail function
func (s *emailService) ScheduleEmail(ctx context.Context, data DeliverEmail, delay *time.Duration) error {
	// Validate email data using validator
	if err := s.validator.Struct(data); err != nil {
		return fmt.Errorf("email data validation failed: %w", err)
	}

	// For immediate sending or short delays, send directly
	if delay == nil || *delay <= 1*time.Minute {
		return s.SendEmail(ctx, data)
	}

	// Use worker queue for delayed sending if available
	if s.workerQueue != nil {
		// First render the email template to get the final HTML
		html, err := s.renderEmailTemplate(data)
		if err != nil {
			return fmt.Errorf("failed to render email template: %w", err)
		}

		// Create schedule email args for worker queue
		scheduleArgs := workerqueue.ScheduleEmailArgs{
			To:      data.To,
			Subject: s.getEmailSubject(DeliverEmailType(data.Email)),
			Body:    html,
			From:    s.config.FromEmail,
			JobKey:  fmt.Sprintf("email_%s_%s", data.Email, data.To), // Unique key for deduplication
		}

		// Calculate when to run the job
		runAt := time.Now().Add(*delay)

		// Enqueue the job with the specified delay
		_, err = s.workerQueue.Enqueue(ctx, "scheduleEmail", scheduleArgs, &workerqueue.JobOptions{
			RunAt: &runAt,
		})

		if err != nil {
			return fmt.Errorf("failed to enqueue email job: %w", err)
		}

		return nil
	}

	// Fallback: send immediately if worker queue is not available
	// This ensures the email service works even without worker queue setup
	return s.SendEmail(ctx, data)
}

// SendEmail sends an email immediately
// Aligned with trigger.dev's sendEmail function
func (s *emailService) SendEmail(ctx context.Context, data DeliverEmail) error {
	// Validate email data using validator
	if err := s.validator.Struct(data); err != nil {
		return fmt.Errorf("email data validation failed: %w", err)
	}

	// Render email template
	html, err := s.renderEmailTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Create email object
	email := Email{
		From:    s.config.FromEmail,
		To:      data.To,
		Subject: s.getEmailSubject(DeliverEmailType(data.Email)),
		HTML:    html,
		ReplyTo: s.config.ReplyToEmail,
	}

	// Send through provider
	if err := s.provider.SendEmail(ctx, email); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// renderEmailTemplate renders the appropriate template for the email type
func (s *emailService) renderEmailTemplate(data DeliverEmail) (string, error) {
	emailType := DeliverEmailType(data.Email)

	// Parse the specific email data based on type
	var templateData interface{}
	var err error

	switch emailType {
	case EmailTypeMagicLink:
		var magicLinkData MagicLinkEmailData
		err = json.Unmarshal(data.Data, &magicLinkData)
		templateData = magicLinkData
	case EmailTypeWelcome:
		var welcomeData WelcomeEmailData
		err = json.Unmarshal(data.Data, &welcomeData)
		templateData = welcomeData
	case EmailTypeInvite:
		var inviteData InviteEmailData
		err = json.Unmarshal(data.Data, &inviteData)
		templateData = inviteData
	case EmailTypeConnectIntegration:
		var connectData ConnectIntegrationEmailData
		err = json.Unmarshal(data.Data, &connectData)
		templateData = connectData
	case EmailTypeWorkflowFailed:
		var workflowFailedData WorkflowFailedEmailData
		err = json.Unmarshal(data.Data, &workflowFailedData)
		templateData = workflowFailedData
	case EmailTypeWorkflowIntegration:
		var workflowIntegrationData WorkflowIntegrationEmailData
		err = json.Unmarshal(data.Data, &workflowIntegrationData)
		templateData = workflowIntegrationData
	default:
		return "", fmt.Errorf("unsupported email type: %s", emailType)
	}

	if err != nil {
		return "", fmt.Errorf("failed to unmarshal email data: %w", err)
	}

	// Render template
	return s.templateEngine.RenderTemplate(string(emailType), templateData)
}

// getEmailSubject returns the appropriate subject for each email type
// Aligned with trigger.dev's email subjects
func (s *emailService) getEmailSubject(emailType DeliverEmailType) string {
	switch emailType {
	case EmailTypeMagicLink:
		return "Sign in to KongFlow"
	case EmailTypeWelcome:
		return "Welcome to KongFlow"
	case EmailTypeInvite:
		return "You've been invited to join KongFlow"
	case EmailTypeConnectIntegration:
		return "Connect your integration"
	case EmailTypeWorkflowFailed:
		return "Workflow failed"
	case EmailTypeWorkflowIntegration:
		return "Workflow integration update"
	default:
		return "KongFlow notification"
	}
}

// isDevelopment checks if we're in development environment
// Aligned with trigger.dev's environment detection logic
func isDevelopment() bool {
	// TODO: Implement proper environment detection
	// This should check environment variables like NODE_ENV equivalent
	return true // Default to development for now
}
