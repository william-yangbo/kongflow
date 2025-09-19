# Jobs Repository 集成测试计划

基于 TestContainers 的专业级集成测试方案

---

## 📋 目标与原则

### 🎯 测试目标

- **数据完整性验证**：确保复杂外键关系和级联操作的正确性
- **事务行为验证**：验证跨表操作的 ACID 特性
- **性能基准测试**：JSONB 查询和 GIN 索引的实际性能
- **业务逻辑保证**：版本管理、队列分配等核心业务流程正确性

### 🔍 设计原则

- **80/20 法则**：专注于覆盖 80% 核心使用场景的 20% 关键测试
- **真实环境模拟**：使用 TestContainers + PostgreSQL，完全匹配生产环境
- **测试隔离性**：每个测试独立运行，无状态污染
- **可维护性**：清晰的测试结构，易于扩展和维护

---

## 🏗 架构设计

### TestContainers 基础架构

```go
// JobsRepositoryTestSuite 基于 TestContainers 的集成测试套件
type JobsRepositoryTestSuite struct {
    suite.Suite

    // 核心组件
    db   *database.TestDB  // TestContainers PostgreSQL 实例
    repo Repository        // Jobs Repository 接口

    // 测试基础数据 - 减少重复创建
    testOrgID      uuid.UUID
    testProjectID  uuid.UUID
    testEnvID      uuid.UUID
    testEndpointID uuid.UUID  // 依赖 endpoints 服务
    testQueueID    uuid.UUID
}
```

### 生命周期管理

```go
// SetupSuite - 测试套件级别的初始化（一次性）
func (suite *JobsRepositoryTestSuite) SetupSuite() {
    // 1. 启动 TestContainers PostgreSQL
    suite.db = database.SetupTestDB(suite.T())

    // 2. 运行数据库迁移
    suite.db.RunMigrations()

    // 3. 初始化 Repository
    suite.repo = NewRepository(suite.db.Pool)
}

// TearDownSuite - 测试套件清理
func (suite *JobsRepositoryTestSuite) TearDownSuite() {
    suite.db.Cleanup(suite.T())
}

// SetupTest - 每个测试方法的初始化
func (suite *JobsRepositoryTestSuite) SetupTest() {
    suite.cleanupTestData()
    suite.setupBaseTestData()
}
```

### 数据清理策略

```go
func (suite *JobsRepositoryTestSuite) cleanupTestData() {
    ctx := context.Background()

    // 按依赖关系逆序清理 - 关键：避免外键约束冲突
    testTables := []string{
        "event_examples",    // 依赖 job_versions
        "job_aliases",       // 依赖 job_versions, jobs
        "event_records",     // 独立表
        "job_versions",      // 依赖 jobs, endpoints, job_queues
        "job_queues",        // 依赖 runtime_environments
        "jobs",              // 依赖 organizations, projects
        "endpoints",         // 依赖关系清理
        "runtime_environments",
        "projects",
        "organizations",
    }

    for _, table := range testTables {
        _, err := suite.db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
        require.NoError(suite.T(), err, "Failed to clean table: %s", table)
    }
}
```

---

## 🧪 核心测试场景

基于 80/20 原则，重点覆盖以下关键场景：

### 1. Job 生命周期测试 (20% - 覆盖基础 CRUD)

