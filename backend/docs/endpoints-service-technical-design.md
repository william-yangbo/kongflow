# Endpoints Service 技术设计详解

## 🔧 核心技术决策

### 1. 数据库设计对齐策略

#### 🎯 **严格字段映射**

```go
// trigger.dev Prisma Model -> kongflow PostgreSQL
trigger.dev: model Endpoint {
  id                      String  @id @default(cuid())           -> id UUID PRIMARY KEY DEFAULT gen_random_uuid()
  slug                    String                                 -> slug VARCHAR(100) NOT NULL
  url                     String                                 -> url TEXT NOT NULL
  indexingHookIdentifier  String?                                -> indexing_hook_identifier VARCHAR(10) NOT NULL
  environmentId           String                                 -> environment_id UUID NOT NULL
  organizationId          String                                 -> organization_id UUID NOT NULL
  projectId               String                                 -> project_id UUID NOT NULL
  createdAt               DateTime @default(now())               -> created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
  updatedAt               DateTime @updatedAt                    -> updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
}
```

#### 🔍 **唯一约束对齐**

```sql
-- trigger.dev: @@unique([environmentId, slug])
-- kongflow: UNIQUE(environment_id, slug)
ALTER TABLE endpoints ADD CONSTRAINT endpoints_environment_slug_unique
    UNIQUE(environment_id, slug);
```

### 2. 错误处理对齐策略

#### 🚨 **CreateEndpointError 精确映射**

```go
// internal/services/endpoints/errors.go
package endpoints

import "fmt"

// CreateEndpointError 对齐 trigger.dev CreateEndpointError
type CreateEndpointError struct {
    Code    CreateEndpointErrorCode `json:"code"`
    Message string                  `json:"message"`
}

type CreateEndpointErrorCode string

const (
    // 精确对齐 trigger.dev 错误代码
    CreateEndpointErrorFailedPing   CreateEndpointErrorCode = "FAILED_PING"
    CreateEndpointErrorFailedUpsert CreateEndpointErrorCode = "FAILED_UPSERT"
)

func (e *CreateEndpointError) Error() string {
    return e.Message
}

// NewCreateEndpointError 工厂方法 (对齐 trigger.dev 构造函数)
func NewCreateEndpointError(code CreateEndpointErrorCode, message string) *CreateEndpointError {
    return &CreateEndpointError{
        Code:    code,
        Message: message,
    }
}
```

### 3. WorkerQueue 任务集成设计

#### 📋 **IndexEndpoint 任务对齐**

```go
// internal/services/endpoints/tasks.go
package endpoints

import (
    "context"
    "encoding/json"
    "kongflow/backend/internal/services/workerqueue"
)

// IndexEndpointTask 对齐 trigger.dev indexEndpoint 任务
type IndexEndpointTask struct {
    ID         string      `json:"id"`         // endpoint.id
    Source     *string     `json:"source,omitempty"`     // "INTERNAL", "MANUAL", etc.
    SourceData interface{} `json:"sourceData,omitempty"`
    Reason     *string     `json:"reason,omitempty"`
}

// RegisterJobTask 对齐 trigger.dev registerJob 任务
type RegisterJobTask struct {
    Job        interface{} `json:"job"`        // JobMetadata from endpointApi
    EndpointID string      `json:"endpointId"`
}

// RegisterSourceTask 对齐 trigger.dev registerSource 任务
type RegisterSourceTask struct {
    Source     interface{} `json:"source"`     // SourceMetadata from endpointApi
    EndpointID string      `json:"endpointId"`
}

// RegisterDynamicTriggerTask 对齐 trigger.dev registerDynamicTrigger 任务
type RegisterDynamicTriggerTask struct {
    DynamicTrigger interface{} `json:"dynamicTrigger"`
    EndpointID     string      `json:"endpointId"`
}

// RegisterDynamicScheduleTask 对齐 trigger.dev registerDynamicSchedule 任务
type RegisterDynamicScheduleTask struct {
    DynamicSchedule interface{} `json:"dynamicSchedule"`
    EndpointID      string      `json:"endpointId"`
}
```

### 4. 服务依赖注入设计

#### 🏗️ **Service 接口完整定义**

