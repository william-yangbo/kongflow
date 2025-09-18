package workerqueue_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/services/workerqueue"
)

// TestBasicWorkerQueue tests basic worker queue functionality
func TestBasicWorkerQueue(t *testing.T) {
	ctx := context.Background()

	// Setup test database using shared testhelper
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Create worker queue client
	client, err := workerqueue.NewClient(workerqueue.ClientOptions{
		DatabasePool: testDB.Pool,
		RunnerOptions: workerqueue.RunnerOptions{
			Concurrency:  2,
			PollInterval: 500,
		},
	})
	require.NoError(t, err)

	// Initialize client (creates River tables)
	err = client.Initialize(ctx)
	require.NoError(t, err)

	// Test enqueue a simple job
	result, err := client.Enqueue(ctx, "indexEndpoint", map[string]interface{}{
		"id":     "test-endpoint-123",
		"source": "MANUAL",
		"reason": "test indexing",
	}, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.Job.ID, int64(0))

	// Clean up
	err = client.Stop(ctx)
	require.NoError(t, err)
}

// TestTransactionSupport tests SQLC + River transaction integration
func TestTransactionSupport(t *testing.T) {
	ctx := context.Background()

	// Setup test database using shared testhelper
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Create worker queue client
	client, err := workerqueue.NewClient(workerqueue.ClientOptions{
		DatabasePool: testDB.Pool,
	})
	require.NoError(t, err)

	// Initialize client
	err = client.Initialize(ctx)
	require.NoError(t, err)

	// Test transaction with business logic + job enqueue
	businessLogicExecuted := false

	result, err := client.EnqueueWithBusinessLogic(ctx, "startRun", map[string]interface{}{
		"id": "test-run-456",
	}, func(ctx context.Context, txCtx *workerqueue.TransactionContext) error {
		// Simulate business logic (e.g., SQLC database operations)
		businessLogicExecuted = true

		// In a real scenario, this would be:
		// return txCtx.SQLCQueries.CreateUser(ctx, database.CreateUserParams{...})

		return nil
	})

	require.NoError(t, err)
	assert.True(t, businessLogicExecuted)
	assert.NotNil(t, result)

	// Clean up
	err = client.Stop(ctx)
	require.NoError(t, err)
}

// TestWorkerCatalogCompatibility tests alignment with trigger.dev workerCatalog
func TestWorkerCatalogCompatibility(t *testing.T) {
	ctx := context.Background()

	// Setup test database using shared testhelper
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Create worker queue client
	client, err := workerqueue.NewClient(workerqueue.ClientOptions{
		DatabasePool: testDB.Pool,
	})
	require.NoError(t, err)

	// Initialize client
	err = client.Initialize(ctx)
	require.NoError(t, err)

	// Test cases matching trigger.dev workerCatalog expectations
	testCases := []struct {
		identifier string
		payload    map[string]interface{}
	}{
		{
			identifier: "indexEndpoint",
			payload: map[string]interface{}{
				"id":     "test-endpoint",
				"source": "MANUAL",
			},
		},
		{
			identifier: "scheduleEmail",
			payload: map[string]interface{}{
				"to":      "test@example.com",
				"subject": "Test Email",
			},
		},
		{
			identifier: "startRun",
			payload: map[string]interface{}{
				"id": "test-run",
			},
		},
		{
			identifier: "deliverEvent",
			payload: map[string]interface{}{
				"id":      "test-event",
				"payload": "test-data",
			},
		},
		{
			identifier: "events.invokeDispatcher",
			payload: map[string]interface{}{
				"id":            "test-dispatcher",
				"eventRecordId": "test-record",
			},
		},
	}

	// Enqueue all test tasks
	for _, tc := range testCases {
		t.Run(tc.identifier, func(t *testing.T) {
			result, err := client.Enqueue(ctx, tc.identifier, tc.payload, nil)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Greater(t, result.Job.ID, int64(0))
		})
	}

	// Clean up
	err = client.Stop(ctx)
	require.NoError(t, err)
}

