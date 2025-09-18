# KongFlow SQLC Architecture

## 🏗️ 混合架构：共享基础层 + 服务特定层

KongFlow 采用基于 **SQLC 官方最佳实践** 的混合架构，结合共享基础实体和服务特定实体，实现最佳的代码复用和服务隔离。

### 📁 目录结构

```
kongflow/backend/
├── sqlc.yaml                         # 主配置文件 (混合架构配置)
├── db/                               # 数据库相关文件
│   └── migrations/                   # 统一数据源 (权威定义)
│       ├── 001_secret_store.sql
│       ├── 002_shared_entities.sql   # 🆕 共享基础实体
│       └── 003_api_vote_service.sql
└── internal/
    ├── shared/                       # 🆕 共享数据库层 (核心实体)
    │   ├── queries/                  # 共享查询
    │   │   ├── users.sql            # User 相关查询
    │   │   ├── organizations.sql    # Organization 相关查询
    │   │   └── projects.sql         # Project 相关查询
    │   ├── db.go                     # SQLC生成
    │   ├── models.go                 # 🔥 共享模型 (User, Organization, Project)
    │   └── *.sql.go                  # 生成的查询方法
    ├── secretstore/                  # SecretStore服务
    │   ├── queries/                  # 专属SQL查询
    │   │   └── secret_store.sql
    │   ├── db.go                     # SQLC生成 (包隔离)
    │   ├── models.go                 # 🔥 只包含SecretStore模型
    │   ├── secret_store.sql.go
    │   ├── repository.go             # 业务逻辑层 (组合 shared + 自己的查询)
    │   └── service.go
    ├── apiauth/                      # 🆕 ApiAuth服务 (trigger.dev对齐)
    │   ├── queries/
    │   │   ├── personal_tokens.sql  # PersonalAccessToken
    │   │   └── api_keys.sql         # ApiKey
    │   ├── db.go                     # SQLC生成
    │   ├── models.go                 # 🔥 服务特定模型 (PersonalAccessToken, ApiKey)
    │   ├── *.sql.go
    │   └── service.go                # 业务逻辑 (组合 shared + apiauth 查询)
    ├── apivote/                      # ApiVote服务
    │   ├── queries/
    │   │   └── api_vote.sql
    │   ├── db.go                     # SQLC生成 (包隔离)
    │   ├── models.go                 # 🔥 只包含ApiIntegrationVote模型
    │   ├── api_vote.sql.go
    │   └── service.go
    └── ...                           # 其他服务
```

### 🎯 混合架构设计原则

1. **🏛️ 单一数据源**: db/migrations 作为所有表结构的权威定义
2. **� 分层复用**: 共享基础实体 (User, Organization, Project) + 服务特定实体
3. **�🔧 智能过滤**: 使用 `omit_unused_structs` 自动消除无关模型
4. **📁 服务隔离**: 每个服务有独立的 queries 目录和生成代码
5. **🛡️ 测试兼容**: 保持与现有 testhelper 和工具链的完全兼容
6. **⚙️ 包组合**: 业务逻辑层组合使用 shared + 服务特定查询

### 🔥 混合架构优势

**共享基础层**：

- ✅ **User, Organization, Project**: 在 `shared` 包中统一定义
- ✅ **类型安全**: 跨服务使用相同的模型类型，避免转换
- ✅ **维护简单**: 核心实体的变更只需要修改一个地方

**服务特定层**：

- ✅ **PersonalAccessToken, ApiKey**: 在 `apiauth` 包中独立定义
- ✅ **业务隔离**: 服务特定的数据模型不会影响其他服务
- ✅ **智能过滤**: SQLC 自动过滤，确保每个服务只生成需要的模型

### 📚 最佳实践参考

- **🏆 SQLC 官方推荐**: `omit_unused_structs` 智能过滤模式
- **🌊 River 队列**: 单一包策略和模块化 SQL 文件组织
- **⚡ Trigger.dev**: Schema 兼容性设计和数据模型对齐
- **📖 SQLC 示例**: ondeck, authors 等官方多包示例项目

### 🚀 添加新服务流程

#### 共享实体相关服务（如 ApiAuth）：

1. **确定依赖**: 识别需要使用的共享实体 (User, Organization, Project)
2. **创建服务目录**: `internal/newservice/`
3. **创建查询目录**: `internal/newservice/queries/`
4. **添加迁移文件**: `db/migrations/xxx_newservice.sql`
5. **更新 sqlc.yaml**: 添加服务配置 (包含 `omit_unused_structs: true`)
6. **创建查询文件**: `internal/newservice/queries/newservice.sql`
7. **生成代码**: `sqlc generate`
8. **组合使用**: 在业务逻辑中组合 `shared.Queries` + `newservice.Queries`

#### 独立服务（如 SecretStore）：

1. **创建服务目录**: `internal/newservice/`
2. **创建查询目录**: `internal/newservice/queries/`
3. **添加迁移文件**: `db/migrations/xxx_newservice.sql`
4. **更新 sqlc.yaml**: 添加服务配置 (包含 `omit_unused_structs: true`)
5. **创建查询文件**: `internal/newservice/queries/newservice.sql`
6. **生成代码**: `sqlc generate`
7. **验证模型**: 确认只生成相关表模型

### 🔧 混合架构 SQLC 配置

#### 共享数据库层配置：

```yaml
- name: shared
  engine: 'postgresql'
  queries: './internal/shared/queries'
  schema: './db/migrations'
  gen:
    go:
      out: './internal/shared'
      package: 'shared'
      sql_package: 'pgx/v5'
      emit_json_tags: true
      emit_interface: true
      emit_prepared_queries: false
      emit_exact_table_names: true
      omit_unused_structs: true # 🔥 只包含 User, Organization, Project
```

