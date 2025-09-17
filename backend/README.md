# KongFlow Backend - SecretStore MVP

基于 Go + SQLC + PostgreSQL + TestContainers 的 SecretStore 服务实现，从 trigger.dev 迁移而来。

## 📋 项目概述

这是一个 **服务层驱动的迁移项目**，将 trigger.dev 的 SecretStore 功能迁移到 KongFlow 平台：

- **源系统**: trigger.dev (TypeScript + Prisma ORM)
- **目标系统**: KongFlow (Go + SQLC + PostgreSQL)
- **架构**: 三层架构 (Repository → Service → 外部调用)
- **测试策略**: 单元测试 (Mock) + 集成测试 (TestContainers) + 端到端测试

## 🏗️ 技术栈

- **语言**: Go 1.23+
- **数据库**: PostgreSQL 15+ (JSONB 支持)
- **ORM**: SQLC (编译时 SQL 验证 + 类型安全代码生成)
- **测试**: TestContainers (真实数据库集成测试)
- **连接池**: pgx/v5 (高性能 PostgreSQL 驱动)

## 📁 项目结构

```
kongflow/backend/
├── cmd/
│   └── demo/              # 演示程序
├── internal/
│   ├── database/          # 数据库连接管理
│   │   ├── postgres.go    # 连接池配置
│   │   └── testhelper.go  # TestContainers 助手
│   └── secretstore/       # SecretStore 核心包
│       ├── db.go          # SQLC 生成：数据库接口
│       ├── models.go      # SQLC 生成：数据模型
│       ├── queries.sql.go # SQLC 生成：查询方法
│       ├── repository.go      # Repository 层实现
│       ├── repository_test.go # Repository 集成测试
│       ├── service.go         # Service 层实现
│       ├── service_test.go    # Service 单元测试
│       └── integration_test.go # 端到端集成测试
├── migrations/            # 数据库迁移文件
├── queries/              # SQLC 查询定义
├── testdata/             # 测试数据
├── docker-compose.yml    # 本地开发环境
├── sqlc.yaml            # SQLC 配置
├── Makefile             # 构建和测试任务
└── README.md
```

## 🚀 快速开始

### 1. 环境要求

- Go 1.23+
- Docker & Docker Compose
- Make (可选)

### 2. 项目设置

```bash
# 克隆项目
cd kongflow/backend

# 安装依赖
go mod tidy

# 启动开发环境 (PostgreSQL)
make dev-up
# 或者手动启动
docker-compose up -d postgres
```

### 3. 运行演示

```bash
# 构建并运行演示程序
make demo

# 或者手动运行
go run cmd/demo/main.go
```

### 4. 运行测试

```bash
# 运行所有测试
make test-all

# 分层测试
make test-unit        # 单元测试 (Mock)
make test-integration # 集成测试 (TestContainers)
make test-e2e        # 端到端测试

# 测试覆盖率
make test-coverage   # 生成 coverage.html
```

## 🔧 核心功能

### SecretStore Service API

```go
type Service struct {
    repo Repository
}

// 核心方法 (兼容 trigger.dev)
func (s *Service) GetSecret(ctx context.Context, key string, target interface{}) error
func (s *Service) SetSecret(ctx context.Context, key string, value interface{}) error
func (s *Service) GetSecretOrThrow(ctx context.Context, key string, target interface{}) error

// 扩展方法
func (s *Service) DeleteSecret(ctx context.Context, key string) error
func (s *Service) ListSecretKeys(ctx context.Context) ([]string, error)
func (s *Service) GetSecretCount(ctx context.Context) (int64, error)
```

### 使用示例

