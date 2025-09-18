// Package testutil provides testing utilities for email service with real River worker queue
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/workerqueue"
)

// TestHarness provides a complete testing environment with real River worker queue
type TestHarness struct {
	// Database
	Container testcontainers.Container
	Pool      *pgxpool.Pool

	// Worker Queue
	Manager *workerqueue.Manager

	// Email Service
	EmailService   email.EmailService
	EmailProvider  *MockEmailProvider
	TemplateEngine *MockTemplateEngine

	// Test Email Sender (tracks sent emails)
	TestEmailSender *TestEmailSender

	// Test utilities
	Logger  *slog.Logger
	cleanup func()
}

// MockEmailProvider implements email.EmailProvider for testing
type MockEmailProvider struct {
	SentEmails []email.Email
}

func (m *MockEmailProvider) SendEmail(ctx context.Context, emailMsg email.Email) error {
	m.SentEmails = append(m.SentEmails, emailMsg)
	return nil
}

// MockTemplateEngine implements email.TemplateEngine for testing
type MockTemplateEngine struct{}

func (m *MockTemplateEngine) RenderTemplate(templateName string, data interface{}) (string, error) {
	switch templateName {
	case "welcome":
		return "<h1>Welcome!</h1><p>Welcome to KongFlow</p>", nil
	case "magic_link":
		return "<h1>Sign In</h1><p>Click to sign in</p>", nil
	case "invite":
		return "<h1>Invitation</h1><p>You've been invited</p>", nil
	case "connect_integration":
		return "<h1>Connect Integration</h1><p>Connect your integration</p>", nil
	case "workflow_failed":
		return "<h1>Workflow Failed</h1><p>Your workflow failed</p>", nil
	case "workflow_integration":
		return "<h1>Workflow Integration</h1><p>Workflow integration message</p>", nil
	default:
		return fmt.Sprintf("<h1>Test Template</h1><p>Template: %s</p>", templateName), nil
	}
}

// TestEmailSender implements workerqueue.EmailSender for email service integration
type TestEmailSender struct {
	EmailService email.EmailService
	SentEmails   []workerqueue.EmailData
}

func (t *TestEmailSender) SendEmail(ctx context.Context, emailData workerqueue.EmailData) error {
	// Record the email data for verification
	t.SentEmails = append(t.SentEmails, emailData)

	// Convert workerqueue.EmailData to email.DeliverEmail format
	// Map subject to appropriate template type
	templateType := "welcome" // Default
	var templateData interface{} = map[string]interface{}{
		"name": "Test User",
	}

	// Map different email types based on subject patterns to valid DeliverEmail types
	switch {
	case emailData.Subject == "Order Confirmation" || emailData.From == "orders@kongflow.dev":
		templateType = "invite" // Use valid email type
	case emailData.Subject == "System Notification" || emailData.From == "system@kongflow.dev":
		templateType = "workflow_failed" // Use valid email type
	case emailData.From == "delayed@kongflow.dev":
		templateType = "connect_integration" // Use valid email type
	default:
		templateType = "welcome"
	}

	data, _ := json.Marshal(templateData)
	deliverEmail := email.DeliverEmail{
		Email: templateType,
		To:    emailData.To,
		Data:  data,
	}

	return t.EmailService.SendEmail(ctx, deliverEmail)
}

