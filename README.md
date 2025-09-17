# KongFlow

KongFlow 是一个基于 Go 的工作流引擎，提供高性能的任务调度和管理功能。

## 项目结构

```
kongflow/
├── backend/                    # Go 后端服务
│   ├── cmd/                   # 命令行程序
│   │   └── demo/              # 演示程序
│   ├── internal/              # 内部包
│   │   ├── database/          # 数据库连接和配置
│   │   └── secretstore/       # 密钥存储服务
│   ├── db/                    # SQLC 生成的数据库代码
│   ├── migrations/            # 数据库迁移文件
│   ├── go.mod                 # Go 模块依赖
│   ├── go.sum                 # Go 模块校验
│   ├── sqlc.yaml              # SQLC 配置
│   ├── Makefile               # 构建脚本
│   └── README.md              # 后端文档
└── README.md                  # 项目主文档
```

## 核心功能

### SecretStore MVP

- **类型安全的密钥存储**: 使用 PostgreSQL JSONB 字段存储结构化数据
- **兼容 trigger.dev 接口**: 提供 `SetSecret`, `GetSecret`, `GetSecretOrThrow` 等方法
- **三层架构设计**: Repository (数据访问) → Service (业务逻辑) → API
- **TestContainers 集成测试**: 真实数据库环境的端到端测试

## 技术栈

- **后端**: Go 1.25+
- **数据库**: PostgreSQL 15+ with JSONB
- **ORM**: SQLC (类型安全的 SQL 代码生成)
- **测试**: TestContainers for Go
- **容器**: Docker & Docker Compose

## 快速开始

### 1. 安装依赖

```bash
cd backend
go mod download
```

### 2. 运行演示

```bash
# 自动启动 PostgreSQL 容器并运行演示
./run-demo.sh
```

### 3. 运行测试

```bash
# 运行所有测试（包括集成测试）
go test ./... -v

# 只运行单元测试
go test ./internal/secretstore -v
```

## 开发指南

### 数据库操作

项目使用 SQLC 生成类型安全的数据库访问代码：

```bash
# 生成数据库代码
make sqlc-generate

# 运行数据库迁移
make migrate-up
```

### 测试策略

- **单元测试**: 使用 Mock 测试业务逻辑
- **集成测试**: 使用 TestContainers 测试数据库交互
- **端到端测试**: 完整的工作流测试

### 示例用法

```go
// 创建 SecretStore 服务
repo := secretstore.NewRepository(pool)
service := secretstore.NewService(repo)

// 存储密钥
err := service.SetSecret(ctx, "oauth.github", map[string]interface{}{
    "client_id": "github_client_123",
    "client_secret": "github_secret_456",
})

// 读取密钥
var config struct {
    ClientID     string `json:"client_id"`
    ClientSecret string `json:"client_secret"`
}
err := service.GetSecret(ctx, "oauth.github", &config)
```

## 架构设计

### 服务层驱动的迁移方式

KongFlow 采用服务层驱动的架构，便于从 trigger.dev 迁移：

1. **接口兼容**: 保持与 trigger.dev SecretStore 相同的 API 语义
2. **数据模型**: 使用 JSONB 支持灵活的数据结构
3. **类型安全**: 通过泛型和接口确保编译时类型检查
4. **可扩展性**: 模块化设计支持功能逐步迁移

## 贡献指南

1. Fork 本仓库
2. 创建特性分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'Add amazing feature'`
4. 推送到分支: `git push origin feature/amazing-feature`
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 项目状态

- ✅ SecretStore MVP 实现完成
- ✅ TestContainers 集成测试
- ✅ 三层架构设计
- ✅ SQLC 类型安全数据访问
- 🚧 工作流引擎核心功能 (开发中)
- 🚧 API 服务层 (计划中)
- 🚧 Web 管理界面 (计划中)
