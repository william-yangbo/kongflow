# SecretStore MVP 迁移

## 目标

用 Go + SQLC + PostgreSQL 实现 trigger.dev 的 SecretStore 基本功能

## SQLC 集成评估

### SQLC 特点

- **编译时验证**：SQL 查询在编译时检查，避免运行时错误
- **类型安全**：自动生成强类型 Go 代码
- **JSONB 支持**：PostgreSQL JSONB 类型映射为 `[]byte`，需要手动序列化
- **接口生成**：可生成 `Querier` 接口，便于测试 mock
- **事务支持**：生成 `WithTx()` 方法支持事务

### 最佳实践

- **分离关注点**：SQLC 负责数据访问，Service 层处理业务逻辑
- **JSON 处理**：Service 层封装 JSON 序列化/反序列化
- **错误处理**：Service 层包装数据库错误，提供业务语义
- **测试策略**：Repository 用真实数据库测试，Service 层 mock Repository

## MVP 范围

- 数据表：key-value 存储 (JSONB)
- **内部服务**：Go package，不需要 HTTP API
- 功能：GetSecret/SetSecret 方法
- 测试：基本功能验证

## 项目目录结构

```
kongflow/backend/
├── cmd/
│   └── migrate/           # 数据库迁移工具
│       └── main.go
├── internal/
│   ├── database/          # 数据库连接管理
│   │   ├── postgres.go
│   │   └── testhelper.go  # TestContainers 测试助手
│   └── secretstore/       # SecretStore 服务包
│       ├── db.go          # SQLC 生成：数据库接口
│       ├── models.go      # SQLC 生成：数据模型
│       ├── queries.sql.go # SQLC 生成：查询方法
│       ├── repository.go      # Repository 接口和实现
│       ├── repository_test.go # Repository 集成测试 (TestContainers)
│       ├── service.go         # 业务逻辑层
│       └── service_test.go    # Service 单元测试 (Mock)
├── migrations/
│   └── 001_secret_store.sql # 数据库 schema
├── queries/
│   └── secret_store.sql   # SQLC 查询定义
├── testdata/              # 测试数据
│   └── secrets.json
├── go.mod
├── go.sum
├── sqlc.yaml             # SQLC 配置
└── docker-compose.yml    # 本地 PostgreSQL
```

**测试策略：**

- **单元测试** (Service 层)：使用 Mock Repository，快速验证业务逻辑
- **集成测试** (Repository 层)：使用 TestContainers + 真实 PostgreSQL，验证数据访问
- **端到端测试**：Repository + Service 完整流程，使用 TestContainers

**设计原则：**

- `internal/` - 私有包，符合 Go 约定
- **三层架构**：Repository (数据访问) → Service (业务逻辑) → 外部调用
- 生成的代码和手写代码分离
- 测试文件就近放置，按类型分层测试

## 实施步骤 (总计：3.5 小时)

### 1. 项目设置 (30 分钟)

```bash
cd kongflow/backend
go mod init kongflow/backend
# 核心依赖
go get github.com/jackc/pgx/v5/pgxpool
# 测试依赖
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go get github.com/stretchr/testify/suite
# TestContainers 集成测试
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/wait
# SQLC 安装
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### 2. 数据库 (30 分钟)

```sql
-- migrations/001_secret_store.sql
CREATE TABLE secret_store (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

```yaml
# sqlc.yaml
version: '2'
sql:
  - engine: 'postgresql'
    queries: './queries'
    schema: './migrations'
    gen:
      go:
        out: './internal/secretstore'
        package: 'secretstore'
        sql_package: 'pgx/v5'
        emit_json_tags: true
        emit_interface: true
        emit_prepared_queries: false
        emit_exact_table_names: true
```

````sql
-- queries/secret_store.sql
-- name: GetSecretStore :one
SELECT * FROM secret_store WHERE key = $1;

-- name: UpsertSecretStore :exec
INSERT INTO secret_store (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = NOW();

-- name: DeleteSecretStore :exec
DELETE FROM secret_store WHERE key = $1;
```### 3. Go 代码 (1.5 小时)

```go
// internal/database/postgres.go
package database

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
    Host     string
    Port     int
    User     string
    Password string
    Database string
    SSLMode  string
}

func (c Config) DSN() string {
    return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
        c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
    return pgxpool.New(ctx, config.DSN())
}
```

```go
// internal/database/testhelper.go
package database

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
    Container testcontainers.Container
    Pool      *pgxpool.Pool
    Config    Config
}

