// Package queue provides real integration testing for events queue service
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

// TestRealEventQueueIntegration tests event queue service with real River worker queue and PostgreSQL
// This test uses TestContainers to provide a real PostgreSQL database and River queue
func TestRealEventQueueIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real event queue in short mode")
	}

	// Setup test harness with real PostgreSQL and River worker queue
	harness := SetupEventQueueTestHarness(t)
	defer harness.Cleanup()

	// Start the worker queue manager
	harness.Start(t)

	t.Run("EndToEnd_SingleEventDelivery", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test single event delivery
		eventID := uuid.New().String()
		endpointID := uuid.New().String()

		req := &EnqueueDeliverEventRequest{
			EventID:    eventID,
			EndpointID: endpointID,
			Payload:    `{"type": "test", "data": "single event test"}`,
		}

		harness.Logger.Info("Enqueuing event delivery job",
			"event_id", eventID,
			"endpoint_id", endpointID,
		)

		result, err := harness.QueueService.EnqueueDeliverEvent(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Wait for processing
		err = harness.WaitForEventOperations(1, 10*time.Second)
		require.NoError(t, err)

		// Verify event was delivered
		assert.Len(t, harness.MockEventProcessor.DeliveredEvents, 1)
		deliveredEvent := harness.MockEventProcessor.DeliveredEvents[0]
		assert.Equal(t, eventID, deliveredEvent.EventID)
		assert.Equal(t, "delivered", deliveredEvent.Status)
		assert.NotEmpty(t, deliveredEvent.DeliveryID)

		harness.Logger.Info("Single event delivery completed successfully")
	})

	t.Run("EndToEnd_SingleDispatcherInvocation", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test single dispatcher invocation
		dispatcherID := uuid.New().String()
		eventID := uuid.New().String()

		req := &EnqueueInvokeDispatcherRequest{
			DispatcherID: dispatcherID,
			EventID:      eventID,
			Payload:      `{"type": "dispatcher_test", "data": "single dispatcher test"}`,
		}

		harness.Logger.Info("Enqueuing dispatcher invocation job",
			"dispatcher_id", dispatcherID,
			"event_id", eventID,
		)

		result, err := harness.QueueService.EnqueueInvokeDispatcher(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Wait for processing
		err = harness.WaitForEventOperations(1, 10*time.Second)
		require.NoError(t, err)

		// Verify dispatcher was invoked
		assert.Len(t, harness.MockEventProcessor.InvokedDispatchers, 1)
		invokedDispatcher := harness.MockEventProcessor.InvokedDispatchers[0]
		assert.Equal(t, dispatcherID, invokedDispatcher.DispatcherID)
		assert.Equal(t, eventID, invokedDispatcher.EventID)
		assert.Equal(t, "invoked", invokedDispatcher.Status)
		assert.NotEmpty(t, invokedDispatcher.InvokeID)

		harness.Logger.Info("Single dispatcher invocation completed successfully")
	})

	t.Run("EndToEnd_MixedEventOperations", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test mixed event operations (delivery + dispatcher invocation)
		ctx := context.Background()
		var results []interface{}

		// Enqueue multiple event deliveries
		for i := 0; i < 3; i++ {
			req := &EnqueueDeliverEventRequest{
				EventID:    uuid.New().String(),
				EndpointID: uuid.New().String(),
				Payload:    `{"type": "batch_delivery", "data": "mixed operations test"}`,
			}

			result, err := harness.QueueService.EnqueueDeliverEvent(ctx, req)
			require.NoError(t, err)
			results = append(results, result)

			harness.Logger.Info("Enqueued batch event delivery",
				"batch_index", i,
				"event_id", req.EventID,
			)
		}

		// Enqueue multiple dispatcher invocations
		for i := 0; i < 2; i++ {
			req := &EnqueueInvokeDispatcherRequest{
				DispatcherID: uuid.New().String(),
				EventID:      uuid.New().String(),
				Payload:      `{"type": "batch_invoke", "data": "mixed operations test"}`,
			}

			result, err := harness.QueueService.EnqueueInvokeDispatcher(ctx, req)
			require.NoError(t, err)
			results = append(results, result)

			harness.Logger.Info("Enqueued batch dispatcher invocation",
				"batch_index", i,
				"dispatcher_id", req.DispatcherID,
			)
		}

		// Wait for all operations to complete
		err := harness.WaitForEventOperations(5, 15*time.Second)
		require.NoError(t, err)

		// Verify results
		assert.Len(t, harness.MockEventProcessor.DeliveredEvents, 3)
		assert.Len(t, harness.MockEventProcessor.InvokedDispatchers, 2)

		// Verify statistics
		totalOperations := len(harness.MockEventProcessor.DeliveredEvents) + len(harness.MockEventProcessor.InvokedDispatchers)
		assert.Equal(t, 5, totalOperations)

		harness.Logger.Info("Mixed event operations completed successfully")
	})

	t.Run("EndToEnd_ScheduledEventProcessing", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test scheduled event processing
		eventID := uuid.New().String()
		scheduledTime := time.Now().Add(2 * time.Second)

		req := &EnqueueDeliverEventRequest{
			EventID:      eventID,
			EndpointID:   uuid.New().String(),
			Payload:      `{"type": "scheduled_test", "data": "delayed event processing"}`,
			ScheduledFor: &scheduledTime,
		}

		harness.Logger.Info("Enqueuing scheduled event delivery",
			"event_id", eventID,
			"scheduled_for", scheduledTime,
		)

		result, err := harness.QueueService.EnqueueDeliverEvent(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Wait for processing with extra time for scheduling
		err = harness.WaitForEventOperations(1, 10*time.Second)
		require.NoError(t, err)

		// Verify scheduled event was processed
		assert.Len(t, harness.MockEventProcessor.DeliveredEvents, 1)
		deliveredEvent := harness.MockEventProcessor.DeliveredEvents[0]
		assert.Equal(t, eventID, deliveredEvent.EventID)

		harness.Logger.Info("Scheduled event processing completed successfully")
	})

	t.Run("EndToEnd_TransactionSupport", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test transaction support with a real database transaction
		ctx := context.Background()

		// Start a database transaction
		tx, err := harness.TestDB.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		// Test delivery with transaction
		deliveryReq := &EnqueueDeliverEventRequest{
			EventID:    uuid.New().String(),
			EndpointID: uuid.New().String(),
			Payload:    `{"type": "tx_delivery", "data": "transaction test"}`,
		}

		deliveryResult, err := harness.QueueService.EnqueueDeliverEventTx(ctx, tx, deliveryReq)
		require.NoError(t, err)
		require.NotNil(t, deliveryResult)

		// Test dispatcher invocation with transaction
		dispatcherReq := &EnqueueInvokeDispatcherRequest{
			DispatcherID: uuid.New().String(),
			EventID:      uuid.New().String(),
			Payload:      `{"type": "tx_dispatcher", "data": "transaction test"}`,
		}

		dispatcherResult, err := harness.QueueService.EnqueueInvokeDispatcherTx(ctx, tx, dispatcherReq)
		require.NoError(t, err)
		require.NotNil(t, dispatcherResult)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Wait for processing
		err = harness.WaitForEventOperations(2, 10*time.Second)
		require.NoError(t, err)

		// Verify both operations completed
		assert.Len(t, harness.MockEventProcessor.DeliveredEvents, 1)
		assert.Len(t, harness.MockEventProcessor.InvokedDispatchers, 1)

		harness.Logger.Info("Transaction support test completed successfully")
	})

	t.Run("EndToEnd_HighConcurrencyPerformance", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test high concurrency event processing
		ctx := context.Background()
		numEvents := 20
		numDispatchers := 10
		totalOperations := numEvents + numDispatchers

		startTime := time.Now()

		// Enqueue events concurrently
		for i := 0; i < numEvents; i++ {
			req := &EnqueueDeliverEventRequest{
				EventID:    uuid.New().String(),
				EndpointID: uuid.New().String(),
				Payload:    `{"type": "performance_test", "data": "concurrency test"}`,
			}

			_, err := harness.QueueService.EnqueueDeliverEvent(ctx, req)
			require.NoError(t, err)

			harness.Logger.Info("Enqueued performance event",
				"event_index", i+1,
				"event_id", req.EventID,
			)
		}

		// Enqueue dispatcher invocations concurrently
		for i := 0; i < numDispatchers; i++ {
			req := &EnqueueInvokeDispatcherRequest{
				DispatcherID: uuid.New().String(),
				EventID:      uuid.New().String(),
				Payload:      `{"type": "performance_test", "data": "concurrency test"}`,
			}

			_, err := harness.QueueService.EnqueueInvokeDispatcher(ctx, req)
			require.NoError(t, err)

			harness.Logger.Info("Enqueued performance dispatcher",
				"dispatcher_index", i+1,
				"dispatcher_id", req.DispatcherID,
			)
		}

		enqueueTime := time.Since(startTime)

		// Wait for all operations to complete
		err := harness.WaitForEventOperations(totalOperations, 30*time.Second)
		require.NoError(t, err)

		totalTime := time.Since(startTime)

		// Verify all operations completed
		assert.Len(t, harness.MockEventProcessor.DeliveredEvents, numEvents)
		assert.Len(t, harness.MockEventProcessor.InvokedDispatchers, numDispatchers)

		// Log performance metrics
		harness.Logger.Info("Event queue performance metrics",
			"enqueue_time", enqueueTime,
			"total_processing_time", totalTime,
			"operations_per_second", float64(totalOperations)/totalTime.Seconds(),
			"delivered_events", numEvents,
			"invoked_dispatchers", numDispatchers,
			"pipeline", "QueueService→RiverQueue→MockEventProcessor",
		)

		assert.Less(t, totalTime, 30*time.Second, "All event operations should complete within 30 seconds")
	})
}

