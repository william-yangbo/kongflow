// Package workerqueue provides convenience functions for job insertion
package workerqueue

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// JobOptions provides enhanced options for job insertion
// This structure aligns with trigger.dev's ZodWorkerEnqueueOptions
type JobOptions struct {
	// Queue assignment (corresponds to trigger.dev queueName)
	// Can be a static string or resolved dynamically from payload
	QueueName string

	// Scheduling (corresponds to trigger.dev runAt)
	RunAt *time.Time

	// Priority (lower number = higher priority, like trigger.dev)
	Priority int

	// Retry configuration (corresponds to trigger.dev maxAttempts)
	MaxAttempts int

	// Job identification and deduplication (corresponds to trigger.dev jobKey)
	JobKey string

	// Job key mode (corresponds to trigger.dev jobKeyMode)
	JobKeyMode string // "replace", "preserve_run_at", "unsafe_dedupe"

	// Tags for job organization (corresponds to trigger.dev tags)
	Tags []string

	// Flags for job metadata (corresponds to trigger.dev flags)
	Flags []string

	// UniqueOpts for River-specific uniqueness constraints
	UniqueOpts *river.UniqueOpts
}

// InsertOpts provides River-specific insertion options
type InsertOpts struct {
	// All the standard River InsertOpts
	*river.InsertOpts
}

// TransactionContext provides unified transaction handling for SQLC + River
type TransactionContext struct {
	Tx pgx.Tx

	// Future: can add SQLC Queries here when needed
	// SQLCQueries *database.Queries
}

// BusinessLogicFunc defines the signature for transaction business logic
type BusinessLogicFunc func(ctx context.Context, txCtx *TransactionContext) error