func SetupTestDB(t *testing.T) *TestDB {
    ctx := context.Background()

    // 启动 PostgreSQL TestContainer
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second)),
    )
    require.NoError(t, err)

    // 获取连接信息
    host, err := container.Host(ctx)
    require.NoError(t, err)

    port, err := container.MappedPort(ctx, "5432")
    require.NoError(t, err)

    config := Config{
        Host:     host,
        Port:     port.Int(),
        User:     "testuser",
        Password: "testpass",
        Database: "testdb",
        SSLMode:  "disable",
    }

    // 创建连接池
    pool, err := NewPool(ctx, config)
    require.NoError(t, err)

    // 运行迁移
    err = runMigrations(ctx, pool)
    require.NoError(t, err)

    return &TestDB{
        Container: container,
        Pool:      pool,
        Config:    config,
    }
}

func (db *TestDB) Cleanup(t *testing.T) {
    ctx := context.Background()
    if db.Pool != nil {
        db.Pool.Close()
    }
    if db.Container != nil {
        require.NoError(t, db.Container.Terminate(ctx))
    }
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
    migration := `
        CREATE TABLE IF NOT EXISTS secret_store (
            key TEXT PRIMARY KEY,
            value JSONB NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
    `
    _, err := pool.Exec(ctx, migration)
    return err
}
```

```go
// internal/secretstore/repository.go
package secretstore

import (
    "context"
    "errors"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var ErrSecretNotFound = errors.New("secret not found")

type Repository interface {
    GetSecret(ctx context.Context, key string) (*SecretStore, error)
    UpsertSecret(ctx context.Context, key string, value []byte) error
    DeleteSecret(ctx context.Context, key string) error
}

type repository struct {
    queries *Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
    return &repository{
        queries: New(db),
    }
}

func (r *repository) GetSecret(ctx context.Context, key string) (*SecretStore, error) {
    secret, err := r.queries.GetSecretStore(ctx, key)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrSecretNotFound
        }
        return nil, err
    }
    return &secret, nil
}

func (r *repository) UpsertSecret(ctx context.Context, key string, value []byte) error {
    return r.queries.UpsertSecretStore(ctx, UpsertSecretStoreParams{
        Key:   key,
        Value: value,
    })
}

func (r *repository) DeleteSecret(ctx context.Context, key string) error {
    return r.queries.DeleteSecretStore(ctx, key)
}
````

```go
// internal/secretstore/service.go
package secretstore

import (
    "context"
    "encoding/json"
    "fmt"
)

type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

// GetSecret 获取密钥并反序列化到 target
func (s *Service) GetSecret(ctx context.Context, key string, target interface{}) error {
    secret, err := s.repo.GetSecret(ctx, key)
    if err != nil {
        return fmt.Errorf("failed to get secret %s: %w", key, err)
    }

    if err := json.Unmarshal(secret.Value, target); err != nil {
        return fmt.Errorf("failed to unmarshal secret %s: %w", key, err)
    }

    return nil
}

// SetSecret 序列化 value 并存储
func (s *Service) SetSecret(ctx context.Context, key string, value interface{}) error {
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
    }

    if err := s.repo.UpsertSecret(ctx, key, data); err != nil {
        return fmt.Errorf("failed to set secret %s: %w", key, err)
    }

    return nil
}

// GetSecretOrThrow 如果不存在则返回错误
func (s *Service) GetSecretOrThrow(ctx context.Context, key string, target interface{}) error {
    if err := s.GetSecret(ctx, key, target); err != nil {
        if errors.Is(err, ErrSecretNotFound) {
            return fmt.Errorf("secret %s not found", key)
        }
        return err
    }
    return nil
}
```

````yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: kongflow_dev
      POSTGRES_USER: kong
      POSTGRES_PASSWORD: flow2025
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
volumes:
  postgres_data:
```### 4. 测试 (30 分钟)

```go
````

### 4. 测试 (1 小时)

#### 4.1 Repository 集成测试 (TestContainers)

```go
// internal/secretstore/repository_test.go
package secretstore

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"

    "kongflow/backend/internal/database"
)

type RepositoryTestSuite struct {
    suite.Suite
    db   *database.TestDB
    repo Repository
}

func (suite *RepositoryTestSuite) SetupSuite() {
    suite.db = database.SetupTestDB(suite.T())
    suite.repo = NewRepository(suite.db.Pool)
}

func (suite *RepositoryTestSuite) TearDownSuite() {
    suite.db.Cleanup(suite.T())
}

func (suite *RepositoryTestSuite) SetupTest() {
    // 清理测试数据
    _, err := suite.db.Pool.Exec(context.Background(), "DELETE FROM secret_store")
    require.NoError(suite.T(), err)
}

func (suite *RepositoryTestSuite) TestUpsertAndGetSecret() {
    ctx := context.Background()

    // 准备测试数据
    testKey := "integration-test-key"
    testValue := map[string]interface{}{
        "api_key": "secret-123",
        "config":  map[string]string{"env": "test"},
    }
    valueBytes, err := json.Marshal(testValue)
    require.NoError(suite.T(), err)

    // 测试插入
    err = suite.repo.UpsertSecret(ctx, testKey, valueBytes)
    assert.NoError(suite.T(), err)

    // 测试获取
    secret, err := suite.repo.GetSecret(ctx, testKey)
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), testKey, secret.Key)
    assert.JSONEq(suite.T(), string(valueBytes), string(secret.Value))
    assert.NotZero(suite.T(), secret.CreatedAt)
    assert.NotZero(suite.T(), secret.UpdatedAt)
}

func (suite *RepositoryTestSuite) TestUpsertUpdateExisting() {
    ctx := context.Background()
    testKey := "update-test-key"

    // 插入初始值
    initialValue, _ := json.Marshal(map[string]string{"version": "1"})
    err := suite.repo.UpsertSecret(ctx, testKey, initialValue)
    require.NoError(suite.T(), err)

    // 更新值
    updatedValue, _ := json.Marshal(map[string]string{"version": "2"})
    err = suite.repo.UpsertSecret(ctx, testKey, updatedValue)
    assert.NoError(suite.T(), err)

    // 验证更新
    secret, err := suite.repo.GetSecret(ctx, testKey)
    assert.NoError(suite.T(), err)
    assert.JSONEq(suite.T(), string(updatedValue), string(secret.Value))
}

func (suite *RepositoryTestSuite) TestGetSecretNotFound() {
    ctx := context.Background()

    secret, err := suite.repo.GetSecret(ctx, "nonexistent-key")
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, ErrSecretNotFound)
    assert.Nil(suite.T(), secret)
}

func (suite *RepositoryTestSuite) TestDeleteSecret() {
    ctx := context.Background()
    testKey := "delete-test-key"

    // 插入数据
    testValue, _ := json.Marshal(map[string]string{"temp": "data"})
    err := suite.repo.UpsertSecret(ctx, testKey, testValue)
    require.NoError(suite.T(), err)

    // 删除数据
    err = suite.repo.DeleteSecret(ctx, testKey)
    assert.NoError(suite.T(), err)

    // 验证删除
    secret, err := suite.repo.GetSecret(ctx, testKey)
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, ErrSecretNotFound)
    assert.Nil(suite.T(), secret)
}

func TestRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}
```

