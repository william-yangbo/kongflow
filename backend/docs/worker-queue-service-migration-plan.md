# Worker Queue Service 迁移计划

> **项目**: kongflow backend  
> **目标**: 迁移 trigger.dev 的 ZodWorker 和 Worker 服务到 Go + River Queue  
> **状态**: 设计阶段  
> **创建时间**: 2025-09-18

## 📋 概述

本文档详细规划从 trigger.dev 的 TypeScript ZodWorker 系统迁移到 Go 语言基于 River Queue 的 Worker Queue Service 的完整实施方案。

### 🎯 迁移目标

1. **功能对齐**: 100% 复现 trigger.dev 的任务队列核心功能
2. **性能提升**: 利用 Go 语言和 River Queue 的性能优势
3. **类型安全**: 保持编译时类型检查和运行时安全
4. **架构简洁**: 避免过度工程，保持与原系统对齐

### 📊 功能对比分析

| 功能特性         | Trigger.dev ZodWorker | 迁移后 Go + River         | 状态        |
| ---------------- | --------------------- | ------------------------- | ----------- |
| 类型安全任务定义 | ✅ Zod Schema         | ✅ Go Structs + JSON Tags | 🎯 完全对齐 |
| 多队列支持       | ✅ 动态队列名         | 🔄 静态队列映射           | 🟡 等效实现 |
| 任务重试机制     | ✅ maxAttempts        | ✅ MaxAttempts            | 🎯 完全对齐 |
| 优先级控制       | ✅ priority           | ✅ Priority 1-4           | 🎯 完全对齐 |
| 延迟执行         | ✅ runAt              | ✅ ScheduledAt            | 🎯 完全对齐 |
| 事务支持         | ✅ Prisma Tx          | ✅ pgx.Tx                 | 🎯 完全对齐 |
| 批量插入         | ❌ 单任务插入         | ✅ InsertMany             | 🟢 功能增强 |
| 任务去重         | ✅ jobKeyMode         | ✅ UniqueOpts             | 🎯 完全对齐 |
| 错误处理         | ✅ 基础处理           | ✅ 自定义 ErrorHandler    | 🟢 功能增强 |

## 🔧 SQLC + River Queue 集成架构

### SQLC 事务支持模式

基于深入研究 SQLC 源码和文档，SQLC 提供了优雅的事务支持机制：

```go
// SQLC 生成的接口支持多种数据库连接类型
type DBTX interface {
    Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
    Query(context.Context, string, ...interface{}) (pgx.Rows, error)
    QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// SQLC 的 WithTx 模式 - 完美契合 River Queue
func (q *Queries) WithTx(tx pgx.Tx) *Queries {
    return &Queries{db: tx}
}
```

### River + SQLC 事务集成

```go
// kongflow 中的事务模式
func (c *Client) InsertTaskWithBusinessLogic(ctx context.Context, args TaskArgs, businessData interface{}) error {
    return pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
        // 1. 使用 SQLC 执行业务逻辑
        queries := c.sqlcQueries.WithTx(tx)

        if err := queries.InsertBusinessData(ctx, businessData); err != nil {
            return err
        }

        // 2. 使用 River 在同一事务中插入任务
        _, err := c.riverClient.InsertTx(ctx, tx, args, &river.InsertOpts{
            Queue: "business-queue",
            MaxAttempts: 3,
        })

        return err
    })
}
```

### 事务一致性保证

**原子性操作**:

- SQLC 操作和 River 任务插入在同一事务中
- 要么全部成功，要么全部回滚
- 确保业务数据和任务队列的强一致性

**快照隔离**:

- 任务在事务提交前不可见
- 避免任务在业务数据可用前执行
- 符合 trigger.dev 的事务语义

### 性能优化策略

**批量操作支持**:

```go
// River 的批量插入 - trigger.dev 不支持
tasks := []river.InsertManyParams{
    {Args: EmailTaskArgs{To: "user1@example.com"}},
    {Args: EmailTaskArgs{To: "user2@example.com"}},
    // ... 更多任务
}

results, err := client.InsertManyTx(ctx, tx, tasks)
```

## 🏗️ 已有实现借鉴分析

### 现有 Worker 实现优势

通过分析 `bak/bak2/kongflow/backend/internal/worker` 目录的实现，发现了多个优秀的设计模式：

#### 1. 三层架构设计 ⭐️

