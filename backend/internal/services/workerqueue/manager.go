// Package workerqueue provides worker queue functionality using River Queue
package workerqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
)

// TestWorker is a simple worker for testing purposes
type TestWorker struct {
	river.WorkerDefaults[IndexEndpointArgs]
	logger *slog.Logger
}

// Work processes test jobs
func (w *TestWorker) Work(ctx context.Context, job *river.Job[IndexEndpointArgs]) error {
	w.logger.Info("Processing index endpoint job", "job_id", job.ID, "args", job.Args)
	return nil
}

// StartRunWorker handles start run jobs
type StartRunWorker struct {
	river.WorkerDefaults[StartRunArgs]
	logger *slog.Logger
}

// Work processes start run jobs
func (w *StartRunWorker) Work(ctx context.Context, job *river.Job[StartRunArgs]) error {
	w.logger.Info("Processing start run job", "job_id", job.ID, "args", job.Args)
	return nil
}

// DeliverEventWorker handles event delivery jobs
type DeliverEventWorker struct {
	river.WorkerDefaults[DeliverEventArgs]
	logger *slog.Logger
}

// Work processes event delivery jobs
func (w *DeliverEventWorker) Work(ctx context.Context, job *river.Job[DeliverEventArgs]) error {
	w.logger.Info("Processing deliver event job", "job_id", job.ID, "args", job.Args)
	return nil
}

// InvokeDispatcherWorker handles dispatcher invocation jobs
type InvokeDispatcherWorker struct {
	river.WorkerDefaults[InvokeDispatcherArgs]
	logger *slog.Logger
}

// Work processes dispatcher invocation jobs
func (w *InvokeDispatcherWorker) Work(ctx context.Context, job *river.Job[InvokeDispatcherArgs]) error {
	w.logger.Info("Processing invoke dispatcher job", "job_id", job.ID, "args", job.Args)
	return nil
}

// EmailSender defines the interface for sending emails
// This avoids circular dependencies with the email package
type EmailSender interface {
	SendEmail(ctx context.Context, email EmailData) error
}

// EmailData represents the email data needed to send an email
type EmailData struct {
	To      string
	Subject string
	Body    string
	From    string
}

// ScheduleEmailWorker handles email scheduling jobs
type ScheduleEmailWorker struct {
	river.WorkerDefaults[ScheduleEmailArgs]
	logger      *slog.Logger
	emailSender EmailSender
}

// Work processes email scheduling jobs
func (w *ScheduleEmailWorker) Work(ctx context.Context, job *river.Job[ScheduleEmailArgs]) error {
	w.logger.Info("Processing schedule email job", "job_id", job.ID, "args", job.Args)

	if w.emailSender == nil {
		w.logger.Error("EmailSender not configured for ScheduleEmailWorker")
		return fmt.Errorf("email sender not configured")
	}

	// Convert ScheduleEmailArgs to EmailData
	emailData := EmailData{
		To:      job.Args.To,
		Subject: job.Args.Subject,
		Body:    job.Args.Body,
		From:    job.Args.From,
	}

	// Send the email using the configured email sender
	if err := w.emailSender.SendEmail(ctx, emailData); err != nil {
		w.logger.Error("Failed to send email",
			"job_id", job.ID,
			"to", job.Args.To,
			"subject", job.Args.Subject,
			"error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	w.logger.Info("Successfully sent email",
		"job_id", job.ID,
		"to", job.Args.To,
		"subject", job.Args.Subject)

	return nil
}

// Manager manages the River Queue client and workers lifecycle
type Manager struct {
	riverClient *river.Client[pgx.Tx]
	dbPool      *pgxpool.Pool
	workers     *river.Workers
	config      Config
	logger      *slog.Logger
	emailSender EmailSender

	// Future: can add SQLC support when needed
	// sqlcQueries *database.Queries
}

// NewManager creates a new worker manager with the given configuration
// EmailSender can be nil for testing or when email functionality is not needed
func NewManager(config Config, dbPool *pgxpool.Pool, logger *slog.Logger, emailSender EmailSender) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	workers := river.NewWorkers()

	// Register all workers to satisfy River's requirement
	river.AddWorker(workers, &TestWorker{logger: logger})
	river.AddWorker(workers, &StartRunWorker{logger: logger})
	river.AddWorker(workers, &DeliverEventWorker{logger: logger})
	river.AddWorker(workers, &InvokeDispatcherWorker{logger: logger})
	river.AddWorker(workers, &ScheduleEmailWorker{logger: logger, emailSender: emailSender})

	riverConfig := &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			string(QueueDefault): {
				MaxWorkers: config.MaxWorkers,
			},
			string(QueueExecution): {
				MaxWorkers: config.ExecutionMaxWorkers,
			},
			string(QueueEvents): {
				MaxWorkers: config.EventsMaxWorkers,
			},
			string(QueueMaintenance): {
				MaxWorkers: config.MaintenanceMaxWorkers,
			},
		},
		Workers:           workers,
		JobTimeout:        config.JobTimeout,
		FetchCooldown:     config.FetchCooldown,
		FetchPollInterval: config.FetchPollInterval,
		Schema:            config.Schema,
		TestOnly:          config.TestMode,
	}

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), riverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create river client: %w", err)
	}

	return &Manager{
		riverClient: riverClient,
		dbPool:      dbPool,
		workers:     workers,
		config:      config,
		logger:      logger,
		emailSender: emailSender,
	}, nil
}