#### 4.2 Service 单元测试 (Mock)

```go
// internal/secretstore/service_test.go
package secretstore

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// MockRepository 实现 Repository 接口的 mock
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) GetSecret(ctx context.Context, key string) (*SecretStore, error) {
    args := m.Called(ctx, key)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*SecretStore), args.Error(1)
}

func (m *MockRepository) UpsertSecret(ctx context.Context, key string, value []byte) error {
    args := m.Called(ctx, key, value)
    return args.Error(0)
}

func (m *MockRepository) DeleteSecret(ctx context.Context, key string) error {
    args := m.Called(ctx, key)
    return args.Error(0)
}

func TestService_SetAndGetSecret(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)
    ctx := context.Background()

    // 准备测试数据
    testKey := "test-key"
    testValue := map[string]string{"hello": "world"}
    expectedJSON := `{"hello":"world"}`

    // Mock repository 行为
    mockRepo.On("UpsertSecret", ctx, testKey, []byte(expectedJSON)).Return(nil)
    mockRepo.On("GetSecret", ctx, testKey).Return(&SecretStore{
        Key:   testKey,
        Value: []byte(expectedJSON),
    }, nil)

    // 测试 SetSecret
    err := service.SetSecret(ctx, testKey, testValue)
    assert.NoError(t, err)

    // 测试 GetSecret
    var result map[string]string
    err = service.GetSecret(ctx, testKey, &result)
    assert.NoError(t, err)
    assert.Equal(t, testValue, result)

    // 验证 mock 调用
    mockRepo.AssertExpectations(t)
}

func TestService_GetSecretNotFound(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)
    ctx := context.Background()

    // Mock repository 返回 not found 错误
    mockRepo.On("GetSecret", ctx, "nonexistent").Return(nil, ErrSecretNotFound)

    var result map[string]string
    err := service.GetSecret(ctx, "nonexistent", &result)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to get secret")
}

func TestService_InvalidJSON(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)
    ctx := context.Background()

    // 测试无效的 JSON 数据
    mockRepo.On("GetSecret", ctx, "invalid-json").Return(&SecretStore{
        Key:   "invalid-json",
        Value: []byte(`{invalid json`),
    }, nil)

    var result map[string]string
    err := service.GetSecret(ctx, "invalid-json", &result)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to unmarshal")
}
```

#### 4.3 端到端集成测试

```go
// internal/secretstore/integration_test.go
package secretstore

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"

    "kongflow/backend/internal/database"
)

type IntegrationTestSuite struct {
    suite.Suite
    db      *database.TestDB
    service *Service
}

func (suite *IntegrationTestSuite) SetupSuite() {
    suite.db = database.SetupTestDB(suite.T())
    repo := NewRepository(suite.db.Pool)
    suite.service = NewService(repo)
}

func (suite *IntegrationTestSuite) TearDownSuite() {
    suite.db.Cleanup(suite.T())
}

func (suite *IntegrationTestSuite) SetupTest() {
    // 清理测试数据
    _, err := suite.db.Pool.Exec(context.Background(), "DELETE FROM secret_store")
    require.NoError(suite.T(), err)
}

func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
    ctx := context.Background()

    // 测试数据
    secretKey := "oauth.github.client"
    secretData := map[string]interface{}{
        "client_id":     "github_client_123",
        "client_secret": "github_secret_456",
        "scopes":        []string{"read:user", "repo"},
        "metadata": map[string]string{
            "provider": "github",
            "env":      "test",
        },
    }

    // 1. 设置 Secret
    err := suite.service.SetSecret(ctx, secretKey, secretData)
    assert.NoError(suite.T(), err)

    // 2. 获取 Secret
    var retrieved map[string]interface{}
    err = suite.service.GetSecret(ctx, secretKey, &retrieved)
    assert.NoError(suite.T(), err)

    // 验证数据完整性
    assert.Equal(suite.T(), secretData["client_id"], retrieved["client_id"])
    assert.Equal(suite.T(), secretData["client_secret"], retrieved["client_secret"])

    // 验证数组类型
    scopes, ok := retrieved["scopes"].([]interface{})
    assert.True(suite.T(), ok)
    assert.Len(suite.T(), scopes, 2)

    // 3. 更新 Secret
    updatedData := map[string]interface{}{
        "client_id":     "github_client_789",
        "client_secret": "github_secret_updated",
        "scopes":        []string{"read:user", "repo", "admin:org"},
    }

    err = suite.service.SetSecret(ctx, secretKey, updatedData)
    assert.NoError(suite.T(), err)

    // 4. 验证更新
    var updated map[string]interface{}
    err = suite.service.GetSecret(ctx, secretKey, &updated)
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), "github_client_789", updated["client_id"])

    updatedScopes, ok := updated["scopes"].([]interface{})
    assert.True(suite.T(), ok)
    assert.Len(suite.T(), updatedScopes, 3)

    // 5. 测试不存在的 Secret
    var notFound map[string]interface{}
    err = suite.service.GetSecret(ctx, "nonexistent.key", &notFound)
    assert.Error(suite.T(), err)
    assert.Contains(suite.T(), err.Error(), "failed to get secret")
}

func TestIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(IntegrationTestSuite))
}
```

#### 4.4 测试数据文件

```json
// testdata/secrets.json
{
  "oauth.github": {
    "client_id": "test_github_client",
    "client_secret": "test_github_secret",
    "scopes": ["read:user", "repo"]
  },
  "database.postgres": {
    "host": "localhost",
    "port": 5432,
    "username": "testuser",
    "password": "testpass",
    "database": "testdb",
    "ssl_mode": "disable"
  },
  "api.external": {
    "base_url": "https://api.example.com",
    "timeout": 30,
    "retry_count": 3,
    "headers": {
      "User-Agent": "KongFlow/1.0"
    }
  }
}
```

#### 4.5 运行测试

```bash
# 环境准备和测试

# 1. 生成 SQLC 代码
sqlc generate

# 2. 运行单元测试（Service 层，快速）
go test ./internal/secretstore -run "TestService_" -v

# 3. 运行集成测试（Repository 层，使用 TestContainers）
go test ./internal/secretstore -run "TestRepositoryTestSuite" -v

# 4. 运行端到端测试（完整流程）
go test ./internal/secretstore -run "TestIntegrationTestSuite" -v

# 5. 运行所有测试
go test ./internal/secretstore -v

# 6. 生成覆盖率报告
go test ./internal/secretstore -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 7. 基准测试
go test ./internal/secretstore -bench=. -benchmem
```

## 完成标准

- [ ] SQLC 生成代码无错误
- [ ] Repository 层数据访问正常 (TestContainers 集成测试)
- [ ] Service 层 JSON 序列化正确 (Mock 单元测试)
- [ ] 端到端流程测试通过 (完整集成测试)
- [ ] 单元测试覆盖率 >= 80%
- [ ] 集成测试覆盖核心数据流
- [ ] 匹配 trigger.dev SecretStore 接口语义

**测试覆盖范围：**

1. **单元测试** (快速反馈)

   - Service 层业务逻辑
   - JSON 序列化/反序列化
   - 错误处理

2. **集成测试** (真实环境)

   - Repository 层数据访问
   - PostgreSQL JSONB 操作
   - SQLC 生成代码验证

3. **端到端测试** (完整流程)
   - Service + Repository 协作
   - 复杂数据结构处理
   - 真实使用场景模拟

**MVP 后续扩展：**

- 性能基准测试 (Benchmark)
- 错误处理优化和测试
- 并发安全性测试
- Provider 模式支持 (AWS Param Store)
- 监控和指标收集