```go
func (suite *JobsRepositoryTestSuite) TestJobCRUDOperations() {
    ctx := context.Background()

    // 1. 创建 Job
    createParams := CreateJobParams{
        Slug:           "data-processor",
        Title:          "Data Processing Job",
        Internal:       false,
        OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
        ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
    }

    job, err := suite.repo.CreateJob(ctx, createParams)
    require.NoError(suite.T(), err)
    assert.NotEqual(suite.T(), pgtype.UUID{}, job.ID)

    // 2. 读取验证
    retrieved, err := suite.repo.GetJobByID(ctx, job.ID)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), job.Slug, retrieved.Slug)

    // 3. Slug 查询
    bySlug, err := suite.repo.GetJobBySlug(ctx,
        suite.goUUIDToPgtype(suite.testProjectID), "data-processor")
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), job.ID, bySlug.ID)

    // 4. 更新操作
    updateParams := UpdateJobParams{
        ID:       job.ID,
        Title:    "Updated Data Processor",
        Internal: true,
    }
    updated, err := suite.repo.UpdateJob(ctx, updateParams)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), "Updated Data Processor", updated.Title)
    assert.True(suite.T(), updated.Internal)

    // 5. 删除操作
    err = suite.repo.DeleteJob(ctx, job.ID)
    require.NoError(suite.T(), err)

    // 验证删除
    _, err = suite.repo.GetJobByID(ctx, job.ID)
    assert.Error(suite.T(), err)
}
```

### 2. Job 版本管理测试 (20% - 核心业务逻辑)

```go
func (suite *JobsRepositoryTestSuite) TestJobVersionManagement() {
    ctx := context.Background()

    // 前置：创建 Job
    job := suite.createTestJob(ctx, "version-test-job")

    // 测试场景：创建多个版本
    versions := []struct {
        version string
        spec    string
    }{
        {"1.0.0", `{"triggers": [{"type": "webhook"}]}`},
        {"1.1.0", `{"triggers": [{"type": "webhook", "version": "v2"}]}`},
        {"2.0.0", `{"triggers": [{"type": "schedule", "cron": "0 */6 * * *"}]}`},
    }

    var createdVersions []JobVersions

    for _, v := range versions {
        versionParams := CreateJobVersionParams{
            JobID:               job.ID,
            Version:            v.version,
            EventSpecification: []byte(v.spec),
            Properties:         []byte(`{"timeout": 300}`),
            EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
            EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
            OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
            ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
            QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
            StartPosition:      JobStartPositionInitial,
            PreprocessRuns:     false,
        }

        version, err := suite.repo.CreateJobVersion(ctx, versionParams)
        require.NoError(suite.T(), err)
        createdVersions = append(createdVersions, version)
    }

    // 验证最新版本查询
    latestParams := GetLatestJobVersionParams{
        JobID:         job.ID,
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
    }
    latest, err := suite.repo.GetLatestJobVersion(ctx, latestParams)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), "2.0.0", latest.Version)

    // 验证版本计数
    countParams := CountLaterJobVersionsParams{
        JobID:         job.ID,
        Version:       "1.0.0",
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
    }
    count, err := suite.repo.CountLaterJobVersions(ctx, countParams)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), int64(2), count) // 1.1.0 和 2.0.0
}
```

### 3. JSONB 查询和索引测试 (15% - 性能关键)

