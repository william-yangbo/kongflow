package queue

import (
	"context"
	"testing"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockManager 模拟WorkerQueue Manager，用于测试
type MockManager struct {
	mock.Mock
}

func (m *MockManager) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func (m *MockManager) EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, tx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func TestRiverQueueService_EnqueueDeliverEvent(t *testing.T) {
	tests := []struct {
		name        string
		request     *EnqueueDeliverEventRequest
		setupMock   func(*MockManager)
		expectError bool
	}{
		{
			name: "successful_enqueue_with_defaults",
			request: &EnqueueDeliverEventRequest{
				EventID:    "test-event-123",
				EndpointID: "test-endpoint-456",
				Payload:    `{"type": "test", "data": "hello world"}`,
			},
			setupMock: func(m *MockManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    1,
						State: "available",
					},
				}
				m.On("EnqueueJob", mock.Anything, "deliver_event", mock.AnythingOfType("workerqueue.DeliverEventArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "successful_enqueue_with_scheduled_time",
			request: &EnqueueDeliverEventRequest{
				EventID:      "test-event-456",
				EndpointID:   "test-endpoint-789",
				Payload:      `{"type": "scheduled", "data": "scheduled event"}`,
				ScheduledFor: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
			},
			setupMock: func(m *MockManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    2,
						State: "scheduled",
					},
				}
				m.On("EnqueueJob", mock.Anything, "deliver_event", mock.AnythingOfType("workerqueue.DeliverEventArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockManager{}
			tt.setupMock(mockManager)

			service := &riverQueueService{
				manager: mockManager,
			}

			result, err := service.EnqueueDeliverEvent(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestRiverQueueService_EnqueueInvokeDispatcher(t *testing.T) {
	tests := []struct {
		name        string
		request     *EnqueueInvokeDispatcherRequest
		setupMock   func(*MockManager)
		expectError bool
	}{
		{
			name: "successful_enqueue_with_defaults",
			request: &EnqueueInvokeDispatcherRequest{
				DispatcherID: "test-dispatcher-123",
				EventID:      "test-event-456",
				Payload:      `{"type": "dispatcher_test", "data": "invoke dispatcher"}`,
			},
			setupMock: func(m *MockManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    1,
						State: "available",
					},
				}
				m.On("EnqueueJob", mock.Anything, "invoke_dispatcher", mock.AnythingOfType("workerqueue.InvokeDispatcherArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "successful_enqueue_with_scheduled_time",
			request: &EnqueueInvokeDispatcherRequest{
				DispatcherID: "test-dispatcher-789",
				EventID:      "test-event-012",
				Payload:      `{"type": "scheduled_dispatcher", "data": "scheduled invocation"}`,
				ScheduledFor: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
			},
			setupMock: func(m *MockManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    2,
						State: "scheduled",
					},
				}
				m.On("EnqueueJob", mock.Anything, "invoke_dispatcher", mock.AnythingOfType("workerqueue.InvokeDispatcherArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockManager{}
			tt.setupMock(mockManager)

			service := &riverQueueService{
				manager: mockManager,
			}

			result, err := service.EnqueueInvokeDispatcher(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestRiverQueueService_EnqueueDeliverEventTx(t *testing.T) {
	mockManager := &MockManager{}

	req := &EnqueueDeliverEventRequest{
		EventID:    "test-event-tx-123",
		EndpointID: "test-endpoint-tx-456",
		Payload:    `{"type": "transaction_test", "data": "transaction delivery"}`,
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    1,
			State: "available",
		},
	}

	mockManager.On("EnqueueJobTx", mock.Anything, mock.Anything, "deliver_event", mock.AnythingOfType("workerqueue.DeliverEventArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		manager: mockManager,
	}

	actualResult, err := service.EnqueueDeliverEventTx(context.Background(), nil, req)

	assert.NoError(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, result, actualResult)
	mockManager.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueInvokeDispatcherTx(t *testing.T) {
	mockManager := &MockManager{}

	req := &EnqueueInvokeDispatcherRequest{
		DispatcherID: "test-dispatcher-tx-123",
		EventID:      "test-event-tx-456",
		Payload:      `{"type": "transaction_test", "data": "transaction invocation"}`,
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    1,
			State: "available",
		},
	}

	mockManager.On("EnqueueJobTx", mock.Anything, mock.Anything, "invoke_dispatcher", mock.AnythingOfType("workerqueue.InvokeDispatcherArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		manager: mockManager,
	}

	actualResult, err := service.EnqueueInvokeDispatcherTx(context.Background(), nil, req)

	assert.NoError(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, result, actualResult)
	mockManager.AssertExpectations(t)
}

func TestRiverQueueService_JobOptionsConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		scheduledFor *time.Time
		expectedOpts func(*time.Time) *workerqueue.JobOptions
	}{
		{
			name:         "default_options",
			scheduledFor: nil,
			expectedOpts: func(*time.Time) *workerqueue.JobOptions {
				return &workerqueue.JobOptions{
					QueueName: string(workerqueue.QueueEvents),
					Priority:  int(workerqueue.PriorityHigh),
					RunAt:     nil,
				}
			},
		},
		{
			name:         "with_scheduled_time",
			scheduledFor: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
			expectedOpts: func(scheduledFor *time.Time) *workerqueue.JobOptions {
				return &workerqueue.JobOptions{
					QueueName: string(workerqueue.QueueEvents),
					Priority:  int(workerqueue.PriorityHigh),
					RunAt:     scheduledFor,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockManager{}
			service := &riverQueueService{manager: mockManager}

			expectedOpts := tt.expectedOpts(tt.scheduledFor)

			// Test with deliver event
			deliveryReq := &EnqueueDeliverEventRequest{
				EventID:      "test-options-event",
				EndpointID:   "test-options-endpoint",
				Payload:      `{"test": "options"}`,
				ScheduledFor: tt.scheduledFor,
			}

			result := &rivertype.JobInsertResult{
				Job: &rivertype.JobRow{ID: 1, State: "available"},
			}

			mockManager.On("EnqueueJob", mock.Anything, "deliver_event", mock.AnythingOfType("workerqueue.DeliverEventArgs"), mock.MatchedBy(func(opts *workerqueue.JobOptions) bool {
				return opts.QueueName == expectedOpts.QueueName &&
					opts.Priority == expectedOpts.Priority &&
					((opts.RunAt == nil && expectedOpts.RunAt == nil) ||
						(opts.RunAt != nil && expectedOpts.RunAt != nil))
			})).Return(result, nil)

			_, err := service.EnqueueDeliverEvent(context.Background(), deliveryReq)
			assert.NoError(t, err)

			mockManager.AssertExpectations(t)
		})
	}
}