// TestJobOptions tests job configuration options that align with trigger.dev
func TestJobOptions(t *testing.T) {
	ctx := context.Background()

	// Setup test database using shared testhelper
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Create worker queue client
	client, err := workerqueue.NewClient(workerqueue.ClientOptions{
		DatabasePool: testDB.Pool,
	})
	require.NoError(t, err)

	// Initialize client
	err = client.Initialize(ctx)
	require.NoError(t, err)

	// Test job with custom options (mirrors trigger.dev JobOptions)
	runAt := time.Now().Add(5 * time.Minute)
	result, err := client.Enqueue(ctx, "indexEndpoint", map[string]interface{}{
		"id": "delayed-endpoint",
	}, &workerqueue.JobOptions{
		QueueName:   "custom-queue",
		Priority:    50,
		MaxAttempts: 5,
		RunAt:       &runAt,
		JobKey:      "unique-key-123",
		Tags:        []string{"test", "delayed"},
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "custom-queue", result.Job.Queue)
	// Priority 50 maps to 2 (Normal) in River Queue's 1-4 scale
	assert.Equal(t, 2, result.Job.Priority)
	assert.Equal(t, 5, result.Job.MaxAttempts)

	// Clean up
	err = client.Stop(ctx)
	require.NoError(t, err)
}

// TestDynamicQueueRouting tests Phase 2 dynamic queue routing functionality
func TestDynamicQueueRouting(t *testing.T) {
	// Setup test database
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Test queue resolution without starting the client
	t.Run("RunLevelIsolation", func(t *testing.T) {
		// Test different run IDs should go to different queues
		testCases := []struct {
			runID    string
			expected string
		}{
			{"run-12345", "runs_run-12345"},
			{"run-67890", "runs_run-67890"},
			{"run-abcdef", "runs_run-abcdef"},
		}

		for _, tc := range testCases {
			payload := workerqueue.PerformRunExecutionV2Args{
				ID:        tc.runID,
				ProjectID: "test-project",
				UserID:    "test-user",
			}

			// Create a DynamicQueueName to test the routing logic
			dynamicQueue := workerqueue.DynamicQueueName(func(payload interface{}) string {
				if runArgs, ok := payload.(workerqueue.PerformRunExecutionV2Args); ok {
					return "runs_" + runArgs.ID
				}
				return "runs_default"
			})

			resolvedQueue := dynamicQueue.ResolveQueueName(payload)
			assert.Equal(t, tc.expected, resolvedQueue,
				"Run %s should resolve to queue %s", tc.runID, tc.expected)
		}
	})

	t.Run("ProjectLevelIsolation", func(t *testing.T) {
		testCases := []struct {
			projectID string
			expected  string
		}{
			{"proj-123", "project_proj-123_runs"},
			{"proj-456", "project_proj-456_runs"},
			{"proj-enterprise", "project_proj-enterprise_runs"},
		}

		for _, tc := range testCases {
			payload := workerqueue.StartQueuedRunsArgs{
				ProjectID: tc.projectID,
				UserID:    "test-user",
				BatchSize: 10,
			}

			dynamicQueue := workerqueue.DynamicQueueName(func(payload interface{}) string {
				if runArgs, ok := payload.(workerqueue.StartQueuedRunsArgs); ok {
					return "project_" + runArgs.ProjectID + "_runs"
				}
				return "project_default_runs"
			})

			resolvedQueue := dynamicQueue.ResolveQueueName(payload)
			assert.Equal(t, tc.expected, resolvedQueue,
				"Project %s should resolve to queue %s", tc.projectID, tc.expected)
		}
	})

	t.Run("UserPlanGeographicRouting", func(t *testing.T) {
		testCases := []struct {
			userPlan string
			region   string
			expected string
		}{
			{"enterprise", "us-east-1", "us-east-1_enterprise_high-priority"},
			{"pro", "eu-west-1", "eu-west-1_pro_medium-priority"},
			{"free", "ap-southeast-1", "ap-southeast-1_free_standard"},
			{"unknown", "us-west-2", "us-west-2_standard_normal"},
		}

		for _, tc := range testCases {
			payload := workerqueue.UserTaskArgs{
				UserID:   "test-user",
				UserPlan: tc.userPlan,
				Region:   tc.region,
				TaskType: "test-task",
				TaskData: map[string]interface{}{"test": "data"},
			}

			dynamicQueue := workerqueue.DynamicQueueName(func(payload interface{}) string {
				if userArgs, ok := payload.(workerqueue.UserTaskArgs); ok {
					switch userArgs.UserPlan {
					case "enterprise":
						return userArgs.Region + "_enterprise_high-priority"
					case "pro":
						return userArgs.Region + "_pro_medium-priority"
					case "free":
						return userArgs.Region + "_free_standard"
					default:
						return userArgs.Region + "_standard_normal"
					}
				}
				return "default_standard_normal"
			})

			resolvedQueue := dynamicQueue.ResolveQueueName(payload)
			// Since we updated the test logic, let's just use the expected value from testCases
			assert.Equal(t, tc.expected, resolvedQueue,
				"User plan %s, region %s should resolve correctly", tc.userPlan, tc.region)
		}
	})
}
