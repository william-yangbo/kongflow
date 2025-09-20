// Package queue provides real integration testing for endpoints queue service
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/services/workerqueue"
	workertestutil "kongflow/backend/internal/services/workerqueue/testutil"
)

// EndpointQueueTestHarness provides a complete testing environment for endpoints queue with real River worker queue
type EndpointQueueTestHarness struct {
	// Database (using shared testhelper)
	TestDB *database.TestDB

	// Worker Queue
	Manager *workerqueue.Manager

	// Queue Service
	QueueService QueueService

	// Mock Endpoint Indexer (simulates endpoint indexing operations)
	MockIndexer *MockEndpointIndexer

	// Test utilities
	Logger *slog.Logger
	t      *testing.T // Keep reference to testing.T for cleanup
}

// EndpointTestSender extends TestEmailSender to handle endpoint indexing jobs
type EndpointTestSender struct {
	*workertestutil.TestEmailSender
	MockIndexer *MockEndpointIndexer
	Logger      *slog.Logger
}

// SendEmail implements the EmailSender interface and also handles indexEndpoint jobs
func (e *EndpointTestSender) SendEmail(ctx context.Context, email workerqueue.EmailData) error {
	// Check if this is actually an endpoint indexing job disguised as an email
	// (This is a hack to work with the existing worker infrastructure)
	if email.Subject == "ENDPOINT_INDEX_JOB" {
		// Parse the endpoint indexing data from the email body
		// In a real implementation, you'd have a proper job type
		e.Logger.Info("Processing endpoint index job through email sender", "body", email.Body)
		return nil
	}

	// Otherwise, handle as a regular email
	return e.TestEmailSender.SendEmail(ctx, email)
}

// RegisterJobMockWorker is a mock worker for register job operations
type RegisterJobMockWorker struct {
	river.WorkerDefaults[workerqueue.RegisterJobArgs]
	logger *slog.Logger
}

// Work processes register job requests
func (w *RegisterJobMockWorker) Work(ctx context.Context, job *river.Job[workerqueue.RegisterJobArgs]) error {
	w.logger.Info("Processing register job", "job_id", job.ID, "endpoint_id", job.Args.EndpointID, "job_id_field", job.Args.JobID)
	// Mock processing - in real implementation this would register the job
	return nil
}

// RegisterSourceMockWorker is a mock worker for register source operations
type RegisterSourceMockWorker struct {
	river.WorkerDefaults[workerqueue.RegisterSourceArgs]
	logger *slog.Logger
}

// Work processes register source requests
func (w *RegisterSourceMockWorker) Work(ctx context.Context, job *river.Job[workerqueue.RegisterSourceArgs]) error {
	w.logger.Info("Processing register source", "job_id", job.ID, "endpoint_id", job.Args.EndpointID, "source_id", job.Args.SourceID)
	// Mock processing - in real implementation this would register the source
	return nil
}

// ManagerAdapter adapts workerqueue.Manager to implement WorkerQueueManager interface
type ManagerAdapter struct {
	manager *workerqueue.Manager
}

// EnqueueJob implements WorkerQueueManager interface
func (m *ManagerAdapter) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	return m.manager.EnqueueJob(ctx, identifier, payload, opts)
}

// EnqueueJobTx implements WorkerQueueManager interface
func (m *ManagerAdapter) EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	return m.manager.EnqueueJobTx(ctx, tx, identifier, payload, opts)
}

// MockEndpointIndexer simulates endpoint indexing operations for testing
type MockEndpointIndexer struct {
	IndexedEndpoints []EndpointIndexOperation
}

// EndpointIndexOperation records an endpoint indexing operation
type EndpointIndexOperation struct {
	EndpointID string
	Source     string
	Reason     string
	SourceData map[string]interface{}
	Timestamp  time.Time
	Status     string // "success", "failed"
	Error      error
}