```go
// internal/services/endpoints/service.go
package endpoints

import (
    "context"
    "database/sql"

    "kongflow/backend/internal/services/apiauth"
    "kongflow/backend/internal/services/endpointapi"
    "kongflow/backend/internal/services/logger"
    "kongflow/backend/internal/services/ulid"
    "kongflow/backend/internal/services/workerqueue"
)

// Service 端点管理服务
type Service interface {
    CreateEndpoint(ctx context.Context, req *CreateEndpointRequest) (*CreateEndpointResponse, error)
    IndexEndpoint(ctx context.Context, req *IndexEndpointRequest) (*IndexEndpointResponse, error)
}

// Dependencies 服务依赖
type Dependencies struct {
    DB          *sql.DB
    Logger      logger.Logger
    ULID        ulid.Generator
    WorkerQueue workerqueue.Client

    // EndpointAPI 工厂函数 (避免循环依赖)
    NewEndpointAPIClient func(apiKey, url, endpointID string) endpointapi.Client
}

// service 服务实现
type service struct {
    repo                 Repository
    logger               logger.Logger
    ulid                 ulid.Generator
    workerQueue          workerqueue.Client
    newEndpointAPIClient func(apiKey, url, endpointID string) endpointapi.Client
}

// NewService 创建服务实例
func NewService(deps *Dependencies) Service {
    repo := NewRepository(deps.DB)

    return &service{
        repo:                 repo,
        logger:               deps.Logger,
        ulid:                 deps.ULID,
        workerQueue:          deps.WorkerQueue,
        newEndpointAPIClient: deps.NewEndpointAPIClient,
    }
}
```

### 5. 数据模型设计

#### 📊 **完整实体定义**

```go
// internal/services/endpoints/types.go
package endpoints

import (
    "time"
    "github.com/google/uuid"
    "database/sql/driver"
    "encoding/json"
)

// Endpoint 端点实体 (对齐 trigger.dev Endpoint)
type Endpoint struct {
    ID                     uuid.UUID `json:"id" db:"id"`
    Slug                   string    `json:"slug" db:"slug"`
    URL                    string    `json:"url" db:"url"`
    IndexingHookIdentifier string    `json:"indexingHookIdentifier" db:"indexing_hook_identifier"`

    // 关联关系
    EnvironmentID  uuid.UUID `json:"environmentId" db:"environment_id"`
    OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
    ProjectID      uuid.UUID `json:"projectId" db:"project_id"`

    // 时间戳
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// EndpointIndex 端点索引 (对齐 trigger.dev EndpointIndex)
type EndpointIndex struct {
    ID         uuid.UUID `json:"id" db:"id"`
    EndpointID uuid.UUID `json:"endpointId" db:"endpoint_id"`

    // JSONB 字段
    Stats      IndexStats    `json:"stats" db:"stats"`
    Data       IndexData     `json:"data" db:"data"`

    // 索引来源
    Source     EndpointIndexSource `json:"source" db:"source"`
    SourceData interface{}         `json:"sourceData,omitempty" db:"source_data"`
    Reason     *string             `json:"reason,omitempty" db:"reason"`

    // 时间戳
    CreatedAt time.Time `json:"createdAt" db:"created_at"`
    UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// EndpointIndexSource 索引来源枚举 (对齐 trigger.dev)
type EndpointIndexSource string

const (
    EndpointIndexSourceManual   EndpointIndexSource = "MANUAL"
    EndpointIndexSourceAPI      EndpointIndexSource = "API"
    EndpointIndexSourceInternal EndpointIndexSource = "INTERNAL"
    EndpointIndexSourceHook     EndpointIndexSource = "HOOK"
)

// IndexStats 索引统计 (对齐 trigger.dev)
type IndexStats struct {
    Jobs             int `json:"jobs"`
    Sources          int `json:"sources"`
    DynamicTriggers  int `json:"dynamicTriggers"`
    DynamicSchedules int `json:"dynamicSchedules"`
}

// IndexData 索引数据 (对齐 trigger.dev)
type IndexData struct {
    Jobs             []interface{} `json:"jobs"`
    Sources          []interface{} `json:"sources"`
    DynamicTriggers  []interface{} `json:"dynamicTriggers"`
    DynamicSchedules []interface{} `json:"dynamicSchedules"`
}

// JSONB 支持
func (s IndexStats) Value() (driver.Value, error) {
    return json.Marshal(s)
}

func (s *IndexStats) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    return json.Unmarshal(value.([]byte), s)
}

func (d IndexData) Value() (driver.Value, error) {
    return json.Marshal(d)
}

func (d *IndexData) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    return json.Unmarshal(value.([]byte), d)
}
```

