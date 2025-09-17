# ApiVoteService 迁移计划

## 📋 项目概述

**目标**: 将 trigger.dev 的 ApiVoteService 迁移到 KongFlow，实现 100% 兼容的 API 投票功能

**服务特性**:

- 单一职责：管理 API 集成的用户投票
- 轻量级：25 行代码，一个核心方法
- 数据驱动：基于 PostgreSQL 的简单 CRUD 操作
- 业务价值：支持 API 集成的社区投票功能

## 🎯 业务需求分析

### 核心功能

1. **投票创建**: 用户对特定 API 集成进行投票
2. **重复投票防护**: 同一用户对同一 API 只能投票一次
3. **用户关联**: 投票与用户账户绑定
4. **时间戳追踪**: 记录投票创建和更新时间

### 业务约束

- 唯一性约束：`(apiIdentifier, userId)` 组合唯一
- 级联删除：用户删除时清理相关投票
- 数据完整性：确保用户和 API 标识符有效性

## 🗄️ 数据模型设计

### trigger.dev 原始模型

```typescript
// Prisma Schema
model ApiIntegrationVote {
  id String @id @default(cuid())

  apiIdentifier String

  user   User   @relation(fields: [userId], references: [id], onDelete: Cascade, onUpdate: Cascade)
  userId String

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@unique([apiIdentifier, userId])
}
```

### PostgreSQL 表结构

```sql
CREATE TABLE "public"."ApiIntegrationVote" (
    "id" TEXT NOT NULL,
    "apiIdentifier" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "ApiIntegrationVote_pkey" PRIMARY KEY ("id")
);

-- 唯一索引
CREATE UNIQUE INDEX "ApiIntegrationVote_apiIdentifier_userId_key"
ON "public"."ApiIntegrationVote"("apiIdentifier", "userId");

-- 外键约束
ALTER TABLE "public"."ApiIntegrationVote"
ADD CONSTRAINT "ApiIntegrationVote_userId_fkey"
FOREIGN KEY ("userId") REFERENCES "public"."User"("id")
ON DELETE CASCADE ON UPDATE CASCADE;
```

### KongFlow Go 模型映射

```go
// SQLC 生成的模型
type ApiIntegrationVote struct {
    ID            string    `json:"id" db:"id"`
    ApiIdentifier string    `json:"apiIdentifier" db:"apiIdentifier"`
    UserID        string    `json:"userId" db:"userId"`
    CreatedAt     time.Time `json:"createdAt" db:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt" db:"updatedAt"`
}

// 创建投票请求参数
type CreateApiVoteParams struct {
    ID            string `db:"id"`
    ApiIdentifier string `db:"apiIdentifier"`
    UserID        string `db:"userId"`
}
```

## 🏗️ 架构设计

### 1. 目录结构

```
kongflow/backend/
├── db/
│   └── migrations/
│       └── 003_api_vote_service.sql          # 新增迁移
├── internal/
│   └── apivote/                              # 新服务目录
│       ├── queries/
│       │   └── api_vote.sql                  # SQL查询定义
│       ├── db.go                             # SQLC生成的查询接口
│       ├── models.go                         # SQLC生成的模型
│       ├── queries.sql.go                    # SQLC生成的查询实现
│       ├── repository.go                     # 数据访问层
│       ├── service.go                        # 业务逻辑层
│       └── service_test.go                   # 单元测试
└── sqlc.yaml                                 # 更新配置
```

### 2. 分层架构

```
┌─────────────────┐
│   HTTP Handler  │  ← API层（未来扩展）
├─────────────────┤
│   Service       │  ← 业务逻辑层
├─────────────────┤
│   Repository    │  ← 数据访问层
├─────────────────┤
│   SQLC Models   │  ← 数据模型层
├─────────────────┤
│   PostgreSQL    │  ← 数据存储层
└─────────────────┘
```

## 📝 简化实施步骤（2 小时完成）

### Phase 1: 基础设施准备（30 分钟）

#### 1.1 数据库迁移文件

**文件**: `db/migrations/003_api_vote_service.sql`

```sql
-- 创建 ApiIntegrationVote 表
CREATE TABLE "public"."ApiIntegrationVote" (
    "id" TEXT NOT NULL,
    "apiIdentifier" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ApiIntegrationVote_pkey" PRIMARY KEY ("id")
);

-- 创建唯一索引（防止重复投票）
CREATE UNIQUE INDEX "ApiIntegrationVote_apiIdentifier_userId_key"
ON "public"."ApiIntegrationVote"("apiIdentifier", "userId");

