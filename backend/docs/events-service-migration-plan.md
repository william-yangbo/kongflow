# Events Service 迁移计划

## 📋 迁移概览

### 🎯 **迁移目标**

将 trigger.dev 的 Events 服务迁移到 KongFlow，确保严格对齐功能和架构，实现高质量的事件驱动系统。

### 📊 **服务范围**

- **DeliverEvent Service**: 事件分发核心逻辑
- **IngestSendEvent Service**: 事件摄取和发送
- **InvokeDispatcher Service**: 调度器调用管理

---

## 🔍 源系统分析

### 📁 **trigger.dev Events 服务架构**

#### 1. **DeliverEventService** (155 行)

```typescript
// 核心职责：事件分发到匹配的调度器
class DeliverEventService {
  public async call(id: string); // 分发指定事件
  #evaluateEventRule(); // 评估事件过滤规则
}
```

**关键功能**:

- 🔍 查找可能的事件调度器
- 🎯 基于过滤规则匹配调度器
- 🚀 异步调用匹配的调度器
- ✅ 标记事件为已分发

#### 2. **IngestSendEvent** (125 行)

```typescript
// 核心职责：事件摄取、存储和初始化分发
class IngestSendEvent {
  public async call(environment, event, options); // 摄取事件
  #calculateDeliverAt(options); // 计算延迟投递时间
}
```

**关键功能**:

- 📥 接收和验证事件数据
- 💾 创建 EventRecord 数据库记录
- ⏰ 支持延迟投递 (deliverAt/deliverAfter)
- 🎯 关联外部账户
- 🚀 触发事件分发作业

#### 3. **InvokeDispatcherService** (156 行)

```typescript
// 核心职责：调度器调用，创建作业运行
class InvokeDispatcherService {
  public async call(id, eventRecordId); // 调用调度器
}
```

**关键功能**:

- 🔍 查找并验证事件调度器
- 🎯 支持作业版本和动态触发器调度
- 🏃 创建作业运行实例
- ✅ 处理调度器状态管理

---

## 📊 数据模型分析

### 🗄️ **核心数据表**

#### 1. **EventRecord** (事件记录表)

```sql
CREATE TABLE event_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL,
    name TEXT NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    payload JSONB NOT NULL,
    context JSONB,
    source TEXT DEFAULT 'trigger.dev',
    organization_id UUID REFERENCES organizations(id),
    environment_id UUID REFERENCES runtime_environments(id),
    project_id UUID REFERENCES projects(id),
    external_account_id UUID REFERENCES external_accounts(id),
    deliver_at TIMESTAMPTZ DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    is_test BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(event_id, environment_id)
);
```

#### 2. **EventDispatcher** (事件调度器表)

```sql
CREATE TABLE event_dispatchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event TEXT NOT NULL,
    source TEXT NOT NULL,
    payload_filter JSONB,
    context_filter JSONB,
    manual BOOLEAN DEFAULT FALSE,
    dispatchable_id TEXT NOT NULL,
    dispatchable JSONB NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    environment_id UUID REFERENCES runtime_environments(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(dispatchable_id, environment_id)
);
```

---

## 🏗️ 目标架构设计

### 📂 **目录结构**

```
internal/services/events/
├── 📄 service.go              # 核心服务接口定义
├── 📄 repository.go           # 数据仓储层接口
├── 📄 deliver_event.go        # 事件分发服务实现
├── 📄 ingest_send_event.go    # 事件摄取服务实现
├── 📄 invoke_dispatcher.go    # 调度器调用服务实现
├── 📄 event_matcher.go        # 事件过滤匹配器
├── 📄 models.go               # 数据模型定义
├── 📄 helpers.go              # 辅助函数
├── 📄 service_test.go         # 综合服务测试
├── 📁 queries/                # SQL 查询文件
│   ├── event_records.sql      # EventRecord 相关查询
│   ├── event_dispatchers.sql  # EventDispatcher 相关查询
│   └── ...
└── 📄 README.md               # 服务使用文档
```

### 🎯 **服务接口设计**

#### **核心服务接口**

```go
// Service Events 服务接口，严格对齐 trigger.dev 实现
type Service interface {
    // 事件摄取 - 对齐 IngestSendEvent.call
    IngestSendEvent(ctx context.Context, env *apiauth.AuthenticatedEnvironment,
                   event *SendEventRequest, opts *SendEventOptions) (*EventRecord, error)

    // 事件分发 - 对齐 DeliverEventService.call
    DeliverEvent(ctx context.Context, eventID string) error

    // 调度器调用 - 对齐 InvokeDispatcherService.call
    InvokeDispatcher(ctx context.Context, dispatcherID string, eventRecordID string) error

    // 事件查询
    GetEventRecord(ctx context.Context, id string) (*EventRecord, error)
    ListEventRecords(ctx context.Context, params ListEventRecordsParams) (*ListEventRecordsResponse, error)

    // 调度器管理
    GetEventDispatcher(ctx context.Context, id string) (*EventDispatcher, error)
    ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) (*ListEventDispatchersResponse, error)
}
```