## 🔄 关键业务流程设计

### 1. CreateEndpoint 完整流程

```go
// internal/services/endpoints/create_endpoint.go
func (s *service) CreateEndpoint(ctx context.Context, req *CreateEndpointRequest) (*CreateEndpointResponse, error) {
    // Step 1: 参数验证
    if err := s.validateCreateRequest(req); err != nil {
        return nil, err
    }

    // Step 2: Ping 验证 (对齐 trigger.dev)
    client := s.newEndpointAPIClient(req.Environment.APIKey, req.URL, req.ID)
    pong, err := client.Ping(ctx)
    if err != nil {
        s.logger.Error("failed to ping endpoint", map[string]interface{}{
            "url": req.URL,
            "error": err.Error(),
        })
        return nil, NewCreateEndpointError(CreateEndpointErrorFailedPing, err.Error())
    }

    if !pong.OK {
        s.logger.Error("endpoint ping failed", map[string]interface{}{
            "url": req.URL,
            "error": pong.Error,
        })
        return nil, NewCreateEndpointError(CreateEndpointErrorFailedPing, pong.Error)
    }

    // Step 3: 事务处理 (对齐 trigger.dev $transaction)
    var result *Endpoint
    err = s.repo.WithTx(ctx, func(tx Repository) error {
        // 生成 indexingHookIdentifier (对齐 trigger.dev customAlphabet)
        hookIdentifier := s.generateHookIdentifier()

        // Upsert 端点 (对齐 trigger.dev upsert 逻辑)
        endpoint, err := tx.UpsertEndpoint(ctx, &UpsertEndpointParams{
            Where: EndpointWhereUniqueInput{
                EnvironmentIDSlug: &EnvironmentIDSlugCompoundUniqueInput{
                    EnvironmentID: req.Environment.ID,
                    Slug:          req.ID,
                },
            },
            Create: EndpointCreateInput{
                Environment: EnvironmentConnectInput{
                    ID: req.Environment.ID,
                },
                Organization: OrganizationConnectInput{
                    ID: req.Environment.OrganizationID,
                },
                Project: ProjectConnectInput{
                    ID: req.Environment.ProjectID,
                },
                Slug:                   req.ID,
                URL:                    req.URL,
                IndexingHookIdentifier: hookIdentifier,
            },
            Update: EndpointUpdateInput{
                URL: &req.URL,
            },
        })
        if err != nil {
            return err
        }

        // 调度 indexEndpoint 任务 (对齐 trigger.dev)
        queueName := fmt.Sprintf("endpoint-%s", endpoint.ID)
        err = s.workerQueue.Enqueue(ctx, "indexEndpoint", &IndexEndpointTask{
            ID:     endpoint.ID.String(),
            Source: stringPtr("INTERNAL"),
        }, &workerqueue.JobOptions{
            Queue: queueName,
        })
        if err != nil {
            return err
        }

        result = endpoint
        return nil
    })

    if err != nil {
        s.logger.Error("failed to create endpoint", map[string]interface{}{
            "url": req.URL,
            "slug": req.ID,
            "error": err.Error(),
        })
        return nil, NewCreateEndpointError(CreateEndpointErrorFailedUpsert, err.Error())
    }

    // Step 4: 构造响应
    return &CreateEndpointResponse{
        ID:                     result.ID,
        Slug:                   result.Slug,
        URL:                    result.URL,
        IndexingHookIdentifier: result.IndexingHookIdentifier,
        EnvironmentID:          result.EnvironmentID,
        OrganizationID:         result.OrganizationID,
        ProjectID:              result.ProjectID,
        CreatedAt:              result.CreatedAt,
        UpdatedAt:              result.UpdatedAt,
    }, nil
}

// generateHookIdentifier 生成索引钩子标识符 (替代 trigger.dev customAlphabet)
func (s *service) generateHookIdentifier() string {
    // 对齐 trigger.dev: customAlphabet("0123456789abcdefghijklmnopqrstuvxyz", 10)
    return s.ulid.Generate()[:10] // 截取前10位
}
```

### 2. IndexEndpoint 完整流程

