package jobs

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"kongflow/backend/internal/services/apiauth"
	"kongflow/backend/internal/services/events"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository 简化的模拟仓储，专注核心测试（80/20 原则）
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetJobByID(ctx context.Context, id pgtype.UUID) (Jobs, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Jobs), args.Error(1)
}

func (m *MockRepository) UpsertJob(ctx context.Context, params UpsertJobParams) (Jobs, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(Jobs), args.Error(1)
}

func (m *MockRepository) UpsertJobQueue(ctx context.Context, params UpsertJobQueueParams) (JobQueues, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobQueues), args.Error(1)
}

func (m *MockRepository) UpsertJobVersion(ctx context.Context, params UpsertJobVersionParams) (JobVersions, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobVersions), args.Error(1)
}

func (m *MockRepository) GetLatestJobVersion(ctx context.Context, params GetLatestJobVersionParams) (JobVersions, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobVersions), args.Error(1)
}

func (m *MockRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	args := m.Called(ctx, fn)
	// 实际执行传入的函数
	if fn != nil {
		return fn(m)
	}
	return args.Error(0)
}

// 其他接口方法的简化实现
func (m *MockRepository) CreateJob(ctx context.Context, params CreateJobParams) (Jobs, error) {
	return Jobs{}, nil
}
func (m *MockRepository) GetJobBySlug(ctx context.Context, projectID pgtype.UUID, slug string) (Jobs, error) {
	args := m.Called(ctx, projectID, slug)
	return args.Get(0).(Jobs), args.Error(1)
}
func (m *MockRepository) ListJobsByProject(ctx context.Context, params ListJobsByProjectParams) ([]Jobs, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]Jobs), args.Error(1)
}
func (m *MockRepository) CountJobsByProject(ctx context.Context, projectID pgtype.UUID) (int64, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockRepository) UpdateJob(ctx context.Context, params UpdateJobParams) (Jobs, error) {
	return Jobs{}, nil
}
func (m *MockRepository) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockRepository) CreateJobVersion(ctx context.Context, params CreateJobVersionParams) (JobVersions, error) {
	return JobVersions{}, nil
}
func (m *MockRepository) GetJobVersionByID(ctx context.Context, id pgtype.UUID) (JobVersions, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(JobVersions), args.Error(1)
}
func (m *MockRepository) GetJobVersionByJobAndVersion(ctx context.Context, params GetJobVersionByJobAndVersionParams) (JobVersions, error) {
	return JobVersions{}, nil
}
func (m *MockRepository) ListJobVersionsByJob(ctx context.Context, params ListJobVersionsByJobParams) ([]JobVersions, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]JobVersions), args.Error(1)
}

// MockEventsService 是 Events 服务的模拟实现
type MockEventsService struct {
	mock.Mock
}

func (m *MockEventsService) IngestSendEvent(ctx context.Context, env *apiauth.AuthenticatedEnvironment,
	event *events.SendEventRequest, opts *events.SendEventOptions) (*events.EventRecordResponse, error) {
	args := m.Called(ctx, env, event, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*events.EventRecordResponse), args.Error(1)
}

func (m *MockEventsService) DeliverEvent(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

func (m *MockEventsService) InvokeDispatcher(ctx context.Context, dispatcherID string, eventRecordID string) error {
	args := m.Called(ctx, dispatcherID, eventRecordID)
	return args.Error(0)
}

func (m *MockEventsService) GetEventRecord(ctx context.Context, id string) (*events.EventRecordResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*events.EventRecordResponse), args.Error(1)
}

func (m *MockEventsService) ListEventRecords(ctx context.Context, params events.ListEventRecordsParams) (*events.ListEventRecordsResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*events.ListEventRecordsResponse), args.Error(1)
}

func (m *MockEventsService) GetEventDispatcher(ctx context.Context, id string) (*events.EventDispatcherResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*events.EventDispatcherResponse), args.Error(1)
}

func (m *MockEventsService) ListEventDispatchers(ctx context.Context, params events.ListEventDispatchersParams) (*events.ListEventDispatchersResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*events.ListEventDispatchersResponse), args.Error(1)
}