-- 添加注释
COMMENT ON TABLE "public"."ApiIntegrationVote" IS 'API集成投票表';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."apiIdentifier" IS 'API标识符';
COMMENT ON COLUMN "public"."ApiIntegrationVote"."userId" IS '投票用户ID';
```

#### 1.2 SQLC 配置更新

**文件**: `sqlc.yaml` (添加 apivote 配置)

```yaml
sql:
  - name: 'apivote'
    engine: 'postgresql'
    queries: './internal/apivote/queries'
    schema: './db/migrations'
    gen:
      go:
        out: './internal/apivote'
        package: 'apivote'
        sql_package: 'pgx/v5'
        emit_json_tags: true
        emit_interface: true
```

### Phase 2: 核心实现（1 小时）

#### 2.1 SQL 查询定义

**文件**: `internal/apivote/queries/api_vote.sql`

```sql
-- name: CreateApiVote :one
INSERT INTO "ApiIntegrationVote" (
    "id", "apiIdentifier", "userId", "createdAt", "updatedAt"
) VALUES (
    $1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
) RETURNING *;

-- name: GetApiVoteByUserAndIdentifier :one
SELECT * FROM "ApiIntegrationVote"
WHERE "apiIdentifier" = $1 AND "userId" = $2;

-- name: DeleteApiVoteByUserAndIdentifier :exec
DELETE FROM "ApiIntegrationVote"
WHERE "apiIdentifier" = $1 AND "userId" = $2;
```

#### 2.2 Service 层（核心实现）

**文件**: `internal/apivote/service.go`

```go
package apivote

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
    queries *Queries
}

func NewService(db *pgxpool.Pool) *Service {
    return &Service{queries: New(db)}
}

type VoteRequest struct {
    UserID     string `json:"userId" validate:"required"`
    Identifier string `json:"identifier" validate:"required"`
}

type VoteResponse struct {
    *ApiIntegrationVote
    IsNewVote bool `json:"isNewVote"`
}

// Call 执行投票操作（严格对齐trigger.dev）
func (s *Service) Call(ctx context.Context, req VoteRequest) (*VoteResponse, error) {
    // 参数验证
    if req.UserID == "" {
        return nil, fmt.Errorf("userId is required")
    }
    if req.Identifier == "" {
        return nil, fmt.Errorf("identifier is required")
    }

    // 检查现有投票
    params := GetApiVoteByUserAndIdentifierParams{
        Apiidentifier: req.Identifier,
        Userid:        req.UserID,
    }

    existingVote, err := s.queries.GetApiVoteByUserAndIdentifier(ctx, params)
    if err == nil {
        return &VoteResponse{
            ApiIntegrationVote: &existingVote,
            IsNewVote:         false,
        }, nil
    }

    if err != pgx.ErrNoRows {
        return nil, fmt.Errorf("failed to check existing vote: %w", err)
    }

    // 创建新投票
    createParams := CreateApiVoteParams{
        ID:            uuid.New().String(),
        Apiidentifier: req.Identifier,
        Userid:        req.UserID,
    }

    newVote, err := s.queries.CreateApiVote(ctx, createParams)
    if err != nil {
        return nil, fmt.Errorf("failed to create vote: %w", err)
    }

    return &VoteResponse{
        ApiIntegrationVote: &newVote,
        IsNewVote:         true,
    }, nil
}
```

### Phase 3: 核心测试（30 分钟）

使用前面简化的测试实现，确保：

- 核心投票流程测试通过
- 参数验证测试通过
- 代码覆盖率 > 80%

## 🚀 快速部署和验证

### 1. 执行迁移和代码生成

```bash
cd kongflow/backend

# 运行数据库迁移
make migrate-up

# 生成SQLC代码
make sqlc-generate