```go
// internal/services/endpoints/index_endpoint.go
func (s *service) IndexEndpoint(ctx context.Context, req *IndexEndpointRequest) (*IndexEndpointResponse, error) {
    // Step 1: 获取端点信息
    endpoint, err := s.repo.GetEndpointByID(ctx, req.ID)
    if err != nil {
        return nil, fmt.Errorf("endpoint not found: %w", err)
    }

    // Step 2: 获取环境信息 (需要 API Key)
    environment, err := s.getEnvironmentWithAPIKey(ctx, endpoint.EnvironmentID)
    if err != nil {
        return nil, fmt.Errorf("environment not found: %w", err)
    }

    // Step 3: 调用端点索引 API (对齐 trigger.dev)
    client := s.newEndpointAPIClient(environment.APIKey, endpoint.URL, endpoint.Slug)
    indexData, err := client.IndexEndpoint(ctx)
    if err != nil {
        s.logger.Error("failed to index endpoint", map[string]interface{}{
            "endpointId": endpoint.ID.String(),
            "url": endpoint.URL,
            "error": err.Error(),
        })
        return nil, fmt.Errorf("failed to index endpoint: %w", err)
    }

    // Step 4: 处理索引数据 (对齐 trigger.dev 逻辑)
    queueName := fmt.Sprintf("endpoint-%s", endpoint.ID)
    stats := &IndexStats{}

    var indexResult *EndpointIndex
    err = s.repo.WithTx(ctx, func(tx Repository) error {
        // 处理 Jobs (对齐 trigger.dev)
        for _, job := range indexData.Jobs {
            if !job.Enabled {
                continue // 跳过未启用的作业
            }

            stats.Jobs++

            // 调度 registerJob 任务
            err := s.workerQueue.Enqueue(ctx, "registerJob", &RegisterJobTask{
                Job:        job,
                EndpointID: endpoint.ID.String(),
            }, &workerqueue.JobOptions{
                Queue: queueName,
            })
            if err != nil {
                return fmt.Errorf("failed to enqueue registerJob: %w", err)
            }
        }

        // 处理 Sources (对齐 trigger.dev)
        for _, source := range indexData.Sources {
            stats.Sources++

            err := s.workerQueue.Enqueue(ctx, "registerSource", &RegisterSourceTask{
                Source:     source,
                EndpointID: endpoint.ID.String(),
            }, &workerqueue.JobOptions{
                Queue: queueName,
            })
            if err != nil {
                return fmt.Errorf("failed to enqueue registerSource: %w", err)
            }
        }

        // 处理 Dynamic Triggers (对齐 trigger.dev)
        for _, dynamicTrigger := range indexData.DynamicTriggers {
            stats.DynamicTriggers++

            err := s.workerQueue.Enqueue(ctx, "registerDynamicTrigger", &RegisterDynamicTriggerTask{
                DynamicTrigger: dynamicTrigger,
                EndpointID:     endpoint.ID.String(),
            }, &workerqueue.JobOptions{
                Queue: queueName,
            })
            if err != nil {
                return fmt.Errorf("failed to enqueue registerDynamicTrigger: %w", err)
            }
        }

        // 处理 Dynamic Schedules (对齐 trigger.dev)
        for _, dynamicSchedule := range indexData.DynamicSchedules {
            stats.DynamicSchedules++

            err := s.workerQueue.Enqueue(ctx, "registerDynamicSchedule", &RegisterDynamicScheduleTask{
                DynamicSchedule: dynamicSchedule,
                EndpointID:      endpoint.ID.String(),
            }, &workerqueue.JobOptions{
                Queue: queueName,
            })
            if err != nil {
                return fmt.Errorf("failed to enqueue registerDynamicSchedule: %w", err)
            }
        }

        // 创建端点索引记录 (对齐 trigger.dev)
        index, err := tx.CreateEndpointIndex(ctx, &EndpointIndex{
            EndpointID: endpoint.ID,
            Stats:      *stats,
            Data: IndexData{
                Jobs:             indexData.Jobs,
                Sources:          indexData.Sources,
                DynamicTriggers:  indexData.DynamicTriggers,
                DynamicSchedules: indexData.DynamicSchedules,
            },
            Source:     req.Source,
            SourceData: req.SourceData,
            Reason:     req.Reason,
        })
        if err != nil {
            return fmt.Errorf("failed to create endpoint index: %w", err)
        }

        indexResult = index
        return nil
    })

    if err != nil {
        s.logger.Error("failed to process endpoint index", map[string]interface{}{
            "endpointId": endpoint.ID.String(),
            "error": err.Error(),
        })
        return nil, err
    }

    // Step 5: 记录成功日志
    s.logger.Debug("endpoint indexed successfully", map[string]interface{}{
        "endpointId": endpoint.ID.String(),
        "stats": stats,
    })

    return &IndexEndpointResponse{
        ID:         indexResult.ID,
        EndpointID: indexResult.EndpointID,
        Stats:      indexResult.Stats,
        Data:       indexResult.Data,
        Source:     indexResult.Source,
        SourceData: indexResult.SourceData,
        Reason:     indexResult.Reason,
        CreatedAt:  indexResult.CreatedAt,
        UpdatedAt:  indexResult.UpdatedAt,
    }, nil
}
```