#### 服务特定层配置：

```yaml
- name: apiauth
  engine: 'postgresql'
  queries: './internal/apiauth/queries'
  schema: './db/migrations'
  gen:
    go:
      out: './internal/apiauth'
      package: 'apiauth'
      sql_package: 'pgx/v5'
      emit_json_tags: true
      emit_interface: true
      emit_prepared_queries: false
      emit_exact_table_names: true
      omit_unused_structs: true # 🔥 只包含 PersonalAccessToken, ApiKey 等
```

### 💡 业务逻辑层组合使用示例

```go
// internal/apiauth/service.go
package apiauth

import (
    "github.com/kongflow/backend/internal/shared"
)

type Service struct {
    sharedQueries  *shared.Queries     // User, Organization, Project 查询
    apiAuthQueries *Queries            // PersonalAccessToken, ApiKey 查询
}

func (s *Service) CreatePersonalToken(userID string, orgID string) {
    // 使用 shared.Queries 获取 User 和 Organization
    user, err := s.sharedQueries.GetUser(ctx, userID)
    org, err := s.sharedQueries.GetOrganization(ctx, orgID)

    // 使用 apiauth.Queries 创建 token
    token, err := s.apiAuthQueries.CreatePersonalAccessToken(ctx, ...)
}
```

### 📊 架构对比分析

| 方面           | 传统多包模式 | 纯服务隔离模式 | KongFlow 混合模式 |
| -------------- | ------------ | -------------- | ----------------- |
| **模型重复**   | ❌ 100%重复  | ❌ 跨服务重复  | ✅ 0%重复         |
| **配置复杂度** | 🟡 中等      | ✅ 简单        | 🟡 中等           |
| **测试兼容性** | ❌ 需要适配  | ✅ 完全兼容    | ✅ 完全兼容       |
| **维护成本**   | ❌ 高        | 🟡 中等        | ✅ 低             |
| **类型安全**   | 🟡 需要转换  | ❌ 类型不一致  | ✅ 完全类型安全   |
| **扩展性**     | 🟡 一般      | ✅ 优秀        | ✅ 优秀           |
| **代码复用**   | ❌ 差        | ❌ 需要重复    | ✅ 最佳           |
| **维护成本**   | ❌ 高        | ✅ 低          |
| **扩展性**     | 🟡 一般      | ✅ 优秀        |

### ✅ 混合架构优势

#### 🎯 **核心优势**

- **零模型重复**: 共享实体在 `shared` 包中统一定义，服务特定实体各自隔离
- **单一数据源**: db/migrations 作为权威定义，避免不一致
- **智能生成**: 每个包只包含相关的表模型和查询方法
- **完全兼容**: 与现有测试框架和工具链无缝集成
- **类型安全**: 跨服务使用相同的共享模型类型，避免转换

#### 🚀 **开发效率**

- **配置清晰**: 共享层 + 服务层的分层配置，职责明确
- **组合灵活**: 业务逻辑层可灵活组合共享查询和服务特定查询
- **调试友好**: 清晰的文件组织，容易定位问题
- **版本控制**: 标准的 migration 历史，便于回滚和审计

#### 🏗️ **企业级特性**

- **可扩展性**: 支持无限制添加新服务，共享层提供稳定基础
- **团队协作**: 共享实体由平台团队维护，服务实体由业务团队开发
- **生产就绪**: River、SQLC 官方项目验证的成熟模式
- **工具生态**: 与主流数据库工具和 CI/CD 流水线完全兼容

### 🛡️ 质量保证

#### 📊 **测试策略**

- **85%+ 覆盖率**: 每个服务维持高质量测试覆盖
- **隔离测试**: 使用 TestContainers 确保测试环境一致性
- **迁移测试**: 自动验证数据库 schema 的正确性

#### 🔍 **代码质量**

- **SQLC 生成**: 避免手写 SQL 绑定代码的错误
- **类型检查**: Go 编译器保证类型安全
- **代码审查**: 清晰的文件边界便于 Code Review

### 🎯 最佳实践总结

> **KongFlow 采用的混合架构是 SQLC 官方推荐的智能过滤技术与实际生产环境需求的完美结合，提供了共享基础实体的复用性和服务特定实体的隔离性。**

这种架构模式已经通过以下项目验证：

- ✅ **River**: 高性能作业队列系统 (centralized dbsqlc pattern)
- ✅ **多个 SQLC 官方示例**: ondeck, authors, booktest (single package + omit_unused_structs)
- ✅ **KongFlow**: trigger.dev 迁移项目，严格对齐验证

#### 🔄 架构演进路径

1. **Phase 1**: 创建 `shared` 包，迁移 User, Organization, Project 基础实体
2. **Phase 2**: 实施 ApiAuth 服务，验证混合架构的有效性
3. **Phase 3**: 逐步迁移其他服务，完善生态系统

#### 🎯 适用场景

**使用混合架构的情况**：

- ✅ 多个服务需要使用相同的基础实体 (User, Organization, Project)
- ✅ 服务有自己特定的数据模型需求
- ✅ 需要保持跨服务的类型安全和一致性
- ✅ 团队希望平衡代码复用和服务隔离

**使用纯服务隔离的情况**：

- ✅ 服务完全独立，不共享任何数据模型
- ✅ 微服务架构，强调完全的服务边界
- ✅ 服务间通过 API 而非共享数据库交互
