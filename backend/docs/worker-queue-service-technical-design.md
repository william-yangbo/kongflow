# Worker Queue Service 技术设计详解

> **补充文档**: 详细技术设计和 trigger.dev 对齐策略  
> **关联**: worker-queue-service-migration-plan.md  
> **更新时间**: 2025-09-18

## 🎯 trigger.dev 对齐策略

### 1. 任务类型完整对齐

基于 trigger.dev 的 `workerCatalog`，我们需要实现以下 25+ 任务类型：

```go
// internal/services/workerqueue/tasks/types.go
package tasks

import "time"

// 组织管理任务
type OrganizationCreatedArgs struct {
    ID string `json:"id"`
}
func (OrganizationCreatedArgs) Kind() string { return "organizationCreated" }

// 端点管理任务
type IndexEndpointArgs struct {
    ID         string      `json:"id"`
    Source     *string     `json:"source,omitempty"`     // "MANUAL", "API", "INTERNAL", "HOOK"
    SourceData interface{} `json:"sourceData,omitempty"`
    Reason     *string     `json:"reason,omitempty"`
}
func (IndexEndpointArgs) Kind() string { return "indexEndpoint" }

// 邮件任务
type ScheduleEmailArgs struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    HTML    string `json:"html"`
    Text    string `json:"text"`
}
func (ScheduleEmailArgs) Kind() string { return "scheduleEmail" }

// GitHub 集成任务
type GitHubAppInstallationDeletedArgs struct {
    ID string `json:"id"`
}
func (GitHubAppInstallationDeletedArgs) Kind() string { return "githubAppInstallationDeleted" }

type GitHubPushArgs struct {
    Branch    string `json:"branch"`
    CommitSHA string `json:"commitSha"`
    Repository string `json:"repository"`
}
func (GitHubPushArgs) Kind() string { return "githubPush" }

// VM 管理任务
type StopVMArgs struct {
    ID string `json:"id"`
}
func (StopVMArgs) Kind() string { return "stopVM" }

// 部署任务
type StartInitialProjectDeploymentArgs struct {
    ID string `json:"id"`
}
func (StartInitialProjectDeploymentArgs) Kind() string { return "startInitialProjectDeployment" }

// 运行任务
type StartRunArgs struct {
    ID string `json:"id"`
}
func (StartRunArgs) Kind() string { return "startRun" }

type PerformRunExecutionArgs struct {
    ID string `json:"id"`
}
func (PerformRunExecutionArgs) Kind() string { return "performRunExecution" }

type PerformTaskOperationArgs struct {
    ID string `json:"id"`
}
func (PerformTaskOperationArgs) Kind() string { return "performTaskOperation" }

type RunFinishedArgs struct {
    ID string `json:"id"`
}
func (RunFinishedArgs) Kind() string { return "runFinished" }

// HTTP 源请求任务
type DeliverHTTPSourceRequestArgs struct {
    ID string `json:"id"`
}
func (DeliverHTTPSourceRequestArgs) Kind() string { return "deliverHttpSourceRequest" }

// OAuth 任务
type RefreshOAuthTokenArgs struct {
    OrganizationID string `json:"organizationId"`
    ConnectionID   string `json:"connectionId"`
}
func (RefreshOAuthTokenArgs) Kind() string { return "refreshOAuthToken" }

// 作业注册任务
type RegisterJobArgs struct {
    EndpointID string      `json:"endpointId"`
    Job        interface{} `json:"job"` // JobMetadataSchema
}
func (RegisterJobArgs) Kind() string { return "registerJob" }

// 源注册任务
type RegisterSourceArgs struct {
    EndpointID string      `json:"endpointId"`
    Source     interface{} `json:"source"` // SourceMetadataSchema
}
func (RegisterSourceArgs) Kind() string { return "registerSource" }

// 动态触发器任务
type RegisterDynamicTriggerArgs struct {
    EndpointID     string      `json:"endpointId"`
    DynamicTrigger interface{} `json:"dynamicTrigger"` // DynamicTriggerEndpointMetadataSchema
}
func (RegisterDynamicTriggerArgs) Kind() string { return "registerDynamicTrigger" }

// 动态调度任务
type RegisterDynamicScheduleArgs struct {
    EndpointID      string      `json:"endpointId"`
    DynamicSchedule interface{} `json:"dynamicSchedule"` // RegisterDynamicSchedulePayloadSchema
}
func (RegisterDynamicScheduleArgs) Kind() string { return "registerDynamicSchedule" }

// 源激活任务
type ActivateSourceArgs struct {
    ID            string   `json:"id"`
    OrphanedEvents []string `json:"orphanedEvents,omitempty"`
}
func (ActivateSourceArgs) Kind() string { return "activateSource" }

// 队列运行任务
type StartQueuedRunsArgs struct {
    ID string `json:"id"`
}
func (StartQueuedRunsArgs) Kind() string { return "startQueuedRuns" }

// 事件处理任务
type DeliverEventArgs struct {
    ID string `json:"id"`
}
func (DeliverEventArgs) Kind() string { return "deliverEvent" }

type InvokeDispatcherArgs struct {
    ID            string `json:"id"`
    EventRecordID string `json:"eventRecordId"`
}
func (InvokeDispatcherArgs) Kind() string { return "events.invokeDispatcher" }

type DeliverScheduledEventArgs struct {
    ID      string      `json:"id"`
    Payload interface{} `json:"payload"` // ScheduledPayloadSchema
}
func (DeliverScheduledEventArgs) Kind() string { return "events.deliverScheduled" }

// 连接任务
type MissingConnectionCreatedArgs struct {
    ID string `json:"id"`
}
func (MissingConnectionCreatedArgs) Kind() string { return "missingConnectionCreated" }

type ConnectionCreatedArgs struct {
    ID string `json:"id"`
}
func (ConnectionCreatedArgs) Kind() string { return "connectionCreated" }
```