// SetEmailSender sets the email sender for the manager after creation
// This allows for dependency injection without circular dependencies
func (m *Manager) SetEmailSender(emailSender EmailSender) {
	m.emailSender = emailSender
	// TODO: Update already registered workers if needed
	// For now, workers are registered at creation time with the email sender
}

// Start starts the worker manager and begins processing jobs
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting worker manager",
		"max_workers", m.config.MaxWorkers,
		"execution_workers", m.config.ExecutionMaxWorkers,
		"events_workers", m.config.EventsMaxWorkers,
	)

	if err := m.riverClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start river client: %w", err)
	}

	m.logger.Info("Worker manager started successfully")
	return nil
}

// Stop gracefully stops the worker manager
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("Stopping worker manager")

	if err := m.riverClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop river client: %w", err)
	}

	m.logger.Info("Worker manager stopped successfully")
	return nil
}

// WithTransaction provides SQLC + River transaction support
// This enables atomic operations across database changes and job insertions
func (m *Manager) WithTransaction(ctx context.Context, fn BusinessLogicFunc) error {
	tx, err := m.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txCtx := &TransactionContext{
		Tx: tx,
		// Future: add SQLC queries here when needed
		// SQLCQueries: m.sqlcQueries.WithTx(tx),
	}

	if err := fn(ctx, txCtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// EnqueueInTransaction inserts a job within an existing transaction
// This aligns with trigger.dev's transaction semantics
func (m *Manager) EnqueueInTransaction(ctx context.Context, txCtx *TransactionContext, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	args, err := m.createJobArgsWithJobKey(identifier, payload, opts)
	if err != nil {
		return nil, err
	}

	riverOpts := m.convertToRiverOpts(opts)

	// Use River's transaction support
	return m.riverClient.InsertTx(ctx, txCtx.Tx, args, riverOpts)
}

// Client returns the underlying River client for job insertion
func (m *Manager) Client() *river.Client[pgx.Tx] {
	return m.riverClient
}

// InsertJob inserts a job into the queue
func (m *Manager) InsertJob(ctx context.Context, args JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return m.riverClient.Insert(ctx, args, opts)
}

// InsertJobTx inserts a job into the queue within a transaction
func (m *Manager) InsertJobTx(ctx context.Context, tx pgx.Tx, args JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return m.riverClient.InsertTx(ctx, tx, args, opts)
}

// EnqueueJob inserts a job into the queue using string identifier (trigger.dev compatible)
func (m *Manager) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	args, err := m.createJobArgsWithJobKey(identifier, payload, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create job args: %w", err)
	}

	riverOpts := m.convertToRiverOpts(opts)
	return m.riverClient.Insert(ctx, args, riverOpts)
}

// EnqueueJobTx inserts a job into the queue within a transaction using string identifier
func (m *Manager) EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	args, err := m.createJobArgsWithJobKey(identifier, payload, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create job args: %w", err)
	}

	riverOpts := m.convertToRiverOpts(opts)
	return m.riverClient.InsertTx(ctx, tx, args, riverOpts)
}

// DequeueJob cancels a job by job key (placeholder implementation)
func (m *Manager) DequeueJob(ctx context.Context, jobKey string) error {
	// River doesn't have direct dequeue functionality
	// This would typically involve job cancellation
	m.logger.Debug("DequeueJob called", "jobKey", jobKey)
	return fmt.Errorf("dequeue functionality not yet implemented for job key: %s", jobKey)
}

// Health returns the health status of the worker manager
func (m *Manager) Health() bool {
	// For now, we'll consider the manager healthy if the client exists
	return m.riverClient != nil
}

// Stats returns basic statistics about the worker manager
func (m *Manager) Stats() map[string]interface{} {
	return map[string]interface{}{
		"config": map[string]interface{}{
			"max_workers":             m.config.MaxWorkers,
			"execution_max_workers":   m.config.ExecutionMaxWorkers,
			"events_max_workers":      m.config.EventsMaxWorkers,
			"maintenance_max_workers": m.config.MaintenanceMaxWorkers,
			"job_timeout":             m.config.JobTimeout.String(),
			"fetch_cooldown":          m.config.FetchCooldown.String(),
		},
	}
}