```go
// Layer 1: 核心管理层 - Manager
type Manager struct {
    riverClient *river.Client[pgx.Tx]
    dbPool      *pgxpool.Pool
    workers     *river.Workers
    config      Config
    logger      *slog.Logger
}

// Layer 2: 兼容适配层 - TriggerCompatibleWorker
type TriggerCompatibleWorker struct {
    manager   *Manager
    catalog   TaskCatalog
    recurring map[string]RecurringTaskConfig
    logger    *slog.Logger
}

// Layer 3: 任务执行层 - 具体 Worker 实现
type IndexEndpointWorker struct {
    river.WorkerDefaults[IndexEndpointArgs]
    indexer EndpointIndexer
    logger  *slog.Logger
}
```

**优势**: 清晰的关注点分离，易于测试和维护

#### 2. 类型安全的任务系统 ⭐️

```go
// 已有实现的 JobArgs 接口设计
type JobArgs interface {
    Kind() string
}

// 任务参数与配置合一
type IndexEndpointArgs struct {
    ID         string                 `json:"id"`
    Source     IndexSource           `json:"source"`
    SourceData map[string]interface{} `json:"source_data,omitempty"`
    JobKey     string                `json:"job_key,omitempty" river:"unique"`
}

func (IndexEndpointArgs) Kind() string { return "index_endpoint" }

func (IndexEndpointArgs) InsertOpts() river.InsertOpts {
    return river.InsertOpts{
        Queue:       string(QueueDefault),
        Priority:    int(PriorityNormal),
        MaxAttempts: 7,
        UniqueOpts: river.UniqueOpts{
            ByArgs:   true,
            ByPeriod: 15 * time.Minute,
        },
    }
}
```

**优势**:

- 任务定义即配置，减少重复
- 编译时类型安全
- 支持任务级别的配置覆盖

#### 3. Trigger.dev 兼容层设计 ⭐️

```go
// TaskCatalog 映射 trigger.dev 的 workerCatalog
type TaskCatalog map[string]TaskDefinition

type TaskDefinition struct {
    QueueName   string
    Priority    int
    MaxAttempts int
    JobKeyMode  string // "replace", "preserve_run_at", "unsafe_dedupe"
    Handler     TaskHandler
}

// 完全兼容的 API 设计
func (w *TriggerCompatibleWorker) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error)

func (w *TriggerCompatibleWorker) EnqueueTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *JobOptions) (*rivertype.JobInsertResult, error)
```

**优势**:

- 最小化迁移成本
- 保持 API 一致性
- 渐进式迁移支持

#### 4. 配置层次化管理 ⭐️

```go
// 全局默认配置
func DefaultConfig() Config {
    return Config{
        MaxWorkers:            10,
        ExecutionMaxWorkers:   5,
        EventsMaxWorkers:      20,
        MaintenanceMaxWorkers: 2,
        FetchCooldown:         100 * time.Millisecond,
        JobTimeout:            1 * time.Minute,
    }
}

// 任务级配置覆盖
func (w *TriggerCompatibleWorker) mergeJobOptions(taskDef TaskDefinition, opts *JobOptions) *JobOptions {
    merged := &JobOptions{
        QueueName:   taskDef.QueueName,
        Priority:    taskDef.Priority,
        MaxAttempts: taskDef.MaxAttempts,
        JobKeyMode:  taskDef.JobKeyMode,
    }

    // 运行时选项覆盖
    if opts != nil {
        if opts.QueueName != "" {
            merged.QueueName = opts.QueueName
        }
        // ... 其他覆盖逻辑
    }

    return merged
}
```

**优势**: 灵活的配置继承，支持细粒度控制

#### 5. 智能重试策略 ⭐️

```go
// 自定义重试策略实现
func (w *IndexEndpointWorker) NextRetry(job *river.Job[IndexEndpointArgs]) time.Time {
    // 指数退避重试: 2^attempt * 30秒, 最多重试5次
    if job.Attempt >= 5 {
        return time.Time{} // 不再重试
    }

    backoffSeconds := int(30 * (1 << job.Attempt)) // 30, 60, 120, 240, 480 seconds
    return time.Now().Add(time.Duration(backoffSeconds) * time.Second)
}
```

**优势**: 避免雪崩效应，提高系统稳定性

### 可直接借鉴的代码模式

1. **任务参数设计**: 采用已有的 `JobArgs` 接口 + `InsertOpts()` 方法
2. **Manager 结构**: 保持三层架构，清晰的职责分离
3. **兼容层 API**: 直接使用 `TriggerCompatibleWorker` 的接口设计
4. **配置合并**: 采用 `mergeJobOptions` 的配置层次化模式
5. **队列常量**: 使用枚举定义队列名、优先级等常量