### 2. 队列配置完全对齐

```go
// internal/services/workerqueue/config.go
package workerqueue

import (
    "time"
    "github.com/riverqueue/river"
)

// QueueMappingConfig 完全对齐 trigger.dev 的队列配置
type QueueMappingConfig struct {
    // 精确映射 trigger.dev 的任务配置
    TaskConfigs map[string]TaskQueueConfig
}

type TaskQueueConfig struct {
    QueueName   string        `json:"queue_name"`
    Priority    int           `json:"priority"`     // 对齐 trigger.dev 的 priority
    MaxAttempts int           `json:"max_attempts"` // 对齐 trigger.dev 的 maxAttempts
    Timeout     time.Duration `json:"timeout"`
}

// GetTaskQueueConfig 返回与 trigger.dev 完全对齐的任务配置
func GetTaskQueueConfig() map[string]TaskQueueConfig {
    return map[string]TaskQueueConfig{
        // 事件处理 - 对齐 trigger.dev 配置
        "events.invokeDispatcher": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     30 * time.Second,
        },
        "events.deliverScheduled": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 5, // 对齐 maxAttempts: 5
            Timeout:     60 * time.Second,
        },

        // 连接处理 - 对齐 trigger.dev 配置
        "connectionCreated": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     30 * time.Second,
        },
        "missingConnectionCreated": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     30 * time.Second,
        },

        // 运行处理 - 对齐 trigger.dev 配置
        "runFinished": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "startQueuedRuns": {
            QueueName:   "queue", // 注意：trigger.dev 使用动态队列名 `queue:${payload.id}`
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     300 * time.Second,
        },

        // 作业和源注册 - 对齐 trigger.dev 配置
        "registerJob": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "registerSource": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "registerDynamicTrigger": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "registerDynamicSchedule": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },

        // 源处理 - 对齐 trigger.dev 配置
        "activateSource": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     120 * time.Second,
        },
        "deliverHttpSourceRequest": {
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 5, // 对齐 maxAttempts: 5
            Timeout:     30 * time.Second,
        },

        // 运行执行 - 对齐 trigger.dev 配置
        "startRun": {
            QueueName:   "executions", // 对齐 queueName: "executions"
            Priority:    river.PriorityDefault,
            MaxAttempts: 13, // 对齐 maxAttempts: 13
            Timeout:     600 * time.Second,
        },
        "performRunExecution": {
            QueueName:   "runs", // 注意：trigger.dev 使用动态队列名 `runs:${payload.id}`
            Priority:    river.PriorityDefault,
            MaxAttempts: 1, // 对齐 maxAttempts: 1
            Timeout:     1800 * time.Second, // 30分钟超时
        },
        "performTaskOperation": {
            QueueName:   "tasks", // 注意：trigger.dev 使用动态队列名 `tasks:${payload.id}`
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 对齐 maxAttempts: 3
            Timeout:     300 * time.Second,
        },

        // 内部队列任务 - 对齐 trigger.dev 配置
        "scheduleEmail": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    100,              // 对齐 priority: 100 (高优先级)
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     30 * time.Second,
        },
        "startInitialProjectDeployment": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    50,               // 对齐 priority: 50
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     600 * time.Second,
        },
        "stopVM": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    50,               // 对齐 priority: 50
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     120 * time.Second,
        },
        "organizationCreated": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    50,               // 对齐 priority: 50
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "githubPush": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    50,               // 对齐 priority: 50
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     120 * time.Second,
        },
        "githubAppInstallationDeleted": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    50,               // 对齐 priority: 50
            MaxAttempts: 3,                // 对齐 maxAttempts: 3
            Timeout:     60 * time.Second,
        },
        "indexEndpoint": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 默认值，trigger.dev 未指定
            Timeout:     300 * time.Second,
        },
        "refreshOAuthToken": {
            QueueName:   "internal-queue", // 对齐 queueName: "internal-queue"
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 默认值，trigger.dev 未指定
            Timeout:     60 * time.Second,
        },

        // 事件分发器 - 对齐 trigger.dev 配置
        "deliverEvent": {
            QueueName:   "event-dispatcher", // 对齐 queueName: "event-dispatcher"
            Priority:    river.PriorityDefault,
            MaxAttempts: 3, // 默认值，trigger.dev 未指定
            Timeout:     120 * time.Second,
        },
    }
}
```