```go
func (suite *JobsRepositoryTestSuite) TestJSONBQueriesAndIndexes() {
    ctx := context.Background()

    job := suite.createTestJob(ctx, "jsonb-test-job")

    // 创建包含复杂 JSONB 数据的版本
    complexSpec := `{
        "triggers": [
            {
                "type": "webhook",
                "path": "/api/data",
                "methods": ["POST", "PUT"],
                "auth": {"type": "bearer"}
            },
            {
                "type": "schedule",
                "cron": "0 2 * * *",
                "timezone": "UTC"
            }
        ],
        "steps": [
            {
                "id": "validate",
                "type": "function",
                "handler": "validateInput"
            },
            {
                "id": "process",
                "type": "function",
                "handler": "processData",
                "depends_on": ["validate"]
            }
        ],
        "config": {
            "retries": 3,
            "timeout": 300,
            "env": ["PROD", "STAGING"]
        }
    }`

    versionParams := CreateJobVersionParams{
        JobID:               job.ID,
        Version:            "1.0.0",
        EventSpecification: []byte(complexSpec),
        Properties:         []byte(`{"priority": "high", "category": "data-processing"}`),
        EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
        EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
        OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
        ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
        QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
        StartPosition:      JobStartPositionInitial,
        PreprocessRuns:     false,
    }

    version, err := suite.repo.CreateJobVersion(ctx, versionParams)
    require.NoError(suite.T(), err)

    // 测试 JSONB 路径查询 (使用原生 SQL 验证索引效果)
    var count int
    err = suite.db.Pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM job_versions
        WHERE event_specification @> '{"triggers": [{"type": "webhook"}]}'
        AND job_id = $1`, job.ID).Scan(&count)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), 1, count)

    // 测试属性查询
    err = suite.db.Pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM job_versions
        WHERE properties ->> 'priority' = 'high'
        AND job_id = $1`, job.ID).Scan(&count)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), 1, count)

    // 属性更新测试
    newProperties := `{"priority": "critical", "category": "data-processing", "owner": "team-alpha"}`
    updateParams := UpdateJobVersionPropertiesParams{
        ID:         version.ID,
        Properties: []byte(newProperties),
    }

    updated, err := suite.repo.UpdateJobVersionProperties(ctx, updateParams)
    require.NoError(suite.T(), err)

    // 验证更新结果
    var priority string
    err = suite.db.Pool.QueryRow(ctx, `
        SELECT properties ->> 'priority' FROM job_versions WHERE id = $1`,
        updated.ID).Scan(&priority)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), "critical", priority)
}
```

### 4. 事务完整性测试 (15% - 数据一致性)

```go
func (suite *JobsRepositoryTestSuite) TestTransactionIntegrity() {
    ctx := context.Background()

    // 测试成功提交的事务
    suite.Run("TransactionCommit", func() {
        var jobID pgtype.UUID
        var versionID pgtype.UUID

        err := suite.repo.WithTx(ctx, func(txRepo Repository) error {
            // 1. 创建 Job
            job, err := txRepo.CreateJob(ctx, CreateJobParams{
                Slug:           "tx-test-job",
                Title:          "Transaction Test Job",
                Internal:       false,
                OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
                ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
            })
            if err != nil {
                return err
            }
            jobID = job.ID

            // 2. 创建 Job Version
            version, err := txRepo.CreateJobVersion(ctx, CreateJobVersionParams{
                JobID:               job.ID,
                Version:            "1.0.0",
                EventSpecification: []byte(`{"triggers": []}`),
                EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
                EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
                OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
                ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
                QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
                StartPosition:      JobStartPositionInitial,
                PreprocessRuns:     false,
            })
            if err != nil {
                return err
            }
            versionID = version.ID

            // 3. 创建 Job Alias
            _, err = txRepo.CreateJobAlias(ctx, CreateJobAliasParams{
                JobID:         job.ID,
                VersionID:     version.ID,
                EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
                Name:          "latest",
                Value:         "1.0.0",
            })
            return err
        })

        require.NoError(suite.T(), err)

        // 验证所有记录都已创建
        _, err = suite.repo.GetJobByID(ctx, jobID)
        assert.NoError(suite.T(), err)

        _, err = suite.repo.GetJobVersionByID(ctx, versionID)
        assert.NoError(suite.T(), err)
    })

    // 测试事务回滚
    suite.Run("TransactionRollback", func() {
        initialJobCount := suite.countJobs(ctx)

        err := suite.repo.WithTx(ctx, func(txRepo Repository) error {
            // 创建 Job
            _, err := txRepo.CreateJob(ctx, CreateJobParams{
                Slug:           "rollback-test",
                Title:          "Rollback Test",
                Internal:       false,
                OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
                ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
            })
            if err != nil {
                return err
            }

            // 故意返回错误触发回滚
            return fmt.Errorf("intentional error for rollback test")
        })

        assert.Error(suite.T(), err)
        assert.Contains(suite.T(), err.Error(), "intentional error")

        // 验证没有创建任何记录
        finalJobCount := suite.countJobs(ctx)
        assert.Equal(suite.T(), initialJobCount, finalJobCount)
    })
}
```