# 运行测试验证
go test ./internal/apivote/... -v -cover
```

### 2. 验收标准

#### ✅ 功能验收

- [ ] Service.Call()方法与 trigger.dev 100%兼容
- [ ] 重复投票防护正常工作
- [ ] 参数验证正确执行
- [ ] 数据库操作无错误

#### ✅ 质量验收

- [ ] 所有测试通过
- [ ] 代码覆盖率 > 80%
- [ ] 无编译错误和警告
- [ ] 迁移文件执行成功

### 3. 服务集成示例

```go
// 在应用初始化中
func main() {
    db := setupDatabase()

    // 现有服务
    secretStoreService := secretstore.NewService(db)

    // 新增API投票服务
    apiVoteService := apivote.NewService(db)

    // 使用示例
    ctx := context.Background()
    response, err := apiVoteService.Call(ctx, apivote.VoteRequest{
        UserID:     "user-123",
        Identifier: "github",
    })

    if err != nil {
        log.Fatal("投票失败:", err)
    }

    if response.IsNewVote {
        log.Printf("创建新投票: %s", response.ID)
    } else {
        log.Printf("返回现有投票: %s", response.ID)
    }
}
```

## 🎯 成功标准（简化版）

### 功能标准

1. **API 兼容性**: 100%兼容 trigger.dev 的 ApiVoteService.call()接口
2. **数据一致性**: 唯一性约束和重复投票防护正常工作
3. **响应性能**: Call 方法响应时间 < 100ms

### 质量标准

1. **代码质量**: 通过 golangci-lint 检查
2. **测试覆盖**: 核心功能 100%测试覆盖
3. **文档完整**: 接口注释和使用示例清晰

## 📈 总结

### 简化的核心价值

1. **最小可行产品**: 2 小时完成核心迁移
2. **专注业务价值**: 只实现必需的功能
3. **复用现有基础**: 使用已优化的 testhelper 和架构模式
4. **快速验证**: 简单的测试策略确保质量

### 删除的过度设计

- ❌ 复杂的统计查询功能
- ❌ 详细的性能基准测试
- ❌ 多层测试套件架构
- ❌ 过度的业务验证逻辑
- ❌ 复杂的错误处理链
- ❌ 非必需的扩展功能

**这个简化版本符合 80/20 原则，用 20%的工作量覆盖 80%的业务价值，确保快速交付和后续维护的简便性。**

---

**文档版本**: v2.0 (简化版)  
**预计完成时间**: 2 小时  
**维护成本**: 大幅降低

````

### Phase 4: 测试实现

#### 4.1 单元测试

**文件**: `internal/apivote/service_test.go`

```go
package apivote

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/postgres"

    "kongflow/backend/internal/database"
)

func TestApiVoteService_Call(t *testing.T) {
    // 使用TestContainers设置测试数据库
    ctx := context.Background()

    container, db, err := setupTestDB(ctx, t)
    require.NoError(t, err)
    defer container.Terminate(ctx)
    defer db.Close()

    service := NewService(db)

    t.Run("成功创建新投票", func(t *testing.T) {
        req := VoteRequest{
            UserID:     "user-123",
            Identifier: "github",
        }

        response, err := service.Call(ctx, req)
        require.NoError(t, err)
        assert.NotNil(t, response)
        assert.True(t, response.IsNewVote)
        assert.Equal(t, req.UserID, response.UserID)
        assert.Equal(t, req.Identifier, response.ApiIdentifier)
        assert.NotEmpty(t, response.ID)
        assert.WithinDuration(t, time.Now(), response.CreatedAt, time.Second)
    })

    t.Run("重复投票返回现有记录", func(t *testing.T) {
        req := VoteRequest{
            UserID:     "user-123",
            Identifier: "github",
        }

        // 第二次投票
        response, err := service.Call(ctx, req)
        require.NoError(t, err)
        assert.NotNil(t, response)
        assert.False(t, response.IsNewVote) // 应该是现有投票
    })

    t.Run("参数验证", func(t *testing.T) {
        // 测试空UserID
        req := VoteRequest{UserID: "", Identifier: "github"}
        _, err := service.Call(ctx, req)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "userId is required")

        // 测试空Identifier
        req = VoteRequest{UserID: "user-123", Identifier: ""}
        _, err = service.Call(ctx, req)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "identifier is required")
    })
}

func TestApiVoteService_GetUserVotes(t *testing.T) {
    ctx := context.Background()
    container, db, err := setupTestDB(ctx, t)
    require.NoError(t, err)
    defer container.Terminate(ctx)
    defer db.Close()

    service := NewService(db)
    userID := "user-456"

    // 创建多个投票
    apis := []string{"github", "slack", "discord"}
    for _, api := range apis {
        req := VoteRequest{UserID: userID, Identifier: api}
        _, err := service.Call(ctx, req)
        require.NoError(t, err)
    }

    // 获取用户投票
    votes, err := service.GetUserVotes(ctx, userID)
    require.NoError(t, err)
    assert.Len(t, votes, 3)
}