### 需要优化的部分

1. **SQLC 集成**: 已有实现缺少 SQLC 事务集成，需要增强
2. **错误处理**: 可以增加更详细的错误分类和处理
3. **监控指标**: 增加任务执行的监控和指标收集
4. **测试覆盖**: 补充集成测试和性能测试

## 🔧 技术规范

````

**连接池共享**:
```go
// 复用 kongflow 现有的数据库连接池
func NewWorkerQueueClient(dbPool *pgxpool.Pool, sqlcQueries *sqlc.Queries) *Client {
    riverClient, _ := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
        // River 配置
    })

    return &Client{
        pool:         dbPool,
        riverClient:  riverClient,
        sqlcQueries:  sqlcQueries,
    }
}
````

## 🏗️ 架构设计

### 核心组件架构

```
Worker Queue Service
├── queue/
│   ├── client.go          // River Client 封装
│   ├── config.go          // 队列配置
│   ├── worker_registry.go // Worker 注册管理
│   └── errors.go          // 错误处理
├── tasks/
│   ├── types.go           // 任务类型定义
│   ├── handlers/          // 任务处理器目录
│   │   ├── email.go       // 邮件任务处理器
│   │   ├── runs.go        // 运行任务处理器
│   │   └── ...
│   └── registry.go       // 任务注册
├── examples/
│   └── basic_usage.go    // 使用示例
└── README.md             // 文档
```

### 任务定义结构

```go
// 基础任务接口 - 对齐 trigger.dev 的 JobArgs
type TaskArgs interface {
    Kind() string
}

// 任务处理器接口 - 对齐 ZodWorker 的 handler
type TaskHandler[T TaskArgs] interface {
    Handle(ctx context.Context, task *Task[T]) error
}

// 任务配置 - 对齐 ZodTasks 配置
type TaskConfig struct {
    QueueName   string
    Priority    int
    MaxAttempts int
    Timeout     time.Duration
}
```

## 📋 详细实施计划

### Phase 1: 核心基础设施借鉴与优化 (2-3 天)

#### 1.1 已有实现代码复用 ⭐️

**借鉴目标**: 直接复用 `bak/bak2/kongflow/backend/internal/worker` 的优秀设计

```bash
# 复用已有实现的核心文件
cp bak/bak2/kongflow/backend/internal/worker/types.go internal/services/workerqueue/
cp bak/bak2/kongflow/backend/internal/worker/config.go internal/services/workerqueue/
cp bak/bak2/kongflow/backend/internal/worker/manager.go internal/services/workerqueue/
cp bak/bak2/kongflow/backend/internal/worker/trigger_compatible.go internal/services/workerqueue/

# 需要优化的文件
cp bak/bak2/kongflow/backend/internal/worker/jobs.go internal/services/workerqueue/job_args.go
cp bak/bak2/kongflow/backend/internal/worker/job_consumer.go internal/services/workerqueue/workers.go
```

**优势**:

- 减少 70% 的开发工作量
- 保持已验证的架构设计
- 兼容层 API 已经完备

#### 1.2 SQLC 集成增强

基于已有的 Manager 结构，增加 SQLC 支持：

```go
// 扩展已有的 Manager 结构
type Manager struct {
    riverClient *river.Client[pgx.Tx]
    dbPool      *pgxpool.Pool
    workers     *river.Workers
    config      Config
    logger      *slog.Logger

    // 新增 SQLC 支持
    sqlcQueries *database.Queries  // 复用现有 SQLC 查询
}

// 增强的事务支持方法
func (m *Manager) WithTransaction(ctx context.Context, fn func(TransactionContext) error) error {
    return pgx.BeginFunc(ctx, m.dbPool, func(tx pgx.Tx) error {
        txCtx := &transactionContextImpl{
            qtx:     m.sqlcQueries.WithTx(tx),
            riverTx: m.riverClient.WithTx(tx),
        }
        return fn(txCtx)
    })
}
```

#### 1.3 任务定义系统优化

保持已有的 `JobArgs` 接口，增强任务配置：

```go
// 复用已有的 JobArgs 设计，增加 SQLC 事务支持
type JobArgs interface {
    Kind() string
}

// 扩展已有的任务参数结构
type IndexEndpointArgs struct {
    ID         string                 `json:"id"`
    Source     IndexSource           `json:"source"`
    SourceData map[string]interface{} `json:"source_data,omitempty"`
    JobKey     string                `json:"job_key,omitempty" river:"unique"`

    // 新增: 支持事务上下文
    RequiresTransaction bool `json:"requires_transaction,omitempty"`
}

