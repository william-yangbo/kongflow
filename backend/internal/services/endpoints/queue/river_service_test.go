package queue

import (
	"context"
	"testing"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkerQueueClient 实现 WorkerQueueClient 接口
type MockWorkerQueueClient struct {
	mock.Mock
}

func (m *MockWorkerQueueClient) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, identifier, payload, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func (m *MockWorkerQueueClient) EnqueueWithBusinessLogic(ctx context.Context, identifier string, payload interface{}, businessLogic workerqueue.BusinessLogicFunc) (*rivertype.JobInsertResult, error) {
	args := m.Called(ctx, identifier, payload, businessLogic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}

func TestRiverQueueService_EnqueueIndexEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		request     EnqueueIndexEndpointRequest
		setupMock   func(*MockWorkerQueueClient)
		expectError bool
	}{
		{
			name: "successful_enqueue_with_defaults",
			request: EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceInternal,
				Reason:     "Test indexing",
			},
			setupMock: func(m *MockWorkerQueueClient) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    1,
						State: "available",
					},
				}
				m.On("Enqueue", mock.Anything, "index_endpoint", mock.AnythingOfType("workerqueue.IndexEndpointArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "successful_enqueue_with_custom_options",
			request: EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceAPI,
				Reason:     "API triggered indexing",
				QueueName:  "custom_queue",
				Priority:   1,
				RunAt:      func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
				SourceData: map[string]interface{}{"trigger": "api"},
			},
			setupMock: func(m *MockWorkerQueueClient) {
				result := &rivertype.JobInsertResult{
					Job: &rivertype.JobRow{
						ID:    2,
						State: "scheduled",
					},
				}
				m.On("Enqueue", mock.Anything, "index_endpoint", mock.AnythingOfType("workerqueue.IndexEndpointArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWorkerQueueClient{}
			tt.setupMock(mockClient)

			service := &riverQueueService{
				client: mockClient,
			}

			result, err := service.EnqueueIndexEndpoint(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.Job)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRiverQueueService_EnqueueRegisterJob(t *testing.T) {
	mockClient := &MockWorkerQueueClient{}
	endpointID := uuid.New()

	request := RegisterJobRequest{
		EndpointID:  endpointID,
		JobID:       "test-job-1",
		JobMetadata: map[string]interface{}{"name": "test job", "version": "1.0"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    3,
			State: "available",
		},
	}

	mockClient.On("Enqueue", mock.Anything, "register_job", mock.AnythingOfType("workerqueue.RegisterJobArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		client: mockClient,
	}

	jobResult, err := service.EnqueueRegisterJob(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, jobResult)
	assert.Equal(t, int64(3), jobResult.Job.ID)
	mockClient.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueRegisterSource(t *testing.T) {
	mockClient := &MockWorkerQueueClient{}
	endpointID := uuid.New()

	request := RegisterSourceRequest{
		EndpointID:     endpointID,
		SourceID:       "test-source-1",
		SourceMetadata: map[string]interface{}{"type": "http", "url": "https://api.example.com"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    4,
			State: "available",
		},
	}

	mockClient.On("Enqueue", mock.Anything, "register_source", mock.AnythingOfType("workerqueue.RegisterSourceArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		client: mockClient,
	}

	jobResult, err := service.EnqueueRegisterSource(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, jobResult)
	assert.Equal(t, int64(4), jobResult.Job.ID)
	mockClient.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueRegisterDynamicTrigger(t *testing.T) {
	mockClient := &MockWorkerQueueClient{}
	endpointID := uuid.New()

	request := RegisterDynamicTriggerRequest{
		EndpointID:      endpointID,
		TriggerID:       "test-trigger-1",
		TriggerMetadata: map[string]interface{}{"event": "user.created", "filter": "premium=true"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    5,
			State: "available",
		},
	}

	mockClient.On("Enqueue", mock.Anything, "register_dynamic_trigger", mock.AnythingOfType("workerqueue.RegisterDynamicTriggerArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		client: mockClient,
	}

	jobResult, err := service.EnqueueRegisterDynamicTrigger(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, jobResult)
	assert.Equal(t, int64(5), jobResult.Job.ID)
	mockClient.AssertExpectations(t)
}

func TestRiverQueueService_EnqueueRegisterDynamicSchedule(t *testing.T) {
	mockClient := &MockWorkerQueueClient{}
	endpointID := uuid.New()

	request := RegisterDynamicScheduleRequest{
		EndpointID:       endpointID,
		ScheduleID:       "test-schedule-1",
		ScheduleMetadata: map[string]interface{}{"cron": "0 0 * * *", "timezone": "UTC"},
	}

	result := &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID:    6,
			State: "available",
		},
	}

	mockClient.On("Enqueue", mock.Anything, "register_dynamic_schedule", mock.AnythingOfType("workerqueue.RegisterDynamicScheduleArgs"), mock.AnythingOfType("*workerqueue.JobOptions")).Return(result, nil)

	service := &riverQueueService{
		client: mockClient,
	}

	jobResult, err := service.EnqueueRegisterDynamicSchedule(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, jobResult)
	assert.Equal(t, int64(6), jobResult.Job.ID)
	mockClient.AssertExpectations(t)
}

func TestNewRiverQueueService(t *testing.T) {
	mockClient := &MockWorkerQueueClient{}
	service := NewRiverQueueService(mockClient)

	assert.NotNil(t, service)
	assert.IsType(t, &riverQueueService{}, service)
}
