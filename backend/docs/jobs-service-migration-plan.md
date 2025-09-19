# Jobs Service 迁移计划

## 1. 执行概要

本文档详细制定了从 trigger.dev 的 Jobs 服务到 kongflow/backend 的迁移计划。基于已成功迁移的 endpoints 服务模式，确保与 trigger.dev 的功能严格对齐，同时适配 Go 语言最佳实践。

### 1.1 迁移目标

- **100% 功能对齐**：确保与 trigger.dev Jobs 服务功能完全对等
- **遵循已有模式**：沿用 endpoints 服务的成功经验和架构模式
- **Go 最佳实践**：采用 Go 语言生态中的最佳实践
- **高测试覆盖率**：确保 > 95% 的测试覆盖率

### 1.2 核心服务范围

- **Job Management**: 作业注册、更新、查询、删除
- **JobVersion Management**: 版本控制、事件规范管理
- **JobQueue Management**: 队列管理、并发控制
- **JobIntegration Management**: 作业集成管理
- **Testing Support**: 作业测试功能

## 2. 源系统分析

### 2.1 trigger.dev Jobs 数据模型分析

基于 trigger.dev 的 Prisma Schema，核心数据模型包括：

#### 2.1.1 Job Model

```prisma
model Job {
  id       String  @id @default(cuid())
  slug     String
  title    String
  internal Boolean @default(false)

  organization   Organization @relation(fields: [organizationId], references: [id], onDelete: Cascade)
  organizationId String
  project        Project      @relation(fields: [projectId], references: [id], onDelete: Cascade)
  projectId      String

  versions        JobVersion[]
  runs            JobRun[]
  integrations    JobIntegration[]
  aliases         JobAlias[]
  dynamicTriggers DynamicTrigger[]

  @@unique([projectId, slug])
}
```

#### 2.1.2 JobVersion Model

```prisma
model JobVersion {
  id                 String @id @default(cuid())
  version            String
  eventSpecification Json
  properties         Json?

  job         Job                 @relation(fields: [jobId], references: [id], onDelete: Cascade)
  jobId       String
  endpoint    Endpoint            @relation(fields: [endpointId], references: [id], onDelete: Cascade)
  endpointId  String
  environment RuntimeEnvironment @relation(fields: [environmentId], references: [id], onDelete: Cascade)
  environment String
  queue       JobQueue            @relation(fields: [queueId], references: [id])
  queueId     String

  startPosition  JobStartPosition @default(INITIAL)
  preprocessRuns Boolean          @default(false)

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  runs         JobRun[]
  integrations JobIntegration[]
  aliases      JobAlias[]
  examples     EventExample[]

  @@unique([jobId, version, environmentId])
}
```

#### 2.1.3 JobQueue Model

```prisma
model JobQueue {
  id   String @id @default(cuid())
  name String

  environment   RuntimeEnvironment @relation(fields: [environmentId], references: [id], onDelete: Cascade)
  environmentId String

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  jobCount Int @default(0)
  maxJobs  Int @default(100)

  runs       JobRun[]
  jobVersion JobVersion[]

  @@unique([environmentId, name])
}
```

### 2.2 trigger.dev Jobs 服务分析

#### 2.2.1 RegisterJobService

**核心功能**:

- 作业注册和更新 (Upsert 模式)
- 集成管理
- 版本管理
- 队列管理
- 事件分发器管理
- 别名管理

**关键方法**:

- `call(endpointId: string, metadata: JobMetadata)`
- `#upsertJob(endpoint, environment, metadata)`
- `#upsertEventDispatcher(trigger, job, jobVersion, environment)`
- `#upsertJobIntegration(job, jobVersion, config, integrations, key)`

#### 2.2.2 TestJobService

**核心功能**:

- 作业测试执行
- 事件记录创建
- 运行创建

**关键方法**:

- `call({environmentId, versionId, payload})`

### 2.3 API 接口分析

trigger.dev 的 Jobs 相关 API 通过 endpoints 进行访问，主要包括：

- `POST /api/v1/endpoints` - 端点创建（包含作业注册逻辑）
- `POST /api/v1/endpoints/{endpointSlug}/index` - 端点索引（触发作业发现）

## 3. 目标架构设计

### 3.1 Go 数据模型设计

基于 trigger.dev 的数据模型，设计符合 Go 最佳实践的数据结构：

#### 3.1.1 Job Models