### 3. 动态队列名解决方案

trigger.dev 中有几个任务使用动态队列名，我们需要特殊处理：

```go
// internal/services/workerqueue/dynamic_queue.go
package workerqueue

import (
    "fmt"
    "regexp"
)

// DynamicQueueHandler 处理动态队列名的特殊逻辑
type DynamicQueueHandler struct {
    defaultMapping map[string]string
}

func NewDynamicQueueHandler() *DynamicQueueHandler {
    return &DynamicQueueHandler{
        defaultMapping: map[string]string{
            "startQueuedRuns":      "queue-operations",
            "performRunExecution":  "run-executions",
            "performTaskOperation": "task-operations",
        },
    }
}

// ResolveDynamicQueue 解析动态队列名
// 对齐 trigger.dev 的动态队列逻辑：
// - startQueuedRuns: queueName: (payload) => `queue:${payload.id}`
// - performRunExecution: queueName: (payload) => `runs:${payload.id}`
// - performTaskOperation: queueName: (payload) => `tasks:${payload.id}`
func (h *DynamicQueueHandler) ResolveDynamicQueue(taskKind string, payload interface{}) string {
    switch taskKind {
    case "startQueuedRuns":
        if args, ok := payload.(StartQueuedRunsArgs); ok {
            return fmt.Sprintf("queue-%s", args.ID)
        }
        return "queue-operations"

    case "performRunExecution":
        if args, ok := payload.(PerformRunExecutionArgs); ok {
            return fmt.Sprintf("runs-%s", args.ID)
        }
        return "run-executions"

    case "performTaskOperation":
        if args, ok := payload.(PerformTaskOperationArgs); ok {
            return fmt.Sprintf("tasks-%s", args.ID)
        }
        return "task-operations"

    default:
        // 回退到静态队列映射
        if queue, exists := h.defaultMapping[taskKind]; exists {
            return queue
        }
        return river.QueueDefault
    }
}
```

### 4. 客户端接口完全对齐