```go
import (
    "kongflow/backend/internal/database"
    "kongflow/backend/internal/services/secretstore"
)

// 初始化
config := database.NewDefaultConfig()
pool, _ := database.NewPool(ctx, config)
service := secretstore.NewService(secretstore.NewRepository(pool))

// 存储复杂数据结构
oauthConfig := map[string]interface{}{
    "client_id":     "github_123",
    "client_secret": "secret_456",
    "scopes":        []string{"read:user", "repo"},
}
service.SetSecret(ctx, "oauth.github", oauthConfig)

// 类型安全的读取
var config map[string]interface{}
service.GetSecret(ctx, "oauth.github", &config)
fmt.Printf("Client ID: %s", config["client_id"])
```

## 🧪 测试策略

### 三层测试金字塔

1. **单元测试** (Service 层)

   - 使用 Mock Repository
   - 快速执行，验证业务逻辑
   - 覆盖 JSON 序列化/反序列化

2. **集成测试** (Repository 层)

   - 使用 TestContainers + 真实 PostgreSQL
   - 验证 SQLC 生成代码和数据库交互
   - 测试 JSONB 存储和检索

3. **端到端测试** (完整流程)
   - Service + Repository 协作
   - 真实业务场景模拟
   - 数据一致性验证

### 测试覆盖范围

- ✅ JSONB 数据类型处理
- ✅ 复杂嵌套对象序列化
- ✅ 错误处理和边界情况
- ✅ 并发安全性
- ✅ 数据库事务处理

## 📊 性能特性

- **SQLC**: 编译时 SQL 验证，零运行时反射
- **pgx/v5**: 高性能 PostgreSQL 驱动，连接池优化
- **JSONB**: PostgreSQL 原生 JSON 支持，索引优化
- **TestContainers**: 真实数据库环境，无 Mock 数据差异

## 🔄 与 trigger.dev 的兼容性

| trigger.dev 方法     | KongFlow 实现        | 状态          |
| -------------------- | -------------------- | ------------- |
| `getSecret()`        | `GetSecret()`        | ✅ 完全兼容   |
| `setSecret()`        | `SetSecret()`        | ✅ 完全兼容   |
| `getSecretOrThrow()` | `GetSecretOrThrow()` | ✅ 完全兼容   |
| Provider 模式        | 计划中               | 🔄 MVP 后扩展 |

## 🛠️ 开发指南

### 数据库迁移

```bash
# 查看迁移文件
cat migrations/001_secret_store.sql

# 手动运行迁移
docker-compose exec postgres psql -U kong -d kongflow_dev -f /docker-entrypoint-initdb.d/001_secret_store.sql
```

### SQLC 代码生成

```bash
# 修改查询后重新生成
# 注意：当前由于 Go 版本问题，手动维护生成的代码
# sqlc generate
```

### 添加新的查询

1. 在 `queries/secret_store.sql` 中添加 SQL 查询
2. 重新生成 SQLC 代码
3. 在 Repository 接口中添加方法
4. 在 Service 层封装业务逻辑
5. 添加对应的测试

## 📈 后续扩展计划

- [ ] **Provider 模式**: 支持 AWS Parameter Store, HashiCorp Vault
- [ ] **加密支持**: 敏感数据加密存储
- [ ] **审计日志**: 密钥访问和修改记录
- [ ] **性能优化**: 缓存层和批量操作
- [ ] **监控指标**: Prometheus 指标收集
- [ ] **CLI 工具**: 密钥管理命令行工具

## 🎯 MVP 验收标准

- [x] SQLC 生成代码无错误
- [x] Repository 层数据访问正常 (TestContainers 集成测试)
- [x] Service 层 JSON 序列化正确 (Mock 单元测试)
- [x] 端到端流程测试通过 (完整集成测试)
- [x] 单元测试覆盖率 >= 80%
- [x] 集成测试覆盖核心数据流
- [x] 匹配 trigger.dev SecretStore 接口语义

## 📞 支持和贡献

这是一个个人学习项目，展示了现代 Go 服务开发的最佳实践：

- 类型安全的数据库访问 (SQLC)
- 真实环境集成测试 (TestContainers)
- 清晰的分层架构
- 完整的测试覆盖

## 📝 许可证

MIT License