// 辅助函数：创建测试用的服务
func createTestService() Service {
	mockRepo := &MockRepository{}
	mockEvents := &MockEventsService{}
	return NewService(mockRepo, mockEvents, slog.Default())
}

func createTestServiceWithMocks() (Service, *MockRepository, *MockEventsService) {
	mockRepo := &MockRepository{}
	mockEvents := &MockEventsService{}
	service := NewService(mockRepo, mockEvents, slog.Default())
	return service, mockRepo, mockEvents
}

func (m *MockRepository) CountLaterJobVersions(ctx context.Context, params CountLaterJobVersionsParams) (int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockRepository) UpdateJobVersionProperties(ctx context.Context, params UpdateJobVersionPropertiesParams) (JobVersions, error) {
	return JobVersions{}, nil
}
func (m *MockRepository) DeleteJobVersion(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *MockRepository) CreateJobQueue(ctx context.Context, params CreateJobQueueParams) (JobQueues, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobQueues), args.Error(1)
}
func (m *MockRepository) GetJobQueueByID(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return JobQueues{}, nil
}
func (m *MockRepository) GetJobQueueByName(ctx context.Context, params GetJobQueueByNameParams) (JobQueues, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobQueues), args.Error(1)
}
func (m *MockRepository) ListJobQueuesByEnvironment(ctx context.Context, params ListJobQueuesByEnvironmentParams) ([]JobQueues, error) {
	return nil, nil
}
func (m *MockRepository) UpdateJobQueueCounts(ctx context.Context, params UpdateJobQueueCountsParams) (JobQueues, error) {
	return JobQueues{}, nil
}
func (m *MockRepository) IncrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return JobQueues{}, nil
}
func (m *MockRepository) DecrementJobCount(ctx context.Context, id pgtype.UUID) (JobQueues, error) {
	return JobQueues{}, nil
}
func (m *MockRepository) DeleteJobQueue(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *MockRepository) CreateJobAlias(ctx context.Context, params CreateJobAliasParams) (JobAliases, error) {
	return JobAliases{}, nil
}
func (m *MockRepository) GetJobAliasByID(ctx context.Context, id pgtype.UUID) (JobAliases, error) {
	return JobAliases{}, nil
}
func (m *MockRepository) GetJobAliasByName(ctx context.Context, params GetJobAliasByNameParams) (JobAliases, error) {
	return JobAliases{}, nil
}
func (m *MockRepository) UpsertJobAlias(ctx context.Context, params UpsertJobAliasParams) (JobAliases, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(JobAliases), args.Error(1)
}
func (m *MockRepository) ListJobAliasesByJob(ctx context.Context, params ListJobAliasesByJobParams) ([]JobAliases, error) {
	return nil, nil
}
func (m *MockRepository) DeleteJobAlias(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *MockRepository) DeleteJobAliasesByJob(ctx context.Context, jobID pgtype.UUID) error {
	return nil
}
func (m *MockRepository) CreateEventExample(ctx context.Context, params CreateEventExampleParams) (EventExamples, error) {
	return EventExamples{}, nil
}
func (m *MockRepository) GetEventExampleByID(ctx context.Context, id pgtype.UUID) (EventExamples, error) {
	return EventExamples{}, nil
}
func (m *MockRepository) GetEventExampleBySlug(ctx context.Context, params GetEventExampleBySlugParams) (EventExamples, error) {
	return EventExamples{}, nil
}
func (m *MockRepository) UpsertEventExample(ctx context.Context, params UpsertEventExampleParams) (EventExamples, error) {
	return EventExamples{}, nil
}
func (m *MockRepository) ListEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) ([]EventExamples, error) {
	return nil, nil
}
func (m *MockRepository) DeleteEventExample(ctx context.Context, id pgtype.UUID) error { return nil }
func (m *MockRepository) DeleteEventExamplesByJobVersion(ctx context.Context, jobVersionID pgtype.UUID) error {
	return nil
}
func (m *MockRepository) DeleteEventExamplesNotInList(ctx context.Context, params DeleteEventExamplesNotInListParams) error {
	return nil
}

