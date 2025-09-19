// Package queue provides real integration testing for endpoints queue service
package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealEndpointQueueIntegration tests endpoint queue service with real River worker queue and PostgreSQL
// This test uses TestContainers to provide a real PostgreSQL database and River queue
func TestRealEndpointQueueIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real endpoint queue in short mode")
	}

	// Setup test harness with real PostgreSQL and River worker queue
	harness := SetupEndpointQueueTestHarness(t)
	defer harness.Cleanup()

	// Start the worker queue
	harness.Start(t)
	defer harness.Stop(t)

	ctx := context.Background()

	t.Run("EndToEnd_SingleEndpointIndexing", func(t *testing.T) {
		// Clear any previous operations
		harness.ClearIndexOperations()

		endpointID := uuid.New()
		req := EnqueueIndexEndpointRequest{
			EndpointID: endpointID,
			Source:     EndpointIndexSourceAPI,
			Reason:     "Real queue integration test",
			SourceData: map[string]interface{}{
				"test":        true,
				"integration": "real-queue",
			},
		}

		// Enqueue the indexing job using the queue service
		result, err := harness.QueueService.EnqueueIndexEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Greater(t, result.Job.ID, int64(0))

		harness.Logger.Info("Enqueued endpoint indexing job",
			"job_id", result.Job.ID,
			"endpoint_id", endpointID,
			"source", req.Source)

		// Wait for the job to be processed
		harness.WaitForIndexOperations(t, 1, 10*time.Second)

		// Verify operation was recorded by our mock indexer
		operations := harness.GetIndexedOperations()
		require.Len(t, operations, 1)

		op := operations[0]
		assert.Equal(t, endpointID.String(), op.EndpointID)
		assert.Equal(t, string(EndpointIndexSourceAPI), op.Source)
		assert.Equal(t, "Real queue integration test", op.Reason)
		assert.Equal(t, "success", op.Status)
		assert.True(t, op.SourceData["test"].(bool))
		assert.Equal(t, "real-queue", op.SourceData["integration"])
	})

	t.Run("EndToEnd_MultipleEndpointOperations", func(t *testing.T) {
		// Clear any previous operations
		harness.ClearIndexOperations()

		// Create multiple endpoint operations
		endpointOperations := []struct {
			endpointID uuid.UUID
			source     EndpointIndexSource
			reason     string
		}{
			{uuid.New(), EndpointIndexSourceInternal, "Automatic indexing"},
			{uuid.New(), EndpointIndexSourceAPI, "API triggered indexing"},
			{uuid.New(), EndpointIndexSourceHook, "Webhook triggered indexing"},
			{uuid.New(), EndpointIndexSourceManual, "Manual indexing request"},
		}

		// Enqueue all operations
		for i, op := range endpointOperations {
			req := EnqueueIndexEndpointRequest{
				EndpointID: op.endpointID,
				Source:     op.source,
				Reason:     op.reason,
				SourceData: map[string]interface{}{
					"batch_index": i,
					"batch_id":    "multi-endpoint-test",
				},
			}

			result, err := harness.QueueService.EnqueueIndexEndpoint(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, result)

			harness.Logger.Info("Enqueued batch endpoint operation",
				"job_id", result.Job.ID,
				"endpoint_id", op.endpointID,
				"source", op.source,
				"batch_index", i)
		}

		// Wait for all operations to be processed
		harness.WaitForIndexOperations(t, len(endpointOperations), 15*time.Second)

		// Verify all operations were processed
		operations := harness.GetIndexedOperations()
		require.Len(t, operations, len(endpointOperations))

		// Verify each operation
		operationMap := make(map[string]EndpointIndexOperation)
		for _, op := range operations {
			operationMap[op.EndpointID] = op
		}

		for _, expectedOp := range endpointOperations {
			actualOp, exists := operationMap[expectedOp.endpointID.String()]
			require.True(t, exists, "Operation for endpoint %s should exist", expectedOp.endpointID)
			assert.Equal(t, string(expectedOp.source), actualOp.Source)
			assert.Equal(t, expectedOp.reason, actualOp.Reason)
			assert.Equal(t, "success", actualOp.Status)
			assert.Equal(t, "multi-endpoint-test", actualOp.SourceData["batch_id"])
		}
	})

	t.Run("EndToEnd_DelayedEndpointIndexing", func(t *testing.T) {
		// Clear any previous operations
		harness.ClearIndexOperations()

		endpointID := uuid.New()
		delay := 500 * time.Millisecond
		runAt := time.Now().Add(delay)

		req := EnqueueIndexEndpointRequest{
			EndpointID: endpointID,
			Source:     EndpointIndexSourceInternal,
			Reason:     "Delayed indexing test",
			RunAt:      &runAt,
			Priority:   1,
			SourceData: map[string]interface{}{
				"delayed": true,
				"delay":   delay.String(),
			},
		}

		// Enqueue the delayed job
		result, err := harness.QueueService.EnqueueIndexEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		harness.Logger.Info("Enqueued delayed endpoint indexing job",
			"job_id", result.Job.ID,
			"endpoint_id", endpointID,
			"delay", delay)

		// Verify job is not processed immediately
		time.Sleep(100 * time.Millisecond)
		operations := harness.GetIndexedOperations()
		assert.Equal(t, 0, len(operations), "Delayed job should not be processed immediately")

		// Wait for the delayed job to be processed
		harness.WaitForIndexOperations(t, 1, delay+10*time.Second)

		// Verify operation was processed
		operations = harness.GetIndexedOperations()
		require.Len(t, operations, 1)

		op := operations[0]
		assert.Equal(t, endpointID.String(), op.EndpointID)
		assert.Equal(t, "Delayed indexing test", op.Reason)
		assert.Equal(t, "success", op.Status)
		assert.True(t, op.SourceData["delayed"].(bool))

		harness.Logger.Info("Delayed endpoint indexing completed successfully")
	})

	t.Run("EndToEnd_RegisterJobOperations", func(t *testing.T) {
		t.Skip("Skipping register_job tests - worker not implemented in test manager")
		// Clear any previous operations
		harness.ClearIndexOperations()

		endpointID := uuid.New()

		// Test registering a job
		jobReq := RegisterJobRequest{
			EndpointID: endpointID,
			JobID:      "test-job-123",
			JobMetadata: map[string]interface{}{
				"name":        "Test Job",
				"description": "A test job for endpoint queue integration",
				"trigger":     "http",
				"enabled":     true,
			},
		}

		result, err := harness.QueueService.EnqueueRegisterJob(ctx, jobReq)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Greater(t, result.Job.ID, int64(0))

		harness.Logger.Info("Enqueued job registration",
			"job_id", result.Job.ID,
			"endpoint_id", endpointID,
			"register_job_id", jobReq.JobID)

		// Since we don't have actual job registration logic in the mock,
		// we just verify the job was enqueued successfully
		// In a real implementation, this would register the job in the database
	})

	t.Run("EndToEnd_RegisterSourceOperations", func(t *testing.T) {
		t.Skip("Skipping register_source tests - worker not implemented in test manager")
		endpointID := uuid.New()

		// Test registering a source
		sourceReq := RegisterSourceRequest{
			EndpointID: endpointID,
			SourceID:   "webhook-source-456",
			SourceMetadata: map[string]interface{}{
				"type":        "webhook",
				"url":         "https://api.example.com/webhook",
				"events":      []string{"user.created", "user.updated"},
				"auth_method": "bearer_token",
			},
		}

		result, err := harness.QueueService.EnqueueRegisterSource(ctx, sourceReq)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Greater(t, result.Job.ID, int64(0))

		harness.Logger.Info("Enqueued source registration",
			"job_id", result.Job.ID,
			"endpoint_id", endpointID,
			"source_id", sourceReq.SourceID)
	})

	t.Run("EndToEnd_HighConcurrencyPerformance", func(t *testing.T) {
		// Clear any previous operations
		harness.ClearIndexOperations()

		numJobs := 20
		endpointIDs := make([]uuid.UUID, numJobs)

		// Generate unique endpoint IDs
		for i := range endpointIDs {
			endpointIDs[i] = uuid.New()
		}

		// Enqueue all jobs concurrently
		startTime := time.Now()
		for i, endpointID := range endpointIDs {
			req := EnqueueIndexEndpointRequest{
				EndpointID: endpointID,
				Source:     EndpointIndexSourceAPI,
				Reason:     fmt.Sprintf("Performance test %d", i+1),
				Priority:   1,
				SourceData: map[string]interface{}{
					"performance_test": true,
					"job_index":        i,
					"batch_size":       numJobs,
				},
			}

			_, err := harness.QueueService.EnqueueIndexEndpoint(ctx, req)
			require.NoError(t, err)
		}
		enqueueTime := time.Since(startTime)

		// Wait for all jobs to be processed
		harness.WaitForIndexOperations(t, numJobs, 30*time.Second)
		totalTime := time.Since(startTime)

		// Verify all operations were processed
		operations := harness.GetIndexedOperations()
		require.Len(t, operations, numJobs)

		// Verify all operations were successful
		successCount := 0
		for _, op := range operations {
			if op.Status == "success" {
				successCount++
			}
		}
		assert.Equal(t, numJobs, successCount)

		// Performance assertions
		harness.Logger.Info("Endpoint queue performance metrics",
			"enqueue_time", enqueueTime,
			"total_processing_time", totalTime,
			"jobs_per_second", float64(numJobs)/totalTime.Seconds(),
			"pipeline", "QueueService→RiverQueue→MockEndpointIndexer",
		)

		assert.Less(t, totalTime, 30*time.Second, "All endpoint indexing jobs should complete within 30 seconds")
	})
}

