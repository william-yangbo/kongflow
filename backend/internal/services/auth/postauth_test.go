package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kongflow/backend/internal/services/logger"
	"kongflow/backend/internal/services/workerqueue"
	"kongflow/backend/internal/shared"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkerQueueManager is a mock implementation that matches Manager's EnqueueJob method
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

// MockJobInsertResult creates a mock job insert result
func MockJobInsertResult() *rivertype.JobInsertResult {
	return &rivertype.JobInsertResult{
		Job: &rivertype.JobRow{
			ID: int64(12345),
		},
	}
}

func TestPostAuthProcessor_Creation(t *testing.T) {
	mockAnalytics := &MockAnalyticsService{}
	logger := logger.New("test-auth")
	mockWorkerQueue := &MockWorkerQueueManager{}

	processor := NewPostAuthProcessor(mockAnalytics, logger, mockWorkerQueue)

	assert.NotNil(t, processor)
	assert.Equal(t, mockAnalytics, processor.analytics)
	assert.Equal(t, logger, processor.logger)
	assert.Equal(t, mockWorkerQueue, processor.workerQueue)
}

func TestPostAuthProcessor_PostAuthentication_NewUser(t *testing.T) {
	mockAnalytics := &MockAnalyticsService{}
	logger := logger.New("test-auth")
	mockWorkerQueue := &MockWorkerQueueManager{}

	processor := NewPostAuthProcessor(mockAnalytics, logger, mockWorkerQueue)

	// Create test user
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")

	user := &shared.Users{
		ID:    userID,
		Email: "test@example.com",
	}

	ctx := context.Background()

	// Set up expectations
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, true).Return(nil)
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, false).Return(nil)

	mockResult := MockJobInsertResult()
	mockWorkerQueue.On("EnqueueJob", ctx, "scheduleEmail", mock.Anything, mock.AnythingOfType("*workerqueue.JobOptions")).Return(mockResult, nil) // Execute
	err := processor.PostAuthentication(ctx, user, true)

	// Verify
	assert.NoError(t, err)
	mockAnalytics.AssertExpectations(t)
	mockWorkerQueue.AssertExpectations(t)

	// Verify the job was scheduled with correct parameters
	mockWorkerQueue.AssertCalled(t, "EnqueueJob", ctx, "scheduleEmail", mock.Anything, mock.MatchedBy(func(opts *workerqueue.JobOptions) bool {
		return opts != nil &&
			opts.QueueName == "email" &&
			opts.Priority == 2 &&
			opts.MaxAttempts == 3 &&
			opts.RunAt != nil &&
			opts.RunAt.After(time.Now().Add(time.Minute)) && // Should be around 2 minutes from now
			opts.RunAt.Before(time.Now().Add(3*time.Minute)) &&
			len(opts.Tags) == 3 &&
			opts.JobKey == fmt.Sprintf("welcome_email_%s", userID.Bytes)
	}))
}

func TestPostAuthProcessor_PostAuthentication_ExistingUser(t *testing.T) {
	mockAnalytics := &MockAnalyticsService{}
	logger := logger.New("test-auth")
	mockWorkerQueue := &MockWorkerQueueManager{}

	processor := NewPostAuthProcessor(mockAnalytics, logger, mockWorkerQueue)

	// Create test user
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")

	user := &shared.Users{
		ID:    userID,
		Email: "existing@example.com",
	}

	ctx := context.Background()

	// Set up expectations - existing user should not get welcome email
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, false).Return(nil).Twice()

	// No welcome email should be scheduled for existing users
	// mockWorkerQueue should NOT be called

	// Execute
	err := processor.PostAuthentication(ctx, user, false)

	// Verify
	assert.NoError(t, err)
	mockAnalytics.AssertExpectations(t)

	// Verify that no email job was scheduled
	mockWorkerQueue.AssertNotCalled(t, "EnqueueJob")
}

func TestPostAuthProcessor_PostAuthentication_WithAnalyticsError(t *testing.T) {
	mockAnalytics := &MockAnalyticsService{}
	logger := logger.New("test-auth")
	mockWorkerQueue := &MockWorkerQueueManager{}

	processor := NewPostAuthProcessor(mockAnalytics, logger, mockWorkerQueue)

	// Create test user
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")

	user := &shared.Users{
		ID:    userID,
		Email: "test@example.com",
	}

	ctx := context.Background()

	// Set up expectations with analytics error
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, true).Return(assert.AnError)
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, false).Return(assert.AnError)

	// Welcome email should still be scheduled despite analytics errors
	mockResult := MockJobInsertResult()
	mockWorkerQueue.On("EnqueueJob", ctx, "scheduleEmail", mock.Anything, mock.AnythingOfType("*workerqueue.JobOptions")).Return(mockResult, nil)

	// Execute
	err := processor.PostAuthentication(ctx, user, true)

	// Verify - should not fail even with analytics errors
	assert.NoError(t, err)
	mockAnalytics.AssertExpectations(t)
	mockWorkerQueue.AssertExpectations(t)
}

func TestPostAuthProcessor_PostAuthentication_WithWorkerQueueError(t *testing.T) {
	mockAnalytics := &MockAnalyticsService{}
	logger := logger.New("test-auth")
	mockWorkerQueue := &MockWorkerQueueManager{}

	processor := NewPostAuthProcessor(mockAnalytics, logger, mockWorkerQueue)

	// Create test user
	userID := pgtype.UUID{}
	userID.Scan("123e4567-e89b-12d3-a456-426614174000")

	user := &shared.Users{
		ID:    userID,
		Email: "test@example.com",
	}

	ctx := context.Background()

	// Set up expectations
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, true).Return(nil)
	mockAnalytics.On("UserIdentify", ctx, mock.Anything, false).Return(nil)

	// Worker queue fails
	mockWorkerQueue.On("EnqueueJob", ctx, "scheduleEmail", mock.Anything, mock.AnythingOfType("*workerqueue.JobOptions")).Return(nil, assert.AnError)

	// Execute
	err := processor.PostAuthentication(ctx, user, true)

	// Verify - should not fail even with worker queue errors
	assert.NoError(t, err)
	mockAnalytics.AssertExpectations(t)
	mockWorkerQueue.AssertExpectations(t)
}
