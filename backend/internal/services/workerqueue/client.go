// Package workerqueue provides a simplified client API that aligns with trigger.dev's ZodWorker
package workerqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/rivertype"
)

// Client provides the main API for worker queue operations
// This mirrors trigger.dev's ZodWorker class structure
type Client struct {
	manager       *Manager
	triggerWorker *TriggerCompatibleWorker
	logger        *slog.Logger
}

// ClientOptions provides configuration for creating a new Client
// This aligns with trigger.dev's ZodWorkerOptions structure
type ClientOptions struct {
	DatabasePool  *pgxpool.Pool `json:"-"`
	RunnerOptions RunnerOptions `json:"runnerOptions"`
	TaskCatalog   TaskCatalog   `json:"-"`
	Logger        *slog.Logger  `json:"-"`
}

// RunnerOptions aligns with trigger.dev's runnerOptions
type RunnerOptions struct {
	Concurrency  int `json:"concurrency"`  // Default: 5 (matches trigger.dev)
	PollInterval int `json:"pollInterval"` // Default: 1000ms (matches trigger.dev)
}

// NewClient creates a new worker queue client
// This mirrors trigger.dev's ZodWorker constructor
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	// Set defaults that match trigger.dev
	if opts.RunnerOptions.Concurrency == 0 {
		opts.RunnerOptions.Concurrency = 5
	}
	if opts.RunnerOptions.PollInterval == 0 {
		opts.RunnerOptions.PollInterval = 1000
	}

	// Create manager configuration
	config := DefaultConfig()
	config.MaxWorkers = opts.RunnerOptions.Concurrency

	// Create manager
	manager, err := NewManager(config, opts.DatabasePool, opts.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Create task catalog if not provided
	if opts.TaskCatalog == nil {
		opts.TaskCatalog = createDefaultTaskCatalog()
	}

	// Create trigger-compatible worker
	triggerWorker := NewTriggerCompatibleWorker(TriggerWorkerOptions{
		Name:    "KongFlow Worker",
		Manager: manager,
		Catalog: opts.TaskCatalog,
		Logger:  opts.Logger,
	})

	return &Client{
		manager:       manager,
		triggerWorker: triggerWorker,
		logger:        opts.Logger,
	}, nil
}

// Initialize starts the worker client (mirrors trigger.dev's initialize())
func (c *Client) Initialize(ctx context.Context) error {
	c.logger.Info("Initializing worker queue client")

	// Ensure River tables exist
	if err := c.manager.EnsureRiverTables(ctx); err != nil {
		return fmt.Errorf("failed to ensure river tables: %w", err)
	}

	// Initialize trigger-compatible worker
	return c.triggerWorker.Initialize(ctx)
}

// Stop gracefully stops the worker client (mirrors trigger.dev's stop())
func (c *Client) Stop(ctx context.Context) error {
	c.logger.Info("Stopping worker queue client")
	return c.triggerWorker.Stop(ctx)
}

// Enqueue adds a job to the queue (mirrors trigger.dev's enqueue())
func (c *Client) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error) {
	return c.triggerWorker.Enqueue(ctx, identifier, payload, opts)
}

// EnqueueWithBusinessLogic executes business logic and enqueues a job in a transaction
// This provides SQLC + River transaction support that trigger.dev lacks
func (c *Client) EnqueueWithBusinessLogic(ctx context.Context, identifier string, payload interface{}, businessLogic BusinessLogicFunc) (*rivertype.JobInsertResult, error) {
	var result *rivertype.JobInsertResult

	err := c.manager.WithTransaction(ctx, func(ctx context.Context, txCtx *TransactionContext) error {
		// Execute business logic first
		if err := businessLogic(ctx, txCtx); err != nil {
			return fmt.Errorf("business logic failed: %w", err)
		}

		// Then enqueue the job in the same transaction
		var err error
		result, err = c.manager.EnqueueInTransaction(ctx, txCtx, identifier, payload, nil)
		return err
	})

	return result, err
}