// SetupTestHarness creates a complete testing environment with PostgreSQL container and River worker queue
func SetupTestHarness(t *testing.T) *TestHarness {
	t.Helper()

	ctx := context.Background()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "test_db",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	err = container.Start(ctx)
	require.NoError(t, err)

	// Get connection info
	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Build connection string
	connStr := fmt.Sprintf("postgres://test_user:test_password@%s:%s/test_db?sslmode=disable", host, port.Port())
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Test connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Setup email components
	emailProvider := &MockEmailProvider{SentEmails: make([]email.Email, 0)}
	templateEngine := &MockTemplateEngine{}

	emailConfig := email.EmailConfig{
		FromEmail:     "test@kongflow.dev",
		ReplyToEmail:  "noreply@kongflow.dev",
		ImagesBaseURL: "https://test.kongflow.dev",
	}

	emailService := email.New(emailProvider, templateEngine, emailConfig)

	// Setup test email sender that integrates with email service
	testEmailSender := &TestEmailSender{
		EmailService: emailService,
		SentEmails:   make([]workerqueue.EmailData, 0),
	}

	// Setup worker queue manager
	config := workerqueue.DefaultConfig()
	config.TestMode = true
	config.MaxWorkers = 2
	config.ExecutionMaxWorkers = 1
	config.EventsMaxWorkers = 1
	config.MaintenanceMaxWorkers = 1

	manager, err := workerqueue.NewManager(config, pool, logger, testEmailSender)
	require.NoError(t, err)

	// Ensure River tables are created
	err = manager.EnsureRiverTables(ctx)
	require.NoError(t, err)

	// Setup cleanup function
	cleanup := func() {
		if manager != nil {
			_ = manager.Stop(ctx)
		}
		if pool != nil {
			pool.Close()
		}
		if container != nil {
			_ = container.Terminate(ctx)
		}
	}

	return &TestHarness{
		Container:       container,
		Pool:            pool,
		Manager:         manager,
		EmailService:    emailService,
		EmailProvider:   emailProvider,
		TemplateEngine:  templateEngine,
		TestEmailSender: testEmailSender,
		Logger:          logger,
		cleanup:         cleanup,
	}
}

// Start starts the worker queue manager
func (th *TestHarness) Start(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	err := th.Manager.Start(ctx)
	require.NoError(t, err)

	// Wait for workers to be ready
	time.Sleep(1 * time.Second)
}

// Stop stops the worker queue manager
func (th *TestHarness) Stop(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	err := th.Manager.Stop(ctx)
	require.NoError(t, err)
}

// Cleanup cleans up all resources
func (th *TestHarness) Cleanup() {
	if th.cleanup != nil {
		th.cleanup()
	}
}

// WaitForEmailJobs waits for email jobs to be processed and returns the number of emails sent
func (th *TestHarness) WaitForEmailJobs(t *testing.T, expectedCount int, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)
			sentCount := len(th.EmailProvider.SentEmails)
			queuedCount := len(th.TestEmailSender.SentEmails)

			th.Logger.Error("Timeout waiting for emails",
				"expected", expectedCount,
				"sent_emails", sentCount,
				"queued_emails", queuedCount,
				"elapsed", elapsed,
				"timeout", timeout)

			t.Fatalf("Timeout after %v waiting for %d emails. Sent: %d, Queued: %d",
				elapsed, expectedCount, sentCount, queuedCount)
		case <-ticker.C:
			sentCount := len(th.EmailProvider.SentEmails)
			if sentCount >= expectedCount {
				elapsed := time.Since(startTime)
				th.Logger.Info("Email jobs completed",
					"expected", expectedCount,
					"actual", sentCount,
					"elapsed", elapsed)
				return
			}

			// Log progress every second
			if int(time.Since(startTime)/time.Second)%5 == 0 {
				queuedCount := len(th.TestEmailSender.SentEmails)
				th.Logger.Debug("Waiting for emails",
					"expected", expectedCount,
					"sent", sentCount,
					"queued", queuedCount,
					"elapsed", time.Since(startTime))
			}
		}
	}
}

// GetSentEmails returns all emails sent by the email provider
func (th *TestHarness) GetSentEmails() []email.Email {
	return th.EmailProvider.SentEmails
}

// GetQueuedEmails returns all emails queued through the worker queue
func (th *TestHarness) GetQueuedEmails() []workerqueue.EmailData {
	return th.TestEmailSender.SentEmails
}

// ClearEmails clears all recorded emails and waits briefly for any pending operations
func (th *TestHarness) ClearEmails() {
	th.EmailProvider.SentEmails = make([]email.Email, 0)
	th.TestEmailSender.SentEmails = make([]workerqueue.EmailData, 0)

	// Wait briefly to ensure any in-flight operations complete
	time.Sleep(100 * time.Millisecond)
}
