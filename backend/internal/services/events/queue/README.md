# Events Queue Service

Events 服务的 WorkerQueue 集成模块，提供异步事件分发和调度器调用能力，严格对齐 trigger.dev 架构。

## 📋 概览

该模块实现了 Events 服务与 River WorkerQueue 系统的集成，支持以下核心功能：

- **异步事件分发**: 对齐 `trigger.dev` 的 `deliverEvent` 作业
- **调度器调用**: 对齐 `trigger.dev` 的 `events.invokeDispatcher` 作业
- **事务支持**: 在数据库事务中安全地入队作业
- **延迟投递**: 支持计划在未来特定时间执行的事件

## 🏗️ 架构设计

### 接口设计

```go
type QueueService interface {
    // 将事件分发任务加入队列
    EnqueueDeliverEvent(ctx context.Context, req EnqueueDeliverEventRequest) (*rivertype.JobInsertResult, error)

    // 将调度器调用任务加入队列
    EnqueueInvokeDispatcher(ctx context.Context, req EnqueueInvokeDispatcherRequest) (*rivertype.JobInsertResult, error)
}
```

### trigger.dev 对齐

#### DeliverEvent 作业

**trigger.dev:**

```typescript
await workerQueue.enqueue(
  'deliverEvent',
  { id: eventLog.id },
  { runAt: eventLog.deliverAt, tx }
);
```

**KongFlow:**

```go
queueReq := queue.EnqueueDeliverEventRequest{
    EventRecordID: uuid.UUID(record.ID.Bytes),
    RunAt:         deliverAt,
    WithinTx:      true,
}
_, err := s.queueSvc.EnqueueDeliverEvent(ctx, queueReq)
```

#### InvokeDispatcher 作业

**trigger.dev:**

```typescript
await workerQueue.enqueue(
  'events.invokeDispatcher',
  { id: eventDispatcher.id, eventRecordId: eventRecord.id },
  { tx }
);
```

**KongFlow:**

```go
queueReq := queue.EnqueueInvokeDispatcherRequest{
    DispatcherID:  uuid.UUID(dispatcher.ID.Bytes),
    EventRecordID: uuid.UUID(eventRecord.ID.Bytes),
    WithinTx:      true,
}
_, err := s.queueSvc.EnqueueInvokeDispatcher(ctx, queueReq)
```

## 🔧 使用方式

### 1. 创建队列服务

```go
import (
    "kongflow/backend/internal/services/events/queue"
    "kongflow/backend/internal/services/workerqueue"
)

// 创建 WorkerQueue 客户端
workerClient := workerqueue.NewClient(db, logger)

// 创建 Events 队列服务
queueSvc := queue.NewRiverQueueService(workerClient)

// 创建 Events 服务（集成队列）
eventsService := events.NewService(repository, queueSvc, logger)
```

### 2. 事件分发

```go
// 在 IngestSendEvent 中自动触发
req := &events.SendEventRequest{
    ID:      "event-123",
    Name:    "user.login",
    Payload: map[string]interface{}{"user_id": "456"},
}

opts := &events.SendEventOptions{
    DeliverAfter: 60, // 60秒后分发
}

response, err := eventsService.IngestSendEvent(ctx, env, req, opts)
// ✅ 自动将 deliverEvent 作业加入队列
```

### 3. 调度器调用

```go
// 在 DeliverEvent 中自动触发
err := eventsService.DeliverEvent(ctx, eventRecordID)
// ✅ 自动为匹配的调度器加入 invokeDispatcher 作业队列
```

## 🎯 特性

### 队列配置

| 特性            | trigger.dev | KongFlow | 说明             |
| --------------- | ----------- | -------- | ---------------- |
| **Queue Name**  | 默认队列    | `events` | 专用事件队列     |
| **Priority**    | 默认        | `HIGH`   | 事件处理高优先级 |
| **Retry**       | 内置        | 5 次重试 | 自动错误重试     |
| **Uniqueness**  | ByArgs      | ByArgs   | 防止重复作业     |
| **Transaction** | 支持        | 支持     | 事务安全         |

### 延迟投递

```go
// 立即投递
req := EnqueueDeliverEventRequest{
    EventRecordID: eventID,
    // RunAt 为 nil，立即执行
}

// 延迟投递
futureTime := time.Now().Add(30 * time.Minute)
req := EnqueueDeliverEventRequest{
    EventRecordID: eventID,
    RunAt:         &futureTime, // 30分钟后执行
}
```

### 事务支持

```go
// 在事务中安全入队
req := EnqueueDeliverEventRequest{
    EventRecordID: eventID,
    WithinTx:      true, // 事务中执行
}

// 使用 EnqueueWithBusinessLogic 保证事务一致性
```

## 🧪 测试

### 单元测试

```bash
go test ./internal/services/events/queue -v
```

### Mock 客户端

```go
type MockWorkerQueueClient struct {
    mock.Mock
}

func (m *MockWorkerQueueClient) Enqueue(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error) {
    args := m.Called(ctx, identifier, payload, opts)
    return args.Get(0).(*rivertype.JobInsertResult), args.Error(1)
}
```

## 🔍 监控和调试

### 队列状态查看

```bash
# 查看 events 队列状态
river job list --queue=events

# 查看失败作业
river job list --state=failed --queue=events

# 手动重试失败作业
river job retry <job_id>
```

### 日志记录

```go
logger.Debug("Enqueued dispatcher invocation",
    "dispatcher_id", dispatcher.ID.Bytes,
    "event_record_id", eventRecord.ID.Bytes)
```

## 🚀 性能优化

### 批量操作

```go
// 未来可扩展：批量调度器调用
type EnqueueBatchInvokeDispatcherRequest struct {
    Dispatchers   []DispatcherEventPair
    WithinTx      bool
}
```

### 优先级队列

```go
// 根据事件类型动态设置优先级
req := EnqueueDeliverEventRequest{
    EventRecordID: eventID,
    Priority:      int(workerqueue.PriorityCritical), // 关键事件
}
```

## 📈 扩展计划

### Phase 2 增强

- [ ] **批量作业处理**: 减少队列开销
- [ ] **智能重试策略**: 指数退避算法
- [ ] **死信队列**: 失败作业收集分析
- [ ] **队列分片**: 基于项目/环境的队列分离
- [ ] **性能监控**: 队列延迟和吞吐量指标

### 集成点

- [ ] **Runs Service**: 作业运行创建集成
- [ ] **Jobs Service**: 作业版本调度集成
- [ ] **Dynamic Triggers**: 动态触发器调度集成
- [ ] **Webhooks**: Webhook 事件异步处理

---

_严格对齐 trigger.dev 架构，提供企业级的异步事件处理能力_ 🚀