// 测试辅助函数
func createTestJob() Jobs {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return Jobs{
		ID:             uuidToPgUUID(uuid.New()),
		Slug:           "test-job",
		Title:          "Test Job",
		Internal:       false,
		OrganizationID: uuidToPgUUID(uuid.New()),
		ProjectID:      uuidToPgUUID(uuid.New()),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func createTestJobVersion() JobVersions {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return JobVersions{
		ID:                 uuidToPgUUID(uuid.New()),
		JobID:              uuidToPgUUID(uuid.New()),
		Version:            "1.0.0",
		EventSpecification: []byte(`{"name":"test.event","source":"api","type":"object"}`),
		Properties:         []byte(`{"description":"Test job"}`),
		EndpointID:         uuidToPgUUID(uuid.New()),
		EnvironmentID:      uuidToPgUUID(uuid.New()),
		OrganizationID:     uuidToPgUUID(uuid.New()),
		ProjectID:          uuidToPgUUID(uuid.New()),
		QueueID:            uuidToPgUUID(uuid.New()),
		StartPosition:      JobStartPositionLATEST,
		PreprocessRuns:     false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func createTestJobQueue() JobQueues {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return JobQueues{
		ID:            uuidToPgUUID(uuid.New()),
		Name:          "default",
		EnvironmentID: uuidToPgUUID(uuid.New()),
		JobCount:      0,
		MaxJobs:       100,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// 核心测试用例 - 80/20 原则：专注最重要的功能

func TestService_GetJob_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	jobID := uuid.New()
	expectedJob := createTestJob()
	expectedJob.ID = uuidToPgUUID(jobID)

	mockRepo.On("GetJobByID", mock.Anything, uuidToPgUUID(jobID)).Return(expectedJob, nil)

	result, err := service.GetJob(context.Background(), jobID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, jobID, result.ID)
	assert.Equal(t, expectedJob.Slug, result.Slug)

	mockRepo.AssertExpectations(t)
}

func TestService_GetJob_NotFound(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	jobID := uuid.New()
	mockRepo.On("GetJobByID", mock.Anything, uuidToPgUUID(jobID)).Return(Jobs{}, assert.AnError)

	result, err := service.GetJob(context.Background(), jobID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get job")

	mockRepo.AssertExpectations(t)
}

func TestService_TestJob_Success(t *testing.T) {
	service, mockRepo, mockEvents := createTestServiceWithMocks()

	// 准备测试数据
	versionID := uuid.New()
	environmentID := uuid.New()

	testVersion := createTestJobVersion()
	testVersion.ID = uuidToPgUUID(versionID)
	testVersion.EnvironmentID = uuidToPgUUID(environmentID)

	// 确保 EventSpecification 包含正确的 JSON
	testVersion.EventSpecification = []byte(`{"name":"test.event","source":"api","type":"object"}`)

	// 设置 Repository mock 期望
	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(testVersion, nil)

	// 设置 Events service mock 期望
	mockEventRecord := &events.EventRecordResponse{
		ID:      "test-record-id",
		EventID: uuid.New().String(),
	}
	mockEvents.On("IngestSendEvent", mock.Anything, mock.AnythingOfType("*apiauth.AuthenticatedEnvironment"),
		mock.AnythingOfType("*events.SendEventRequest"), mock.AnythingOfType("*events.SendEventOptions")).Return(mockEventRecord, nil)

	// 构建请求
	request := TestJobRequest{
		VersionID:     versionID,
		EnvironmentID: environmentID,
		Payload:       map[string]interface{}{"test": "data"},
	}

	// 执行测试
	result, err := service.TestJob(context.Background(), request)

	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.RunID)
	assert.NotEmpty(t, result.EventID)
	assert.Equal(t, "pending", result.Status)
	assert.Equal(t, "Test job submitted successfully", result.Message)

	mockRepo.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}

func TestHelpers_UUID_Conversion(t *testing.T) {
	originalUUID := uuid.New()

	pgUUID := uuidToPgUUID(originalUUID)
	convertedUUID := pgUUIDToUUID(pgUUID)

	assert.Equal(t, originalUUID, convertedUUID)
	assert.True(t, pgUUID.Valid)
}

func TestHelpers_JSONB_Conversion(t *testing.T) {
	originalMap := map[string]interface{}{
		"name": "test",
		"type": "object",
	}

	jsonbData, err := mapToJsonb(originalMap)
	require.NoError(t, err)
	convertedMap := jsonbToMap(jsonbData)

	assert.Equal(t, originalMap["name"], convertedMap["name"])
	assert.Equal(t, originalMap["type"], convertedMap["type"])
}

// ========== 核心缺失测试用例 (80/20 原则) ==========

func TestService_RegisterJob_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	endpointID := uuid.New()
	request := RegisterJobRequest{
		ID:      "data-processor",
		Name:    "Data Processing Job",
		Version: "1.0.0",
		Event: EventSpecification{
			Name:   "data.processed",
			Source: "api",
		},
		Trigger: TriggerMetadata{
			Type: "static",
			Rule: &TriggerRule{
				Source: "api",
				Event:  "data.processed",
			},
		},
	}

	// 创建预期的返回值
	expectedJob := createTestJob()
	expectedQueue := createTestJobQueue()
	expectedVersion := createTestJobVersion()

	// 设置 mock 期望
	mockRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpsertJob", mock.Anything, mock.Anything).Return(expectedJob, nil)
	mockRepo.On("UpsertJobQueue", mock.Anything, mock.Anything).Return(expectedQueue, nil)
	mockRepo.On("UpsertJobVersion", mock.Anything, mock.Anything).Return(expectedVersion, nil)
	// manageJobAlias 相关的 mock
	mockRepo.On("CountLaterJobVersions", mock.Anything, mock.Anything).Return(int64(0), nil)
	mockRepo.On("UpsertJobAlias", mock.Anything, mock.Anything).Return(JobAliases{}, nil)

	result, err := service.RegisterJob(context.Background(), endpointID, request)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, pgUUIDToUUID(expectedJob.ID), result.ID)
	assert.Equal(t, expectedJob.Slug, result.Slug)
	assert.NotNil(t, result.CurrentVersion)

	mockRepo.AssertExpectations(t)
}

func TestService_RegisterJob_ValidationError(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	endpointID := uuid.New()
	invalidRequest := RegisterJobRequest{
		// 缺少必需字段
		ID:   "",
		Name: "",
	}

	result, err := service.RegisterJob(context.Background(), endpointID, invalidRequest)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid request")
}

func TestService_GetJobBySlug_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	projectID := uuid.New()
	slug := "test-job"
	expectedJob := createTestJob()
	expectedJob.Slug = slug

	mockRepo.On("GetJobBySlug", mock.Anything, mock.Anything, slug).Return(expectedJob, nil)

	result, err := service.GetJobBySlug(context.Background(), projectID, slug)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, slug, result.Slug)

	mockRepo.AssertExpectations(t)
}

func TestService_ListJobs_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	projectID := uuid.New()
	params := ListJobsParams{
		ProjectID: projectID,
		Limit:     10,
		Offset:    0,
	}

	expectedJobs := []Jobs{createTestJob(), createTestJob()}
	expectedCount := int64(2)

	mockRepo.On("ListJobsByProject", mock.Anything, mock.Anything).Return(expectedJobs, nil)
	mockRepo.On("CountJobsByProject", mock.Anything, mock.Anything).Return(expectedCount, nil)

	result, err := service.ListJobs(context.Background(), params)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Jobs, 2)
	assert.Equal(t, expectedCount, result.Total)
	assert.False(t, result.HasMore)

	mockRepo.AssertExpectations(t)
}

