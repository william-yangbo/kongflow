// Package testutil provides shared testing utilities for workerqueue module
package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/services/workerqueue"
)

// EmailJobTemplate represents a template for creating test email jobs
type EmailJobTemplate struct {
	ToPattern      string // e.g., "user%d@example.com"
	SubjectPattern string // e.g., "Test Email %d"
	BodyPattern    string // e.g., "This is test email #%d"
	FromEmail      string
	JobKeyPattern  string // e.g., "test_job_%d"
}

// DefaultEmailJobTemplate returns a standard template for test email jobs
func DefaultEmailJobTemplate() EmailJobTemplate {
	return EmailJobTemplate{
		ToPattern:      "user%d@example.com",
		SubjectPattern: "Test Email %d",
		BodyPattern:    "This is test email number %d for integration testing",
		FromEmail:      "test@kongflow.dev",
		JobKeyPattern:  "test_job_%d",
	}
}

// CreateEmailJobs generates a slice of ScheduleEmailArgs for testing
func CreateEmailJobs(count int, template EmailJobTemplate) []workerqueue.ScheduleEmailArgs {
	jobs := make([]workerqueue.ScheduleEmailArgs, count)

	for i := 0; i < count; i++ {
		jobs[i] = workerqueue.ScheduleEmailArgs{
			To:      fmt.Sprintf(template.ToPattern, i+1),
			Subject: fmt.Sprintf(template.SubjectPattern, i+1),
			Body:    fmt.Sprintf(template.BodyPattern, i+1),
			From:    template.FromEmail,
			JobKey:  fmt.Sprintf(template.JobKeyPattern, i+1),
		}
	}

	return jobs
}

// CreateNamedEmailJobs creates email jobs with specific recipients and subjects
func CreateNamedEmailJobs() []workerqueue.ScheduleEmailArgs {
	return []workerqueue.ScheduleEmailArgs{
		{
			To:      "welcome@example.com",
			Subject: "Welcome to KongFlow",
			Body:    "Thank you for joining our platform!",
			From:    "welcome@kongflow.dev",
			JobKey:  "welcome_test",
		},
		{
			To:      "orders@example.com",
			Subject: "Order Confirmation",
			Body:    "Your order has been confirmed and will be processed soon.",
			From:    "orders@kongflow.dev",
			JobKey:  "order_test",
		},
		{
			To:      "support@example.com",
			Subject: "System Notification",
			Body:    "This is a system notification for testing purposes.",
			From:    "system@kongflow.dev",
			JobKey:  "system_test",
		},
	}
}

// WaitForJobCompletion waits for a specific number of jobs to complete with timeout
func WaitForJobCompletion(t *testing.T, checkFunc func() int, expectedCount int, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			actual := checkFunc()
			t.Fatalf("Timeout after %v waiting for %d jobs to complete. Got: %d",
				timeout, expectedCount, actual)
		case <-ticker.C:
			if actual := checkFunc(); actual >= expectedCount {
				t.Logf("Successfully completed %d jobs (expected: %d)", actual, expectedCount)
				return
			}
		}
	}
}

// TestEmailSender is a shared implementation for testing email sending
type TestEmailSender struct {
	SentEmails []workerqueue.EmailData
	logger     *slog.Logger
}

// NewTestEmailSender creates a new test email sender
func NewTestEmailSender(logger *slog.Logger) *TestEmailSender {
	if logger == nil {
		logger = slog.Default()
	}

	return &TestEmailSender{
		SentEmails: make([]workerqueue.EmailData, 0),
		logger:     logger,
	}
}

// SendEmail implements workerqueue.EmailSender interface
func (t *TestEmailSender) SendEmail(ctx context.Context, email workerqueue.EmailData) error {
	t.SentEmails = append(t.SentEmails, email)
	t.logger.Info("Test Email Sent",
		"to", email.To,
		"subject", email.Subject,
		"from", email.From)
	return nil
}

// GetSentCount returns the number of sent emails
func (t *TestEmailSender) GetSentCount() int {
	return len(t.SentEmails)
}

// Clear resets the sent emails list
func (t *TestEmailSender) Clear() {
	t.SentEmails = make([]workerqueue.EmailData, 0)
}

// GetEmailByRecipient returns the first email sent to a specific recipient
func (t *TestEmailSender) GetEmailByRecipient(to string) *workerqueue.EmailData {
	for _, email := range t.SentEmails {
		if email.To == to {
			return &email
		}
	}
	return nil
}

// VerifyEmailSent checks if an email was sent to a specific recipient with expected content
func (t *TestEmailSender) VerifyEmailSent(testing *testing.T, to, expectedSubject string) {
	testing.Helper()

	email := t.GetEmailByRecipient(to)
	require.NotNil(testing, email, "No email found for recipient: %s", to)
	require.Equal(testing, expectedSubject, email.Subject, "Subject mismatch for %s", to)
	require.Equal(testing, to, email.To, "Recipient mismatch")
}

// CreateDelayedEmailJob creates a delayed email job for testing
func CreateDelayedEmailJob(to, subject, body string) workerqueue.ScheduleEmailArgs {
	if to == "" {
		to = "delayed@example.com"
	}
	if subject == "" {
		subject = "Delayed Test Email"
	}
	if body == "" {
		body = "This email was scheduled with a delay for testing purposes"
	}

	return workerqueue.ScheduleEmailArgs{
		To:      to,
		Subject: subject,
		Body:    body,
		From:    "delayed@kongflow.dev",
		JobKey:  "delayed_test",
	}
}
