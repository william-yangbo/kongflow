# KongFlow SQLC Architecture

## 多服务 SQLC 组织架构

KongFlow 采用模块化的 SQLC 架构，支持多个服务的独立开发和维护。

### 🏗️ 目录结构

```
kongflow/backend/
├── sqlc.yaml                    # 主配置文件
├── db/                          # 数据库相关文件
│   └── migrations/              # 所有服务的迁移文件
│       ├── 001_secret_store.sql
│       ├── 002_job_queue.sql    # 未来
│       └── 003_workflow.sql     # 未来
└── internal/
    ├── secretstore/            # SecretStore服务
    │   ├── queries/            # 专属SQL查询
    │   │   └── secret_store.sql
    │   ├── db.go               # SQLC生成代码
    │   ├── models.go
    │   ├── queries.sql.go
    │   ├── repository.go       # 业务逻辑层
    │   └── service.go
    ├── jobqueue/              # JobQueue服务（未来）
    │   ├── queries/
    │   └── ...
    └── workflow/              # Workflow服务（未来）
        ├── queries/
        └── ...
```

### 🎯 设计原则

1. **服务隔离**: 每个服务有独立的 queries 目录
2. **配置集中**: 单个 sqlc.yaml 管理所有服务
3. **Schema 共享**: 所有服务共享 db/migrations
4. **代码分离**: 每个服务生成独立的 SQLC 代码

### 📚 最佳实践参考

- **SQLC 官方**: ondeck, authors 等多包示例
- **River 队列**: 模块化 SQL 文件组织
- **Trigger.dev**: Schema 兼容性设计

### 🚀 添加新服务

1. 创建服务目录: `internal/newservice/`
2. 创建查询目录: `internal/newservice/queries/`
3. 在 sqlc.yaml 中添加配置
4. 添加迁移文件到 db/migrations/
5. 运行`sqlc generate`

### 🔧 SQLC 配置

每个服务的 SQLC 配置包含：

```yaml
- name: servicename
  engine: 'postgresql'
  queries: './internal/servicename/queries'
  schema: './db/migrations'
  gen:
    go:
      out: './internal/servicename'
      package: 'servicename'
      sql_package: 'pgx/v5'
      emit_json_tags: true
      emit_interface: true
      emit_prepared_queries: false
      emit_exact_table_names: true
```

### ✅ 优势

- **可扩展性**: 轻松添加新服务
- **隔离性**: 服务间 SQL 代码独立
- **一致性**: 统一的代码生成配置
- **维护性**: 清晰的文件组织结构