func TestService_DeleteJob_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	jobID := uuid.New()
	mockRepo.On("DeleteJob", mock.Anything, mock.Anything).Return(nil)

	err := service.DeleteJob(context.Background(), jobID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_GetJobVersion_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	versionID := uuid.New()
	expectedVersion := createTestJobVersion()
	expectedVersion.ID = uuidToPgUUID(versionID)

	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(expectedVersion, nil)

	result, err := service.GetJobVersion(context.Background(), versionID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, versionID, result.ID)
	assert.Equal(t, expectedVersion.Version, result.Version)

	mockRepo.AssertExpectations(t)
}

func TestService_ListJobVersions_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	jobID := uuid.New()
	expectedVersions := []JobVersions{createTestJobVersion(), createTestJobVersion()}

	mockRepo.On("ListJobVersionsByJob", mock.Anything, mock.Anything).Return(expectedVersions, nil)

	result, err := service.ListJobVersions(context.Background(), jobID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Versions, 2)
	assert.Equal(t, 2, result.Total)

	mockRepo.AssertExpectations(t)
}

func TestService_CreateJobQueue_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	request := CreateJobQueueRequest{
		Name:          "test-queue",
		EnvironmentID: uuid.New(),
		MaxJobs:       50,
	}

	expectedQueue := createTestJobQueue()
	expectedQueue.Name = request.Name
	expectedQueue.MaxJobs = request.MaxJobs

	mockRepo.On("CreateJobQueue", mock.Anything, mock.Anything).Return(expectedQueue, nil)

	result, err := service.CreateJobQueue(context.Background(), request)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, request.Name, result.Name)
	assert.Equal(t, request.MaxJobs, result.MaxJobs)

	mockRepo.AssertExpectations(t)
}

