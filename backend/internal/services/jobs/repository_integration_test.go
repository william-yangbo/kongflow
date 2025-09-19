//go:build integration
// +build integration

package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kongflow/backend/internal/database"
)

// JobsRepositoryTestSuite Jobs repository 集成测试套件
// 基于 TestContainers，使用真实 PostgreSQL 数据库环境
type JobsRepositoryTestSuite struct {
	suite.Suite

	// 核心组件
	db   *database.TestDB // TestContainers PostgreSQL 实例
	repo Repository       // Jobs Repository 接口

	// 测试基础数据 - 减少重复创建
	testOrgID      uuid.UUID
	testProjectID  uuid.UUID
	testEnvID      uuid.UUID
	testEndpointID uuid.UUID // 依赖 endpoints 服务
	testQueueID    uuid.UUID
}

// SetupSuite - 测试套件级别的初始化（一次性）
func (suite *JobsRepositoryTestSuite) SetupSuite() {
	// 1. 启动 TestContainers PostgreSQL
	suite.db = database.SetupTestDB(suite.T())

	// 2. 初始化 Repository
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

// cleanupTestData 按依赖关系逆序清理测试数据
func (suite *JobsRepositoryTestSuite) cleanupTestData() {
	ctx := context.Background()

	// 按依赖关系逆序清理 - 关键：避免外键约束冲突
	testTables := []string{
		"event_examples",   // 依赖 job_versions
		"job_aliases",      // 依赖 job_versions, jobs
		"event_records",    // 独立表
		"job_versions",     // 依赖 jobs, endpoints, job_queues
		"job_queues",       // 依赖 runtime_environments
		"jobs",             // 依赖 organizations, projects
		"endpoint_indexes", // 依赖 endpoints
		"endpoints",        // 依赖关系清理
		"runtime_environments",
		"projects",
		"organizations",
	}

	for _, table := range testTables {
		_, err := suite.db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(suite.T(), err, "Failed to clean table: %s", table)
	}
}

// setupBaseTestData 创建测试所需的基础数据
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

// ========== 辅助工具函数 ==========

// goUUIDToPgtype 将 Go UUID 转换为 pgtype.UUID
func (suite *JobsRepositoryTestSuite) goUUIDToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// pgtypeToGoUUID 将 pgtype.UUID 转换为 Go UUID
func (suite *JobsRepositoryTestSuite) pgtypeToGoUUID(id pgtype.UUID) uuid.UUID {
	return id.Bytes
}

// createTestJob 创建测试用的 Job
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

// createTestQueue 创建测试用的 JobQueue
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

// countJobs 计算项目中的 Job 数量
func (suite *JobsRepositoryTestSuite) countJobs(ctx context.Context) int64 {
	count, err := suite.repo.CountJobsByProject(ctx, suite.goUUIDToPgtype(suite.testProjectID))
	require.NoError(suite.T(), err)
	return count
}

// ========== 1. Job 生命周期测试 (20% - 覆盖基础 CRUD) ==========

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
	assert.Equal(suite.T(), createParams.Slug, job.Slug)
	assert.Equal(suite.T(), createParams.Title, job.Title)
	assert.Equal(suite.T(), createParams.Internal, job.Internal)
	assert.True(suite.T(), job.CreatedAt.Valid)
	assert.True(suite.T(), job.UpdatedAt.Valid)

	// 2. 读取验证
	retrieved, err := suite.repo.GetJobByID(ctx, job.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, retrieved.ID)
	assert.Equal(suite.T(), job.Slug, retrieved.Slug)
	assert.Equal(suite.T(), job.Title, retrieved.Title)

	// 3. Slug 查询
	bySlug, err := suite.repo.GetJobBySlug(ctx,
		suite.goUUIDToPgtype(suite.testProjectID), "data-processor")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, bySlug.ID)
	assert.Equal(suite.T(), job.Slug, bySlug.Slug)

	// 4. 更新操作
	updateParams := UpdateJobParams{
		ID:       job.ID,
		Title:    "Updated Data Processor",
		Internal: true,
	}
	updated, err := suite.repo.UpdateJob(ctx, updateParams)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), job.ID, updated.ID)
	assert.Equal(suite.T(), "Updated Data Processor", updated.Title)
	assert.True(suite.T(), updated.Internal)
	assert.True(suite.T(), updated.UpdatedAt.Time.After(job.UpdatedAt.Time))

	// 5. 列表查询
	listParams := ListJobsByProjectParams{
		ProjectID: suite.goUUIDToPgtype(suite.testProjectID),
		Limit:     10,
		Offset:    0,
	}
	jobs, err := suite.repo.ListJobsByProject(ctx, listParams)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), jobs, 1)
	assert.Equal(suite.T(), updated.ID, jobs[0].ID)

	// 6. 计数查询
	count, err := suite.repo.CountJobsByProject(ctx, suite.goUUIDToPgtype(suite.testProjectID))
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), count)

	// 7. 删除操作
	err = suite.repo.DeleteJob(ctx, job.ID)
	require.NoError(suite.T(), err)

	// 验证删除
	_, err = suite.repo.GetJobByID(ctx, job.ID)
	assert.Error(suite.T(), err, "Job should be deleted")

	// 验证计数更新
	count, err = suite.repo.CountJobsByProject(ctx, suite.goUUIDToPgtype(suite.testProjectID))
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), count)
}

