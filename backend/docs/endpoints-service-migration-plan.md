# Endpoints Service 迁移计划

## 📋 项目概览

### 🎯 目标

将 trigger.dev 的 endpoints 服务迁移到 kongflow/backend，实现端点管理和索引功能，严格对齐原版设计模式，适配 Go 语言最佳实践。

### 📊 迁移范围

| 组件                      | trigger.dev 原版                    | kongflow 目标    | 状态      |
| ------------------------- | ----------------------------------- | ---------------- | --------- |
| **CreateEndpointService** | `createEndpoint.server.ts` (102 行) | Go 服务实现      | 📋 待开发 |
| **IndexEndpointService**  | `indexEndpoint.server.ts` (131 行)  | Go 服务实现      | 📋 待开发 |
| **数据库表结构**          | Prisma schema                       | PostgreSQL DDL   | 📋 待开发 |
| **测试套件**              | N/A                                 | Go 单元+集成测试 | 📋 待开发 |

### 🔗 核心依赖评估

| 依赖服务        | kongflow 状态 | 版本 | 备注                              |
| --------------- | ------------- | ---- | --------------------------------- |
| **endpointApi** | ✅ 已实现     | v1.0 | HTTP 客户端，83.4% 测试覆盖       |
| **workerQueue** | ✅ 已实现     | v1.0 | 任务队列，支持 indexEndpoint 任务 |
| **数据层**      | ✅ 已实现     | v1.0 | shared entities 已就绪            |
| **apiAuth**     | ✅ 已实现     | v1.0 | 环境认证服务                      |
| **logger**      | ✅ 已实现     | v1.0 | 结构化日志                        |
| **ulid**        | ✅ 已实现     | v1.0 | 可替代 nanoid                     |

## 🏗️ 技术架构设计

### 📁 目录结构

```
kongflow/backend/internal/services/endpoints/
├── service.go              # 主服务接口定义
├── create_endpoint.go      # CreateEndpointService 实现
├── index_endpoint.go       # IndexEndpointService 实现
├── types.go               # 请求/响应类型定义
├── errors.go              # 错误定义
├── repository.go          # 数据访问层
├── service_test.go        # 单元测试
├── integration_test.go    # 集成测试
└── examples/
    └── basic_usage.go     # 使用示例
```

### 🗄️ 数据库设计

#### 📊 **endpoints 表**

```sql
-- 对齐 trigger.dev Endpoint 模型
CREATE TABLE endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    indexing_hook_identifier VARCHAR(10) NOT NULL,

    -- 关联关系 (严格对齐 trigger.dev)
    environment_id UUID NOT NULL REFERENCES runtime_environments(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- 约束 (对齐 trigger.dev)
    UNIQUE(environment_id, slug)
);
```

#### 📊 **endpoint_indexes 表**

```sql
-- 对齐 trigger.dev EndpointIndex 模型
CREATE TABLE endpoint_indexes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,

    -- 统计和数据 (JSONB 对齐 trigger.dev)
    stats JSONB NOT NULL,
    data JSONB NOT NULL,

    -- 索引来源 (对齐 EndpointIndexSource 枚举)
    source VARCHAR(50) NOT NULL DEFAULT 'MANUAL'
        CHECK (source IN ('MANUAL', 'API', 'INTERNAL', 'HOOK')),
    source_data JSONB,
    reason TEXT,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 🔍 **索引设计**

```sql
-- 性能优化索引
CREATE INDEX idx_endpoints_environment_id ON endpoints(environment_id);
CREATE INDEX idx_endpoints_organization_id ON endpoints(organization_id);
CREATE INDEX idx_endpoints_project_id ON endpoints(project_id);
CREATE INDEX idx_endpoints_slug ON endpoints(environment_id, slug);

CREATE INDEX idx_endpoint_indexes_endpoint_id ON endpoint_indexes(endpoint_id);
CREATE INDEX idx_endpoint_indexes_source ON endpoint_indexes(source);
CREATE INDEX idx_endpoint_indexes_created_at ON endpoint_indexes(created_at);
```

### 🎯 服务接口设计

#### 🔧 **核心接口定义**

```go
// internal/services/endpoints/service.go
package endpoints