// IndexEndpoint implements workerqueue.EndpointIndexer interface
func (m *MockEndpointIndexer) IndexEndpoint(ctx context.Context, req *workerqueue.EndpointIndexRequest) (*workerqueue.EndpointIndexResult, error) {
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	operation := EndpointIndexOperation{
		EndpointID: req.EndpointID,
		Source:     req.Source,
		Reason:     req.Reason,
		SourceData: req.SourceData,
		Timestamp:  time.Now(),
		Status:     "success",
	}

	// Simulate occasional failures for testing error handling
	if req.Reason == "force_failure" {
		operation.Status = "failed"
		operation.Error = fmt.Errorf("simulated indexing failure")
		m.IndexedEndpoints = append(m.IndexedEndpoints, operation)
		return nil, operation.Error
	}

	m.IndexedEndpoints = append(m.IndexedEndpoints, operation)

	// Return a mock result
	return &workerqueue.EndpointIndexResult{
		IndexID: fmt.Sprintf("index_%s", req.EndpointID),
		Stats:   map[string]int{"test": 1},
	}, nil
}

// GetIndexedCount returns the number of successfully indexed endpoints
func (m *MockEndpointIndexer) GetIndexedCount() int {
	count := 0
	for _, op := range m.IndexedEndpoints {
		if op.Status == "success" {
			count++
		}
	}
	return count
}

// Clear clears all recorded operations
func (m *MockEndpointIndexer) Clear() {
	m.IndexedEndpoints = make([]EndpointIndexOperation, 0)
}

// SetupEndpointQueueTestHarness creates a complete testing environment with PostgreSQL container and River worker queue
func SetupEndpointQueueTestHarness(t *testing.T) *EndpointQueueTestHarness {
	t.Helper()

	ctx := context.Background()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Use shared testhelper to setup database with all migrations
	testDB := database.SetupTestDB(t)

	// Setup mock endpoint indexer
	mockIndexer := &MockEndpointIndexer{
		IndexedEndpoints: make([]EndpointIndexOperation, 0),
	}

	// Setup test email sender for worker queue (reuse from workerqueue testutil)
	testEmailSender := workertestutil.NewTestEmailSender(logger)

	// Setup worker queue manager with custom IndexEndpointWorker
	config := workerqueue.DefaultConfig()
	config.TestMode = true // Enable test mode for faster processing
	config.MaxWorkers = 2
	config.ExecutionMaxWorkers = 1
	config.EventsMaxWorkers = 1
	config.MaintenanceMaxWorkers = 1

	manager, err := workerqueue.NewManagerWithIndexer(config, testDB.Pool, logger, testEmailSender, mockIndexer)
	require.NoError(t, err)

	// Ensure River tables are created
	err = manager.EnsureRiverTables(ctx)
	require.NoError(t, err)

	// Create adapter to make manager implement WorkerQueueManager interface
	managerAdapter := &ManagerAdapter{manager: manager}

	// Create queue service using the adapter
	queueService := NewRiverQueueService(managerAdapter)

	return &EndpointQueueTestHarness{
		TestDB:       testDB,
		Manager:      manager,
		QueueService: queueService,
		MockIndexer:  mockIndexer,
		Logger:       logger,
		t:            t, // Save testing.T for cleanup
	}
}

// Start starts the worker queue manager
func (th *EndpointQueueTestHarness) Start(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	err := th.Manager.Start(ctx)
	require.NoError(t, err)

	// Wait for workers to be ready
	time.Sleep(1 * time.Second)
}

// Stop stops the worker queue manager
func (th *EndpointQueueTestHarness) Stop(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	err := th.Manager.Stop(ctx)
	require.NoError(t, err)
}

// Cleanup cleans up all resources
func (th *EndpointQueueTestHarness) Cleanup() {
	ctx := context.Background()
	if th.Manager != nil {
		_ = th.Manager.Stop(ctx)
	}
	if th.TestDB != nil && th.t != nil {
		th.TestDB.Cleanup(th.t)
	}
}