#### **数据传输对象**

```go
// SendEventRequest 发送事件请求，对齐 trigger.dev RawEvent
type SendEventRequest struct {
    ID        string                 `json:"id" validate:"required"`
    Name      string                 `json:"name" validate:"required"`
    Payload   map[string]interface{} `json:"payload"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Source    string                 `json:"source,omitempty"`
    Timestamp *time.Time             `json:"timestamp,omitempty"`
}

// SendEventOptions 发送选项，对齐 trigger.dev SendEventOptions
type SendEventOptions struct {
    AccountID    *string    `json:"account_id,omitempty"`
    DeliverAt    *time.Time `json:"deliver_at,omitempty"`
    DeliverAfter *int       `json:"deliver_after,omitempty"` // 秒数
}

// EventFilter 事件过滤器，对齐 trigger.dev EventFilter
type EventFilter struct {
    Payload map[string]interface{} `json:"payload"`
    Context map[string]interface{} `json:"context"`
}
```

### 🔧 **核心组件设计**

#### 1. **EventMatcher** (事件匹配器)

```go
// EventMatcher 事件过滤匹配器，对齐 trigger.dev EventMatcher
type EventMatcher struct {
    event *EventRecord
}

func NewEventMatcher(event *EventRecord) *EventMatcher
func (m *EventMatcher) Matches(filter *EventFilter) bool
func patternMatches(payload interface{}, pattern interface{}) bool
```

#### 2. **Repository 接口**

```go
// Repository Events 数据仓储接口
type Repository interface {
    // EventRecord 操作
    CreateEventRecord(ctx context.Context, params CreateEventRecordParams) (*EventRecord, error)
    GetEventRecordByID(ctx context.Context, id string) (*EventRecord, error)
    UpdateEventRecordDeliveredAt(ctx context.Context, id string, deliveredAt time.Time) error
    ListEventRecords(ctx context.Context, params ListEventRecordsParams) ([]*EventRecord, error)

    // EventDispatcher 操作
    GetEventDispatcherByID(ctx context.Context, id string) (*EventDispatcher, error)
    FindEventDispatchers(ctx context.Context, params FindEventDispatchersParams) ([]*EventDispatcher, error)
    ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) ([]*EventDispatcher, error)

    // 事务支持
    WithTx(ctx context.Context, fn func(Repository) error) error
}
```

---

## 🛣️ 实施计划

### 📅 **阶段划分**

#### **Phase 1: 基础架构 (2-3 天)**

- [ ] 创建基础目录结构
- [ ] 设计数据库迁移脚本
- [ ] 实现基础数据模型
- [ ] 创建 Repository 接口
- [ ] 配置 SQLC 生成

#### **Phase 2: 核心服务实现 (4-5 天)**

- [ ] 实现 IngestSendEvent 服务
- [ ] 实现 DeliverEvent 服务
- [ ] 实现 InvokeDispatcher 服务
- [ ] 实现 EventMatcher 组件
- [ ] 集成 WorkerQueue 异步处理

#### **Phase 3: 测试和优化 (2-3 天)**

- [ ] 编写单元测试 (80/20 原则)
- [ ] 集成测试
- [ ] 性能优化
- [ ] 文档完善

---

## 🗄️ 数据库迁移

### **迁移脚本: 001_create_events_tables.sql**

```sql
-- 创建事件记录表
CREATE TABLE IF NOT EXISTS event_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL,
    name TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL DEFAULT '{}',
    context JSONB DEFAULT '{}',
    source TEXT NOT NULL DEFAULT 'trigger.dev',

    -- 关联字段
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES runtime_environments(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    external_account_id UUID REFERENCES external_accounts(id) ON DELETE SET NULL,

    -- 投递控制
    deliver_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,

    -- 元数据
    is_test BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 唯一约束
    CONSTRAINT event_records_event_id_environment_id_key UNIQUE(event_id, environment_id)
);