```go
// internal/services/workerqueue/client.go
package workerqueue

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Client 完全对齐 trigger.dev 的 ZodWorker 接口
type Client struct {
    river         *river.Client[pgx.Tx]
    workers       *river.Workers
    dynamicQueue  *DynamicQueueHandler
    taskConfigs   map[string]TaskQueueConfig
}

// Config 对齐 trigger.dev 的 ZodWorkerOptions
type Config struct {
    DatabasePool *pgxpool.Pool

    // 对齐 trigger.dev 的 runnerOptions
    RunnerOptions RunnerOptions

    // 任务配置
    TaskConfigs map[string]TaskQueueConfig
}

// RunnerOptions 对齐 trigger.dev 的 RunnerOptions
type RunnerOptions struct {
    Concurrency  int           `json:"concurrency"`   // 对齐 concurrency: 5
    PollInterval time.Duration `json:"poll_interval"` // 对齐 pollInterval: 1000
}

// NewClient 对齐 trigger.dev 的 ZodWorker 构造函数
func NewClient(config Config) (*Client, error) {
    workers := river.NewWorkers()

    // 注册所有任务处理器
    if err := registerAllHandlers(workers); err != nil {
        return nil, fmt.Errorf("failed to register handlers: %w", err)
    }

    riverClient, err := river.NewClient(riverpgxv5.New(config.DatabasePool), &river.Config{
        Queues: map[string]river.QueueConfig{
            "executions":       {MaxWorkers: 10},
            "run-executions":   {MaxWorkers: 8},
            "task-operations":  {MaxWorkers: 5},
            "queue-operations": {MaxWorkers: 3},
            "internal-queue":   {MaxWorkers: 3},
            "event-dispatcher": {MaxWorkers: 8},
            river.QueueDefault: {MaxWorkers: 100},
        },
        Workers:      workers,
        FetchCooldown: 100 * time.Millisecond,
        FetchPollInterval: config.RunnerOptions.PollInterval,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create river client: %w", err)
    }

    return &Client{
        river:        riverClient,
        workers:      workers,
        dynamicQueue: NewDynamicQueueHandler(),
        taskConfigs:  config.TaskConfigs,
    }, nil
}

// Initialize 对齐 trigger.dev 的 initialize() 方法
func (c *Client) Initialize(ctx context.Context) error {
    return c.river.Start(ctx)
}

// Stop 对齐 trigger.dev 的 stop() 方法
func (c *Client) Stop(ctx context.Context) error {
    return c.river.Stop(ctx)
}

// InsertOpts 对齐 trigger.dev 的 ZodWorkerEnqueueOptions
type InsertOpts struct {
    // 对齐 TaskSpec 字段
    QueueName   string
    Priority    int
    MaxAttempts int
    ScheduledAt *time.Time
    JobKey      string
    UniqueOpts  *river.UniqueOpts

    // 对齐 ZodWorkerEnqueueOptions 的事务支持
    Tx pgx.Tx
}

// Insert 对齐 trigger.dev 的 enqueue() 方法
func (c *Client) Insert[T TaskArgs](ctx context.Context, args T, opts *InsertOpts) (*river.Job[T], error) {
    taskKind := args.Kind()

    // 获取任务配置
    config, exists := c.taskConfigs[taskKind]
    if !exists {
        config = TaskQueueConfig{
            QueueName:   river.QueueDefault,
            Priority:    river.PriorityDefault,
            MaxAttempts: 3,
            Timeout:     60 * time.Second,
        }
    }

    // 解析动态队列名（如果适用）
    queueName := config.QueueName
    if isDynamicQueue(taskKind) {
        queueName = c.dynamicQueue.ResolveDynamicQueue(taskKind, args)
    }

    // 构建 River 插入选项
    riverOpts := &river.InsertOpts{
        Queue:       queueName,
        Priority:    config.Priority,
        MaxAttempts: config.MaxAttempts,
    }

    // 应用用户提供的选项（覆盖默认值）
    if opts != nil {
        if opts.QueueName != "" {
            riverOpts.Queue = opts.QueueName
        }
        if opts.Priority > 0 {
            riverOpts.Priority = opts.Priority
        }
        if opts.MaxAttempts > 0 {
            riverOpts.MaxAttempts = opts.MaxAttempts
        }
        if opts.ScheduledAt != nil {
            riverOpts.ScheduledAt = *opts.ScheduledAt
        }
        if opts.JobKey != "" {
            riverOpts.UniqueOpts = &river.UniqueOpts{
                ByArgs: true,
                ByQueue: true,
            }
        }
        if opts.UniqueOpts != nil {
            riverOpts.UniqueOpts = opts.UniqueOpts
        }
    }

    // 执行插入
    if opts != nil && opts.Tx != nil {
        return c.river.InsertTx(ctx, opts.Tx, args, riverOpts)
    }

    return c.river.Insert(ctx, args, riverOpts)
}

// InsertTx 对齐 trigger.dev 的事务支持
func (c *Client) InsertTx[T TaskArgs](ctx context.Context, tx pgx.Tx, args T, opts *InsertOpts) (*river.Job[T], error) {
    if opts == nil {
        opts = &InsertOpts{}
    }
    opts.Tx = tx
    return c.Insert(ctx, args, opts)
}

// isDynamicQueue 检查是否为动态队列任务
func isDynamicQueue(taskKind string) bool {
    dynamicTasks := map[string]bool{
        "startQueuedRuns":      true,
        "performRunExecution":  true,
        "performTaskOperation": true,
    }
    return dynamicTasks[taskKind]
}
```

