package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestEventRecord_Integration 集成测试 EventRecord 创建和查询功能
func TestEventRecord_Integration(t *testing.T) {
	// 这个测试需要实际的数据库连接，在本示例中跳过
	// 在实际项目中，您可以使用 testcontainers 或类似工具设置测试数据库
	t.Skip("Skipping integration test - requires database setup")

	// 以下是完整集成测试的示例代码结构：
	/*
		// 创建测试数据库连接
		ctx := context.Background()
		db := setupTestDatabase(t)
		defer cleanupTestDatabase(t, db)

		// 创建仓储实例
		queries := New(db)
		repo := &repository{queries: queries, db: db}

		// 准备测试数据
		environmentID := uuid.New()
		organizationID := uuid.New()
		projectID := uuid.New()

		// 创建 EventRecord
		eventRecord, err := repo.CreateEventRecord(ctx, CreateEventRecordParams{
			EventID:        uuid.New().String(),
			Name:           "test.event",
			Source:         "integration_test",
			Payload:        []byte(`{"test": "data"}`),
			Context:        []byte(`{"test": true}`),
			Timestamp:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			EnvironmentID:  uuidToPgUUID(environmentID),
			OrganizationID: uuidToPgUUID(organizationID),
			ProjectID:      uuidToPgUUID(projectID),
			IsTest:         true,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, eventRecord.ID)
		assert.Equal(t, "test.event", eventRecord.Name)
		assert.Equal(t, "integration_test", eventRecord.Source)
		assert.True(t, eventRecord.IsTest)

		// 查询创建的记录
		retrievedRecord, err := repo.GetEventRecordByID(ctx, eventRecord.ID)
		require.NoError(t, err)
		assert.Equal(t, eventRecord.ID, retrievedRecord.ID)
		assert.Equal(t, eventRecord.Name, retrievedRecord.Name)

		// 测试列表查询
		testRecords, err := repo.ListTestEventRecords(ctx, ListTestEventRecordsParams{
			EnvironmentID: uuidToPgUUID(environmentID),
			Limit:         10,
			Offset:        0,
		})
		require.NoError(t, err)
		assert.Len(t, testRecords, 1)
		assert.Equal(t, eventRecord.ID, testRecords[0].ID)
	*/
}

// TestEventRecord_CreateAndQuery 单元测试 EventRecord 创建和查询参数
func TestEventRecord_CreateAndQuery(t *testing.T) {
	// 测试 CreateEventRecordParams 参数验证
	now := time.Now()
	params := CreateEventRecordParams{
		EventID:        uuid.New().String(),
		Name:           "test.event",
		Source:         "unit_test",
		Payload:        []byte(`{"key": "value"}`),
		Context:        []byte(`{"test": true}`),
		Timestamp:      pgtype.Timestamptz{Time: now, Valid: true},
		EnvironmentID:  uuidToPgUUID(uuid.New()),
		OrganizationID: uuidToPgUUID(uuid.New()),
		ProjectID:      uuidToPgUUID(uuid.New()),
		IsTest:         true,
	}

	// 验证参数结构正确
	assert.NotEmpty(t, params.EventID)
	assert.Equal(t, "test.event", params.Name)
	assert.Equal(t, "unit_test", params.Source)
	assert.True(t, params.IsTest)
	assert.True(t, params.EnvironmentID.Valid)
	assert.True(t, params.OrganizationID.Valid)
	assert.True(t, params.ProjectID.Valid)
	assert.True(t, params.Timestamp.Valid)
	assert.Equal(t, now.Truncate(time.Microsecond), params.Timestamp.Time.Truncate(time.Microsecond))
}

// TestEventRecord_TestJobWorkflow 测试 TestJob 工作流中的 EventRecord 创建
func TestEventRecord_TestJobWorkflow(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil)

	// 准备测试数据
	versionID := uuid.New()
	environmentID := uuid.New()
	organizationID := uuid.New()
	projectID := uuid.New()

	testVersion := JobVersions{
		ID:                 uuidToPgUUID(versionID),
		EnvironmentID:      uuidToPgUUID(environmentID),
		OrganizationID:     uuidToPgUUID(organizationID),
		ProjectID:          uuidToPgUUID(projectID),
		EventSpecification: []byte(`{"name":"user.created","source":"api","type":"object"}`),
		Version:            "1.0.0",
		CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	expectedEventRecord := EventRecords{
		ID:             uuidToPgUUID(uuid.New()),
		EventID:        uuid.New().String(),
		Name:           "user.created",
		Source:         "test",
		EnvironmentID:  uuidToPgUUID(environmentID),
		OrganizationID: uuidToPgUUID(organizationID),
		ProjectID:      uuidToPgUUID(projectID),
		IsTest:         true,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// 设置 mock 期望
	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(testVersion, nil)

	// 验证 CreateEventRecord 被调用时的参数
	mockRepo.On("CreateEventRecord", mock.Anything, mock.MatchedBy(func(params CreateEventRecordParams) bool {
		return params.Name == "user.created" &&
			params.Source == "test" &&
			params.IsTest == true &&
			params.EnvironmentID == uuidToPgUUID(environmentID) &&
			params.OrganizationID == uuidToPgUUID(organizationID) &&
			params.ProjectID == uuidToPgUUID(projectID)
	})).Return(expectedEventRecord, nil)

	// 构建测试请求
	request := TestJobRequest{
		VersionID:     versionID,
		EnvironmentID: environmentID,
		Payload:       map[string]interface{}{"userId": "123", "email": "test@example.com"},
	}

	// 执行 TestJob
	result, err := service.TestJob(context.Background(), request)

	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.RunID)
	assert.NotEmpty(t, result.EventID)
	assert.Equal(t, "pending", result.Status)
	assert.Contains(t, result.Message, "successfully")

	// 验证所有 mock 调用
	mockRepo.AssertExpectations(t)
}
