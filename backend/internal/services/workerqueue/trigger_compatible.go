// Package workerqueue provides trigger.dev compatible worker functionality
package workerqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
)

// QueueNameResolver defines the interface for resolving queue names
// This allows both static and dynamic queue name assignment
type QueueNameResolver interface {
	ResolveQueueName(payload interface{}) string
}

// StaticQueueName implements QueueNameResolver for static queue names
// This provides backward compatibility with existing configurations
type StaticQueueName string

// ResolveQueueName returns the static queue name regardless of payload
func (s StaticQueueName) ResolveQueueName(payload interface{}) string {
	return string(s)
}

// DynamicQueueName implements QueueNameResolver for dynamic queue names
// This allows runtime queue selection based on payload content
type DynamicQueueName func(payload interface{}) string

// ResolveQueueName calls the function to dynamically determine queue name
func (d DynamicQueueName) ResolveQueueName(payload interface{}) string {
	return d(payload)
}

// TaskCatalog defines available task types (similar to trigger.dev's workerCatalog)
type TaskCatalog map[string]TaskDefinition

// TaskDefinition represents a task configuration
type TaskDefinition struct {
	// Task configuration
	QueueName   QueueNameResolver // ✅ Now supports both static and dynamic queue names
	Priority    int
	MaxAttempts int
	JobKeyMode  string // "replace", "preserve_run_at", "unsafe_dedupe"
	Flags       []string

	// Handler function
	Handler TaskHandler
}

// TaskHandler defines the interface for task handlers
type TaskHandler func(ctx context.Context, payload json.RawMessage, job JobContext) error

// JobContext provides context information about the executing job
type JobContext struct {
	ID        int64
	Attempt   int
	Queue     string
	CreatedAt time.Time
	Metadata  map[string]string
}

// RecurringTaskHandler defines the interface for recurring task handlers
type RecurringTaskHandler func(ctx context.Context, payload RecurringTaskPayload) error

// RecurringTaskPayload represents payload for recurring tasks
type RecurringTaskPayload struct {
	Timestamp  time.Time `json:"timestamp"`
	Backfilled bool      `json:"backfilled"`
}

// TriggerCompatibleWorker provides trigger.dev ZodWorker compatible interface
type TriggerCompatibleWorker struct {
	manager   *Manager
	catalog   TaskCatalog
	recurring map[string]RecurringTaskConfig
	logger    *slog.Logger
}

// RecurringTaskConfig represents configuration for recurring tasks
type RecurringTaskConfig struct {
	Pattern string                // Cron pattern
	Handler RecurringTaskHandler  // Handler function
	Options *RecurringTaskOptions // Additional options
}

// RecurringTaskOptions provides additional configuration for recurring tasks
type RecurringTaskOptions struct {
	Timezone string
	Jitter   time.Duration
}

// TriggerWorkerOptions mirrors trigger.dev's ZodWorkerOptions
type TriggerWorkerOptions struct {
	Name           string
	Manager        *Manager
	Catalog        TaskCatalog
	RecurringTasks map[string]RecurringTaskConfig
	Logger         *slog.Logger
}

// NewTriggerCompatibleWorker creates a new trigger.dev compatible worker
func NewTriggerCompatibleWorker(opts TriggerWorkerOptions) *TriggerCompatibleWorker {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return &TriggerCompatibleWorker{
		manager:   opts.Manager,
		catalog:   opts.Catalog,
		recurring: opts.RecurringTasks,
		logger:    opts.Logger,
	}
}

// Initialize starts the worker (mirrors trigger.dev's initialize())
func (w *TriggerCompatibleWorker) Initialize(ctx context.Context) error {
	w.logger.Info("Initializing trigger-compatible worker")

	// Register recurring tasks if any
	for identifier, config := range w.recurring {
		if err := w.registerRecurringTask(identifier, config); err != nil {
			return fmt.Errorf("failed to register recurring task %s: %w", identifier, err)
		}
	}

	// Start the underlying manager if not already started
	return w.manager.Start(ctx)
}