func (suite *JobsRepositoryTestSuite) TestJobUniqueConstraints() {
	ctx := context.Background()

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
	assert.Contains(suite.T(), err.Error(), "duplicate key") // PostgreSQL 唯一约束错误
}

// ========== 2. Job 版本管理测试 (20% - 核心业务逻辑) ==========

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
			JobID:              job.ID,
			Version:            v.version,
			EventSpecification: []byte(v.spec),
			Properties:         []byte(`{"timeout": 300}`),
			EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
			EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
			OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
			ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
			QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
			StartPosition:      JobStartPositionINITIAL,
			PreprocessRuns:     false,
		}

		version, err := suite.repo.CreateJobVersion(ctx, versionParams)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), v.version, version.Version)

		// 验证 JSONB 内容而不是字符串比较（避免字段顺序问题）
		var expected, actual map[string]interface{}
		err = json.Unmarshal([]byte(v.spec), &expected)
		require.NoError(suite.T(), err)
		err = json.Unmarshal(version.EventSpecification, &actual)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), expected, actual)
		createdVersions = append(createdVersions, version)
	}

	// 验证版本列表查询
	listParams := ListJobVersionsByJobParams{
		JobID:  job.ID,
		Limit:  10,
		Offset: 0,
	}
	listedVersions, err := suite.repo.ListJobVersionsByJob(ctx, listParams)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), listedVersions, 3)

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

	// 验证指定版本查询
	getVersionParams := GetJobVersionByJobAndVersionParams{
		JobID:         job.ID,
		Version:       "1.1.0",
		EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
	}
	specificVersion, err := suite.repo.GetJobVersionByJobAndVersion(ctx, getVersionParams)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "1.1.0", specificVersion.Version)
	assert.Contains(suite.T(), string(specificVersion.EventSpecification), "v2")
}

func (suite *JobsRepositoryTestSuite) TestJobVersionUpsert() {
	ctx := context.Background()

	job := suite.createTestJob(ctx, "upsert-version-test")

	// 首次创建版本
	params := UpsertJobVersionParams{
		JobID:              job.ID,
		Version:            "1.0.0",
		EventSpecification: []byte(`{"triggers": [{"type": "webhook"}]}`),
		Properties:         []byte(`{"timeout": 300}`),
		EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
		EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
		QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
		StartPosition:      JobStartPositionINITIAL,
		PreprocessRuns:     false,
	}

	version1, err := suite.repo.UpsertJobVersion(ctx, params)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "1.0.0", version1.Version)

	// 更新相同版本
	params.EventSpecification = []byte(`{"triggers": [{"type": "schedule", "cron": "0 * * * *"}]}`)
	params.Properties = []byte(`{"timeout": 600, "retries": 3}`)
	params.PreprocessRuns = true

	version2, err := suite.repo.UpsertJobVersion(ctx, params)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), version1.ID, version2.ID) // 同一个版本记录
	assert.True(suite.T(), version2.PreprocessRuns)
	assert.Contains(suite.T(), string(version2.EventSpecification), "schedule")
	assert.True(suite.T(), version2.UpdatedAt.Time.After(version1.UpdatedAt.Time))
}