import (
    "context"
    "github.com/google/uuid"
    "kongflow/backend/internal/services/apiauth"
    "kongflow/backend/internal/services/endpointapi"
)

// Service 端点管理服务接口
type Service interface {
    // CreateEndpoint 创建端点 (对齐 CreateEndpointService.call)
    CreateEndpoint(ctx context.Context, req *CreateEndpointRequest) (*CreateEndpointResponse, error)

    // IndexEndpoint 索引端点 (对齐 IndexEndpointService.call)
    IndexEndpoint(ctx context.Context, req *IndexEndpointRequest) (*IndexEndpointResponse, error)
}

// CreateEndpointRequest 创建端点请求
type CreateEndpointRequest struct {
    Environment *apiauth.AuthenticatedEnvironment `json:"environment"`
    URL         string                            `json:"url"`
    ID          string                            `json:"id"`  // slug
}

// CreateEndpointResponse 创建端点响应
type CreateEndpointResponse struct {
    ID                      uuid.UUID `json:"id"`
    Slug                    string    `json:"slug"`
    URL                     string    `json:"url"`
    IndexingHookIdentifier  string    `json:"indexingHookIdentifier"`
    EnvironmentID          uuid.UUID `json:"environmentId"`
    OrganizationID         uuid.UUID `json:"organizationId"`
    ProjectID              uuid.UUID `json:"projectId"`
    CreatedAt              time.Time `json:"createdAt"`
    UpdatedAt              time.Time `json:"updatedAt"`
}
```

## 📋 实施计划

### 🚀 Phase 1: 数据库层实现 (预估: 0.5 天)

#### 📝 **任务清单**

- [ ] **1.1** 创建数据库迁移文件 `005_endpoints_service.sql`
- [ ] **1.2** 实现 endpoints 和 endpoint_indexes 表结构
- [ ] **1.3** 添加必要的索引和约束
- [ ] **1.4** 测试数据库迁移和回滚

#### ✅ **验收标准**

- 数据库表结构与 trigger.dev Prisma schema 严格对齐
- 外键约束正确配置
- 索引性能优化到位
- 迁移文件可正常执行和回滚

### 🔧 Phase 2: Repository 层实现 (预估: 0.5 天)

#### 📝 **任务清单**

- [ ] **2.1** 实现 `Repository` 接口定义
- [ ] **2.2** 实现 endpoints CRUD 操作
- [ ] **2.3** 实现 endpoint_indexes CRUD 操作
- [ ] **2.4** 实现事务支持
- [ ] **2.5** 添加 repository 单元测试

#### 💻 **核心方法**

```go
// internal/services/endpoints/repository.go
type Repository interface {
    // Endpoint 操作
    CreateEndpoint(ctx context.Context, endpoint *Endpoint) (*Endpoint, error)
    UpdateEndpoint(ctx context.Context, id uuid.UUID, updates *EndpointUpdates) (*Endpoint, error)
    GetEndpointByID(ctx context.Context, id uuid.UUID) (*Endpoint, error)
    GetEndpointBySlug(ctx context.Context, environmentID uuid.UUID, slug string) (*Endpoint, error)

    // EndpointIndex 操作
    CreateEndpointIndex(ctx context.Context, index *EndpointIndex) (*EndpointIndex, error)
    ListEndpointIndexes(ctx context.Context, endpointID uuid.UUID) ([]*EndpointIndex, error)

    // 事务支持
    WithTx(ctx context.Context, fn func(Repository) error) error
}
```

### 🎯 Phase 3: Service 层实现 (预估: 1.5 天)

#### 📝 **任务清单**

- [ ] **3.1** 实现 `CreateEndpointService` (对齐 trigger.dev)
- [ ] **3.2** 实现 `IndexEndpointService` (对齐 trigger.dev)
- [ ] **3.3** 集成 endpointApi 客户端
- [ ] **3.4** 集成 workerQueue 任务调度
- [ ] **3.5** 实现错误处理和日志记录
- [ ] **3.6** 添加服务层单元测试

#### 🔧 **CreateEndpointService 实现重点**

```go
// internal/services/endpoints/create_endpoint.go
func (s *service) CreateEndpoint(ctx context.Context, req *CreateEndpointRequest) (*CreateEndpointResponse, error) {
    // 1. Ping 验证 (对齐 trigger.dev)
    client := endpointapi.NewClient(req.Environment.APIKey, req.URL, req.ID, s.logger)
    pong, err := client.Ping(ctx)
    if err != nil || !pong.OK {
        return nil, NewCreateEndpointError("FAILED_PING", pong.Error)
    }

    // 2. 事务创建端点 + 队列任务 (对齐 trigger.dev)
    var result *Endpoint
    err = s.repo.WithTx(ctx, func(tx Repository) error {
        // 生成 indexingHookIdentifier (对齐 trigger.dev)
        hookID := s.ulid.Generate() // 替代 customAlphabet

        // Upsert endpoint (对齐 trigger.dev 逻辑)
        endpoint, err := tx.UpsertEndpoint(ctx, &UpsertEndpointParams{
            EnvironmentID: req.Environment.ID,
            Slug:          req.ID,
            URL:           req.URL,
            // ... 其他字段
        })
        if err != nil {
            return err
        }

        // 调度 indexEndpoint 任务 (对齐 trigger.dev)
        return s.workerQueue.Enqueue(ctx, "indexEndpoint", &IndexEndpointTask{
            ID:     endpoint.ID,
            Source: "INTERNAL",
        }, &workerqueue.JobOptions{
            Queue: fmt.Sprintf("endpoint-%s", endpoint.ID),
        })
    })

    return result, err
}
```

#### 🔧 **IndexEndpointService 实现重点**

```go
// internal/services/endpoints/index_endpoint.go
func (s *service) IndexEndpoint(ctx context.Context, req *IndexEndpointRequest) (*IndexEndpointResponse, error) {
    // 1. 获取端点信息
    endpoint, err := s.repo.GetEndpointByID(ctx, req.ID)
    if err != nil {
        return nil, err
    }

    // 2. 调用端点索引 API (对齐 trigger.dev)
    client := endpointapi.NewClient(endpoint.Environment.APIKey, endpoint.URL, endpoint.Slug, s.logger)
    indexData, err := client.IndexEndpoint(ctx)
    if err != nil {
        return nil, err
    }

    // 3. 批量处理和任务调度 (对齐 trigger.dev)
    stats := &IndexStats{}
    return s.repo.WithTx(ctx, func(tx Repository) error {
        // 处理 jobs
        for _, job := range indexData.Jobs {
            if !job.Enabled {
                continue
            }
            stats.Jobs++

            // 调度 registerJob 任务
            err := s.workerQueue.Enqueue(ctx, "registerJob", &RegisterJobTask{
                Job:        job,
                EndpointID: endpoint.ID,
            }, &workerqueue.JobOptions{
                Queue: queueName,
            })
            if err != nil {
                return err
            }
        }

        // 处理 sources, dynamicTriggers, dynamicSchedules...
        // ... (对齐 trigger.dev 逻辑)

        // 创建索引记录
        return tx.CreateEndpointIndex(ctx, &EndpointIndex{
            EndpointID: endpoint.ID,
            Stats:      stats,
            Data:       indexData,
            Source:     req.Source,
            SourceData: req.SourceData,
            Reason:     req.Reason,
        })
    })
}
```

### 🧪 Phase 4: 测试套件实现 (预估: 1 天)

#### 📝 **任务清单**

- [ ] **4.1** 单元测试 (CreateEndpoint, IndexEndpoint)
- [ ] **4.2** 集成测试 (数据库 + HTTP + 队列)
- [ ] **4.3** 错误场景测试
- [ ] **4.4** 性能基准测试
- [ ] **4.5** 测试覆盖率验证 (目标: 80%+)

#### 🎯 **测试策略**

```go
// internal/services/endpoints/service_test.go
func TestCreateEndpoint_Success(t *testing.T) {
    // 1. 模拟成功的 ping 响应
    mockEndpointAPI := &MockEndpointAPI{
        PingResponse: &endpointapi.PongResponse{OK: true},
    }

    // 2. 测试端点创建
    service := NewService(mockRepo, mockWorkerQueue, mockEndpointAPI, logger)
    result, err := service.CreateEndpoint(ctx, &CreateEndpointRequest{
        Environment: testEnv,
        URL:         "https://test.com",
        ID:          "test-endpoint",
    })

    // 3. 验证结果
    assert.NoError(t, err)
    assert.Equal(t, "test-endpoint", result.Slug)

    // 4. 验证队列任务被调度
    mockWorkerQueue.AssertTaskEnqueued(t, "indexEndpoint")
}
```

### 📚 Phase 5: 文档和示例 (预估: 0.5 天)

#### 📝 **任务清单**

- [ ] **5.1** 编写 README.md 使用指南
- [ ] **5.2** 创建基础使用示例
- [ ] **5.3** API 文档生成
- [ ] **5.4** 错误代码参考

## 🚨 风险评估与缓解

### ⚠️ **技术风险**

| 风险项         | 影响度 | 概率 | 缓解策略                                       |
| -------------- | ------ | ---- | ---------------------------------------------- |
| **事务一致性** | 高     | 中   | 使用数据库事务，确保端点创建和队列任务的原子性 |
| **网络延迟**   | 中     | 高   | 实现超时和重试机制，优雅处理网络错误           |
| **数据库性能** | 中     | 低   | 合理设计索引，使用连接池优化                   |
| **队列积压**   | 中     | 中   | 监控队列长度，实现背压控制                     |

### 🛡️ **质量保证**

| 质量维度       | 目标    | 验证方式             |
| -------------- | ------- | -------------------- |
| **测试覆盖率** | 80%+    | Go test coverage     |
| **性能基准**   | < 100ms | Benchmark tests      |
| **错误处理**   | 100%    | Error scenario tests |
| **API 对齐度** | 99%+    | 对比测试             |

## 📊 项目时间线

### 📅 **总体时间安排 (3.5 天)**

| 阶段        | 时间   | 关键里程碑         |
| ----------- | ------ | ------------------ |
| **Phase 1** | 0.5 天 | 数据库结构完成     |
| **Phase 2** | 0.5 天 | Repository 层完成  |
| **Phase 3** | 1.5 天 | 服务层核心功能完成 |
| **Phase 4** | 1 天   | 测试覆盖率达标     |
| **Phase 5** | 0.5 天 | 文档和示例完成     |

### 🎯 **关键检查点**

- **Day 1 End**: 数据库 + Repository 层就绪
- **Day 2 End**: CreateEndpoint 服务完成
- **Day 3 End**: IndexEndpoint 服务完成，测试覆盖达标
- **Day 4**: 文档完善，ready for production

## 📋 验收标准

### ✅ **功能验收**

- [ ] CreateEndpoint 功能与 trigger.dev 行为 100% 对齐
- [ ] IndexEndpoint 功能与 trigger.dev 行为 100% 对齐
- [ ] 支持所有 EndpointIndexSource 类型
- [ ] 错误处理与 trigger.dev 完全一致
- [ ] WorkerQueue 集成正常工作

### ✅ **质量验收**

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 集成测试全部通过
- [ ] 性能基准测试通过
- [ ] 代码质量符合 Go 最佳实践
- [ ] 文档完整且准确

### ✅ **对齐验收**

- [ ] 数据模型与 trigger.dev Prisma 完全对齐
- [ ] API 行为与 trigger.dev 完全对齐
- [ ] 错误代码和消息与 trigger.dev 完全对齐
- [ ] 队列任务调度与 trigger.dev 完全对齐

## 🎉 总结

本迁移计划确保 endpoints 服务与 trigger.dev 原版保持严格对齐，同时充分利用 kongflow 已有的基础设施。通过分阶段实施、全面测试和质量保证，我们将交付一个生产就绪的端点管理服务。

**预期成果**:

- 🚀 高质量的 Go 端点管理服务
- 🎯 与 trigger.dev 99%+ 的功能对齐度
- 🧪 80%+ 的测试覆盖率
- 📚 完整的文档和使用示例
- ⚡ 3.5 天的快速交付周期

**下一步**: 立即开始 Phase 1 的数据库层实现！