func (IndexEndpointArgs) Kind() string { return "index_endpoint" }

// 保持已有的 InsertOpts 设计
func (IndexEndpointArgs) InsertOpts() river.InsertOpts {
    return river.InsertOpts{
        Queue:       string(QueueDefault),
        Priority:    int(PriorityNormal),
        MaxAttempts: 7,
        UniqueOpts: river.UniqueOpts{
            ByArgs:   true,
            ByPeriod: 15 * time.Minute,
        },
    }
}
```

### Phase 2: 兼容层完善 (1-2 天)

#### 2.1 TriggerCompatibleWorker 增强

保持已有的兼容层设计，增加 SQLC 集成：

```go
// 扩展已有的 TriggerCompatibleWorker
type TriggerCompatibleWorker struct {
    manager   *Manager
    catalog   TaskCatalog
    recurring map[string]RecurringTaskConfig
    logger    *slog.Logger

    // 新增: SQLC 事务支持
    sqlcQueries *database.Queries
}

// 保持已有的 API，增加事务支持
func (w *TriggerCompatibleWorker) EnqueueWithBusinessLogic(
    ctx context.Context,
    identifier string,
    payload interface{},
    businessLogic func(TransactionContext) error,
) (*rivertype.JobInsertResult, error) {

    return w.manager.WithTransaction(ctx, func(txCtx TransactionContext) error {
        // 1. 执行业务逻辑
        if err := businessLogic(txCtx); err != nil {
            return err
        }

        // 2. 插入任务
        result, err := w.enqueueInTransaction(ctx, txCtx, identifier, payload)
        if err != nil {
            return err
        }

        return nil
    })
}
```

#### 2.2 任务处理器迁移

复用已有的 Worker 实现，增强错误处理：

```go
// 保持已有的 IndexEndpointWorker 设计
type IndexEndpointWorker struct {
    river.WorkerDefaults[IndexEndpointArgs]
    indexer      EndpointIndexer
    logger       *slog.Logger
    sqlcQueries  *database.Queries  // 新增 SQLC 支持
}

// 扩展已有的 Work 方法，增加事务支持
func (w *IndexEndpointWorker) Work(ctx context.Context, job *river.Job[IndexEndpointArgs]) error {
    w.logger.Info("Processing index endpoint job",
        "job_id", job.ID,
        "endpoint_id", job.Args.ID,
        "source", job.Args.Source,
    )

    // 如果需要事务支持
    if job.Args.RequiresTransaction {
        return w.workWithTransaction(ctx, job)
    }

    // 保持原有的处理逻辑
    return w.workRegular(ctx, job)
}
```

### Phase 3: 核心任务类型迁移 (3-4 天)

#### 3.1 保持已有任务定义

直接使用已实现的任务类型：

- ✅ `IndexEndpointArgs` - 端点索引
- ✅ `StartRunArgs` - 运行启动
- ✅ `InvokeDispatcherArgs` - 事件分发
- ✅ `DeliverEventArgs` - 事件投递
- ✅ `PerformRunExecutionV2Args` - 运行执行

#### 3.2 增强任务配置

基于已有的配置层次化设计，增加 SQLC 集成：

```go
// 保持已有的队列映射设计
var QueueMapping = map[string]string{
    "startRun":             "executions",
    "performRunExecution":  "executions",
    "performTaskOperation": "tasks",
    "scheduleEmail":        "internal-queue",
    "deliverEvent":         "event-dispatcher",
    "indexEndpoint":        string(QueueDefault),
}

// 扩展已有的配置管理
func (c *Config) WithSQLCIntegration(queries *database.Queries) *Config {
    enhanced := *c
    enhanced.SQLCQueries = queries
    return &enhanced
}
```

### Phase 4: 测试与部署 (2-3 天)

#### 4.1 集成测试增强

基于已有的测试结构，增加 SQLC 测试：

```go
// 扩展已有的测试工具
type TestHarness struct {
    manager     *Manager
    worker      *TriggerCompatibleWorker
    db          *pgxpool.Pool
    sqlcQueries *database.Queries
    cleanup     func()
}

