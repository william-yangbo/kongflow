package workerqueue_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/services/workerqueue"
	"kongflow/backend/internal/services/workerqueue/testutil"
)

func TestEmailConsumerIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup test database using shared testhelper
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Setup shared test email sender
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	testEmailSender := testutil.NewTestEmailSender(logger)

	// Create worker queue manager with email consumer using DefaultConfig
	config := workerqueue.DefaultConfig()
	config.TestMode = true // Enable test mode for faster processing
	config.MaxWorkers = 2
	config.ExecutionMaxWorkers = 1
	config.EventsMaxWorkers = 1
	config.MaintenanceMaxWorkers = 1

	manager, err := workerqueue.NewManager(config, testDB.Pool, logger, testEmailSender)
	require.NoError(t, err)

	// Ensure River tables are created
	err = manager.EnsureRiverTables(ctx)
	require.NoError(t, err)

	// Start the manager (starts processing)
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, manager.Stop(ctx))
	}()

	// Wait for workers to be ready
	time.Sleep(1 * time.Second)

	t.Run("River Queue Consumer Processing", func(t *testing.T) {
		// Clear any previous emails
		testEmailSender.Clear()

		// Use shared email job creation
		emailJobs := testutil.CreateNamedEmailJobs()

		// Enqueue all jobs using River queue
		for i, emailJob := range emailJobs {
			result, err := manager.EnqueueJob(ctx, "scheduleEmail", emailJob, &workerqueue.JobOptions{
				Priority: 1, // High priority
				Tags:     []string{"test", "river-queue"},
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			slog.Info("River: Enqueued job", "index", i+1, "jobID", result.Job.ID, "to", emailJob.To)
		}

		// Wait for River workers to process jobs
		testutil.WaitForJobCompletion(t, testEmailSender.GetSentCount, len(emailJobs), 15*time.Second)

		// Verify all emails were sent by River workers
		assert.Equal(t, len(emailJobs), testEmailSender.GetSentCount())

		// Verify specific email content using shared utilities
		testEmailSender.VerifyEmailSent(t, "welcome@example.com", "Welcome to KongFlow")
		testEmailSender.VerifyEmailSent(t, "orders@example.com", "Order Confirmation")
		testEmailSender.VerifyEmailSent(t, "support@example.com", "System Notification")
	})

	t.Run("River Queue Delayed Job Scheduling", func(t *testing.T) {
		// Clear previous emails
		testEmailSender.Clear()

		// Create delayed email job using shared utility
		delayedEmailJob := testutil.CreateDelayedEmailJob("", "", "")

		// Enqueue with River's delayed execution
		delay := 3 * time.Second
		runAt := time.Now().Add(delay)
		result, err := manager.EnqueueJob(ctx, "scheduleEmail", delayedEmailJob, &workerqueue.JobOptions{
			Priority: 3, // Low priority
			RunAt:    &runAt,
			Tags:     []string{"test", "river-delayed"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		slog.Info("River: Enqueued delayed job", "jobID", result.Job.ID, "delay", delay)

		// Verify River doesn't execute job immediately
		time.Sleep(1 * time.Second)
		assert.Equal(t, 0, testEmailSender.GetSentCount(), "River should not execute delayed job immediately")

		// Wait for River's delayed job execution
		testutil.WaitForJobCompletion(t, testEmailSender.GetSentCount, 1, 10*time.Second)

		// Verify River processed the delayed job correctly
		assert.Equal(t, 1, testEmailSender.GetSentCount())
		testEmailSender.VerifyEmailSent(t, "delayed@example.com", "Delayed Test Email")
		slog.Info("River delayed job executed successfully!")
	})

	// Clean up
	err = manager.Stop(ctx)
	require.NoError(t, err)

	slog.Info("River Queue Integration Test Complete!")
	slog.Info("✅ River consumer processing")
	slog.Info("✅ River delayed job scheduling")
	slog.Info("✅ River database persistence")
	slog.Info("✅ River worker queue mechanics")
}