```go
// Job 作业主体
type Job struct {
    ID             uuid.UUID `json:"id" db:"id"`
    Slug           string    `json:"slug" db:"slug"`
    Title          string    `json:"title" db:"title"`
    Internal       bool      `json:"internal" db:"internal"`
    OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
    ProjectID      uuid.UUID `json:"project_id" db:"project_id"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// JobVersion 作业版本
type JobVersion struct {
    ID                 uuid.UUID               `json:"id" db:"id"`
    JobID              uuid.UUID               `json:"job_id" db:"job_id"`
    Version            string                  `json:"version" db:"version"`
    EventSpecification map[string]interface{}  `json:"event_specification" db:"event_specification"`
    Properties         *map[string]interface{} `json:"properties,omitempty" db:"properties"`
    EndpointID         uuid.UUID               `json:"endpoint_id" db:"endpoint_id"`
    EnvironmentID      uuid.UUID               `json:"environment_id" db:"environment_id"`
    OrganizationID     uuid.UUID               `json:"organization_id" db:"organization_id"`
    ProjectID          uuid.UUID               `json:"project_id" db:"project_id"`
    QueueID            uuid.UUID               `json:"queue_id" db:"queue_id"`
    StartPosition      JobStartPosition        `json:"start_position" db:"start_position"`
    PreprocessRuns     bool                    `json:"preprocess_runs" db:"preprocess_runs"`
    CreatedAt          time.Time               `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time               `json:"updated_at" db:"updated_at"`
}

// JobQueue 作业队列
type JobQueue struct {
    ID            uuid.UUID `json:"id" db:"id"`
    Name          string    `json:"name" db:"name"`
    EnvironmentID uuid.UUID `json:"environment_id" db:"environment_id"`
    JobCount      int       `json:"job_count" db:"job_count"`
    MaxJobs       int       `json:"max_jobs" db:"max_jobs"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// JobStartPosition 作业开始位置枚举
type JobStartPosition string

const (
    JobStartPositionInitial JobStartPosition = "INITIAL"
    JobStartPositionLatest  JobStartPosition = "LATEST"
)

// JobAlias 作业别名
type JobAlias struct {
    ID            uuid.UUID `json:"id" db:"id"`
    JobID         uuid.UUID `json:"job_id" db:"job_id"`
    VersionID     uuid.UUID `json:"version_id" db:"version_id"`
    EnvironmentID uuid.UUID `json:"environment_id" db:"environment_id"`
    Name          string    `json:"name" db:"name"`
    Value         string    `json:"value" db:"value"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// EventExample 事件示例
type EventExample struct {
    ID           uuid.UUID               `json:"id" db:"id"`
    JobVersionID uuid.UUID               `json:"job_version_id" db:"job_version_id"`
    Slug         string                  `json:"slug" db:"slug"`
    Name         string                  `json:"name" db:"name"`
    Icon         *string                 `json:"icon,omitempty" db:"icon"`
    Payload      *map[string]interface{} `json:"payload,omitempty" db:"payload"`
    CreatedAt    time.Time               `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time               `json:"updated_at" db:"updated_at"`
}
```

#### 3.1.2 Request/Response Models

```go
// RegisterJobRequest 作业注册请求
type RegisterJobRequest struct {
    ID             string                  `json:"id" validate:"required"`
    Name           string                  `json:"name" validate:"required"`
    Version        string                  `json:"version" validate:"required"`
    Internal       bool                    `json:"internal"`
    Event          EventSpecification      `json:"event" validate:"required"`
    Trigger        TriggerMetadata         `json:"trigger" validate:"required"`
    Queue          QueueConfig             `json:"queue,omitempty"`
    Integrations   map[string]Integration  `json:"integrations,omitempty"`
    StartPosition  string                  `json:"startPosition,omitempty"`
    PreprocessRuns bool                    `json:"preprocessRuns"`
}

// EventSpecification 事件规范
type EventSpecification struct {
    Name     string            `json:"name" validate:"required"`
    Source   string            `json:"source,omitempty"`
    Examples []EventExample    `json:"examples,omitempty"`
}

// TriggerMetadata 触发器元数据
type TriggerMetadata struct {
    Type       string                  `json:"type" validate:"required"` // "static" | "scheduled"
    Rule       *TriggerRule            `json:"rule,omitempty"`
    Schedule   *ScheduleMetadata       `json:"schedule,omitempty"`
    Properties *map[string]interface{} `json:"properties,omitempty"`
}

// QueueConfig 队列配置
type QueueConfig struct {
    Name          string `json:"name,omitempty"`
    MaxConcurrent int    `json:"maxConcurrent,omitempty"`
}

// JobResponse 作业响应
type JobResponse struct {
    ID             uuid.UUID               `json:"id"`
    Slug           string                  `json:"slug"`
    Title          string                  `json:"title"`
    Internal       bool                    `json:"internal"`
    OrganizationID uuid.UUID               `json:"organization_id"`
    ProjectID      uuid.UUID               `json:"project_id"`
    CreatedAt      time.Time               `json:"created_at"`
    UpdatedAt      time.Time               `json:"updated_at"`
    CurrentVersion *JobVersionResponse     `json:"current_version,omitempty"`
    Versions       []JobVersionResponse    `json:"versions,omitempty"`
}

// JobVersionResponse 作业版本响应
type JobVersionResponse struct {
    ID                 uuid.UUID               `json:"id"`
    Version            string                  `json:"version"`
    EventSpecification map[string]interface{}  `json:"event_specification"`
    Properties         *map[string]interface{} `json:"properties,omitempty"`
    StartPosition      JobStartPosition        `json:"start_position"`
    PreprocessRuns     bool                    `json:"preprocess_runs"`
    CreatedAt          time.Time               `json:"created_at"`
    UpdatedAt          time.Time               `json:"updated_at"`
}
```

### 3.2 服务接口设计

基于 endpoints 服务的成功模式，设计 Jobs 服务接口：

```go
// Service Jobs 服务接口
type Service interface {
    // Job Management
    RegisterJob(ctx context.Context, endpointID uuid.UUID, req RegisterJobRequest) (*JobResponse, error)
    GetJob(ctx context.Context, id uuid.UUID) (*JobResponse, error)
    GetJobBySlug(ctx context.Context, projectID uuid.UUID, slug string) (*JobResponse, error)
    ListJobs(ctx context.Context, params ListJobsParams) (*ListJobsResponse, error)
    DeleteJob(ctx context.Context, id uuid.UUID) error

    // Job Version Management
    GetJobVersion(ctx context.Context, id uuid.UUID) (*JobVersionResponse, error)
    ListJobVersions(ctx context.Context, jobID uuid.UUID) (*ListJobVersionsResponse, error)

    // Job Queue Management
    GetJobQueue(ctx context.Context, environmentID uuid.UUID, name string) (*JobQueue, error)
    CreateJobQueue(ctx context.Context, req CreateJobQueueRequest) (*JobQueue, error)
    UpdateJobQueue(ctx context.Context, id uuid.UUID, req UpdateJobQueueRequest) (*JobQueue, error)

    // Job Testing
    TestJob(ctx context.Context, req TestJobRequest) (*TestJobResponse, error)
}
```

### 3.3 目录结构设计

遵循 endpoints 服务的目录结构：

```
kongflow/backend/internal/services/jobs/
├── db.go                          # 数据库连接
├── models.go                      # 数据模型 (sqlc 生成)
├── jobs.sql.go                    # Jobs 相关查询 (sqlc 生成)
├── job_versions.sql.go            # JobVersions 相关查询 (sqlc 生成)
├── job_queues.sql.go              # JobQueues 相关查询 (sqlc 生成)
├── job_aliases.sql.go             # JobAliases 相关查询 (sqlc 生成)
├── event_examples.sql.go          # EventExamples 相关查询 (sqlc 生成)
├── querier.go                     # 查询器接口 (sqlc 生成)
├── repository.go                  # 仓储实现
├── repository_integration_test.go # 仓储集成测试
├── service.go                     # 服务实现
├── service_unit_test.go           # 服务单元测试
├── service_validation_test.go     # 服务验证测试
├── queue/                         # 队列相关
│   ├── job_queue_worker.go        # 作业队列工作器
│   └── job_queue_service.go       # 队列服务
└── queries/                       # SQL 查询文件
    ├── jobs.sql                   # Jobs 查询
    ├── job_versions.sql           # JobVersions 查询
    ├── job_queues.sql             # JobQueues 查询
    ├── job_aliases.sql            # JobAliases 查询
    └── event_examples.sql         # EventExamples 查询
```

## 4. 数据库设计

### 4.1 表结构设计

#### 4.1.1 jobs 表

```sql
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    internal BOOLEAN NOT NULL DEFAULT false,
    organization_id UUID NOT NULL,
    project_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(project_id, slug),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX idx_jobs_project_slug ON jobs(project_id, slug);
CREATE INDEX idx_jobs_organization ON jobs(organization_id);
```

#### 4.1.2 job_versions 表

```sql
CREATE TABLE job_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    version VARCHAR(100) NOT NULL,
    event_specification JSONB NOT NULL,
    properties JSONB,
    endpoint_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    project_id UUID NOT NULL,
    queue_id UUID NOT NULL,
    start_position job_start_position NOT NULL DEFAULT 'INITIAL',
    preprocess_runs BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(job_id, version, environment_id),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (queue_id) REFERENCES job_queues(id) ON DELETE RESTRICT
);