func TestService_CreateJobQueue_DefaultMaxJobs(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	request := CreateJobQueueRequest{
		Name:          "test-queue",
		EnvironmentID: uuid.New(),
		MaxJobs:       0, // 使用默认值
	}

	expectedQueue := createTestJobQueue()
	expectedQueue.MaxJobs = DefaultMaxConcurrentRuns

	mockRepo.On("CreateJobQueue", mock.Anything, mock.Anything).Return(expectedQueue, nil)

	result, err := service.CreateJobQueue(context.Background(), request)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(DefaultMaxConcurrentRuns), result.MaxJobs)

	mockRepo.AssertExpectations(t)
}

func TestService_GetJobQueue_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	environmentID := uuid.New()
	queueName := "test-queue"
	expectedQueue := createTestJobQueue()
	expectedQueue.Name = queueName

	mockRepo.On("GetJobQueueByName", mock.Anything, mock.Anything).Return(expectedQueue, nil)

	result, err := service.GetJobQueue(context.Background(), environmentID, queueName)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queueName, result.Name)

	mockRepo.AssertExpectations(t)
}

func TestService_TestJob_InvalidEventSpecification(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, &MockEventsService{}, slog.Default())

	versionID := uuid.New()
	environmentID := uuid.New()

	// 创建无效的 EventSpecification（缺少 name 字段）
	invalidVersion := createTestJobVersion()
	invalidVersion.EventSpecification = []byte(`{"source":"api","type":"object"}`) // 缺少 name

	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(invalidVersion, nil)

	request := TestJobRequest{
		VersionID:     versionID,
		EnvironmentID: environmentID,
		Payload:       map[string]interface{}{"test": "data"},
	}

	result, err := service.TestJob(context.Background(), request)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid event specification")

	mockRepo.AssertExpectations(t)
}

// ========== 错误处理测试 ==========

func TestService_ErrorHandling_RepositoryFailures(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*MockRepository, Service)
	}{
		{
			name: "GetJob_RepositoryError",
			testFunc: func(mockRepo *MockRepository, service Service) {
				jobID := uuid.New()
				mockRepo.On("GetJobByID", mock.Anything, uuidToPgUUID(jobID)).Return(Jobs{}, assert.AnError)

				result, err := service.GetJob(context.Background(), jobID)
				assert.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "DeleteJob_RepositoryError",
			testFunc: func(mockRepo *MockRepository, service Service) {
				jobID := uuid.New()
				mockRepo.On("DeleteJob", mock.Anything, mock.Anything).Return(assert.AnError)

				err := service.DeleteJob(context.Background(), jobID)
				assert.Error(t, err)
			},
		},
		{
			name: "GetJobVersion_RepositoryError",
			testFunc: func(mockRepo *MockRepository, service Service) {
				versionID := uuid.New()
				mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(JobVersions{}, assert.AnError)

				result, err := service.GetJobVersion(context.Background(), versionID)
				assert.Error(t, err)
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, &MockEventsService{}, slog.Default())
			tt.testFunc(mockRepo, service)
			mockRepo.AssertExpectations(t)
		})
	}
}