### 5. 外键约束和级联测试 (10% - 数据完整性)

```go
func (suite *JobsRepositoryTestSuite) TestForeignKeyConstraintsAndCascades() {
    ctx := context.Background()

    // 创建完整的依赖链
    job := suite.createTestJob(ctx, "cascade-test")

    version, err := suite.repo.CreateJobVersion(ctx, CreateJobVersionParams{
        JobID:               job.ID,
        Version:            "1.0.0",
        EventSpecification: []byte(`{"triggers": []}`),
        EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
        EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
        OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
        ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
        QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
        StartPosition:      JobStartPositionInitial,
        PreprocessRuns:     false,
    })
    require.NoError(suite.T(), err)

    alias, err := suite.repo.CreateJobAlias(ctx, CreateJobAliasParams{
        JobID:         job.ID,
        VersionID:     version.ID,
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
        Name:          "latest",
        Value:         "1.0.0",
    })
    require.NoError(suite.T(), err)

    // 测试级联删除：删除 Job 应该删除所有相关记录
    err = suite.repo.DeleteJob(ctx, job.ID)
    require.NoError(suite.T(), err)

    // 验证级联删除
    _, err = suite.repo.GetJobVersionByID(ctx, version.ID)
    assert.Error(suite.T(), err, "Version should be deleted")

    _, err = suite.repo.GetJobAliasByID(ctx, alias.ID)
    assert.Error(suite.T(), err, "Alias should be deleted")

    // 测试外键约束：尝试创建引用不存在 Job 的 Version
    invalidJobID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
    _, err = suite.repo.CreateJobVersion(ctx, CreateJobVersionParams{
        JobID:               invalidJobID,
        Version:            "1.0.0",
        EventSpecification: []byte(`{"triggers": []}`),
        EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
        EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
        OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
        ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
        QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
        StartPosition:      JobStartPositionInitial,
        PreprocessRuns:     false,
    })
    assert.Error(suite.T(), err, "Should fail due to foreign key constraint")
}
```

### 6. 队列管理测试 (10% - 业务逻辑)

```go
func (suite *JobsRepositoryTestSuite) TestJobQueueManagement() {
    ctx := context.Background()

    // 创建测试队列
    queueParams := CreateJobQueueParams{
        Name:          "test-processing-queue",
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
        JobCount:      0,
        MaxJobs:       50,
    }

    queue, err := suite.repo.CreateJobQueue(ctx, queueParams)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), int32(0), queue.JobCount)
    assert.Equal(suite.T(), int32(50), queue.MaxJobs)

    // 测试队列计数管理
    updatedQueue, err := suite.repo.IncrementJobCount(ctx, queue.ID)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), int32(1), updatedQueue.JobCount)

    updatedQueue, err = suite.repo.IncrementJobCount(ctx, queue.ID)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), int32(2), updatedQueue.JobCount)

    updatedQueue, err = suite.repo.DecrementJobCount(ctx, queue.ID)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), int32(1), updatedQueue.JobCount)

    // 测试 Upsert 行为
    upsertParams := UpsertJobQueueParams{
        Name:          "test-processing-queue", // 相同名称
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
        JobCount:      0,
        MaxJobs:       100, // 更新最大作业数
    }

    upserted, err := suite.repo.UpsertJobQueue(ctx, upsertParams)
    require.NoError(suite.T(), err)
    assert.Equal(suite.T(), queue.ID, upserted.ID) // 应该是同一个队列
    assert.Equal(suite.T(), int32(100), upserted.MaxJobs) // 最大作业数已更新

    // 测试按环境列出队列
    listParams := ListJobQueuesByEnvironmentParams{
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
        Limit:         10,
        Offset:        0,
    }

    queues, err := suite.repo.ListJobQueuesByEnvironment(ctx, listParams)
    require.NoError(suite.T(), err)
    assert.Len(suite.T(), queues, 1)
    assert.Equal(suite.T(), queue.ID, queues[0].ID)
}
```

