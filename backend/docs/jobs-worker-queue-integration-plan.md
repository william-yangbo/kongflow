# Jobs 服务 Worker Queue 集成计划

## 🎯 目标

为 Jobs 服务添加 Worker Queue 对接，实现完整的测试作业执行流程，对齐 trigger.dev 功能。

## 📋 Phase 2: Jobs 服务 Worker Queue 集成 (2-3 天)

### 2.1 服务架构改进 (0.5 天)

**任务：**

- [ ] 更新 Jobs Service 构造函数，添加 WorkerQueue 依赖
- [ ] 定义 JobRun 相关的数据结构和接口
- [ ] 创建 RunCreationService 组件

**输出：**

- 支持 Worker Queue 的 Jobs Service
- JobRun 数据模型定义
- 创建 Run 的服务组件

### 2.2 TestJob 完整实现 (1 天)

**任务：**

- [ ] 实现 CreateRunService，对齐 trigger.dev 逻辑
- [ ] 更新 TestJob 方法，添加 Run 创建和队列提交
- [ ] 创建测试作业的 Worker 定义
- [ ] 实现作业执行状态跟踪

**输出：**

- 完整的 TestJob 实现
- 测试作业 Worker
- 作业状态管理

### 2.3 Worker 集成和测试 (0.5 天)

**任务：**

- [ ] 注册测试作业 Worker 到 River 队列
- [ ] 创建集成测试验证完整流程
- [ ] 实现作业结果回调机制

**输出：**

- 端到端测试作业执行
- 完整的集成测试
- 结果回调系统

## 🔧 技术实现细节

### WorkerQueue 集成模式

```go
// 1. 更新 Service 接口和实现
type Service interface {
    // 现有方法...
    TestJob(ctx context.Context, req TestJobRequest) (*TestJobResponse, error)
}

type service struct {
    repo        Repository
    logger      *slog.Logger
    workerQueue WorkerQueueClient  // 新增
    runService  *RunCreationService // 新增
}

// 2. WorkerQueue 客户端接口
type WorkerQueueClient interface {
    EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
}

// 3. 新的 TestJob 实现
func (s *service) TestJob(ctx context.Context, req TestJobRequest) (*TestJobResponse, error) {
    // 1. 创建 EventRecord (已实现)
    eventRecord, err := s.createEventRecord(ctx, req)

    // 2. 创建 JobRun (新增)
    run, err := s.runService.CreateRun(ctx, CreateRunRequest{
        EventID:       eventRecord.ID,
        JobVersionID:  req.VersionID,
        EnvironmentID: req.EnvironmentID,
        IsTest:        true,
    })

    // 3. 提交到 Worker Queue (新增)
    result, err := s.workerQueue.EnqueueJob(ctx, "executeJobRun", ExecuteJobRunPayload{
        RunID:         run.ID,
        EventRecordID: eventRecord.ID,
        JobVersionID:  req.VersionID,
    }, &workerqueue.JobOptions{
        QueueName: "job-execution",
        Priority:  1,
        Tags:      []string{"test", "job-run"},
    })

    return &TestJobResponse{
        RunID:   run.ID,
        EventID: eventRecord.EventID,
        Status:  "queued", // 实际状态
        Message: "Test job queued for execution",
    }, nil
}
```

### Worker 定义

```go
// 4. 作业执行 Worker
type ExecuteJobRunWorker struct {
    river.WorkerDefaults[ExecuteJobRunPayload]
    repo   Repository
    logger *slog.Logger
}

type ExecuteJobRunPayload struct {
    RunID         uuid.UUID `json:"run_id"`
    EventRecordID uuid.UUID `json:"event_record_id"`
    JobVersionID  uuid.UUID `json:"job_version_id"`
}

func (w *ExecuteJobRunWorker) Work(ctx context.Context, job *river.Job[ExecuteJobRunPayload]) error {
    // 1. 获取作业版本和事件数据
    // 2. 模拟作业执行（或调用实际端点）
    // 3. 更新 JobRun 状态
    // 4. 记录执行结果
    return nil
}
```

## 🧪 测试策略

### 集成测试流程

1. **TestJob E2E 测试**：

   - 调用 TestJob API
   - 验证 EventRecord 创建
   - 验证 JobRun 创建
   - 验证 Worker Queue 任务提交
   - 验证作业执行完成
   - 验证状态更新

2. **Worker 单元测试**：

   - 模拟作业执行逻辑
   - 测试错误处理
   - 测试状态更新

3. **性能测试**：
   - 并发测试作业提交
   - 队列吞吐量测试

## 📈 预期结果

完成后，Jobs 服务将：

1. ✅ **完全对齐 trigger.dev**：

   - 支持完整的测试作业流程
   - 实际执行作业并返回结果
   - 支持作业状态跟踪

2. ✅ **集成 Worker Queue**：

   - 异步执行测试作业
   - 支持队列优先级和标签
   - 支持作业重试和错误处理

3. ✅ **企业级功能**：
   - 支持高并发作业执行
   - 完整的监控和日志
   - 事务性操作保证

## 🚀 下一步行动

如果需要继续迭代，请确认：

1. 是否开始 Phase 2 的实现？
2. 是否有特定的优先级要求？
3. 是否需要调整技术方案？