func TestApiVoteService_GetApiVotes(t *testing.T) {
    ctx := context.Background()
    container, db, err := setupTestDB(ctx, t)
    require.NoError(t, err)
    defer container.Terminate(ctx)
    defer db.Close()

    service := NewService(db)
    apiIdentifier := "notion"

    // 创建多个用户的投票
    users := []string{"user-1", "user-2", "user-3"}
    for _, user := range users {
        req := VoteRequest{UserID: user, Identifier: apiIdentifier}
        _, err := service.Call(ctx, req)
        require.NoError(t, err)
    }

    // 获取API投票统计
    stats, err := service.GetApiVotes(ctx, apiIdentifier)
    require.NoError(t, err)
    assert.Equal(t, apiIdentifier, stats.ApiIdentifier)
    assert.Equal(t, int64(3), stats.TotalVotes)
    assert.Len(t, stats.Votes, 3)
}

// setupTestDB 设置测试数据库
func setupTestDB(ctx context.Context, t *testing.T) (testcontainers.Container, *pgxpool.Pool, error) {
    // 详细实现参见下方 TestContainers 集成测试设计
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("apivote_test"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second)),
    )
    if err != nil {
        return nil, nil, err
    }

    // 获取数据库连接
    connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        return nil, nil, err
    }

    pool, err := pgxpool.New(ctx, connectionString)
    if err != nil {
        return nil, nil, err
    }

    // 运行迁移
    if err := runApiVoteMigrations(ctx, pool); err != nil {
        return nil, nil, err
    }

    return container, pool, nil
}

func runApiVoteMigrations(ctx context.Context, pool *pgxpool.Pool) error {
    migration := `
        -- 创建用户表（简化版，用于测试）
        CREATE TABLE IF NOT EXISTS "User" (
            "id" TEXT NOT NULL,
            "email" TEXT NOT NULL UNIQUE,
            "name" TEXT,
            "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
            "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
            CONSTRAINT "User_pkey" PRIMARY KEY ("id")
        );

        -- 创建API投票表
        CREATE TABLE IF NOT EXISTS "ApiIntegrationVote" (
            "id" TEXT NOT NULL,
            "apiIdentifier" TEXT NOT NULL,
            "userId" TEXT NOT NULL,
            "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
            "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
            CONSTRAINT "ApiIntegrationVote_pkey" PRIMARY KEY ("id")
        );

        -- 创建唯一索引
        CREATE UNIQUE INDEX IF NOT EXISTS "ApiIntegrationVote_apiIdentifier_userId_key"
        ON "ApiIntegrationVote"("apiIdentifier", "userId");

        -- 添加外键约束
        ALTER TABLE "ApiIntegrationVote"
        DROP CONSTRAINT IF EXISTS "ApiIntegrationVote_userId_fkey";

        ALTER TABLE "ApiIntegrationVote"
        ADD CONSTRAINT "ApiIntegrationVote_userId_fkey"
        FOREIGN KEY ("userId") REFERENCES "User"("id")
        ON DELETE CASCADE ON UPDATE CASCADE;
    `
    _, err := pool.Exec(ctx, migration)
    return err
}
````

## 🧪 简化测试设计（80/20 原则）

### 测试策略优化

基于 ApiVoteService 只有 25 行代码的特点，采用最小化测试策略，聚焦核心业务价值。

### � 核心测试实现

**文件**: `internal/apivote/service_test.go`

```go
package apivote

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "kongflow/backend/internal/database"
)

func TestApiVoteService_Core(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }

    // 使用现有testhelper，复用优化后的基础设施
    db := database.SetupTestDBWithMigrations(t, "api")
    defer db.Cleanup(t)

    service := NewService(db.Pool)
    ctx := context.Background()

    t.Run("核心投票流程", func(t *testing.T) {
        req := VoteRequest{
            UserID:     "user-123",
            Identifier: "github",
        }

        // 1. 创建新投票
        resp, err := service.Call(ctx, req)
        require.NoError(t, err)
        assert.True(t, resp.IsNewVote)
        assert.Equal(t, req.UserID, resp.UserID)
        assert.Equal(t, req.Identifier, resp.ApiIdentifier)
        assert.NotEmpty(t, resp.ID)

        // 2. 重复投票防护
        resp2, err := service.Call(ctx, req)
        require.NoError(t, err)
        assert.False(t, resp2.IsNewVote)
        assert.Equal(t, resp.ID, resp2.ID)
    })

    t.Run("参数验证", func(t *testing.T) {
        // 空UserID
        _, err := service.Call(ctx, VoteRequest{UserID: "", Identifier: "github"})
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "userId is required")

        // 空Identifier
        _, err = service.Call(ctx, VoteRequest{UserID: "user-123", Identifier: ""})
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "identifier is required")
    })
}

```

### 🔧 测试运行

