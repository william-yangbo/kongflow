package email_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/services/email"
	"kongflow/backend/internal/services/email/testutil"
	"kongflow/backend/internal/services/workerqueue"
	workertestutil "kongflow/backend/internal/services/workerqueue/testutil"
)

// TestRealWorkerQueueIntegration tests email service with real River worker queue
// This test uses Testcontainers to provide a real PostgreSQL database and River queue
func TestRealWorkerQueueIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real worker queue in short mode")
	}

	// Setup test harness with real PostgreSQL and River worker queue
	harness := testutil.SetupTestHarness(t)
	defer harness.Cleanup()

	// Start the worker queue
	harness.Start(t)
	defer harness.Stop(t)

	ctx := context.Background()

	t.Run("EndToEnd_SingleEmailWithTemplate", func(t *testing.T) {
		// Clear any previous emails
		harness.ClearEmails()

		// Use shared utility to create email job
		emailJob := workerqueue.ScheduleEmailArgs{
			To:      "test@example.com",
			Subject: "Real Queue Test",
			Body:    "This email was processed by real River worker queue",
			From:    "welcome@kongflow.dev", // Use welcome sender to get correct template
			JobKey:  "single_test",          // Different JobKey to avoid conflicts
		}

		// Enqueue the job using the manager
		jobResult, err := harness.Manager.EnqueueJob(ctx, "scheduleEmail", emailJob, &workerqueue.JobOptions{
			Priority: 1,
			Tags:     []string{"test", "end-to-end"},
		})
		require.NoError(t, err)
		require.NotNil(t, jobResult)
		assert.Greater(t, jobResult.Job.ID, int64(0))

		// Wait for the job to be processed
		harness.WaitForEmailJobs(t, 1, 10*time.Second)

		// Verify email was sent through the email service (with template processing)
		sentEmails := harness.GetSentEmails()
		require.Len(t, sentEmails, 1)
		assert.Equal(t, "test@example.com", sentEmails[0].To)
		assert.Equal(t, "Welcome to KongFlow", sentEmails[0].Subject) // Template processed

		// Verify email was queued through worker queue (original data)
		queuedEmails := harness.GetQueuedEmails()
		require.Len(t, queuedEmails, 1)
		assert.Equal(t, "test@example.com", queuedEmails[0].To)
		assert.Equal(t, "Real Queue Test", queuedEmails[0].Subject) // Original subject
	})

	t.Run("EndToEnd_MultipleEmailTemplates", func(t *testing.T) {
		// Clear any previous emails
		harness.ClearEmails()

		// Use shared utility to create multiple email jobs
		emailJobs := workertestutil.CreateNamedEmailJobs()

		// Enqueue all jobs
		jobResults := make([]*rivertype.JobInsertResult, 0, len(emailJobs))
		for _, job := range emailJobs {
			result, err := harness.Manager.EnqueueJob(ctx, "scheduleEmail", job, &workerqueue.JobOptions{
				Priority: 1,
				Tags:     []string{"test", "end-to-end-bulk"},
			})
			require.NoError(t, err)
			jobResults = append(jobResults, result)
		}

		// Verify all jobs were enqueued
		for _, result := range jobResults {
			assert.Greater(t, result.Job.ID, int64(0))
		}

		// Wait for all jobs to be processed
		harness.WaitForEmailJobs(t, len(emailJobs), 15*time.Second)

		// Verify all emails were sent through email service (with templates)
		sentEmails := harness.GetSentEmails()
		require.Len(t, sentEmails, len(emailJobs))

		queuedEmails := harness.GetQueuedEmails()
		require.Len(t, queuedEmails, len(emailJobs))

		// Verify email contents using shared utilities
		emailMap := make(map[string]email.Email)
		for _, emailObj := range sentEmails {
			emailMap[emailObj.To] = emailObj
		}

		assert.Contains(t, emailMap, "welcome@example.com")
		assert.Contains(t, emailMap, "orders@example.com")
		assert.Contains(t, emailMap, "support@example.com")
	})

	t.Run("EndToEnd_DelayedEmailProcessing", func(t *testing.T) {
		// Clear any previous emails
		harness.ClearEmails()

		// Use shared utility to create delayed email job
		emailJob := workertestutil.CreateDelayedEmailJob("delayed-endtoend@example.com", "End-to-End Delayed Test",
			"This tests the complete pipeline with a delayed email")

		// Enqueue job with proper delay using RunAt option
		delay := 500 * time.Millisecond // Shorter delay for testing
		runAt := time.Now().Add(delay)
		jobResult, err := harness.Manager.EnqueueJob(ctx, "scheduleEmail", emailJob, &workerqueue.JobOptions{
			Priority: 1,
			RunAt:    &runAt, // This enables delayed execution
			Tags:     []string{"test", "end-to-end-delayed"},
		})
		require.NoError(t, err)
		require.NotNil(t, jobResult)
		assert.Greater(t, jobResult.Job.ID, int64(0))

		// Verify job is not processed immediately
		time.Sleep(100 * time.Millisecond)
		sentEmails := harness.GetSentEmails()
		assert.Equal(t, 0, len(sentEmails), "Delayed job should not be processed immediately")

		// Wait for the delayed job to be processed (total wait: delay + processing time)
		harness.WaitForEmailJobs(t, 1, delay+10*time.Second)

		// Verify email was sent through complete pipeline
		sentEmails = harness.GetSentEmails()
		require.Len(t, sentEmails, 1)
		assert.Equal(t, "delayed-endtoend@example.com", sentEmails[0].To)

		// Also verify queued emails to ensure worker queue processed it
		queuedEmails := harness.GetQueuedEmails()
		require.Len(t, queuedEmails, 1)
		assert.Equal(t, "delayed-endtoend@example.com", queuedEmails[0].To)

		harness.Logger.Info("End-to-end delayed processing completed",
			"recipient", sentEmails[0].To,
			"subject", sentEmails[0].Subject)
	})

	t.Run("EndToEnd_EmailTemplateProcessing", func(t *testing.T) {
		// Clear any previous emails
		harness.ClearEmails()

		// Test complete pipeline: WorkerQueue → Email Service → Template Engine → Email Provider
		welcomeData := email.WelcomeEmailData{
			Name: stringPtr("John Doe"),
		}

		data, err := json.Marshal(welcomeData)
		require.NoError(t, err)

		deliverEmail := email.DeliverEmail{
			Email: string(email.EmailTypeWelcome),
			To:    "john@example.com",
			Data:  data,
		}

		// Send email directly through email service to test complete template processing
		err = harness.EmailService.SendEmail(ctx, deliverEmail)
		require.NoError(t, err)

		// Verify template was processed through entire pipeline
		sentEmails := harness.GetSentEmails()
		require.Len(t, sentEmails, 1)
		assert.Equal(t, "john@example.com", sentEmails[0].To)
		assert.Equal(t, "Welcome to KongFlow", sentEmails[0].Subject)
		// The mock template engine returns specific HTML for welcome template
		assert.Contains(t, sentEmails[0].HTML, "Welcome!")

		harness.Logger.Info("Template processing pipeline completed successfully")
	})

	t.Run("EndToEnd_HighConcurrencyPerformance", func(t *testing.T) {
		// Clear any previous emails
		harness.ClearEmails()

		// Use shared template to generate high-concurrency test emails
		template := workertestutil.DefaultEmailJobTemplate()
		template.ToPattern = "bulk%d@example.com"
		template.SubjectPattern = "Performance Test %d"
		template.BodyPattern = "This is performance test email number %d"

		numJobs := 20
		emailJobs := workertestutil.CreateEmailJobs(numJobs, template)

		// Enqueue all jobs concurrently to test complete system performance
		startTime := time.Now()
		for _, job := range emailJobs {
			_, err := harness.Manager.EnqueueJob(ctx, "scheduleEmail", job, &workerqueue.JobOptions{
				Priority: 1,
				Tags:     []string{"test", "performance"},
			})
			require.NoError(t, err)
		}
		enqueueTime := time.Since(startTime)

		// Wait for all jobs to be processed through complete pipeline
		harness.WaitForEmailJobs(t, numJobs, 30*time.Second)
		totalTime := time.Since(startTime)

		// Verify all emails were processed through entire system
		sentEmails := harness.GetSentEmails()
		require.Len(t, sentEmails, numJobs)

		queuedEmails := harness.GetQueuedEmails()
		require.Len(t, queuedEmails, numJobs)

		// Performance assertions for end-to-end processing
		harness.Logger.Info("End-to-end performance metrics",
			"enqueue_time", enqueueTime,
			"total_processing_time", totalTime,
			"jobs_per_second", float64(numJobs)/totalTime.Seconds(),
			"pipeline", "WorkerQueue→EmailService→TemplateEngine→EmailProvider",
		)

		assert.Less(t, totalTime, 30*time.Second, "Complete pipeline should process all jobs within 30 seconds")
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