### 7. Upsert 逻辑测试 (5% - 边界场景)

```go
func (suite *JobsRepositoryTestSuite) TestUpsertLogic() {
    ctx := context.Background()

    // Job Upsert 测试
    suite.Run("JobUpsert", func() {
        // 首次 Upsert - 应该创建新记录
        params := UpsertJobParams{
            Slug:           "upsert-test",
            Title:          "Initial Title",
            Internal:       false,
            OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
            ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
        }

        job1, err := suite.repo.UpsertJob(ctx, params)
        require.NoError(suite.T(), err)
        assert.Equal(suite.T(), "Initial Title", job1.Title)

        // 再次 Upsert - 应该更新现有记录
        params.Title = "Updated Title"
        params.Internal = true

        job2, err := suite.repo.UpsertJob(ctx, params)
        require.NoError(suite.T(), err)
        assert.Equal(suite.T(), job1.ID, job2.ID) // 相同 ID
        assert.Equal(suite.T(), "Updated Title", job2.Title)
        assert.True(suite.T(), job2.Internal)
        assert.True(suite.T(), job2.UpdatedAt.Time.After(job1.UpdatedAt.Time))
    })

    // JobVersion Upsert 测试
    suite.Run("JobVersionUpsert", func() {
        job := suite.createTestJob(ctx, "version-upsert-test")

        // 首次创建
        params := UpsertJobVersionParams{
            JobID:               job.ID,
            Version:            "1.0.0",
            EventSpecification: []byte(`{"triggers": [{"type": "webhook"}]}`),
            Properties:         []byte(`{"timeout": 300}`),
            EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
            EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
            OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
            ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
            QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
            StartPosition:      JobStartPositionInitial,
            PreprocessRuns:     false,
        }

        version1, err := suite.repo.UpsertJobVersion(ctx, params)
        require.NoError(suite.T(), err)

        // 更新相同版本
        params.EventSpecification = []byte(`{"triggers": [{"type": "schedule", "cron": "0 * * * *"}]}`)
        params.Properties = []byte(`{"timeout": 600, "retries": 3}`)
        params.PreprocessRuns = true

        version2, err := suite.repo.UpsertJobVersion(ctx, params)
        require.NoError(suite.T(), err)
        assert.Equal(suite.T(), version1.ID, version2.ID)
        assert.True(suite.T(), version2.PreprocessRuns)

        // 验证 JSONB 更新
        var timeout int
        err = suite.db.Pool.QueryRow(ctx, `
            SELECT (properties ->> 'timeout')::int FROM job_versions WHERE id = $1`,
            version2.ID).Scan(&timeout)
        require.NoError(suite.T(), err)
        assert.Equal(suite.T(), 600, timeout)
    })
}
```

### 8. 错误处理和边界测试 (5% - 健壮性)