CREATE TYPE job_start_position AS ENUM ('INITIAL', 'LATEST');

CREATE INDEX idx_job_versions_job ON job_versions(job_id);
CREATE INDEX idx_job_versions_environment ON job_versions(environment_id);
CREATE INDEX idx_job_versions_endpoint ON job_versions(endpoint_id);
```

#### 4.1.3 job_queues 表

```sql
CREATE TABLE job_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    environment_id UUID NOT NULL,
    job_count INTEGER NOT NULL DEFAULT 0,
    max_jobs INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(environment_id, name),
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

CREATE INDEX idx_job_queues_environment ON job_queues(environment_id);
```

#### 4.1.4 job_aliases 表

```sql
CREATE TABLE job_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    version_id UUID NOT NULL,
    environment_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT 'latest',
    value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(job_id, environment_id, name),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES job_versions(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES runtime_environments(id) ON DELETE CASCADE
);

CREATE INDEX idx_job_aliases_job_env ON job_aliases(job_id, environment_id);
```

#### 4.1.5 event_examples 表

```sql
CREATE TABLE event_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_version_id UUID NOT NULL,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(500) NOT NULL,
    icon VARCHAR(255),
    payload JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(slug, job_version_id),
    FOREIGN KEY (job_version_id) REFERENCES job_versions(id) ON DELETE CASCADE
);

