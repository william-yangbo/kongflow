package queue

import (
	"context"
	"testing"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkerQueueManager 实现 WorkerQueueManager 接口
type MockWorkerQueueManager struct {
	mock.Mock
}

func (m *MockWorkerQueueManager) EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func (m *MockWorkerQueueManager) EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, tx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func TestRiverQueueService_EnqueueIndexEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		request     *EnqueueIndexEndpointRequest
		setupMock   func(*MockWorkerQueueManager)
		expectError bool
	}{
		{
			name: "successful_enqueue_with_defaults",
			request: &EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceInternal,
				Reason:     "Test indexing",
			},
			setupMock: func(m *MockWorkerQueueManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    1,
						State: "available",
					},
				}
				m.On("EnqueueJob", mock.Anything, "index_endpoint", mock.AnythingOfType("workerqueue.IndexEndpointArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "successful_enqueue_with_custom_options",
			request: &EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceAPI,
				Reason:     "API triggered indexing",
				QueueName:  "custom_queue",
				Priority:   1,
				RunAt:      func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
				SourceData: map[string]interface{}{"trigger": "api"},
			},
			setupMock: func(m *MockWorkerQueueManager) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    2,
						State: "scheduled",
					},
				}
				m.On("EnqueueJob", mock.Anything, "index_endpoint", mock.AnythingOfType("workerqueue.IndexEndpointArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockWorkerQueueManager{}
			tt.setupMock(mockManager)

			service := &riverQueueService{
				manager: mockManager,
			}

			result, err := service.EnqueueIndexEndpoint(context.Background(), tt.request)

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

func TestRiverQueueService_EnqueueIndexEndpointTx(t *testing.T) {
	mockManager := &MockWorkerQueueManager{}
	
	endpointID := uuid.New()
	req := &EnqueueIndexEndpointRequest{
		EndpointID: endpointID,
		Source:     EndpointIndexSourceAPI,
		Reason:     "Transaction test",
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    1,
			State: "available",
		},
	}

	// 使用简单的接口类型作为mock
	mockManager.On("EnqueueJobTx", mock.Anything, mock.Anything, "index_endpoint", mock.AnythingOfType("workerqueue.IndexEndpointArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		manager: mockManager,
	}

	// 使用nil作为事务，因为我们只关心业务逻辑
	actualResult, err := service.EnqueueIndexEndpointTx(context.Background(), nil, req)

	assert.NoError(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, result, actualResult)
	mockManager.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueRegisterJob(t *testing.T) {
	mockManager := &MockWorkerQueueManager{}
	
	endpointID := uuid.New()
	req := &RegisterJobRequest{
		EndpointID:  endpointID,
		JobID:       "test-job",
		JobMetadata: map[string]interface{}{"name": "Test Job"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    1,
			State: "available",
		},
	}

	mockManager.On("EnqueueJob", mock.Anything, "register_job", mock.AnythingOfType("workerqueue.RegisterJobArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		manager: mockManager,
	}

	actualResult, err := service.EnqueueRegisterJob(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, result, actualResult)
	mockManager.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueRegisterJobTx(t *testing.T) {
	mockManager := &MockWorkerQueueManager{}
	
	endpointID := uuid.New()
	req := &RegisterJobRequest{
		EndpointID:  endpointID,
		JobID:       "test-job",
		JobMetadata: map[string]interface{}{"name": "Test Job"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    1,
			State: "available",
		},
	}

	mockManager.On("EnqueueJobTx", mock.Anything, mock.Anything, "register_job", mock.AnythingOfType("workerqueue.RegisterJobArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		manager: mockManager,
	}

	actualResult, err := service.EnqueueRegisterJobTx(context.Background(), nil, req)

	assert.NoError(t, err)
	assert.NotNil(t, actualResult)
	assert.Equal(t, result, actualResult)
	mockManager.AssertExpectations(t)
}

func TestBuildJobOptions(t *testing.T) {
	tests := []struct {
		name      string
		queueName string
		priority  int
		runAt     interface{}
		expected  *workerqueue.JobOptions
	}{
		{
			name:      "defaults",
			queueName: "",
			priority:  0,
			runAt:     nil,
			expected: &workerqueue.JobOptions{
				QueueName: string(workerqueue.QueueDefault),
				Priority:  int(workerqueue.PriorityNormal),
			},
		},
		{
			name:      "custom_values",
			queueName: "custom",
			priority:  5,
			runAt:     nil,
			expected: &workerqueue.JobOptions{
				QueueName: "custom",
				Priority:  5,
			},
		},
		{
			name:      "with_run_at",
			queueName: "test",
			priority:  1,
			runAt:     func() *time.Time { t := time.Now(); return &t }(),
			expected: &workerqueue.JobOptions{
				QueueName: "test",
				Priority:  1,
				RunAt:     func() *time.Time { t := time.Now(); return &t }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJobOptions(tt.queueName, tt.priority, tt.runAt)
			
			assert.Equal(t, tt.expected.QueueName, result.QueueName)
			assert.Equal(t, tt.expected.Priority, result.Priority)
			
			if tt.runAt != nil {
				assert.NotNil(t, result.RunAt)
			} else {
				assert.Nil(t, result.RunAt)
			}
		})
	}
}