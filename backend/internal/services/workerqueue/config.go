// Package workerqueue provides worker queue functionality using River Queue
package workerqueue

import (
	"time"
)

// Config contains configuration for the worker system
type Config struct {
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string

	// MaxWorkers is the maximum number of workers for the default queue
	MaxWorkers int

	// ExecutionMaxWorkers is the maximum number of workers for the execution queue
	ExecutionMaxWorkers int

	// EventsMaxWorkers is the maximum number of workers for the events queue
	EventsMaxWorkers int

	// MaintenanceMaxWorkers is the maximum number of workers for maintenance tasks
	MaintenanceMaxWorkers int

	// FetchCooldown is the minimum time between job fetches
	FetchCooldown time.Duration

	// JobTimeout is the default timeout for jobs
	JobTimeout time.Duration

	// FetchPollInterval is the interval for polling new jobs
	FetchPollInterval time.Duration

	// Schema is the database schema to use for River tables
	Schema string

	// TestMode indicates if running in test mode
	TestMode bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxWorkers:            10,
		ExecutionMaxWorkers:   5,
		EventsMaxWorkers:      20,
		MaintenanceMaxWorkers: 2,
		FetchCooldown:         100 * time.Millisecond,
		JobTimeout:            1 * time.Minute,
		FetchPollInterval:     1 * time.Second,
		Schema:                "public",
		TestMode:              false,
	}
}

// QueueNames contains the standard queue names used by KongFlow
type QueueNames struct {
	Default     string
	Execution   string
	Events      string
	Maintenance string
}

// StandardQueues returns the standard queue configuration
func StandardQueues() QueueNames {
	return QueueNames{
		Default:     "default",
		Execution:   "execution",
		Events:      "events",
		Maintenance: "maintenance",
	}
}