CREATE INDEX idx_event_examples_job_version ON event_examples(job_version_id);
```

### 4.2 SQLC 查询设计

#### 4.2.1 jobs.sql

```sql
-- name: CreateJob :one
INSERT INTO jobs (
    slug, title, internal, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, slug, title, internal, organization_id, project_id, created_at, updated_at;

-- name: GetJobByID :one
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs
WHERE id = $1;

-- name: GetJobBySlug :one
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs
WHERE project_id = $1 AND slug = $2;

-- name: UpsertJob :one
INSERT INTO jobs (
    slug, title, internal, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, slug)
DO UPDATE SET
    title = EXCLUDED.title,
    internal = EXCLUDED.internal,
    updated_at = NOW()
RETURNING id, slug, title, internal, organization_id, project_id, created_at, updated_at;

-- name: ListJobsByProject :many
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = $1;
```

## 5. 实施计划

### 5.1 Phase 1: 基础设施搭建 (2-3 天)

#### 5.1.1 数据库层

- [ ] 创建数据库迁移文件
- [ ] 编写 SQLC 查询文件
- [ ] 生成 Go 模型和查询代码
- [ ] 编写仓储层实现
- [ ] 编写仓储集成测试

#### 5.1.2 服务层框架

- [ ] 创建服务接口定义
- [ ] 实现基础服务结构
- [ ] 配置依赖注入
- [ ] 设置日志记录

### 5.2 Phase 2: 核心功能实现 (4-5 天)

#### 5.2.1 Job 管理功能

- [ ] 实现 RegisterJob 方法
- [ ] 实现 GetJob 和 GetJobBySlug 方法
- [ ] 实现 ListJobs 方法
- [ ] 实现 DeleteJob 方法

#### 5.2.2 JobVersion 管理功能

- [ ] 实现版本创建和更新逻辑
- [ ] 实现版本查询功能
- [ ] 实现事件规范管理
- [ ] 实现属性管理

#### 5.2.3 JobQueue 管理功能

- [ ] 实现队列创建和更新
- [ ] 实现队列查询功能
- [ ] 实现并发控制逻辑

### 5.3 Phase 3: 高级功能实现 (3-4 天)

#### 5.3.1 集成和别名管理

- [ ] 实现 JobIntegration 管理
- [ ] 实现 JobAlias 管理
- [ ] 实现事件示例管理

#### 5.3.2 测试功能

- [ ] 实现 TestJob 功能
- [ ] 集成事件记录系统
- [ ] 集成运行创建系统

### 5.4 Phase 4: 测试和文档 (2-3 天)

#### 5.4.1 测试完善

- [ ] 编写全面的单元测试
- [ ] 编写集成测试
- [ ] 编写端到端测试
- [ ] 实现测试覆盖率 > 95%

#### 5.4.2 文档和示例

- [ ] 编写 API 文档
- [ ] 创建使用示例
- [ ] 编写迁移指南

## 6. 测试策略

### 6.1 单元测试

基于 endpoints 服务的测试模式：

#### 6.1.1 服务层测试

```go
func TestJobService_RegisterJob(t *testing.T) {
    tests := []struct {
        name    string
        request RegisterJobRequest
        want    *JobResponse
        wantErr bool
    }{
        {
            name: "successful job registration",
            request: RegisterJobRequest{
                ID:      "test-job",
                Name:    "Test Job",
                Version: "1.0.0",
                Event: EventSpecification{
                    Name: "test.event",
                    Source: "test",
                },
                Trigger: TriggerMetadata{
                    Type: "static",
                    Rule: &TriggerRule{
                        Event: "test.event",
                        Source: "test",
                    },
                },
            },
            want: &JobResponse{
                Slug: "test-job",
                Title: "Test Job",
            },
            wantErr: false,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试实现...
        })
    }
}
```

#### 6.1.2 仓储层测试

```go
func TestJobRepository_CreateJob(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    repo := NewRepository(db)

    tests := []struct {
        name    string
        params  CreateJobParams
        want    Job
        wantErr bool
    }{
        {
            name: "create job successfully",
            params: CreateJobParams{
                Slug: "test-job",
                Title: "Test Job",
                Internal: false,
                OrganizationID: testutil.ValidUUID(),
                ProjectID: testutil.ValidUUID(),
            },
            wantErr: false,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试实现...
        })
    }
}
```

### 6.2 集成测试

#### 6.2.1 数据库集成测试

- 使用真实 PostgreSQL 数据库
- 测试复杂查询和事务
- 验证外键约束和触发器

#### 6.2.2 服务集成测试

- 测试服务间交互
- 验证端到端流程
- 测试并发场景

### 6.3 Mock 测试

创建 Mock 实现用于单元测试：

```go
type MockJobRepository struct {
    jobs       []Job
    jobVersions []JobVersion
    jobQueues   []JobQueue
}

