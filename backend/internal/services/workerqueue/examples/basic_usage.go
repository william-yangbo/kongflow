// Package main provides examples of how to use the Worker Queue Service
// This demonstrates usage patterns that align with trigger.dev's ZodWorker
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"kongflow/backend/internal/services/workerqueue"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Example 1: Basic worker queue setup
	if err := basicSetupExample(ctx, logger); err != nil {
		log.Fatalf("Basic example failed: %v", err)
	}

	logger.Info("All examples completed successfully")
}

// basicSetupExample demonstrates basic worker queue usage
// This mirrors trigger.dev's worker.server.ts setup pattern
func basicSetupExample(ctx context.Context, logger *slog.Logger) error {
	logger.Info("Running basic setup example")

	// Create database connection
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/kongflow?sslmode=disable"
	}

	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}
	defer dbPool.Close()

	// Initialize Worker Queue Client (using proper ClientOptions)
	client, err := workerqueue.NewClient(workerqueue.ClientOptions{
		DatabasePool: dbPool,
		RunnerOptions: workerqueue.RunnerOptions{
			Concurrency:  5,    // matches trigger.dev default
			PollInterval: 1000, // matches trigger.dev default
		},
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create worker queue client: %w", err)
	}

	// Initialize the queue system (creates tables if needed)
	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize worker queue: %w", err)
	}

	// Enqueue a job (mirrors trigger.dev's trigger() function)
	jobID, err := client.Enqueue(ctx, "index-endpoint", &workerqueue.IndexEndpointArgs{
		ID:     "endpoint-123",
		Source: workerqueue.IndexSourceAPI,
		Reason: "Test indexing from basic example",
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	logger.Info("Job enqueued successfully", "job_id", jobID)
	return nil
}
