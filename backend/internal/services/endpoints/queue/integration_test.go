package queue

import (
	"context"
	"testing"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 这个集成测试展示了如何在真实环境中使用队列服务
func TestQueueServiceIntegration(t *testing.T) {
	// 跳过集成测试，因为需要数据库连接
	t.Skip("Integration test - requires database setup")

	// 这里展示如何在真实环境中初始化队列服务
	// 实际使用时，您需要：
	// 1. 设置数据库连接池
	// 2. 创建 workerqueue.Client
	// 3. 创建队列服务

	/*
		// 1. 设置数据库连接池
		dbPool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost/test")
		require.NoError(t, err)
		defer dbPool.Close()

		// 2. 创建 workerqueue.Client
		workerClient, err := workerqueue.NewClient(workerqueue.ClientOptions{
			DatabasePool: dbPool,
			RunnerOptions: workerqueue.RunnerOptions{
				Concurrency:  5,
				PollInterval: 1000,
			},
			Logger: slog.Default(),
		})
		require.NoError(t, err)

		// 3. 创建队列服务
		queueService := NewRiverQueueService(workerClient)

		// 4. 测试端点索引队列
		endpointID := uuid.New()
		indexRequest := EnqueueIndexEndpointRequest{
			EndpointID: endpointID,
			Source:     EndpointIndexSourceAPI,
			Reason:     "Integration test",
			SourceData: map[string]interface{}{
				"test": true,
			},
		}

		result, err := queueService.EnqueueIndexEndpoint(context.Background(), indexRequest)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Job)
	*/
}

// 这个测试展示队列服务与实际的 workerqueue.Client 的集成
func TestQueueServiceWithRealClient(t *testing.T) {
	t.Skip("Requires actual workerqueue.Client setup with database")

	// 展示如何集成真实的 workerqueue.Client
	// 在生产环境中，您会这样使用：

	/*
		// 创建真实的 worker queue client
		client, err := workerqueue.NewClient(workerqueue.ClientOptions{
			DatabasePool: dbPool, // 您的数据库连接池
			RunnerOptions: workerqueue.RunnerOptions{
				Concurrency:  10,
				PollInterval: 500,
			},
			Logger: logger, // 您的日志记录器
		})
		require.NoError(t, err)

		// 创建队列服务
		queueService := NewRiverQueueService(client)

		// 测试所有队列操作
		ctx := context.Background()
		endpointID := uuid.New()

		// 1. 索引端点
		indexResult, err := queueService.EnqueueIndexEndpoint(ctx, EnqueueIndexEndpointRequest{
			EndpointID: endpointID,
			Source:     EndpointIndexSourceInternal,
			Reason:     "Automated indexing",
		})
		require.NoError(t, err)
		assert.NotNil(t, indexResult)

		// 2. 注册作业
		jobResult, err := queueService.EnqueueRegisterJob(ctx, RegisterJobRequest{
			EndpointID:  endpointID,
			JobID:       "test-job",
			JobMetadata: map[string]interface{}{"name": "Test Job"},
		})
		require.NoError(t, err)
		assert.NotNil(t, jobResult)

		// 3. 注册源
		sourceResult, err := queueService.EnqueueRegisterSource(ctx, RegisterSourceRequest{
			EndpointID:     endpointID,
			SourceID:       "test-source",
			SourceMetadata: map[string]interface{}{"type": "webhook"},
		})
		require.NoError(t, err)
		assert.NotNil(t, sourceResult)

		// 4. 注册动态触发器
		triggerResult, err := queueService.EnqueueRegisterDynamicTrigger(ctx, RegisterDynamicTriggerRequest{
			EndpointID:      endpointID,
			TriggerID:       "test-trigger",
			TriggerMetadata: map[string]interface{}{"event": "user.created"},
		})
		require.NoError(t, err)
		assert.NotNil(t, triggerResult)

		// 5. 注册动态调度
		scheduleResult, err := queueService.EnqueueRegisterDynamicSchedule(ctx, RegisterDynamicScheduleRequest{
			EndpointID:       endpointID,
			ScheduleID:       "test-schedule",
			ScheduleMetadata: map[string]interface{}{"cron": "0 0 * * *"},
		})
		require.NoError(t, err)
		assert.NotNil(t, scheduleResult)
	*/
}

// 验证队列配置和选项的测试
func TestQueueOptionsConfiguration(t *testing.T) {
	// 测试各种队列配置选项是否正确设置

	tests := []struct {
		name     string
		request  EnqueueIndexEndpointRequest
		expected struct {
			queueName string
			priority  int
		}
	}{
		{
			name: "default_options",
			request: EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceInternal,
			},
			expected: struct {
				queueName string
				priority  int
			}{
				queueName: string(workerqueue.QueueDefault),
				priority:  int(workerqueue.PriorityNormal),
			},
		},
		{
			name: "custom_options",
			request: EnqueueIndexEndpointRequest{
				EndpointID: uuid.New(),
				Source:     EndpointIndexSourceAPI,
				QueueName:  "high_priority",
				Priority:   int(workerqueue.PriorityHigh),
			},
			expected: struct {
				queueName string
				priority  int
			}{
				queueName: "high_priority",
				priority:  int(workerqueue.PriorityHigh),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockWorkerQueueManager{}

			// 设置 mock 期望，验证正确的选项被传递
			mockManager.On("EnqueueJob",
				mock.Anything,
				"index_endpoint",
				mock.AnythingOfType("workerqueue.IndexEndpointArgs"),
				mock.MatchedBy(func(opts *workerqueue.JobOptions) bool {
					// 验证队列名称
					if opts.QueueName != tt.expected.queueName {
						return false
					}
					// 验证优先级
					if opts.Priority != tt.expected.priority {
						return false
					}
					return true
				}),
			).Return(&rivertype.JobInsertResult{
				Job: &rivertype.JobRow{ID: 1, State: "available"},
			}, nil)

			service := &riverQueueService{manager: mockManager}

			_, err := service.EnqueueIndexEndpoint(context.Background(), &tt.request)
			assert.NoError(t, err)

			mockManager.AssertExpectations(t)
		})
	}
}