```go
func (suite *JobsRepositoryTestSuite) TestErrorHandlingAndEdgeCases() {
    ctx := context.Background()

    // 测试不存在的记录查询
    suite.Run("NotFoundErrors", func() {
        nonExistentID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

        _, err := suite.repo.GetJobByID(ctx, nonExistentID)
        assert.Error(suite.T(), err)

        _, err = suite.repo.GetJobVersionByID(ctx, nonExistentID)
        assert.Error(suite.T(), err)

        _, err = suite.repo.GetJobQueueByID(ctx, nonExistentID)
        assert.Error(suite.T(), err)
    })

    // 测试唯一约束冲突
    suite.Run("UniqueConstraintViolations", func() {
        // 创建第一个 Job
        params := CreateJobParams{
            Slug:           "unique-test",
            Title:          "First Job",
            Internal:       false,
            OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
            ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
        }

        _, err := suite.repo.CreateJob(ctx, params)
        require.NoError(suite.T(), err)

        // 尝试创建相同 slug 的 Job - 应该失败
        params.Title = "Duplicate Job"
        _, err = suite.repo.CreateJob(ctx, params)
        assert.Error(suite.T(), err)
        assert.Contains(suite.T(), err.Error(), "unique") // PostgreSQL 唯一约束错误
    })

    // 测试 NULL/空值处理
    suite.Run("NullValueHandling", func() {
        job := suite.createTestJob(ctx, "null-test")

        // 创建带有最小必需字段的版本
        params := CreateJobVersionParams{
            JobID:               job.ID,
            Version:            "1.0.0",
            EventSpecification: []byte(`{}`), // 空 JSON 对象
            Properties:         nil,          // NULL properties
            EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
            EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
            OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
            ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
            QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
            StartPosition:      JobStartPositionInitial,
            PreprocessRuns:     false,
        }

        version, err := suite.repo.CreateJobVersion(ctx, params)
        require.NoError(suite.T(), err)

        // 验证 NULL properties 处理
        retrieved, err := suite.repo.GetJobVersionByID(ctx, version.ID)
        require.NoError(suite.T(), err)
        assert.Equal(suite.T(), []byte(nil), retrieved.Properties)
    })
}
```

---

## 🔧 辅助工具函数

```go
// 测试数据创建辅助函数
func (suite *JobsRepositoryTestSuite) createTestJob(ctx context.Context, slug string) Jobs {
    params := CreateJobParams{
        Slug:           slug,
        Title:          fmt.Sprintf("Test Job: %s", slug),
        Internal:       false,
        OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
        ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
    }

    job, err := suite.repo.CreateJob(ctx, params)
    require.NoError(suite.T(), err)
    return job
}

func (suite *JobsRepositoryTestSuite) createTestQueue(ctx context.Context, name string) JobQueues {
    params := CreateJobQueueParams{
        Name:          name,
        EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
        JobCount:      0,
        MaxJobs:       100,
    }

    queue, err := suite.repo.CreateJobQueue(ctx, params)
    require.NoError(suite.T(), err)
    return queue
}

// UUID 转换辅助函数
func (suite *JobsRepositoryTestSuite) goUUIDToPgtype(id uuid.UUID) pgtype.UUID {
    return pgtype.UUID{Bytes: id, Valid: true}
}

func (suite *JobsRepositoryTestSuite) pgtypeToGoUUID(id pgtype.UUID) uuid.UUID {
    return id.Bytes
}

// 计数辅助函数
func (suite *JobsRepositoryTestSuite) countJobs(ctx context.Context) int64 {
    count, err := suite.repo.CountJobsByProject(ctx, suite.goUUIDToPgtype(suite.testProjectID))
    require.NoError(suite.T(), err)
    return count
}

// 基础测试数据设置
func (suite *JobsRepositoryTestSuite) setupBaseTestData() {
    ctx := context.Background()

    // 创建组织
    suite.testOrgID = uuid.New()
    _, err := suite.db.Pool.Exec(ctx, `
        INSERT INTO organizations (id, title, slug)
        VALUES ($1, 'Test Organization', 'test-org')`,
        suite.testOrgID)
    require.NoError(suite.T(), err)

    // 创建项目
    suite.testProjectID = uuid.New()
    _, err = suite.db.Pool.Exec(ctx, `
        INSERT INTO projects (id, name, slug, organization_id)
        VALUES ($1, 'Test Project', 'test-project', $2)`,
        suite.testProjectID, suite.testOrgID)
    require.NoError(suite.T(), err)

    // 创建环境
    suite.testEnvID = uuid.New()
    _, err = suite.db.Pool.Exec(ctx, `
        INSERT INTO runtime_environments (id, slug, api_key, type, organization_id, project_id)
        VALUES ($1, 'test-env', 'test-api-key', 'DEVELOPMENT', $2, $3)`,
        suite.testEnvID, suite.testOrgID, suite.testProjectID)
    require.NoError(suite.T(), err)

    // 创建端点 (依赖 endpoints 服务)
    suite.testEndpointID = uuid.New()
    _, err = suite.db.Pool.Exec(ctx, `
        INSERT INTO endpoints (id, slug, url, indexing_hook_identifier, environment_id, organization_id, project_id)
        VALUES ($1, 'test-endpoint', 'https://test.example.com', 'test-hook', $2, $3, $4)`,
        suite.testEndpointID, suite.testEnvID, suite.testOrgID, suite.testProjectID)
    require.NoError(suite.T(), err)

    // 创建默认队列
    suite.testQueueID = uuid.New()
    _, err = suite.db.Pool.Exec(ctx, `
        INSERT INTO job_queues (id, name, environment_id, job_count, max_jobs)
        VALUES ($1, 'default', $2, 0, 100)`,
        suite.testQueueID, suite.testEnvID)
    require.NoError(suite.T(), err)
}
```