func (m *MockJobRepository) CreateJob(ctx context.Context, params CreateJobParams) (Job, error) {
    // Mock 实现
}

// 更多 Mock 方法...
```

## 7. 性能考虑

### 7.1 数据库优化

#### 7.1.1 索引策略

- `jobs` 表：`(project_id, slug)` 唯一索引
- `job_versions` 表：`(job_id, version, environment_id)` 唯一索引
- `job_queues` 表：`(environment_id, name)` 唯一索引

#### 7.1.2 查询优化

- 使用连接查询减少 N+1 问题
- 实现分页查询
- 使用 JSONB 索引优化 JSON 字段查询

### 7.2 缓存策略

#### 7.2.1 Redis 缓存

- 缓存热点作业信息
- 缓存作业版本信息
- 设置合理的 TTL

#### 7.2.2 应用级缓存

- 内存缓存作业元数据
- 缓存队列配置信息

## 8. 安全考虑

### 8.1 输入验证

#### 8.1.1 请求验证

```go
func validateRegisterJobRequest(req RegisterJobRequest) error {
    if req.ID == "" {
        return errors.New("job ID is required")
    }
    if req.Name == "" {
        return errors.New("job name is required")
    }
    if req.Version == "" {
        return errors.New("job version is required")
    }
    // 更多验证逻辑...
    return nil
}
```

#### 8.1.2 SQL 注入防护

- 使用 SQLC 生成的参数化查询
- 避免动态 SQL 构建

### 8.2 权限控制

#### 8.2.1 组织级别权限

- 验证用户是否属于相应组织
- 验证用户对项目的访问权限

#### 8.2.2 操作级别权限

- 读取权限验证
- 写入权限验证
- 删除权限验证

## 9. 监控和日志

### 9.1 日志策略

#### 9.1.1 结构化日志

```go
logger.Info("Registering job",
    slog.String("job_id", req.ID),
    slog.String("job_name", req.Name),
    slog.String("version", req.Version),
    slog.String("organization_id", orgID.String()),
    slog.String("project_id", projectID.String()),
)
```

#### 9.1.2 错误日志

- 记录所有错误详情
- 包含上下文信息
- 便于问题追踪

### 9.2 指标监控

#### 9.2.1 业务指标

- 作业注册成功率
- 作业注册延迟
- 队列使用率

#### 9.2.2 技术指标

- 数据库连接池状态
- 内存使用情况
- API 响应时间

## 10. 迁移步骤

### 10.1 准备阶段

#### 10.1.1 环境准备

1. 创建开发分支 `feature/jobs-service`
2. 配置测试数据库
3. 准备测试数据

#### 10.1.2 依赖确认

1. 确认 endpoints 服务已稳定
2. 确认相关数据库表已创建
3. 确认测试框架已就绪

### 10.2 实施阶段

#### 10.2.1 Week 1: 基础设施

- Day 1-2: 数据库设计和迁移
- Day 3-4: SQLC 配置和代码生成
- Day 5: 仓储层实现

#### 10.2.2 Week 2: 核心功能

- Day 1-2: Job 管理功能
- Day 3-4: JobVersion 管理功能
- Day 5: JobQueue 管理功能

#### 10.2.3 Week 3: 高级功能和测试

- Day 1-2: 集成和别名管理
- Day 3: 测试功能实现
- Day 4-5: 全面测试和文档

### 10.3 验收阶段

#### 10.3.1 功能验收

- [ ] 所有核心功能正常工作
- [ ] 与 trigger.dev 功能对齐
- [ ] 性能指标达标

#### 10.3.2 质量验收

- [ ] 测试覆盖率 > 95%
- [ ] 代码审查通过
- [ ] 文档完整

## 11. 风险评估和缓解

### 11.1 技术风险

#### 11.1.1 数据模型复杂性

**风险**: Jobs 服务涉及多个关联表，数据模型复杂

**缓解措施**:

- 详细的数据建模和验证
- 逐步实现，先简单后复杂
- 充分的集成测试

#### 11.1.2 性能问题

**风险**: 大量作业时可能出现性能问题

**缓解措施**:

- 合理的数据库索引设计
- 实施缓存策略
- 性能测试和监控

### 11.2 业务风险

#### 11.1.1 功能对齐偏差

**风险**: 与 trigger.dev 功能不完全对齐

**缓解措施**:

- 详细的功能对比分析
- 逐功能验证和测试
- 持续的对比验证

#### 11.2.2 数据一致性

**风险**: 复杂的关联关系可能导致数据不一致

**缓解措施**:

- 使用数据库事务
- 实施数据验证
- 定期数据一致性检查

### 11.3 项目风险

#### 11.3.1 开发时间超期

**风险**: 复杂度可能导致开发时间超出预期

**缓解措施**:

- 详细的任务分解
- 每日进度跟踪
- 及时调整计划

#### 11.3.2 质量风险

**风险**: 功能复杂可能影响代码质量

**缓解措施**:

- 严格的代码审查
- 全面的测试覆盖
- 持续集成和质量检查

## 12. 成功标准

### 12.1 功能标准

- [ ] 100% 实现 trigger.dev Jobs 服务的核心功能
- [ ] 所有 API 接口正常工作
- [ ] 数据完整性和一致性得到保证

### 12.2 质量标准

- [ ] 单元测试覆盖率 ≥ 95%
- [ ] 集成测试覆盖率 ≥ 90%
- [ ] 所有边界情况得到测试
- [ ] 代码审查通过

### 12.3 性能标准

- [ ] API 响应时间 < 100ms (P95)
- [ ] 数据库查询时间 < 50ms (P95)
- [ ] 支持并发访问

### 12.4 文档标准

- [ ] 完整的 API 文档
- [ ] 详细的使用示例
- [ ] 运维和监控指南

## 13. 总结

Jobs Service 是 kongflow 系统的核心组件之一，负责作业的全生命周期管理。本迁移计划严格遵循 trigger.dev 的设计模式，同时充分利用已经验证的 endpoints 服务架构。

通过分阶段实施、全面测试和风险控制，确保迁移过程平稳进行，最终交付一个高质量、高性能的 Jobs 服务，为 kongflow 系统的作业管理提供坚实基础。

---

**文档版本**: 1.0  
**创建时间**: 2025-01-19  
**最后更新**: 2025-01-19  
**创建人**: AI Assistant  
**状态**: 待审核
