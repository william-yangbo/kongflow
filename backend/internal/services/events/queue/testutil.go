package queue

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/database"
	"kongflow/backend/internal/services/workerqueue"
)

// EventQueueTestHarness provides a complete testing environment for events queue
type EventQueueTestHarness struct {
	TestDB             *database.TestDB
	RiverClient        *river.Client[pgx.Tx]
	QueueService       QueueService
	MockEventProcessor *MockEventProcessor
	Logger             *slog.Logger
	t                  *testing.T
}

// TestWorkerQueueManager implements WorkerQueueManager interface for testing
type TestWorkerQueueManager struct {
	riverClient *river.Client[pgx.Tx]
	logger      *slog.Logger
}

// EnqueueJob implements the WorkerQueueManager interface
func (m *TestWorkerQueueManager) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	var insertOpts river.InsertOpts
	if opts != nil {
		insertOpts.Queue = opts.QueueName
		insertOpts.Priority = opts.Priority
		if opts.RunAt != nil {
			insertOpts.ScheduledAt = *opts.RunAt
		}
	}

	switch identifier {
	case "deliver_event":
		args, ok := payload.(workerqueue.DeliverEventArgs)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for deliver_event: %T", payload)
		}
		return m.riverClient.Insert(ctx, args, &insertOpts)

	case "invoke_dispatcher":
		args, ok := payload.(workerqueue.InvokeDispatcherArgs)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for invoke_dispatcher: %T", payload)
		}
		return m.riverClient.Insert(ctx, args, &insertOpts)

	default:
		return nil, fmt.Errorf("unknown job identifier: %s", identifier)
	}
}

// EnqueueJobTx implements the WorkerQueueManager interface
func (m *TestWorkerQueueManager) EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	var insertOpts river.InsertOpts
	if opts != nil {
		insertOpts.Queue = opts.QueueName
		insertOpts.Priority = opts.Priority
		if opts.RunAt != nil {
			insertOpts.ScheduledAt = *opts.RunAt
		}
	}

	switch identifier {
	case "deliver_event":
		args, ok := payload.(workerqueue.DeliverEventArgs)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for deliver_event: %T", payload)
		}
		return m.riverClient.InsertTx(ctx, tx, args, &insertOpts)

	case "invoke_dispatcher":
		args, ok := payload.(workerqueue.InvokeDispatcherArgs)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for invoke_dispatcher: %T", payload)
		}
		return m.riverClient.InsertTx(ctx, tx, args, &insertOpts)

	default:
		return nil, fmt.Errorf("unknown job identifier: %s", identifier)
	}
}

// MockEventProcessor simulates event processing operations
type MockEventProcessor struct {
	DeliveredEvents    []EventDeliveryOperation
	InvokedDispatchers []DispatcherInvocationOperation
	FailureSimulation  map[string]error
}

// EventDeliveryOperation represents a completed event delivery
type EventDeliveryOperation struct {
	EventID    string                 `json:"eventId"`
	Status     string                 `json:"status"`
	DeliveryID string                 `json:"deliveryId"`
	Stats      map[string]interface{} `json:"stats"`
	Error      error                  `json:"error,omitempty"`
}

// DispatcherInvocationOperation represents a completed dispatcher invocation
type DispatcherInvocationOperation struct {
	DispatcherID string                 `json:"dispatcherId"`
	EventID      string                 `json:"eventId"`
	Status       string                 `json:"status"`
	InvokeID     string                 `json:"invokeId"`
	Stats        map[string]interface{} `json:"stats"`
	Error        error                  `json:"error,omitempty"`
}

// SetupEventQueueTestHarness creates a complete test environment for events queue
func SetupEventQueueTestHarness(t *testing.T) *EventQueueTestHarness {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	testDB := database.SetupTestDB(t)

	mockEventProcessor := &MockEventProcessor{
		DeliveredEvents:    make([]EventDeliveryOperation, 0),
		InvokedDispatchers: make([]DispatcherInvocationOperation, 0),
		FailureSimulation:  make(map[string]error),
	}

	workers := river.NewWorkers()

	river.AddWorker(workers, NewMockDeliverEventWorker(mockEventProcessor, logger))
	river.AddWorker(workers, NewMockInvokeDispatcherWorker(mockEventProcessor, logger))

	riverConfig := &river.Config{
		Queues: map[string]river.QueueConfig{
			string(workerqueue.QueueDefault):   {MaxWorkers: 5},
			string(workerqueue.QueueEvents):    {MaxWorkers: 20},
			string(workerqueue.QueueExecution): {MaxWorkers: 5},
		},
		Workers: workers,
	}

	riverClient, err := river.NewClient(riverpgxv5.New(testDB.Pool), riverConfig)
	require.NoError(t, err)

	migrator, err := rivermigrate.New(riverpgxv5.New(testDB.Pool), nil)
	require.NoError(t, err)
	_, err = migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	manager := &TestWorkerQueueManager{
		riverClient: riverClient,
		logger:      logger,
	}

	queueService := NewRiverQueueService(manager)

	return &EventQueueTestHarness{
		TestDB:             testDB,
		RiverClient:        riverClient,
		QueueService:       queueService,
		MockEventProcessor: mockEventProcessor,
		Logger:             logger,
		t:                  t,
	}
}

// Start starts the worker queue manager
func (th *EventQueueTestHarness) Start(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	err := th.RiverClient.Start(ctx)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
}

// Cleanup cleans up all resources
func (th *EventQueueTestHarness) Cleanup() {
	ctx := context.Background()
	if th.RiverClient != nil {
		_ = th.RiverClient.Stop(ctx)
	}
	if th.TestDB != nil && th.t != nil {
		th.TestDB.Cleanup(th.t)
	}
}

// ClearEventOperations clears all recorded operations for fresh testing
func (th *EventQueueTestHarness) ClearEventOperations() error {
	th.MockEventProcessor.DeliveredEvents = th.MockEventProcessor.DeliveredEvents[:0]
	th.MockEventProcessor.InvokedDispatchers = th.MockEventProcessor.InvokedDispatchers[:0]
	return nil
}

// WaitForEventOperations waits for a specified number of event operations to complete
func (th *EventQueueTestHarness) WaitForEventOperations(expectedOperations int, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		totalOperations := len(th.MockEventProcessor.DeliveredEvents) + len(th.MockEventProcessor.InvokedDispatchers)
		if totalOperations >= expectedOperations {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	actualOperations := len(th.MockEventProcessor.DeliveredEvents) + len(th.MockEventProcessor.InvokedDispatchers)
	return fmt.Errorf("timeout waiting for event operations: expected=%d, actual=%d", expectedOperations, actualOperations)
}

// SimulateEventFailure configures the mock processor to simulate a failure for a specific event
func (th *EventQueueTestHarness) SimulateEventFailure(eventID string, err error) {
	th.MockEventProcessor.FailureSimulation[eventID] = err
}

// SimulateDispatcherFailure configures the mock processor to simulate a failure for a specific dispatcher
func (th *EventQueueTestHarness) SimulateDispatcherFailure(dispatcherID string, err error) {
	th.MockEventProcessor.FailureSimulation[dispatcherID] = err
}