---

## 🏃‍♂️ 测试运行配置

### Makefile 目标

```makefile
# 集成测试
.PHONY: test-integration-jobs
test-integration-jobs:
	@echo "Running Jobs Repository Integration Tests..."
	cd backend && go test ./internal/services/jobs -tags=integration -v -timeout=5m

# 全部集成测试
.PHONY: test-integration
test-integration: test-integration-jobs
	@echo "All integration tests completed"

# 测试覆盖率
.PHONY: test-integration-coverage
test-integration-coverage:
	cd backend && go test ./internal/services/jobs -tags=integration -coverprofile=coverage.out -covermode=atomic
	cd backend && go tool cover -html=coverage.out -o coverage.html
```

### 测试构建标签

```go
//go:build integration
// +build integration

package jobs

// 只有在运行集成测试时才编译此文件
```

---

## 📊 成功指标

### 测试覆盖率目标

- **核心 CRUD 操作**：100% 覆盖
- **事务操作**：100% 覆盖
- **JSONB 查询**：90% 覆盖
- **外键约束**：95% 覆盖
- **边界条件**：80% 覆盖

### 性能基准

- **单个 Job 创建**：< 50ms
- **复杂 JSONB 查询**：< 100ms
- **事务提交**：< 200ms
- **级联删除**：< 300ms

### 质量门槛

- 所有测试必须通过
- 无数据库连接泄漏
- 无内存泄漏
- 测试执行时间 < 5 分钟

---

## 🔄 维护和扩展

### 添加新测试的准则

1. **遵循命名约定**：`Test[Entity][Operation][Scenario]`
2. **使用辅助函数**：减少重复代码
3. **测试数据清理**：确保测试间无状态污染
4. **文档更新**：新测试场景需要更新本文档

### 性能优化建议

1. **并行测试**：无依赖的测试可以并行运行
2. **数据库连接池**：合理配置连接池大小
3. **测试数据量**：保持测试数据最小化
4. **索引验证**：定期验证索引效果

---

## 📝 总结

这个集成测试计划基于 **80/20 原则**，重点关注最关键的 20% 测试场景，确保覆盖 80% 的核心功能和风险点。通过 TestContainers 提供真实的 PostgreSQL 环境，确保测试结果的可靠性和生产环境的一致性。

**关键优势**：

- ✅ **真实环境**：TestContainers + PostgreSQL
- ✅ **全面覆盖**：CRUD + 事务 + JSONB + 约束
- ✅ **高效执行**：80/20 原则，专注核心场景
- ✅ **易于维护**：清晰结构，完善的辅助函数
- ✅ **性能验证**：包含性能基准测试

这个测试计划将为 Jobs 服务提供坚实的质量保障，确保数据库层的正确性和可靠性。