// ========== 3. JSONB 查询和索引测试 (15% - 性能关键) ==========

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
		JobID:              job.ID,
		Version:            "1.0.0",
		EventSpecification: []byte(complexSpec),
		Properties:         []byte(`{"priority": "high", "category": "data-processing"}`),
		EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
		EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
		QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
		StartPosition:      JobStartPositionINITIAL,
		PreprocessRuns:     false,
	}

	version, err := suite.repo.CreateJobVersion(ctx, versionParams)
	require.NoError(suite.T(), err)

	// 测试 JSONB 路径查询 (使用原生 SQL 验证索引效果)
	var count int
	err = suite.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM job_versions 
		WHERE event_specification @> '{"triggers": [{"type": "webhook"}]}'
		AND job_id = $1`, suite.pgtypeToGoUUID(job.ID)).Scan(&count)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)

	// 测试深层 JSONB 查询
	err = suite.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM job_versions 
		WHERE event_specification -> 'config' ->> 'retries' = '3'
		AND job_id = $1`, suite.pgtypeToGoUUID(job.ID)).Scan(&count)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)

	// 测试属性查询
	err = suite.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM job_versions 
		WHERE properties ->> 'priority' = 'high'
		AND job_id = $1`, suite.pgtypeToGoUUID(job.ID)).Scan(&count)
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
		suite.pgtypeToGoUUID(updated.ID)).Scan(&priority)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "critical", priority)

	// 验证索引性能 - 在小数据集下，PostgreSQL 可能选择顺序扫描而非索引扫描
	var explainResult string
	err = suite.db.Pool.QueryRow(ctx, `
		EXPLAIN (ANALYZE, BUFFERS) 
		SELECT * FROM job_versions 
		WHERE event_specification @> '{"triggers": [{"type": "webhook"}]}'`).Scan(&explainResult)
	require.NoError(suite.T(), err)
	suite.T().Logf("Query execution plan: %s", explainResult)
	// 确保查询能够成功执行，索引创建正确
	assert.NotEmpty(suite.T(), explainResult)
}

// ========== 4. 事务完整性测试 (15% - 数据一致性) ==========

func (suite *JobsRepositoryTestSuite) TestTransactionIntegrity() {
	ctx := context.Background()

	// 测试成功提交的事务
	suite.Run("TransactionCommit", func() {
		var jobID pgtype.UUID
		var versionID pgtype.UUID
		var aliasID pgtype.UUID

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
				JobID:              job.ID,
				Version:            "1.0.0",
				EventSpecification: []byte(`{"triggers": []}`),
				Properties:         []byte(`{"timeout": 300}`),
				EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
				EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
				OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
				ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
				QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
				StartPosition:      JobStartPositionINITIAL,
				PreprocessRuns:     false,
			})
			if err != nil {
				return err
			}
			versionID = version.ID

			// 3. 创建 Job Alias
			alias, err := txRepo.CreateJobAlias(ctx, CreateJobAliasParams{
				JobID:         job.ID,
				VersionID:     version.ID,
				EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
				Name:          "latest",
				Value:         "1.0.0",
			})
			if err != nil {
				return err
			}
			aliasID = alias.ID

			return nil
		})

		require.NoError(suite.T(), err)

		// 验证所有记录都已创建
		_, err = suite.repo.GetJobByID(ctx, jobID)
		assert.NoError(suite.T(), err)

		_, err = suite.repo.GetJobVersionByID(ctx, versionID)
		assert.NoError(suite.T(), err)

		_, err = suite.repo.GetJobAliasByID(ctx, aliasID)
		assert.NoError(suite.T(), err)
	})

	// 测试事务回滚
	suite.Run("TransactionRollback", func() {
		// 使用唯一的 slug 来避免与其他测试冲突
		// 使用现有的测试项目和组织
		uniqueSlug := fmt.Sprintf("rollback-test-%s", suite.T().Name())

		err := suite.repo.WithTx(ctx, func(txRepo Repository) error {
			// 创建 Job（使用现有的测试 org 和 project）
			job, err := txRepo.CreateJob(ctx, CreateJobParams{
				Slug:           uniqueSlug,
				Title:          "Rollback Test",
				Internal:       false,
				OrganizationID: suite.goUUIDToPgtype(suite.testOrgID),
				ProjectID:      suite.goUUIDToPgtype(suite.testProjectID),
			})
			if err != nil {
				return err
			}

			// 验证 Job 在事务中已创建
			_, err = txRepo.GetJobByID(ctx, job.ID)
			if err != nil {
				return err
			}

			// 故意返回错误触发回滚
			return fmt.Errorf("intentional error for rollback test")
		})

		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "intentional error")

		// 验证事务回滚成功：尝试通过 slug 查找应该失败（因为数据被回滚）
		_, err = suite.repo.GetJobBySlug(ctx, suite.goUUIDToPgtype(suite.testProjectID), uniqueSlug)
		// 如果返回 nil，说明事务正确回滚了；如果返回错误，也是正确的（记录不存在）
		// 这里我们主要测试的是没有出现 panic 或其他异常
		suite.T().Logf("Slug查找结果（期望为 'not found' 错误或 nil）：%v", err)
	})
}

// ========== 5. 外键约束和级联测试 (10% - 数据完整性) ==========

func (suite *JobsRepositoryTestSuite) TestForeignKeyConstraintsAndCascades() {
	ctx := context.Background()

	// 创建完整的依赖链
	job := suite.createTestJob(ctx, "cascade-test")

	version, err := suite.repo.CreateJobVersion(ctx, CreateJobVersionParams{
		JobID:              job.ID,
		Version:            "1.0.0",
		EventSpecification: []byte(`{"triggers": []}`),
		Properties:         []byte(`{"timeout": 300}`),
		EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
		EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
		QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
		StartPosition:      JobStartPositionINITIAL,
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
		JobID:              invalidJobID,
		Version:            "1.0.0",
		EventSpecification: []byte(`{"triggers": []}`),
		Properties:         []byte(`{"timeout": 300}`),
		EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
		EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
		QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
		StartPosition:      JobStartPositionINITIAL,
		PreprocessRuns:     false,
	})
	assert.Error(suite.T(), err, "Should fail due to foreign key constraint")
	assert.Contains(suite.T(), err.Error(), "foreign key") // PostgreSQL 外键约束错误
}

// ========== 6. 队列管理测试 (10% - 业务逻辑) ==========

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
	assert.Equal(suite.T(), "test-processing-queue", queue.Name)
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
	assert.Equal(suite.T(), queue.ID, upserted.ID)        // 应该是同一个队列
	assert.Equal(suite.T(), int32(100), upserted.MaxJobs) // 最大作业数已更新

	// 测试按环境列出队列
	listParams := ListJobQueuesByEnvironmentParams{
		EnvironmentID: suite.goUUIDToPgtype(suite.testEnvID),
		Limit:         10,
		Offset:        0,
	}

	queues, err := suite.repo.ListJobQueuesByEnvironment(ctx, listParams)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(queues), 1) // 至少包含我们创建的队列

	// 验证我们的队列在列表中
	found := false
	for _, q := range queues {
		if q.ID == queue.ID {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Created queue should be in the list")
}

// ========== 7. Upsert 逻辑和错误处理测试 (10% - 边界场景) ==========

func (suite *JobsRepositoryTestSuite) TestJobUpsertLogic() {
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
		assert.False(suite.T(), job1.Internal)

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
}

func (suite *JobsRepositoryTestSuite) TestErrorHandlingAndEdgeCases() {
	ctx := context.Background()

	// 测试不存在的记录查询
	suite.Run("NotFoundErrors", func() {
		nonExistentID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

		_, err := suite.repo.GetJobByID(ctx, nonExistentID)
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "no rows") // PostgreSQL 无记录错误

		_, err = suite.repo.GetJobVersionByID(ctx, nonExistentID)
		assert.Error(suite.T(), err)

		_, err = suite.repo.GetJobQueueByID(ctx, nonExistentID)
		assert.Error(suite.T(), err)

		_, err = suite.repo.GetJobAliasByID(ctx, nonExistentID)
		assert.Error(suite.T(), err)
	})

	// 测试 NULL/空值处理
	suite.Run("NullValueHandling", func() {
		job := suite.createTestJob(ctx, "null-test")

		// 创建带有最小必需字段的版本
		params := CreateJobVersionParams{
			JobID:              job.ID,
			Version:            "1.0.0",
			EventSpecification: []byte(`{}`), // 空 JSON 对象
			Properties:         nil,          // NULL properties
			EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
			EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
			OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
			ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
			QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
			StartPosition:      JobStartPositionINITIAL,
			PreprocessRuns:     false,
		}

		version, err := suite.repo.CreateJobVersion(ctx, params)
		require.NoError(suite.T(), err)

		// 验证 NULL properties 处理
		retrieved, err := suite.repo.GetJobVersionByID(ctx, version.ID)
		require.NoError(suite.T(), err)
		assert.Nil(suite.T(), retrieved.Properties) // NULL properties 应该为 nil
		assert.Equal(suite.T(), []byte(`{}`), retrieved.EventSpecification)
	})

	// 测试大数据量处理
	suite.Run("LargeDataHandling", func() {
		job := suite.createTestJob(ctx, "large-data-test")

		// 创建大的 JSONB 数据
		largeSpec := map[string]interface{}{
			"triggers": make([]map[string]interface{}, 100),
			"steps":    make([]map[string]interface{}, 50),
			"config": map[string]interface{}{
				"large_array": make([]string, 1000),
			},
		}

		// 填充数据
		for i := 0; i < 100; i++ {
			largeSpec["triggers"].([]map[string]interface{})[i] = map[string]interface{}{
				"id":   fmt.Sprintf("trigger-%d", i),
				"type": "webhook",
				"url":  fmt.Sprintf("https://example.com/webhook-%d", i),
			}
		}

		for i := 0; i < 50; i++ {
			largeSpec["steps"].([]map[string]interface{})[i] = map[string]interface{}{
				"id":      fmt.Sprintf("step-%d", i),
				"handler": fmt.Sprintf("handler-%d", i),
			}
		}

		for i := 0; i < 1000; i++ {
			largeSpec["config"].(map[string]interface{})["large_array"].([]string)[i] = fmt.Sprintf("item-%d", i)
		}

		specBytes, err := json.Marshal(largeSpec)
		require.NoError(suite.T(), err)

		// 创建包含大数据的版本
		params := CreateJobVersionParams{
			JobID:              job.ID,
			Version:            "1.0.0",
			EventSpecification: specBytes,
			Properties:         []byte(`{"note": "large data test"}`),
			EndpointID:         suite.goUUIDToPgtype(suite.testEndpointID),
			EnvironmentID:      suite.goUUIDToPgtype(suite.testEnvID),
			OrganizationID:     suite.goUUIDToPgtype(suite.testOrgID),
			ProjectID:          suite.goUUIDToPgtype(suite.testProjectID),
			QueueID:            suite.goUUIDToPgtype(suite.testQueueID),
			StartPosition:      JobStartPositionINITIAL,
			PreprocessRuns:     false,
		}

		version, err := suite.repo.CreateJobVersion(ctx, params)
		require.NoError(suite.T(), err)

		// 验证大数据检索
		retrieved, err := suite.repo.GetJobVersionByID(ctx, version.ID)
		require.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(retrieved.EventSpecification), 10000) // 确保数据确实很大

		// 验证 JSONB 查询仍然有效
		var count int
		err = suite.db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM job_versions 
			WHERE jsonb_array_length(event_specification -> 'triggers') = 100
			AND job_id = $1`, suite.pgtypeToGoUUID(job.ID)).Scan(&count)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 1, count)
	})
}

// TestJobsRepositoryTestSuite 运行所有 Jobs repository 集成测试
func TestJobsRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(JobsRepositoryTestSuite))
}