-- 创建事件调度器表
CREATE TABLE IF NOT EXISTS event_dispatchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event TEXT NOT NULL,
    source TEXT NOT NULL,
    payload_filter JSONB,
    context_filter JSONB,
    manual BOOLEAN NOT NULL DEFAULT FALSE,

    -- 可调度对象
    dispatchable_id TEXT NOT NULL,
    dispatchable JSONB NOT NULL,

    -- 状态控制
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 关联字段
    environment_id UUID NOT NULL REFERENCES runtime_environments(id) ON DELETE CASCADE,

    -- 元数据
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 唯一约束
    CONSTRAINT event_dispatchers_dispatchable_id_environment_id_key UNIQUE(dispatchable_id, environment_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_event_records_environment_id ON event_records(environment_id);
CREATE INDEX IF NOT EXISTS idx_event_records_deliver_at ON event_records(deliver_at) WHERE delivered_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_event_records_created_at ON event_records(created_at);
CREATE INDEX IF NOT EXISTS idx_event_dispatchers_environment_id ON event_dispatchers(environment_id);
CREATE INDEX IF NOT EXISTS idx_event_dispatchers_event_source ON event_dispatchers(event, source) WHERE enabled = TRUE;

-- 创建更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_event_records_updated_at BEFORE UPDATE ON event_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_event_dispatchers_updated_at BEFORE UPDATE ON event_dispatchers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## 🧪 测试策略

### **测试覆盖范围 (80/20 原则)**

#### **单元测试 (80%覆盖)**

- ✅ **IngestSendEvent** 测试用例:

  - 正常事件摄取流程
  - 延迟投递计算逻辑
  - 外部账户关联
  - 错误处理和边界情况

- ✅ **DeliverEvent** 测试用例:

  - 事件调度器匹配逻辑
  - 事件过滤规则验证
  - 异步调用队列集成
  - 分发状态更新

- ✅ **InvokeDispatcher** 测试用例:

  - 调度器状态验证
  - 作业版本调度
  - 动态触发器调度
  - 错误处理

- ✅ **EventMatcher** 测试用例:
  - 简单模式匹配
  - 复杂嵌套匹配
  - 数组模式匹配
  - 边界情况处理

#### **集成测试 (20%覆盖)**

- 🔄 端到端事件处理流程
- 🗄️ 数据库事务完整性
- 🚀 WorkerQueue 集成
- ⚡ 性能基准测试

---

## 🔗 依赖关系

### **内部依赖**

- ✅ `apiauth`: 环境认证 (已迁移)
- ✅ `workerqueue`: 异步作业队列 (已迁移)
- ✅ `logger`: 结构化日志 (已迁移)
- ⏳ `runs`: 作业运行管理 (待迁移，可模拟接口)

### **外部依赖**

- 🐘 PostgreSQL 数据库
- 🏔️ River 作业队列系统
- 🔧 SQLC 代码生成
- 🧪 Testify 测试框架

---

## ⚡ 性能考虑

### **优化策略**

1. **数据库优化**:

   - 合理的索引设计
   - 批量操作支持
   - 连接池配置

2. **异步处理**:

   - 事件分发异步化
   - 批量调度器调用
   - 错误重试机制

3. **内存优化**:
   - 流式处理大批量事件
   - 合理的缓存策略
   - 避免内存泄漏

---

## 🚀 迁移检查清单

### **实施前检查**

- [ ] 确认所有依赖服务已就绪
- [ ] 数据库迁移脚本测试通过
- [ ] 开发环境配置完成
- [ ] 代码审查流程确认

### **实施中检查**

- [ ] 单元测试通过率 > 90%
- [ ] 代码覆盖率 > 80%
- [ ] 性能基准测试通过
- [ ] 错误处理完整性验证

### **实施后检查**

- [ ] 集成测试全部通过
- [ ] 文档更新完成
- [ ] 部署指南编写
- [ ] 监控指标配置

---

## 📚 参考文档

### **trigger.dev 源码参考**

- `apps/webapp/app/services/events/deliverEvent.server.ts`
- `apps/webapp/app/services/events/ingestSendEvent.server.ts`
- `apps/webapp/app/services/events/invokeDispatcher.server.ts`
- `packages/database/prisma/schema.prisma`

### **KongFlow 已迁移服务**

- `internal/services/endpoints/` - 端点管理服务参考
- `internal/services/jobs/` - 作业管理服务参考
- `internal/services/workerqueue/` - 队列服务集成参考

---

## 🎯 成功标准

### **功能对齐标准**

- ✅ 100%实现 trigger.dev Events 核心功能
- ✅ API 接口完全兼容
- ✅ 数据模型严格对齐
- ✅ 错误处理行为一致

### **质量标准**

- ✅ 单元测试覆盖率 ≥ 80%
- ✅ 集成测试全部通过
- ✅ 代码审查通过
- ✅ 性能满足基准要求

### **交付标准**

- ✅ 完整的服务实现
- ✅ 专业的测试套件
- ✅ 详细的使用文档
- ✅ 部署和运维指南

---

_迁移原则: 保持严格对齐，避免过度工程，注重质量和可维护性_
