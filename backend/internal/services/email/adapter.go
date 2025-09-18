// Package email provides adapter for worker queue integration
package email

import (
	"context"
	"encoding/json"
	"fmt"

	"kongflow/backend/internal/services/workerqueue"
)

// WorkerQueueEmailAdapter adapts the email service to work with worker queue
// This implements the EmailSender interface from worker queue package
type WorkerQueueEmailAdapter struct {
	emailService EmailService
}

// NewWorkerQueueEmailAdapter creates a new adapter for worker queue integration
func NewWorkerQueueEmailAdapter(emailService EmailService) *WorkerQueueEmailAdapter {
	return &WorkerQueueEmailAdapter{
		emailService: emailService,
	}
}

// SendEmail implements the EmailSender interface for worker queue integration
// It converts the simplified workerqueue.EmailData to the full DeliverEmail format
func (a *WorkerQueueEmailAdapter) SendEmail(ctx context.Context, email workerqueue.EmailData) error {
	if a.emailService == nil {
		return fmt.Errorf("email service not configured")
	}

	// For worker queue emails, we'll create a simple welcome email format
	// since the worker queue uses a simplified interface
	welcomeData := WelcomeEmailData{
		Name: &email.Subject, // Use subject as name for simplicity
	}

	dataBytes, err := json.Marshal(welcomeData)
	if err != nil {
		return fmt.Errorf("failed to marshal email data: %w", err)
	}

	// Convert worker queue EmailData to our DeliverEmail format
	deliverEmail := DeliverEmail{
		Email: string(EmailTypeWelcome), // Use welcome type for worker queue emails
		To:    email.To,
		Data:  dataBytes,
	}

	// Use the email service to send the email
	return a.emailService.SendEmail(ctx, deliverEmail)
}

// GetEmailService returns the underlying email service
func (a *WorkerQueueEmailAdapter) GetEmailService() EmailService {
	return a.emailService
}