// Helper methods for job insertion

// createJobArgsWithJobKey converts identifier and payload to appropriate job args and sets JobKey if provided
func (m *Manager) createJobArgsWithJobKey(identifier string, payload interface{}, opts *JobOptions) (JobArgs, error) {
	args, err := m.createJobArgs(identifier, payload)
	if err != nil {
		return nil, err
	}

	// Set JobKey for IndexEndpointArgs if provided in options
	if opts != nil && opts.JobKey != "" {
		if indexArgs, ok := args.(IndexEndpointArgs); ok {
			indexArgs.JobKey = opts.JobKey
			return indexArgs, nil
		}
		// For other job types, JobKey might not be supported yet
		// This is similar to trigger.dev where not all job types support uniqueness
	}

	return args, nil
}

// createJobArgs converts identifier and payload to appropriate job args
func (m *Manager) createJobArgs(identifier string, payload interface{}) (JobArgs, error) {
	switch identifier {
	case "indexEndpoint", "index_endpoint":
		// Convert payload to IndexEndpointArgs
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args IndexEndpointArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to IndexEndpointArgs: %w", err)
		}
		return args, nil
	case "startRun", "start_run":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args StartRunArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to StartRunArgs: %w", err)
		}
		return args, nil
	case "invokeDispatcher", "invoke_dispatcher":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args InvokeDispatcherArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to InvokeDispatcherArgs: %w", err)
		}
		return args, nil
	case "deliverEvent":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args DeliverEventArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to DeliverEventArgs: %w", err)
		}
		return args, nil
	case "performRunExecutionV2":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args PerformRunExecutionV2Args
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to PerformRunExecutionV2Args: %w", err)
		}
		return args, nil
	case "scheduleEmail":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args ScheduleEmailArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to ScheduleEmailArgs: %w", err)
		}
		return args, nil
	case "events.invokeDispatcher":
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var args InvokeDispatcherArgs
		if err := json.Unmarshal(data, &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to InvokeDispatcherArgs: %w", err)
		}
		return args, nil
	default:
		return nil, fmt.Errorf("unknown job identifier: %s", identifier)
	}
}

// convertToRiverOpts converts JobOptions to River InsertOpts
func (m *Manager) convertToRiverOpts(opts *JobOptions) *river.InsertOpts {
	if opts == nil {
		return nil
	}

	riverOpts := &river.InsertOpts{}

	if opts.QueueName != "" {
		riverOpts.Queue = opts.QueueName
	}

	if opts.RunAt != nil {
		riverOpts.ScheduledAt = *opts.RunAt
	}

	if opts.Priority > 0 {
		// River Queue requires priority to be between 1 and 4
		// Map trigger.dev style priorities to River range
		if opts.Priority <= 4 {
			riverOpts.Priority = opts.Priority
		} else {
			// For higher priorities, map to River's scale
			// 1-25: High (1), 26-50: Normal (2), 51-75: Low (3), 76+: Very Low (4)
			switch {
			case opts.Priority <= 25:
				riverOpts.Priority = 1 // High
			case opts.Priority <= 50:
				riverOpts.Priority = 2 // Normal
			case opts.Priority <= 75:
				riverOpts.Priority = 3 // Low
			default:
				riverOpts.Priority = 4 // Very Low
			}
		}
	}

	if opts.MaxAttempts > 0 {
		riverOpts.MaxAttempts = opts.MaxAttempts
	}

	// Handle tags (now []string instead of map)
	if len(opts.Tags) > 0 {
		riverOpts.Tags = make([]string, len(opts.Tags))
		copy(riverOpts.Tags, opts.Tags)
	}

	// Handle uniqueness constraints
	if opts.UniqueOpts != nil {
		riverOpts.UniqueOpts = *opts.UniqueOpts
	} else if opts.JobKey != "" {
		// Map trigger.dev's jobKey to River's unique constraints
		// Use ByArgs with unique struct tag on JobKey field
		riverOpts.UniqueOpts = river.UniqueOpts{
			ByArgs: true, // This will use the JobKey field marked with river:"unique"
		}
	}

	// TODO: Handle JobKeyMode for different uniqueness behaviors
	// - "replace": could use ByState with only running states
	// - "preserve_run_at": could use ByPeriod
	// - "unsafe_dedupe": could use ByArgs only

	return riverOpts
}

// EnsureRiverTables ensures that River Queue tables are created and up to date
func (m *Manager) EnsureRiverTables(ctx context.Context) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(m.dbPool), &rivermigrate.Config{
		Logger: m.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	m.logger.Info("River Queue tables migration completed successfully")
	return nil
}