// 保持已有的测试模式，增加事务测试
func (h *TestHarness) TestTransactionalWorkflow(t *testing.T) {
    // 复用已有的测试逻辑 + SQLC 事务验证
}
```

#### 4.2 性能基准测试

基于已有实现进行性能对比：

- 保持已有的任务吞吐量测试
- 增加 SQLC 事务性能测试
- 对比 trigger.dev 的性能指标

### 🎯 复用优势总结

通过借鉴已有实现，可以获得以下优势：

1. **开发效率**: 减少 70% 的开发工作量
2. **架构稳定**: 已验证的三层架构设计
3. **兼容性**: 现成的 trigger.dev 兼容层
4. **类型安全**: 完整的 Go 类型系统
5. **配置灵活**: 层次化配置管理系统

需要增强的部分：

- SQLC 事务集成 (新增 20% 工作量)
- 错误处理优化 (优化 10% 工作量)
- 测试覆盖增强 (补充测试用例)

**总体评估**: 通过复用已有实现，可以将原计划的 4 周开发时间缩短到 1.5-2 周。
type TaskHandler[T TaskArgs] interface {
Handle(ctx context.Context, task \*Task[T]) error
Config() TaskConfig
}

// TaskConfig 任务配置 - 对齐 trigger.dev ZodTasks
type TaskConfig struct {
QueueName string // 对齐 queueName
Priority int // 对齐 priority
MaxAttempts int // 对齐 maxAttempts
Timeout time.Duration // 对齐超时设置
}

// TransactionalTaskService - 集成 SQLC 事务支持
type TransactionalTaskService struct {
riverClient *river.Client[pgx.Tx]
sqlcQueries *database.Queries // 复用 kongflow 的 SQLC
pool \*pgxpool.Pool
}

````

#### 1.3 事务集成客户端封装

```go
// internal/services/workerqueue/client.go
package workerqueue

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
    "kongflow/backend/internal/database"
)

type Client struct {
    river       *river.Client[pgx.Tx]
    workers     *river.Workers
    sqlcQueries *database.Queries
    pool        *pgxpool.Pool
}

type Config struct {
    DatabasePool *pgxpool.Pool
    SQLCQueries  *database.Queries

    // 对齐 trigger.dev 的 runnerOptions
    RunnerOptions RunnerOptions

    // 任务配置
    TaskConfigs map[string]TaskConfig
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
            "tasks":           {MaxWorkers: 5},
            "internal-queue":  {MaxWorkers: 3},
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
        river:       riverClient,
        workers:     workers,
        sqlcQueries: config.SQLCQueries,
        pool:        config.DatabasePool,
    }, nil
}

// WithTransaction 提供事务支持 - 对齐 trigger.dev 的事务语义
func (c *Client) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx *TransactionContext) error) error {
    return pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
        txCtx := &TransactionContext{
            Tx:          tx,
            River:       c.river,
            SQLCQueries: c.sqlcQueries.WithTx(tx), // SQLC 事务模式
        }
        return fn(ctx, txCtx)
    })
}

// TransactionContext 事务上下文 - 整合 River 和 SQLC
type TransactionContext struct {
    Tx          pgx.Tx
    River       *river.Client[pgx.Tx]
    SQLCQueries *database.Queries
}

// InsertTask 在事务中插入任务 - 对齐 trigger.dev 的 enqueue 方法
func (tc *TransactionContext) InsertTask(ctx context.Context, args TaskArgs, opts *InsertOpts) (*river.JobInsertResult, error) {
    riverOpts := &river.InsertOpts{
        Queue:       getQueueName(args.Kind()),
        Priority:    getTaskPriority(args.Kind()),
        MaxAttempts: getTaskMaxAttempts(args.Kind()),
    }

    if opts != nil {
        if opts.ScheduledAt != nil {
            riverOpts.ScheduledAt = *opts.ScheduledAt
        }
        if opts.UniqueKey != "" {
            riverOpts.UniqueOpts = &river.UniqueOpts{
                ByArgs: true,
                ByQueue: true,
            }
        }
    }

    return tc.River.InsertTx(ctx, tc.Tx, args, riverOpts)
}
````

    Config() TaskConfig

}

// TaskConfig 任务配置
type TaskConfig struct {
QueueName string // 队列名称
Priority int // 优先级 1-4
MaxAttempts int // 最大重试次数
Timeout time.Duration // 超时时间
}

````

#### 1.3 队列客户端封装

```go
// internal/services/workerqueue/client.go
package workerqueue

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type Client struct {
    river   *river.Client[pgx.Tx]
    workers *river.Workers
}

type Config struct {
    DatabasePool *pgxpool.Pool
    Queues       map[string]QueueConfig
}

type QueueConfig struct {
    MaxWorkers int
}

func NewClient(config Config) (*Client, error) {
    // River Client 初始化逻辑
}
````

### Phase 2: 任务系统实现 (3-4 天)

#### 2.1 任务注册系统

```go
// internal/services/workerqueue/registry.go
package workerqueue