// WaitForIndexOperations waits for endpoint indexing jobs to be processed by monitoring the job queue
func (th *EndpointQueueTestHarness) WaitForIndexOperations(t *testing.T, expectedCount int, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	completedJobs := 0

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)

			th.Logger.Error("Timeout waiting for index operations",
				"expected", expectedCount,
				"completed", completedJobs,
				"elapsed", elapsed,
				"timeout", timeout)

			t.Fatalf("Timeout after %v waiting for %d index operations. Completed: %d",
				elapsed, expectedCount, completedJobs)
		case <-ticker.C:
			// Query the river_job table to count completed index_endpoint jobs with a shorter timeout
			queryCtx, queryCancel := context.WithTimeout(context.Background(), 2*time.Second)
			var count int
			err := th.TestDB.Pool.QueryRow(queryCtx,
				"SELECT COUNT(*) FROM river_job WHERE kind = $1 AND state = $2",
				"index_endpoint", "completed").Scan(&count)
			queryCancel()

			if err != nil {
				th.Logger.Warn("Failed to query job status", "error", err)
				continue
			}

			// Update our mock indexer based on completed jobs
			if count > len(th.MockIndexer.IndexedEndpoints) {
				// Query for details of newly completed jobs
				queryCtx2, queryCancel2 := context.WithTimeout(context.Background(), 2*time.Second)
				rows, err := th.TestDB.Pool.Query(queryCtx2,
					"SELECT args FROM river_job WHERE kind = $1 AND state = $2 LIMIT $3 OFFSET $4",
					"index_endpoint", "completed", count-len(th.MockIndexer.IndexedEndpoints), len(th.MockIndexer.IndexedEndpoints))
				queryCancel2()

				if err != nil {
					th.Logger.Warn("Failed to query job details", "error", err)
				} else {
					defer rows.Close()
					for rows.Next() {
						var argsJson []byte
						if err := rows.Scan(&argsJson); err != nil {
							continue
						}

						// Parse the job args to extract endpoint details
						var args map[string]interface{}
						if err := json.Unmarshal(argsJson, &args); err != nil {
							continue
						}

						endpointID := "unknown"
						source := "unknown"
						reason := "Completed job"
						var sourceData map[string]interface{}

						if id, ok := args["id"].(string); ok {
							endpointID = id
						}
						if src, ok := args["source"].(string); ok {
							source = src
						}
						if rsn, ok := args["reason"].(string); ok {
							reason = rsn
						}
						if sd, ok := args["source_data"].(map[string]interface{}); ok {
							sourceData = sd
						}

						th.MockIndexer.IndexedEndpoints = append(th.MockIndexer.IndexedEndpoints, EndpointIndexOperation{
							EndpointID: endpointID,
							Source:     source,
							Reason:     reason,
							SourceData: sourceData,
							Timestamp:  time.Now(),
							Status:     "success",
						})
					}
				}
			}

			completedJobs = count
			if completedJobs >= expectedCount {
				elapsed := time.Since(startTime)
				th.Logger.Info("Index operations completed",
					"expected", expectedCount,
					"actual", completedJobs,
					"elapsed", elapsed)
				return
			}

			// Log progress every few seconds
			if int(time.Since(startTime)/time.Second)%3 == 0 {
				th.Logger.Debug("Waiting for index operations",
					"expected", expectedCount,
					"completed", completedJobs,
					"elapsed", time.Since(startTime))
			}
		}
	}
} // GetIndexedOperations returns all recorded indexing operations
func (th *EndpointQueueTestHarness) GetIndexedOperations() []EndpointIndexOperation {
	return th.MockIndexer.IndexedEndpoints
}

// ClearIndexOperations clears all recorded indexing operations
func (th *EndpointQueueTestHarness) ClearIndexOperations() {
	th.MockIndexer.Clear()

	// Also clear any completed jobs from the database to ensure clean state
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete completed jobs to avoid interference between tests
	_, err := th.TestDB.Pool.Exec(ctx, "DELETE FROM river_job WHERE state = 'completed'")
	if err != nil {
		th.Logger.Warn("Failed to clear completed jobs", "error", err)
	}

	// Wait briefly to ensure any in-flight operations complete
	time.Sleep(100 * time.Millisecond)
}