## 🔍 Repository 层设计详解

```go
// internal/services/endpoints/repository.go
package endpoints

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/google/uuid"
    "github.com/lib/pq"
)

// Repository 数据访问接口
type Repository interface {
    // Endpoint CRUD
    CreateEndpoint(ctx context.Context, endpoint *Endpoint) (*Endpoint, error)
    UpdateEndpoint(ctx context.Context, id uuid.UUID, updates *EndpointUpdates) (*Endpoint, error)
    GetEndpointByID(ctx context.Context, id uuid.UUID) (*Endpoint, error)
    GetEndpointBySlug(ctx context.Context, environmentID uuid.UUID, slug string) (*Endpoint, error)
    UpsertEndpoint(ctx context.Context, params *UpsertEndpointParams) (*Endpoint, error)

    // EndpointIndex CRUD
    CreateEndpointIndex(ctx context.Context, index *EndpointIndex) (*EndpointIndex, error)
    GetEndpointIndexByID(ctx context.Context, id uuid.UUID) (*EndpointIndex, error)
    ListEndpointIndexes(ctx context.Context, endpointID uuid.UUID) ([]*EndpointIndex, error)

    // 事务支持
    WithTx(ctx context.Context, fn func(Repository) error) error
}

// repository 实现
type repository struct {
    db *sql.DB
    tx *sql.Tx
}

// NewRepository 创建仓库实例
func NewRepository(db *sql.DB) Repository {
    return &repository{db: db}
}

// WithTx 事务执行
func (r *repository) WithTx(ctx context.Context, fn func(Repository) error) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    txRepo := &repository{db: r.db, tx: tx}
    err = fn(txRepo)
    if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit()
}

// UpsertEndpoint 更新或创建端点 (对齐 trigger.dev upsert 逻辑)
func (r *repository) UpsertEndpoint(ctx context.Context, params *UpsertEndpointParams) (*Endpoint, error) {
    query := `
        INSERT INTO endpoints (
            slug, url, indexing_hook_identifier,
            environment_id, organization_id, project_id,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
        ON CONFLICT (environment_id, slug)
        DO UPDATE SET
            url = EXCLUDED.url,
            updated_at = NOW()
        RETURNING
            id, slug, url, indexing_hook_identifier,
            environment_id, organization_id, project_id,
            created_at, updated_at
    `

    var endpoint Endpoint
    executor := r.getExecutor()
    err := executor.QueryRowContext(ctx, query,
        params.Create.Slug,
        params.Create.URL,
        params.Create.IndexingHookIdentifier,
        params.Create.Environment.ID,
        params.Create.Organization.ID,
        params.Create.Project.ID,
    ).Scan(
        &endpoint.ID,
        &endpoint.Slug,
        &endpoint.URL,
        &endpoint.IndexingHookIdentifier,
        &endpoint.EnvironmentID,
        &endpoint.OrganizationID,
        &endpoint.ProjectID,
        &endpoint.CreatedAt,
        &endpoint.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("failed to upsert endpoint: %w", err)
    }

    return &endpoint, nil
}

// getExecutor 获取执行器 (支持事务)
func (r *repository) getExecutor() interface {
    QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
    QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
    if r.tx != nil {
        return r.tx
    }
    return r.db
}
```

## 🧪 测试策略详解

### 1. 单元测试设计

```go
// internal/services/endpoints/service_test.go
package endpoints

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "kongflow/backend/internal/services/apiauth"
    "kongflow/backend/internal/services/endpointapi"
    "kongflow/backend/internal/services/workerqueue"
)

// MockRepository 模拟数据访问层
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) CreateEndpoint(ctx context.Context, endpoint *Endpoint) (*Endpoint, error) {
    args := m.Called(ctx, endpoint)
    return args.Get(0).(*Endpoint), args.Error(1)
}

func (m *MockRepository) UpsertEndpoint(ctx context.Context, params *UpsertEndpointParams) (*Endpoint, error) {
    args := m.Called(ctx, params)
    return args.Get(0).(*Endpoint), args.Error(1)
}

func (m *MockRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
    // 简化事务模拟：直接调用函数
    return fn(m)
}

// MockEndpointAPIClient 模拟 endpointApi 客户端
type MockEndpointAPIClient struct {
    mock.Mock
}

func (m *MockEndpointAPIClient) Ping(ctx context.Context) (*endpointapi.PongResponse, error) {
    args := m.Called(ctx)
    return args.Get(0).(*endpointapi.PongResponse), args.Error(1)
}

func (m *MockEndpointAPIClient) IndexEndpoint(ctx context.Context) (*endpointapi.IndexEndpointResponse, error) {
    args := m.Called(ctx)
    return args.Get(0).(*endpointapi.IndexEndpointResponse), args.Error(1)
}

// 测试用例
func TestService_CreateEndpoint_Success(t *testing.T) {
    // 准备
    mockRepo := &MockRepository{}
    mockWorkerQueue := &workerqueue.MockClient{}
    mockEndpointAPI := &MockEndpointAPIClient{}

    service := &service{
        repo:        mockRepo,
        workerQueue: mockWorkerQueue,
        newEndpointAPIClient: func(apiKey, url, endpointID string) endpointapi.Client {
            return mockEndpointAPI
        },
    }

    // 模拟成功的 ping 响应
    mockEndpointAPI.On("Ping", mock.Anything).Return(&endpointapi.PongResponse{
        OK: true,
    }, nil)

    // 模拟成功的端点创建
    expectedEndpoint := &Endpoint{
        ID:                     uuid.New(),
        Slug:                   "test-endpoint",
        URL:                    "https://test.com",
        IndexingHookIdentifier: "abc123",
        EnvironmentID:          uuid.New(),
        OrganizationID:         uuid.New(),
        ProjectID:              uuid.New(),
        CreatedAt:              time.Now(),
        UpdatedAt:              time.Now(),
    }
    mockRepo.On("UpsertEndpoint", mock.Anything, mock.Anything).Return(expectedEndpoint, nil)

    // 模拟成功的队列任务调度
    mockWorkerQueue.On("Enqueue", mock.Anything, "indexEndpoint", mock.Anything, mock.Anything).Return(nil)

    // 执行
    req := &CreateEndpointRequest{
        Environment: &apiauth.AuthenticatedEnvironment{
            ID:             expectedEndpoint.EnvironmentID,
            OrganizationID: expectedEndpoint.OrganizationID,
            ProjectID:      expectedEndpoint.ProjectID,
            APIKey:         "test-api-key",
        },
        URL: "https://test.com",
        ID:  "test-endpoint",
    }

    result, err := service.CreateEndpoint(context.Background(), req)

    // 验证
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, expectedEndpoint.ID, result.ID)
    assert.Equal(t, expectedEndpoint.Slug, result.Slug)
    assert.Equal(t, expectedEndpoint.URL, result.URL)

    // 验证模拟调用
    mockEndpointAPI.AssertExpectations(t)
    mockRepo.AssertExpectations(t)
    mockWorkerQueue.AssertExpectations(t)
}

func TestService_CreateEndpoint_PingFailed(t *testing.T) {
    // 模拟 ping 失败的情况
    mockEndpointAPI := &MockEndpointAPIClient{}
    mockEndpointAPI.On("Ping", mock.Anything).Return(&endpointapi.PongResponse{
        OK:    false,
        Error: "Connection refused",
    }, nil)

    service := &service{
        newEndpointAPIClient: func(apiKey, url, endpointID string) endpointapi.Client {
            return mockEndpointAPI
        },
    }

    req := &CreateEndpointRequest{
        Environment: &apiauth.AuthenticatedEnvironment{
            APIKey: "test-api-key",
        },
        URL: "https://invalid.com",
        ID:  "test-endpoint",
    }

    result, err := service.CreateEndpoint(context.Background(), req)

    // 验证错误处理
    assert.Error(t, err)
    assert.Nil(t, result)

    var createErr *CreateEndpointError
    assert.ErrorAs(t, err, &createErr)
    assert.Equal(t, CreateEndpointErrorFailedPing, createErr.Code)
    assert.Contains(t, createErr.Message, "Connection refused")
}
```

### 2. 集成测试设计

```go
// internal/services/endpoints/integration_test.go
package endpoints

import (
    "context"
    "database/sql"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "kongflow/backend/internal/database"
    "kongflow/backend/internal/services/apiauth"
    "kongflow/backend/internal/services/endpointapi"
    "kongflow/backend/internal/services/logger"
    "kongflow/backend/internal/services/ulid"
    "kongflow/backend/internal/services/workerqueue"
)

func TestService_Integration_CreateAndIndexEndpoint(t *testing.T) {
    // 准备测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // 创建测试 HTTP 服务器模拟端点
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.Header.Get("x-trigger-action") {
        case "PING":
            w.WriteHeader(200)
            w.Write([]byte(`{"ok":true}`))
        case "INDEX_ENDPOINT":
            w.WriteHeader(200)
            w.Write([]byte(`{
                "jobs": [
                    {"id": "job-1", "name": "Test Job", "enabled": true}
                ],
                "sources": [],
                "dynamicTriggers": [],
                "dynamicSchedules": []
            }`))
        }
    }))
    defer server.Close()

    // 创建服务依赖
    logger := logger.NewTestLogger()
    ulidGen := ulid.NewGenerator()
    workerQueue := workerqueue.NewTestClient()

    service := NewService(&Dependencies{
        DB:          db,
        Logger:      logger,
        ULID:        ulidGen,
        WorkerQueue: workerQueue,
        NewEndpointAPIClient: func(apiKey, url, endpointID string) endpointapi.Client {
            return endpointapi.NewClient(apiKey, url, endpointID, logger)
        },
    })

    // 创建测试环境
    env := createTestEnvironment(t, db)

    // 步骤 1: 创建端点
    createReq := &CreateEndpointRequest{
        Environment: env,
        URL:         server.URL,
        ID:          "test-endpoint",
    }

    createResult, err := service.CreateEndpoint(context.Background(), createReq)
    require.NoError(t, err)
    require.NotNil(t, createResult)

    assert.Equal(t, "test-endpoint", createResult.Slug)
    assert.Equal(t, server.URL, createResult.URL)
    assert.NotEmpty(t, createResult.IndexingHookIdentifier)

    // 验证队列任务被调度
    tasks := workerQueue.GetEnqueuedTasks("indexEndpoint")
    assert.Len(t, tasks, 1)

    // 步骤 2: 索引端点
    indexReq := &IndexEndpointRequest{
        ID:     createResult.ID,
        Source: EndpointIndexSourceInternal,
        Reason: stringPtr("Integration test"),
    }

    indexResult, err := service.IndexEndpoint(context.Background(), indexReq)
    require.NoError(t, err)
    require.NotNil(t, indexResult)

    assert.Equal(t, createResult.ID, indexResult.EndpointID)
    assert.Equal(t, 1, indexResult.Stats.Jobs)
    assert.Equal(t, 0, indexResult.Stats.Sources)

    // 验证注册任务被调度
    registerJobTasks := workerQueue.GetEnqueuedTasks("registerJob")
    assert.Len(t, registerJobTasks, 1)
}

// 测试辅助函数
func setupTestDB(t *testing.T) *sql.DB {
    // 设置测试数据库连接
    db, err := database.NewTestDB()
    require.NoError(t, err)

    // 运行迁移
    err = database.RunMigrations(db)
    require.NoError(t, err)

    return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
    db.Close()
}

func createTestEnvironment(t *testing.T, db *sql.DB) *apiauth.AuthenticatedEnvironment {
    // 创建测试组织、项目、环境
    // 返回完整的认证环境
    return &apiauth.AuthenticatedEnvironment{
        ID:             uuid.New(),
        OrganizationID: uuid.New(),
        ProjectID:      uuid.New(),
        APIKey:         "test-api-key",
    }
}

func stringPtr(s string) *string {
    return &s
}
```

## 📚 总结

这份技术设计详解确保了 endpoints 服务迁移的每个细节都与 trigger.dev 原版严格对齐，同时充分利用 Go 语言的优势和最佳实践。通过详细的代码示例、完整的测试策略和清晰的架构设计，我们为成功实施这个迁移项目提供了坚实的技术基础。