// TestEndpointQueueServiceErrorHandling tests error handling in the queue service
func TestEndpointQueueServiceErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with error handling in short mode")
	}

	// Setup test harness
	harness := SetupEndpointQueueTestHarness(t)
	defer harness.Cleanup()

	harness.Start(t)
	defer harness.Stop(t)

	ctx := context.Background()

	t.Run("EndToEnd_SimulatedIndexingFailure", func(t *testing.T) {
		// Clear any previous operations
		harness.ClearIndexOperations()

		endpointID := uuid.New()
		req := EnqueueIndexEndpointRequest{
			EndpointID: endpointID,
			Source:     EndpointIndexSourceManual,
			Reason:     "force_failure", // This will trigger our mock to simulate failure
			SourceData: map[string]interface{}{
				"test_failure": true,
			},
		}

		// Enqueue the job that will fail
		result, err := harness.QueueService.EnqueueIndexEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Wait for the job to be processed (it should fail)
		time.Sleep(3 * time.Second)

		// Verify the failure was recorded
		operations := harness.GetIndexedOperations()
		require.Len(t, operations, 1)

		op := operations[0]
		assert.Equal(t, endpointID.String(), op.EndpointID)
		assert.Equal(t, "failed", op.Status)
		assert.NotNil(t, op.Error)
		assert.Contains(t, op.Error.Error(), "simulated indexing failure")

		harness.Logger.Info("Error handling test completed",
			"endpoint_id", endpointID,
			"status", op.Status,
			"error", op.Error)
	})
}