// Stop gracefully stops the worker (mirrors trigger.dev's stop())
func (w *TriggerCompatibleWorker) Stop(ctx context.Context) error {
	w.logger.Info("Stopping trigger-compatible worker")
	return w.manager.Stop(ctx)
}

// Enqueue adds a job to the queue (mirrors trigger.dev's enqueue())
func (w *TriggerCompatibleWorker) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	w.logger.Debug("Enqueuing job",
		"identifier", identifier,
		"payload", payload,
	)

	// Get task definition from catalog
	taskDef, exists := w.catalog[identifier]
	if !exists {
		return nil, fmt.Errorf("unknown task identifier: %s", identifier)
	}

	// Merge task definition with provided options
	finalOpts := w.mergeJobOptions(taskDef, opts, payload)

	result, err := w.manager.EnqueueJob(ctx, identifier, payload, finalOpts)
	if err != nil {
		w.logger.Error("Failed to enqueue job",
			"identifier", identifier,
			"error", err,
		)
		return nil, err
	}

	w.logger.Debug("Job enqueued successfully",
		"identifier", identifier,
		"job_id", result.Job.ID,
	)

	return result, nil
}

// EnqueueTx adds a job to the queue within a transaction
func (w *TriggerCompatibleWorker) EnqueueTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	taskDef, exists := w.catalog[identifier]
	if !exists {
		return nil, fmt.Errorf("unknown task identifier: %s", identifier)
	}

	finalOpts := w.mergeJobOptions(taskDef, opts, payload)

	return w.manager.EnqueueJobTx(ctx, tx, identifier, payload, finalOpts)
}

// Dequeue removes a job from the queue by job key (mirrors trigger.dev's dequeue())
func (w *TriggerCompatibleWorker) Dequeue(ctx context.Context, jobKey string) error {
	return w.manager.DequeueJob(ctx, jobKey)
}

// Helper methods

func (w *TriggerCompatibleWorker) mergeJobOptions(taskDef TaskDefinition, opts *JobOptions, payload interface{}) *JobOptions {
	merged := &JobOptions{
		QueueName:   taskDef.QueueName.ResolveQueueName(payload), // ✅ Resolve queue name from payload
		Priority:    taskDef.Priority,
		MaxAttempts: taskDef.MaxAttempts,
		JobKeyMode:  taskDef.JobKeyMode,
		Flags:       taskDef.Flags,
	}

	// Override with provided options
	if opts != nil {
		if opts.QueueName != "" {
			merged.QueueName = opts.QueueName
		}
		if opts.Priority > 0 {
			merged.Priority = opts.Priority
		}
		if opts.MaxAttempts > 0 {
			merged.MaxAttempts = opts.MaxAttempts
		}
		if opts.RunAt != nil {
			merged.RunAt = opts.RunAt
		}
		if opts.JobKey != "" {
			merged.JobKey = opts.JobKey
		}
		if opts.JobKeyMode != "" {
			merged.JobKeyMode = opts.JobKeyMode
		}
		if opts.Tags != nil {
			merged.Tags = opts.Tags
		}
		if len(opts.Flags) > 0 {
			merged.Flags = opts.Flags
		}
	}

	return merged
}

func (w *TriggerCompatibleWorker) registerRecurringTask(identifier string, config RecurringTaskConfig) error {
	// This would integrate with River's periodic job functionality
	// For now, we'll log the registration
	w.logger.Info("Registering recurring task",
		"identifier", identifier,
		"pattern", config.Pattern,
	)

	// TODO: Implement actual recurring task registration
	// This would involve using River's cron functionality or implementing a scheduler

	return nil
}

