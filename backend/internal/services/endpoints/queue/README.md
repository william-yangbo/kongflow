# Endpoints Queue Service

这个包提供了专门针对 Endpoints 服务的队列系统集成，基于现有的 River 队列实现异步任务处理。

## 🎯 功能概述

### 核心功能

- **端点索引队列**: 异步处理端点索引任务
- **作业注册队列**: 处理作业定义的注册和更新
- **源注册队列**: 处理数据源的注册和配置
- **动态触发器队列**: 处理动态触发器的注册
- **动态调度队列**: 处理动态调度任务的注册

### 对齐 Trigger.dev

这个实现严格对齐 Trigger.dev 的队列处理模式：

- 任务优先级管理
- 队列分离 (default, execution, events, maintenance)
- 重试机制和错误处理
- 任务去重和唯一性约束

## 🚀 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "log"

    "kongflow/backend/internal/services/endpoints/queue"
    "kongflow/backend/internal/services/workerqueue"

    "github.com/google/uuid"
)

func main() {
    // 1. 创建 workerqueue.Client (需要数据库连接)
    workerClient, err := workerqueue.NewClient(workerqueue.ClientOptions{
        DatabasePool: dbPool, // 您的 pgxpool.Pool
        RunnerOptions: workerqueue.RunnerOptions{
            Concurrency:  10,
            PollInterval: 1000,
        },
        Logger: logger, // 您的日志记录器
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. 创建队列服务
    queueService := queue.NewRiverQueueService(workerClient)

    // 3. 使用队列服务
    endpointID := uuid.New()

    // 索引端点
    result, err := queueService.EnqueueIndexEndpoint(context.Background(), queue.EnqueueIndexEndpointRequest{
        EndpointID: endpointID,
        Source:     queue.EndpointIndexSourceAPI,
        Reason:     "API triggered indexing",
        SourceData: map[string]interface{}{
            "trigger": "manual",
        },
    })
    if err != nil {
        log.Printf("Failed to enqueue index endpoint: %v", err)
    } else {
        log.Printf("Enqueued job ID: %d", result.Job.ID)
    }
}
```

### 2. 在 Endpoints 服务中集成

```go
// 在 endpoints 服务中注入队列服务
type service struct {
    repo         Repository
    apiClient    *endpointapi.Client
    queueService queue.QueueService  // 新增队列服务
}

func NewService(repo Repository, apiClient *endpointapi.Client, queueService queue.QueueService) Service {
    return &service{
        repo:         repo,
        apiClient:    apiClient,
        queueService: queueService,
    }
}

// 在创建端点后触发异步索引
func (s *service) CreateEndpoint(ctx context.Context, req EndpointRequest) (*EndpointResponse, error) {
    // ... 现有的创建逻辑

    // 异步触发索引
    _, err := s.queueService.EnqueueIndexEndpoint(ctx, queue.EnqueueIndexEndpointRequest{
        EndpointID: endpoint.ID,
        Source:     queue.EndpointIndexSourceInternal,
        Reason:     "Auto-triggered after endpoint creation",
    })
    if err != nil {
        // 记录警告但不失败创建
        s.logger.Warn("Failed to enqueue index endpoint", "endpoint_id", endpoint.ID, "error", err)
    }

    return endpoint, nil
}
```

## 📋 API 接口

### QueueService 接口

```go
type QueueService interface {
    // 端点相关队列操作
    EnqueueIndexEndpoint(ctx context.Context, req EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error)

    // 注册相关队列操作
    EnqueueRegisterJob(ctx context.Context, req RegisterJobRequest) (*rivertype.JobInsertResult, error)
    EnqueueRegisterSource(ctx context.Context, req RegisterSourceRequest) (*rivertype.JobInsertResult, error)
    EnqueueRegisterDynamicTrigger(ctx context.Context, req RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error)
    EnqueueRegisterDynamicSchedule(ctx context.Context, req RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error)
}
```

### 队列请求类型

#### EnqueueIndexEndpointRequest

```go
type EnqueueIndexEndpointRequest struct {
    EndpointID uuid.UUID                  `json:"endpoint_id" validate:"required"`
    Source     EndpointIndexSource        `json:"source" validate:"required"`
    Reason     string                     `json:"reason,omitempty"`
    SourceData map[string]interface{}     `json:"source_data,omitempty"`
    QueueName  string                     `json:"queue_name,omitempty"`
    RunAt      *time.Time                 `json:"run_at,omitempty"`
    Priority   int                        `json:"priority,omitempty"`
}
```

#### 索引来源枚举

```go
const (
    EndpointIndexSourceManual   EndpointIndexSource = "MANUAL"   // 手动触发
    EndpointIndexSourceAPI      EndpointIndexSource = "API"      // API 调用触发
    EndpointIndexSourceInternal EndpointIndexSource = "INTERNAL" // 内部系统触发
    EndpointIndexSourceHook     EndpointIndexSource = "HOOK"     // Webhook 触发
)
```

## 🔧 队列配置

### 优先级级别

基于 River 队列的优先级系统 (1=最高, 4=最低):

```go
const (
    PriorityHigh    JobPriority = 1 // 高优先级 (事件、执行)
    PriorityNormal  JobPriority = 2 // 正常优先级 (索引、基础操作)
    PriorityLow     JobPriority = 3 // 低优先级 (维护、清理)
    PriorityVeryLow JobPriority = 4 // 极低优先级 (后台任务)
)
```

### 队列类型

```go
const (
    QueueDefault     JobQueue = "default"     // 默认队列
    QueueExecution   JobQueue = "execution"   // 执行队列
    QueueEvents      JobQueue = "events"      // 事件队列
    QueueMaintenance JobQueue = "maintenance" // 维护队列
)
```

### 任务唯一性

端点索引任务具有唯一性约束：

- 相同端点 + 相同来源在 15 分钟内不会重复索引
- 其他注册任务在 5 分钟内不会重复

## 🧪 测试

### 运行测试

```bash
# 运行单元测试
go test ./internal/services/endpoints/queue/ -v

# 运行特定测试
go test ./internal/services/endpoints/queue/ -run TestRiverQueueService_EnqueueIndexEndpoint -v
```

### Mock 测试

测试文件包含了完整的 Mock 实现，支持：

- 队列服务接口测试
- 各种配置选项验证
- 错误场景测试

## 🔗 依赖关系

- **workerqueue**: 基础队列客户端
- **River**: 底层队列系统
- **PostgreSQL**: 队列存储后端

## 📊 监控和调试

队列任务的监控通过现有的 workerqueue 系统提供：

- 任务状态跟踪
- 重试机制
- 错误日志记录
- 性能指标

## 🔄 与 Trigger.dev 对齐

此实现确保与 Trigger.dev 的队列处理模式完全对齐：

1. **任务结构**: 使用相同的任务参数格式
2. **队列策略**: 相同的优先级和队列分离策略
3. **错误处理**: 相同的重试和错误处理逻辑
4. **唯一性**: 相同的任务去重机制
5. **调度**: 支持延迟执行和调度

这确保了在迁移到 Trigger.dev 或与其集成时的兼容性。