type Registry struct {
    handlers map[string]interface{}
    configs  map[string]TaskConfig
}

func (r *Registry) RegisterTask[T TaskArgs](handler TaskHandler[T]) error {
    kind := (*new(T)).Kind()
    r.handlers[kind] = handler
    r.configs[kind] = handler.Config()
    return nil
}
```

#### 2.2 核心任务类型迁移

根据 trigger.dev 的 workerCatalog，优先实现以下任务类型：

```go
// internal/services/workerqueue/tasks/email.go
type ScheduleEmailArgs struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

func (ScheduleEmailArgs) Kind() string { return "scheduleEmail" }

type EmailTaskHandler struct{}

func (h *EmailTaskHandler) Handle(ctx context.Context, task *Task[ScheduleEmailArgs]) error {
    // 对齐 trigger.dev 的 sendEmail 调用
    return nil
}

func (h *EmailTaskHandler) Config() TaskConfig {
    return TaskConfig{
        QueueName:   "internal-queue",
        Priority:    river.PriorityHigh,
        MaxAttempts: 3,
        Timeout:     30 * time.Second,
    }
}
```

#### 2.3 队列映射策略

```go
// internal/services/workerqueue/mapping.go
package workerqueue

// QueueMapping 提供静态队列映射，替代 trigger.dev 的动态队列名
var QueueMapping = map[string]string{
    "startRun":             "executions",
    "performRunExecution":  "executions",
    "performTaskOperation": "tasks",
    "scheduleEmail":        "internal-queue",
    "deliverEvent":         "event-dispatcher",
    // ... 其他映射
}

func GetQueueName(taskKind string) string {
    if queue, exists := QueueMapping[taskKind]; exists {
        return queue
    }
    return river.QueueDefault
}
```

### Phase 3: 高级功能实现 (2-3 天)

#### 3.1 错误处理和重试

```go
// internal/services/workerqueue/errors.go
package workerqueue

import (
    "context"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/rivertype"
)

type ErrorHandler struct {
    logger Logger
}

func (h *ErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
    h.logger.Error("Task failed",
        "task_kind", job.Kind,
        "task_id", job.ID,
        "error", err,
        "attempt", job.Attempt,
    )
    return nil // 使用默认重试策略
}

func (h *ErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
    h.logger.Error("Task panicked",
        "task_kind", job.Kind,
        "task_id", job.ID,
        "panic", panicVal,
        "trace", trace,
    )
    return &river.ErrorHandlerResult{SetCancelled: true}
}
```

#### 3.2 事务支持

```go
// internal/services/workerqueue/transaction.go
package workerqueue

import (
    "context"
    "github.com/jackc/pgx/v5"
)

func (c *Client) InsertTx[T TaskArgs](ctx context.Context, tx pgx.Tx, args T, opts *InsertOpts) (*river.Job[T], error) {
    riverOpts := &river.InsertOpts{
        Queue:       GetQueueName(args.Kind()),
        Priority:    opts.Priority,
        MaxAttempts: opts.MaxAttempts,
        ScheduledAt: opts.ScheduledAt,
        UniqueOpts:  opts.UniqueOpts,
    }

    return c.river.InsertTx(ctx, tx, args, riverOpts)
}
```

### Phase 4: 测试和文档 (2 天)

#### 4.1 单元测试

```go
// internal/services/workerqueue/client_test.go
package workerqueue

func TestClientInitialization(t *testing.T) {
    // 测试客户端初始化
}

func TestTaskRegistration(t *testing.T) {
    // 测试任务注册
}

func TestTaskExecution(t *testing.T) {
    // 测试任务执行
}
```

#### 4.2 集成测试

```go
// internal/services/workerqueue/integration_test.go
package workerqueue

func TestEndToEndTaskProcessing(t *testing.T) {
    // 端到端任务处理测试
}

func TestTransactionSupport(t *testing.T) {
    // 事务支持测试
}
```

## 🔧 技术规范

### SQLC 集成架构

```go
// internal/services/workerqueue/sqlc_integration.go
package workerqueue

import (
    "context"
    "database/sql"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "path/to/generated/sqlc"
)

// 扩展 SQLC 生成的 Queries 以支持 River Queue
type ExtendedQueries struct {
    *sqlc.Queries
    river *river.Client[pgx.Tx]
}

func NewExtendedQueries(db *pgxpool.Pool, river *river.Client[pgx.Tx]) *ExtendedQueries {
    return &ExtendedQueries{
        Queries: sqlc.New(db),
        river:   river,
    }
}

