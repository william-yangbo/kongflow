// Package queue provides mock workers for testing events queue service
package queue

import (
	"context"
	"fmt"
	"log/slog"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/riverqueue/river"
)

// MockDeliverEventWorker handles event delivery jobs for testing
type MockDeliverEventWorker struct {
	river.WorkerDefaults[workerqueue.DeliverEventArgs]
	processor *MockEventProcessor
	logger    *slog.Logger
}

// NewMockDeliverEventWorker creates a new MockDeliverEventWorker
func NewMockDeliverEventWorker(processor *MockEventProcessor, logger *slog.Logger) *MockDeliverEventWorker {
	return &MockDeliverEventWorker{
		processor: processor,
		logger:    logger,
	}
}

// Work processes an event delivery job
func (w *MockDeliverEventWorker) Work(ctx context.Context, job *river.Job[workerqueue.DeliverEventArgs]) error {
	w.logger.Info("Processing deliver event job",
		"job_id", job.ID,
		"event_id", job.Args.ID,
		"attempt", job.Attempt,
	)

	// Check for simulated failure
	if err, shouldFail := w.processor.FailureSimulation[job.Args.ID]; shouldFail {
		operation := EventDeliveryOperation{
			EventID: job.Args.ID,
			Status:  "failed",
			Error:   err,
		}
		w.processor.DeliveredEvents = append(w.processor.DeliveredEvents, operation)

		w.logger.Error("Event delivery failed", "event_id", job.Args.ID, "error", err)
		return fmt.Errorf("failed to deliver event %s: %w", job.Args.ID, err)
	}

	// Simulate event delivery processing
	operation := EventDeliveryOperation{
		EventID:    job.Args.ID,
		Status:     "delivered",
		DeliveryID: fmt.Sprintf("delivery_%s", job.Args.ID),
		Stats:      map[string]interface{}{"processed": 1, "job_id": job.ID},
	}

	w.processor.DeliveredEvents = append(w.processor.DeliveredEvents, operation)

	w.logger.Info("Event delivery completed successfully",
		"event_id", job.Args.ID,
		"delivery_id", operation.DeliveryID,
		"stats", operation.Stats,
	)

	return nil
}

// MockInvokeDispatcherWorker handles dispatcher invocation jobs for testing
type MockInvokeDispatcherWorker struct {
	river.WorkerDefaults[workerqueue.InvokeDispatcherArgs]
	processor *MockEventProcessor
	logger    *slog.Logger
}

// NewMockInvokeDispatcherWorker creates a new MockInvokeDispatcherWorker
func NewMockInvokeDispatcherWorker(processor *MockEventProcessor, logger *slog.Logger) *MockInvokeDispatcherWorker {
	return &MockInvokeDispatcherWorker{
		processor: processor,
		logger:    logger,
	}
}

// Work processes a dispatcher invocation job
func (w *MockInvokeDispatcherWorker) Work(ctx context.Context, job *river.Job[workerqueue.InvokeDispatcherArgs]) error {
	w.logger.Info("Processing invoke dispatcher job",
		"job_id", job.ID,
		"args", fmt.Sprintf("%+v", job.Args),
	)

	// Check for simulated failure
	if err, shouldFail := w.processor.FailureSimulation[job.Args.ID]; shouldFail {
		operation := DispatcherInvocationOperation{
			DispatcherID: job.Args.ID,
			EventID:      job.Args.EventRecordID,
			Status:       "failed",
			Error:        err,
		}
		w.processor.InvokedDispatchers = append(w.processor.InvokedDispatchers, operation)

		w.logger.Error("Dispatcher invocation failed", "dispatcher_id", job.Args.ID, "error", err)
		return fmt.Errorf("failed to invoke dispatcher %s: %w", job.Args.ID, err)
	}

	// Simulate dispatcher invocation processing
	operation := DispatcherInvocationOperation{
		DispatcherID: job.Args.ID,
		EventID:      job.Args.EventRecordID,
		Status:       "invoked",
		InvokeID:     fmt.Sprintf("invoke_%s", job.Args.ID),
		Stats:        map[string]interface{}{"processed": 1, "job_id": job.ID},
	}

	w.processor.InvokedDispatchers = append(w.processor.InvokedDispatchers, operation)

	w.logger.Info("Dispatcher invocation completed successfully",
		"dispatcher_id", job.Args.ID,
		"event_id", job.Args.EventRecordID,
		"invoke_id", operation.InvokeID,
		"stats", operation.Stats,
	)

	return nil
}