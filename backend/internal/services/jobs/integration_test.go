package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"kongflow/backend/internal/services/apiauth"
	"kongflow/backend/internal/services/events"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestJobsEventsIntegration 测试 Jobs 服务与 Events 服务的集成
func TestJobsEventsIntegration(t *testing.T) {
	// 创建模拟服务
	mockRepo := &MockRepository{}
	mockEvents := &MockEventsService{}
	service := NewService(mockRepo, mockEvents, slog.Default())

	// 准备测试数据
	versionID := uuid.New()
	environmentID := uuid.New()

	// 创建测试作业版本
	testVersion := JobVersions{
		ID:            uuidToPgUUID(versionID),
		EnvironmentID: uuidToPgUUID(environmentID),
		Version:       "1.0.0",
		EventSpecification: mustMarshalJSON(map[string]interface{}{
			"name":   "user.signup",
			"source": "auth-service",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]interface{}{"type": "string"},
					"email":  map[string]interface{}{"type": "string"},
				},
			},
		}),
	}

	// 模拟事件记录响应
	expectedEventRecord := &events.EventRecordResponse{
		ID:      "event-record-123",
		EventID: uuid.New().String(),
	}

	// 设置 Repository 期望
	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(testVersion, nil)

	// 设置 Events service 期望 - 验证传递的参数
	mockEvents.On("IngestSendEvent",
		mock.Anything, // context
		mock.MatchedBy(func(env *apiauth.AuthenticatedEnvironment) bool {
			// 验证环境信息正确传递
			return env != nil && env.Environment.ID.Bytes == environmentID
		}),
		mock.MatchedBy(func(req *events.SendEventRequest) bool {
			// 验证事件请求正确构造
			return req != nil &&
				req.Name == "user.signup" &&
				req.Source == "auth-service" &&
				req.Payload != nil &&
				req.Context != nil &&
				req.Context["test"] == true
		}),
		mock.MatchedBy(func(opts *events.SendEventOptions) bool {
			// 验证事件选项
			return opts != nil && opts.DeliverAt == nil // 测试事件立即投递
		}),
	).Return(expectedEventRecord, nil)

	// 构建测试请求
	request := TestJobRequest{
		VersionID:     versionID,
		EnvironmentID: environmentID,
		Payload: map[string]interface{}{
			"userId": "user-123",
			"email":  "test@example.com",
		},
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

	// 验证所有 mock 期望都被满足
	mockRepo.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}

// TestJobsEventsIntegration_EventCreationError 测试事件创建失败的情况
func TestJobsEventsIntegration_EventCreationError(t *testing.T) {
	// 创建模拟服务
	mockRepo := &MockRepository{}
	mockEvents := &MockEventsService{}
	service := NewService(mockRepo, mockEvents, slog.Default())

	// 准备测试数据
	versionID := uuid.New()
	environmentID := uuid.New()

	testVersion := JobVersions{
		ID:            uuidToPgUUID(versionID),
		EnvironmentID: uuidToPgUUID(environmentID),
		Version:       "1.0.0",
		EventSpecification: mustMarshalJSON(map[string]interface{}{
			"name": "test.event",
		}),
	}

	// 设置 Repository 期望
	mockRepo.On("GetJobVersionByID", mock.Anything, uuidToPgUUID(versionID)).Return(testVersion, nil)

	// 设置 Events service 返回错误
	mockEvents.On("IngestSendEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	// 构建测试请求
	request := TestJobRequest{
		VersionID:     versionID,
		EnvironmentID: environmentID,
		Payload:       map[string]interface{}{"test": "data"},
	}

	// 执行测试
	result, err := service.TestJob(context.Background(), request)

	// 验证结果 - 应该返回错误
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create event record")

	// 验证所有 mock 期望都被满足
	mockRepo.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}

// 辅助函数：将 map 序列化为 JSON
func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