// 事务执行接口 - 统一 SQLC 和 River 的事务处理
type TransactionContext interface {
    // SQLC 查询
    CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (*sqlc.User, error)
    UpdateUserStatus(ctx context.Context, arg sqlc.UpdateUserStatusParams) error

    // River 任务插入
    InsertTask(ctx context.Context, args river.InsertOpts) (*river.JobInsertResult, error)
    InsertTaskMany(ctx context.Context, args []river.InsertManyParams) (*river.JobInsertManyResult, error)
}

type transactionContextImpl struct {
    qtx      *sqlc.Queries  // SQLC 事务查询
    riverTx  *river.Client[pgx.Tx]  // River 事务客户端
}

// WithTransaction 统一事务处理模式
func (s *WorkerService) WithTransaction(ctx context.Context, fn func(TransactionContext) error) error {
    return s.dbService.BeginFunc(ctx, func(tx pgx.Tx) error {
        txCtx := &transactionContextImpl{
            qtx:     s.queries.WithTx(tx),
            riverTx: s.riverClient.WithTx(tx),
        }
        return fn(txCtx)
    })
}

// 实现 TransactionContext 接口
func (t *transactionContextImpl) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (*sqlc.User, error) {
    return t.qtx.CreateUser(ctx, arg)
}

func (t *transactionContextImpl) InsertTask(ctx context.Context, args river.InsertOpts) (*river.JobInsertResult, error) {
    return t.riverTx.Insert(ctx, args.Args, &args)
}
```

### 代码规范

1. **命名约定**

   - 包名：`workerqueue`
   - 接口名：`TaskHandler`, `TaskArgs`, `TransactionContext`
   - 结构体名：`Client`, `Registry`, `Config`, `ExtendedQueries`

2. **错误处理**

   - 使用 `fmt.Errorf` 包装错误
   - 错误信息包含足够的上下文
   - 关键操作添加日志记录

3. **并发安全**
   - 所有公共方法必须是并发安全的
   - 使用 `sync.RWMutex` 保护共享状态

### 配置管理

```go
// internal/services/workerqueue/config.go
package workerqueue

type Config struct {
    // River 相关配置
    DatabasePool    *pgxpool.Pool           `json:"-"`
    Concurrency     int                     `json:"concurrency"`
    PollInterval    time.Duration           `json:"poll_interval"`

    // 队列配置
    Queues map[string]QueueConfig `json:"queues"`

    // SQLC 集成配置
    SQLCQueries     *sqlc.Queries          `json:"-"`

    // 日志配置
    Logger Logger `json:"-"`
}

type QueueConfig struct {
    MaxWorkers int `json:"max_workers"`
}

// 默认配置 - 对齐 trigger.dev 的默认值
func DefaultConfig() Config {
    return Config{
        Concurrency:  5,                    // 对齐 trigger.dev 的 concurrency: 5
        PollInterval: 1 * time.Second,      // 对齐 trigger.dev 的 pollInterval: 1000
        Queues: map[string]QueueConfig{
            "executions":       {MaxWorkers: 10},
            "tasks":           {MaxWorkers: 5},
            "internal-queue":  {MaxWorkers: 3},
            "event-dispatcher": {MaxWorkers: 8},
            river.QueueDefault: {MaxWorkers: 100},
        },
    }
}
```

## 🧪 测试策略

### 测试覆盖目标

- **单元测试覆盖率**: 90%+
- **集成测试**: 覆盖所有核心流程
- **性能测试**: 验证与 trigger.dev 性能对比
- **SQLC 集成测试**: 验证事务一致性和数据完整性

### SQLC + River 测试集成

```go
// internal/services/workerqueue/testutil/sqlc_integration_test.go
package testutil

import (
    "context"
    "testing"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "path/to/generated/sqlc"
)

type TestHarness struct {
    db      *pgxpool.Pool
    queries *sqlc.Queries
    river   *river.Client[pgx.Tx]
    cleanup func()
}

func SetupTestHarness(t *testing.T) *TestHarness {
    ctx := context.Background()

    // 启动测试数据库容器
    container, db := setupTestDB(t)

    // 运行 SQLC 和 River 迁移
    runMigrations(t, db)

    queries := sqlc.New(db)
    riverClient := setupRiverClient(t, db)

    return &TestHarness{
        db:      db,
        queries: queries,
        river:   riverClient,
        cleanup: func() {
            riverClient.Stop(ctx)
            db.Close()
            container.Terminate(ctx)
        },
    }
}