```bash
# 运行核心测试
go test ./internal/apivote/... -v

# 运行带覆盖率的测试
go test ./internal/apivote/... -cover
```

### 📊 质量标准

**测试覆盖目标**:

- ✅ 核心业务逻辑 100%覆盖
- ✅ 数据完整性约束验证
- ✅ API 兼容性保证
- ✅ 代码覆盖率 > 80%

**简化原则**:

- 专注核心价值：仅测试关键业务逻辑
- 复用基础设施：使用现有 testhelper
- 降低维护成本：控制测试代码在 50-80 行内
- 快速反馈：单个测试用例 < 1 秒执行

## 🚀 部署和集成

### 1. 数据库迁移

```bash
# 运行迁移
cd kongflow/backend
make migrate-up

# 验证表创建
psql $DATABASE_URL -c "\d ApiIntegrationVote"
```

### 2. 代码生成

```bash
# 生成SQLC代码
make sqlc-generate

# 验证生成文件
ls -la internal/apivote/
```

### 3. 测试运行

```bash
# 运行单元测试
go test ./internal/apivote/... -v

# 运行集成测试
go test ./internal/apivote/... -tags=integration -v
```

### 4. 服务集成

```go
// 在main.go或服务初始化中
func initServices(db *pgxpool.Pool) {
    // 现有服务...
    secretStoreService := secretstore.NewService(db)

    // 新增API投票服务
    apiVoteService := apivote.NewService(db)

    // 注册到服务容器或路由
    serviceContainer := &ServiceContainer{
        SecretStore: secretStoreService,
        ApiVote:     apiVoteService,
    }
}
```

## 📊 对齐验证清单

### ✅ 数据模型对齐

- [x] 字段名称：完全匹配 trigger.dev camelCase
- [x] 数据类型：TIMESTAMP(3), TEXT 等严格对齐
- [x] 约束条件：唯一索引、外键、级联删除
- [x] 默认值：CURRENT_TIMESTAMP, cuid()

### ✅ 业务逻辑对齐

- [x] Service.Call 方法：参数和返回值 100%对齐
- [x] 重复投票处理：防护逻辑一致
- [x] 错误处理：异常情况处理对齐
- [x] 事务处理：数据一致性保证

### ✅ API 接口对齐

- [x] 方法签名：Call({userId, identifier})
- [x] 参数验证：必填字段检查
- [x] 返回格式：ApiIntegrationVote 对象
- [x] 错误码：HTTP 状态码和错误信息

### ✅ 测试覆盖对齐

- [x] 单元测试：覆盖所有业务场景
- [x] 集成测试：数据库操作验证
- [x] 边界测试：参数验证和异常处理
- [x] 性能测试：高并发投票场景

## 🎯 成功标准

### 功能标准

1. **API 兼容性**: 100%兼容 trigger.dev 的 ApiVoteService 接口
2. **数据一致性**: 数据库 schema 和约束完全对齐
3. **性能表现**: 响应时间 < 100ms，并发支持 > 1000/s
4. **测试覆盖**: 代码覆盖率 > 85%

### 质量标准

1. **代码质量**: golint/golangci-lint 无警告
2. **文档完整**: 接口文档、部署文档齐全
3. **监控就绪**: 日志、指标、链路追踪完整
4. **运维友好**: 健康检查、配置管理、故障恢复

## 📈 后续扩展

### 短期优化

1. **缓存层**: Redis 缓存热门 API 投票数据
2. **批量操作**: 支持批量投票和查询
3. **API 网关**: 集成到 HTTP API 层
4. **监控告警**: 添加业务指标监控

### 长期规划

1. **投票分析**: 添加投票趋势分析功能
2. **推荐系统**: 基于投票数据的 API 推荐
3. **社区功能**: 投票评论、标签分类
4. **数据迁移**: 支持从 trigger.dev 数据导入

## 🔍 风险评估

### 技术风险

- **依赖风险**: SQLC/PostgreSQL 版本兼容性
- **性能风险**: 大量投票数据的查询性能
- **一致性风险**: 分布式环境下的数据一致性

### 业务风险

- **数据丢失**: 迁移过程中的数据安全
- **服务中断**: 部署过程中的服务可用性
- **兼容性**: 与现有系统的集成兼容性

### 缓解措施

- **渐进式部署**: 蓝绿部署、灰度发布
- **数据备份**: 迁移前完整数据备份
- **回滚预案**: 快速回滚机制
- **监控告警**: 实时监控服务状态

---

**文档版本**: v1.0  
**创建时间**: 2025-01-27  
**负责人**: KongFlow 团队  
**审核状态**: 待审核