// createDefaultTaskCatalog creates a task catalog that aligns with trigger.dev's workerCatalog
func createDefaultTaskCatalog() TaskCatalog {
	return TaskCatalog{
		// Core tasks from trigger.dev workerCatalog
		"indexEndpoint": TaskDefinition{
			QueueName:   StaticQueueName("internal-queue"), // ✅ Convert to StaticQueueName
			MaxAttempts: 7,                                 // matches trigger.dev
			Handler:     handleIndexEndpoint,
		},
		"scheduleEmail": TaskDefinition{
			QueueName:   StaticQueueName("internal-queue"), // ✅ Convert to StaticQueueName
			Priority:    100,                               // matches trigger.dev
			MaxAttempts: 3,                                 // matches trigger.dev
			Handler:     handleScheduleEmail,
		},
		"startRun": TaskDefinition{
			QueueName:   StaticQueueName("executions"), // ✅ Convert to StaticQueueName
			MaxAttempts: 13,                            // matches trigger.dev
			Handler:     handleStartRun,
		},
		"performRunExecution": TaskDefinition{
			// ✅ Phase 2: Dynamic queue - run-level isolation
			QueueName: DynamicQueueName(func(payload interface{}) string {
				// Try to cast to PerformRunExecutionV2Args (our enhanced structure)
				if runArgs, ok := payload.(PerformRunExecutionV2Args); ok {
					return fmt.Sprintf("runs_%s", runArgs.ID) // runs_run-12345 (using underscores)
				}
				// Fallback for other payload types
				if payloadMap, ok := payload.(map[string]interface{}); ok {
					if id, exists := payloadMap["id"]; exists {
						return fmt.Sprintf("runs_%v", id)
					}
				}
				return "runs_default" // Safe fallback
			}),
			MaxAttempts: 1, // matches trigger.dev
			Handler:     handlePerformRunExecution,
		},
		"performTaskOperation": TaskDefinition{
			QueueName:   StaticQueueName("tasks"), // ✅ Convert to StaticQueueName
			MaxAttempts: 3,                        // matches trigger.dev
			Handler:     handlePerformTaskOperation,
		},
		"deliverEvent": TaskDefinition{
			// ✅ Phase 2: Dynamic queue - project-level event routing
			QueueName: DynamicQueueName(func(payload interface{}) string {
				// Try to cast to DeliverEventArgs (our enhanced structure)
				if eventArgs, ok := payload.(DeliverEventArgs); ok {
					if eventArgs.ProjectID != "" {
						// Project-specific event queue
						return fmt.Sprintf("events_project_%s", eventArgs.ProjectID)
					}
					if eventArgs.EventType != "" {
						// Event type-specific queue
						return fmt.Sprintf("events_type_%s", eventArgs.EventType)
					}
				}
				// Fallback for map payload
				if payloadMap, ok := payload.(map[string]interface{}); ok {
					if projectId, exists := payloadMap["projectId"]; exists {
						return fmt.Sprintf("events_project_%v", projectId)
					}
					if eventType, exists := payloadMap["eventType"]; exists {
						return fmt.Sprintf("events_type_%v", eventType)
					}
				}
				return "events_default" // Safe fallback to default event queue
			}),
			MaxAttempts: 5, // matches trigger.dev
			Handler:     handleDeliverEvent,
		},
		"events.invokeDispatcher": TaskDefinition{
			QueueName:   StaticQueueName("event-dispatcher"), // ✅ Add missing queue name
			MaxAttempts: 3,                                   // matches trigger.dev
			Handler:     handleInvokeDispatcher,
		},
		"runFinished": TaskDefinition{
			QueueName:   StaticQueueName("executions"), // ✅ Add missing queue name
			MaxAttempts: 3,                             // matches trigger.dev
			Handler:     handleRunFinished,
		},
		"startQueuedRuns": TaskDefinition{
			// ✅ Phase 2: Dynamic queue - project-level isolation
			QueueName: DynamicQueueName(func(payload interface{}) string {
				// Try to cast to StartQueuedRunsArgs (our new structure)
				if runArgs, ok := payload.(StartQueuedRunsArgs); ok {
					return fmt.Sprintf("project_%s_runs", runArgs.ProjectID) // project_proj-123_runs
				}
				// Fallback for map payload
				if payloadMap, ok := payload.(map[string]interface{}); ok {
					if projectId, exists := payloadMap["projectId"]; exists {
						return fmt.Sprintf("project_%v_runs", projectId)
					}
				}
				return "project_default_runs" // Safe fallback
			}),
			MaxAttempts: 3, // matches trigger.dev
			Handler:     handleStartQueuedRuns,
		},
		// ✅ Phase 2: User-level task routing with plan and geographic isolation
		"processUserTask": TaskDefinition{
			QueueName: DynamicQueueName(func(payload interface{}) string {
				// Try to cast to UserTaskArgs
				if userArgs, ok := payload.(UserTaskArgs); ok {
					// Multi-dimensional routing: plan + region
					switch userArgs.UserPlan {
					case "enterprise":
						return fmt.Sprintf("enterprise_%s_high-priority", userArgs.Region)
					case "pro":
						return fmt.Sprintf("pro_%s_medium-priority", userArgs.Region)
					case "free":
						return fmt.Sprintf("free_%s_standard", userArgs.Region)
					default:
						return fmt.Sprintf("standard_%s_normal", userArgs.Region)
					}
				}
				// Fallback for map payload
				if payloadMap, ok := payload.(map[string]interface{}); ok {
					userPlan := "standard"
					region := "default"
					if plan, exists := payloadMap["userPlan"]; exists {
						userPlan = fmt.Sprintf("%v", plan)
					}
					if reg, exists := payloadMap["region"]; exists {
						region = fmt.Sprintf("%v", reg)
					}
					return fmt.Sprintf("%s_%s_tasks", userPlan, region)
				}
				return "standard_default_tasks" // Safe fallback
			}),
			MaxAttempts: 5,
			Handler:     handleUserTask,
		},
	}
}

// Additional handlers for completeness (placeholders for now)
func handleScheduleEmail(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("scheduleEmail handler not implemented")
}

func handlePerformRunExecution(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("performRunExecution handler not implemented")
}

func handlePerformTaskOperation(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("performTaskOperation handler not implemented")
}

func handleRunFinished(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("runFinished handler not implemented")
}

func handleStartQueuedRuns(ctx context.Context, payload json.RawMessage, job JobContext) error {
	return fmt.Errorf("startQueuedRuns handler not implemented")
}

// ✅ Phase 2: User task handler with dynamic queue routing context
func handleUserTask(ctx context.Context, payload json.RawMessage, job JobContext) error {
	// Parse the payload to get user context
	var userArgs UserTaskArgs
	if err := json.Unmarshal(payload, &userArgs); err == nil {
		// Log the dynamic queue routing for demonstration
		fmt.Printf("Processing user task for user %s (plan: %s, region: %s) in queue: %s\n",
			userArgs.UserID, userArgs.UserPlan, userArgs.Region, job.Queue)
	}
	return fmt.Errorf("userTask handler not implemented - processed in queue: %s", job.Queue)
}