func (h *TestHarness) TestTransactionalWorkflow(t *testing.T) {
    ctx := context.Background()

    // 测试：在事务中同时执行 SQLC 查询和 River 任务插入
    err := h.db.BeginFunc(ctx, func(tx pgx.Tx) error {
        qtx := h.queries.WithTx(tx)
        riverTx := h.river.WithTx(tx)

        // 1. 使用 SQLC 创建用户
        user, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
            Name:  "test-user",
            Email: "test@example.com",
        })
        if err != nil {
            return err
        }

        // 2. 使用 River 插入与用户相关的任务
        _, err = riverTx.Insert(ctx, &WelcomeEmailArgs{
            UserID: user.ID,
            Email:  user.Email,
        }, nil)
        if err != nil {
            return err
        }

        // 3. 验证事务回滚行为
        if shouldRollback {
            return fmt.Errorf("forced rollback")
        }

        return nil
    })

    // 验证结果
    if shouldRollback {
        assert.Error(t, err)
        // 验证用户未创建且任务未插入
    } else {
        assert.NoError(t, err)
        // 验证用户已创建且任务已插入
    }
}
```

### 测试环境

```go
// internal/services/workerqueue/testutil/testutil.go
package testutil

import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)
)

func SetupTestDatabase(t *testing.T) *pgxpool.Pool {
    // 使用 testcontainers 设置测试数据库
}

func SetupTestWorkerQueue(t *testing.T) *workerqueue.Client {
    // 设置测试用的 Worker Queue 客户端
}
```

## 📈 性能目标

### 吞吐量目标

- **任务插入**: > 1000 tasks/second
- **任务处理**: > 500 tasks/second
- **延迟**: < 100ms (P95)

### 资源使用

- **内存使用**: < 100MB (空闲状态)
- **数据库连接**: 可配置连接池
- **CPU 使用**: < 10% (空闲状态)

## 🚀 部署和运维

### 数据库迁移

```sql
-- River Queue 所需的表结构会自动创建
-- 无需额外的迁移脚本
```

### 监控指标

- 任务执行成功率
- 任务执行延迟
- 队列积压情况
- 错误率统计

### 运维工具

```go
// cmd/worker-admin/main.go
package main

// 提供队列管理的 CLI 工具
func main() {
    // 队列状态查看
    // 任务重试
    // 错误任务清理
}
```

## 📝 实施检查清单

### Phase 1 检查清单

- [ ] 创建项目目录结构
- [ ] 添加 River Queue 依赖
- [ ] 实现基础类型定义
- [ ] 实现队列客户端封装
- [ ] 编写基础单元测试

### Phase 2 检查清单

- [ ] 实现任务注册系统
- [ ] 迁移核心任务类型 (email, runs, events)
- [ ] 实现队列映射策略
- [ ] 编写任务处理器测试

### Phase 3 检查清单

- [ ] 实现错误处理和重试
- [ ] 实现事务支持
- [ ] 实现批量操作
- [ ] 编写集成测试

### Phase 4 检查清单

- [ ] 完善测试覆盖率
- [ ] 编写文档和示例
- [ ] 性能测试和优化
- [ ] 代码审查和优化

## 🔄 迁移验证

### 功能验证

1. **任务插入验证**

   ```go
   // 验证任务能够正确插入队列
   job, err := client.Insert(ctx, ScheduleEmailArgs{
       To: "test@example.com",
       Subject: "Test",
       Body: "Test body",
   }, nil)
   ```

2. **任务执行验证**

   ```go
   // 验证任务能够正确执行
   // 验证错误处理
   // 验证重试机制
   ```

3. **事务支持验证**

   ```go
   // 验证事务内任务插入
   tx, _ := pool.Begin(ctx)
   defer tx.Rollback(ctx)

   job, err := client.InsertTx(ctx, tx, args, nil)
   tx.Commit(ctx)
   ```

### 性能验证

- 与 trigger.dev 原系统性能对比
- 高负载场景测试
- 内存和 CPU 使用率测试

## 📚 参考资料

- [River Queue 官方文档](https://riverqueue.com)
- [trigger.dev ZodWorker 源码分析](../analysis/zodworker-analysis.md)
- [Go 语言最佳实践](https://golang.org/doc/effective_go)
- [PostgreSQL 任务队列最佳实践](https://www.postgresql.org/docs/current/bgworker.html)

---

**注意**: 本计划严格遵循"保持对齐，避免过度工程"的原则，所有设计决策都以复现 trigger.dev 原有功能为准，并适配 Go 语言最佳实践。