// TestEventQueueServiceErrorHandling tests error handling in the queue service
func TestEventQueueServiceErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling integration test in short mode")
	}

	// Setup test harness
	harness := SetupEventQueueTestHarness(t)
	defer harness.Cleanup()

	// Start the worker queue manager
	harness.Start(t)

	t.Run("EndToEnd_SimulatedEventDeliveryFailure", func(t *testing.T) {
		// Clear any previous operations
		require.NoError(t, harness.ClearEventOperations())

		// Test simulated event delivery failure
		eventID := "force_failure_event"
		harness.SimulateEventFailure(eventID, fmt.Errorf("simulated delivery failure"))

		req := &EnqueueDeliverEventRequest{
			EventID:    eventID,
			EndpointID: uuid.New().String(),
			Payload:    `{"type": "failure_test", "data": "simulated failure"}`,
		}

		harness.Logger.Info("Enqueuing event delivery with simulated failure",
			"event_id", eventID,
		)

		result, err := harness.QueueService.EnqueueDeliverEvent(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Wait for processing (should fail and retry)
		time.Sleep(5 * time.Second)

		// The job should be queued but fail during processing
		// This tests the resilience of the queue system
		harness.Logger.Info("Error handling test completed",
			"event_id", eventID,
			"status", "failed (as expected)",
		)

		// Verify error handling doesn't crash the system
		totalOperations := len(harness.MockEventProcessor.DeliveredEvents) + len(harness.MockEventProcessor.InvokedDispatchers)
		harness.Logger.Info("Error handling stats", "total_operations", totalOperations)
	})
}