### 5. 处理器接口对齐

```go
// internal/services/workerqueue/handler.go
package workerqueue

import (
    "context"
    "time"

    "github.com/riverqueue/river"
)

// TaskHandler 对齐 trigger.dev 的 handler 接口
type TaskHandler[T TaskArgs] interface {
    // Handle 对齐 trigger.dev 的 handler 函数签名
    Handle(ctx context.Context, job *river.Job[T]) error
}

// TaskWorker 实现 River 的 Worker 接口，桥接我们的 TaskHandler
type TaskWorker[T TaskArgs] struct {
    river.WorkerDefaults[T]
    handler TaskHandler[T]
    config  TaskQueueConfig
}

func NewTaskWorker[T TaskArgs](handler TaskHandler[T], config TaskQueueConfig) *TaskWorker[T] {
    return &TaskWorker[T]{
        handler: handler,
        config:  config,
    }
}

// Work 实现 River 的 Worker 接口
func (w *TaskWorker[T]) Work(ctx context.Context, job *river.Job[T]) error {
    return w.handler.Handle(ctx, job)
}

// Timeout 返回任务超时时间
func (w *TaskWorker[T]) Timeout(job *river.Job[T]) time.Duration {
    return w.config.Timeout
}
```

## 🔗 与现有 kongflow 服务集成

### 1. 日志服务集成

```go
// 复用 kongflow 现有的 logger 服务
import "kongflow/backend/internal/services/logger"

type TaskLogger struct {
    logger *logger.Logger
}

func (l *TaskLogger) LogTaskStart(taskKind string, jobID int64) {
    l.logger.Info("Task started",
        "task_kind", taskKind,
        "job_id", jobID,
    )
}

func (l *TaskLogger) LogTaskComplete(taskKind string, jobID int64, duration time.Duration) {
    l.logger.Info("Task completed",
        "task_kind", taskKind,
        "job_id", jobID,
        "duration_ms", duration.Milliseconds(),
    )
}

func (l *TaskLogger) LogTaskError(taskKind string, jobID int64, err error) {
    l.logger.Error("Task failed",
        "task_kind", taskKind,
        "job_id", jobID,
        "error", err,
    )
}
```

### 2. 数据库连接复用

```go
// 复用 kongflow 现有的数据库连接
import "kongflow/backend/internal/database"

func NewClientFromKongflowDB(dbPool *database.Pool) (*Client, error) {
    config := Config{
        DatabasePool: dbPool.PgxPool(), // 获取底层 pgxpool.Pool
        RunnerOptions: RunnerOptions{
            Concurrency:  5,
            PollInterval: 1 * time.Second,
        },
        TaskConfigs: GetTaskQueueConfig(),
    }

    return NewClient(config)
}
```

## 📋 实施优先级

### 高优先级任务 (立即实施)

1. `scheduleEmail` - 邮件发送任务
2. `startRun` - 运行启动任务
3. `deliverEvent` - 事件分发任务
4. `indexEndpoint` - 端点索引任务

### 中优先级任务 (第二阶段)

1. `performRunExecution` - 运行执行任务
2. `registerJob` - 作业注册任务
3. `activateSource` - 源激活任务
4. `connectionCreated` - 连接创建任务

### 低优先级任务 (第三阶段)

1. GitHub 集成相关任务
2. VM 管理任务
3. OAuth 令牌刷新任务

---

**结论**: 通过以上详细的技术设计，我们可以实现与 trigger.dev ZodWorker 99% 功能对齐的 Go 版本 Worker Queue Service，同时获得更好的性能和类型安全性。