// Factory function to create a complete worker setup similar to trigger.dev's worker.server.ts
func CreateKongFlowWorker(manager *Manager, logger *slog.Logger) *TriggerCompatibleWorker {
	// Define task catalog similar to trigger.dev's workerCatalog
	catalog := TaskCatalog{
		"indexEndpoint": TaskDefinition{
			QueueName:   StaticQueueName(QueueDefault), // ✅ Convert to StaticQueueName
			Priority:    1,
			MaxAttempts: 7,
			Handler:     handleIndexEndpoint,
		},
		"startRun": TaskDefinition{
			QueueName:   StaticQueueName(QueueExecution), // ✅ Convert to StaticQueueName
			Priority:    0,
			MaxAttempts: 4,
			Handler:     handleStartRun,
		},
		"invokeDispatcher": TaskDefinition{
			QueueName:   StaticQueueName(QueueEvents), // ✅ Convert to StaticQueueName
			Priority:    0,
			MaxAttempts: 3,
			Handler:     handleInvokeDispatcher,
		},
		"deliverEvent": TaskDefinition{
			QueueName:   StaticQueueName(QueueEvents), // ✅ Convert to StaticQueueName
			Priority:    0,
			MaxAttempts: 5,
			Handler:     handleDeliverEvent,
		},
		"performRunExecutionV2": TaskDefinition{
			QueueName:   StaticQueueName(QueueExecution), // ✅ Convert to StaticQueueName
			Priority:    0,
			MaxAttempts: 12,
			Handler:     handlePerformRunExecutionV2,
		},
	}

	// Define recurring tasks similar to trigger.dev's recurringTasks
	recurringTasks := map[string]RecurringTaskConfig{
		"autoIndexProductionEndpoints": {
			Pattern: "*/5 * * * *", // Every 5 minutes
			Handler: handleAutoIndexEndpoints,
		},
		"purgeOldIndexings": {
			Pattern: "0 * * * *", // Every hour
			Handler: handlePurgeOldIndexings,
		},
	}

	return NewTriggerCompatibleWorker(TriggerWorkerOptions{
		Name:           "kongflow-worker",
		Manager:        manager,
		Catalog:        catalog,
		RecurringTasks: recurringTasks,
		Logger:         logger,
	})
}

// Task handlers (to be implemented)
func handleIndexEndpoint(ctx context.Context, payload json.RawMessage, job JobContext) error {
	// Implementation would go here
	return fmt.Errorf("handleIndexEndpoint not yet implemented")
}

func handleStartRun(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("handleStartRun not yet implemented")
}

func handleInvokeDispatcher(ctx context.Context, payload json.RawMessage, job JobContext) error {
	var args InvokeDispatcherArgs
	if err := json.Unmarshal(payload, &args); err != nil {
		return fmt.Errorf("failed to unmarshal invoke dispatcher args: %w", err)
	}

	// TODO: Get dependencies from the context or implement dependency injection
	// For now, return a placeholder implementation
	// In a real implementation, we would:
	// 1. Get the InvokeDispatcherService from a service container
	// 2. Call service.InvokeDispatcher(ctx, args.ID, args.EventRecordID)

	// Log the invocation for now
	slog.InfoContext(ctx, "Processing invoke dispatcher job",
		"dispatcherId", args.ID,
		"eventRecordId", args.EventRecordID,
		"jobId", job.ID,
		"attempt", job.Attempt)

	// Return success for placeholder
	// TODO: Replace with actual service call
	return nil
}

func handleDeliverEvent(ctx context.Context, payload json.RawMessage, job JobContext) error {
	var args DeliverEventArgs
	if err := json.Unmarshal(payload, &args); err != nil {
		return fmt.Errorf("failed to unmarshal deliver event args: %w", err)
	}

	// TODO: Get dependencies from the context or implement dependency injection
	// For now, return a placeholder implementation
	// In a real implementation, we would:
	// 1. Get the DeliverEventService from a service container
	// 2. Call service.DeliverEvent(ctx, args.ID)

	// Log the delivery for now
	slog.InfoContext(ctx, "Processing deliver event job",
		"eventId", args.ID,
		"jobId", job.ID,
		"attempt", job.Attempt)

	// Return success for placeholder
	// TODO: Replace with actual service call
	return nil
}

func handlePerformRunExecutionV2(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("handlePerformRunExecutionV2 not yet implemented")
}

func handleAutoIndexEndpoints(ctx context.Context, payload RecurringTaskPayload) error {
	return fmt.Errorf("handleAutoIndexEndpoints not yet implemented")
}

func handlePurgeOldIndexings(ctx context.Context, payload RecurringTaskPayload) error {
	return fmt.Errorf("handlePurgeOldIndexings not yet implemented")
}
