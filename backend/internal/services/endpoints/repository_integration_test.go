package endpoints

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kongflow/backend/internal/database"
)

// RepositoryTestSuite endpoints repository集成测试套件
// 基于TestContainers，使用真实PostgreSQL数据库环境
type RepositoryTestSuite struct {
	suite.Suite
	db            *database.TestDB
	repo          Repository
	testOrgID     uuid.UUID
	testProjectID uuid.UUID
	testEnvID     uuid.UUID
}

func (suite *RepositoryTestSuite) SetupSuite() {
	suite.db = database.SetupTestDB(suite.T())
	suite.repo = NewRepository(suite.db.Pool)
}

func (suite *RepositoryTestSuite) TearDownSuite() {
	suite.db.Cleanup(suite.T())
}

func (suite *RepositoryTestSuite) SetupTest() {
	// 清理测试数据，保证测试间隔离
	ctx := context.Background()
	_, err := suite.db.Pool.Exec(ctx, `DELETE FROM endpoint_indexes`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM endpoints`)
	require.NoError(suite.T(), err)

	// 清理并创建测试所需的依赖记录
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM runtime_environments`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM organizations`)
	require.NoError(suite.T(), err)
	_, err = suite.db.Pool.Exec(ctx, `DELETE FROM projects`)
	require.NoError(suite.T(), err)

	// Create test organization
	_, err = suite.db.Pool.Exec(ctx, `
		INSERT INTO organizations (id, title, slug) 
		VALUES ($1, $2, $3)`,
		suite.testOrgID, "Test Organization", "test-org")
	require.NoError(suite.T(), err)

	// 创建测试项目
	suite.testProjectID = uuid.New()
	_, err = suite.db.Pool.Exec(ctx,
		`INSERT INTO projects (id, name, slug, organization_id, created_at, updated_at) 
		 VALUES ($1, 'Test Project', 'test-project', $2, NOW(), NOW())`,
		suite.testProjectID, suite.testOrgID)
	require.NoError(suite.T(), err)

	// 创建测试环境
	suite.testEnvID = uuid.New()
	_, err = suite.db.Pool.Exec(ctx,
		`INSERT INTO runtime_environments (id, slug, api_key, type, organization_id, project_id, created_at, updated_at) 
		 VALUES ($1, 'test-env', 'test-api-key', 'DEVELOPMENT', $2, $3, NOW(), NOW())`,
		suite.testEnvID, suite.testOrgID, suite.testProjectID)
	require.NoError(suite.T(), err)
}

// ========== Endpoint CRUD 测试 ==========

func (suite *RepositoryTestSuite) TestCreateAndGetEndpoint() {
	ctx := context.Background()

	// 准备测试数据
	params := CreateEndpointParams{
		Slug:                   "api-webhook",
		Url:                    "https://api.example.com/webhook",
		IndexingHookIdentifier: "hook-123",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	// 测试创建
	created, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), created)

	// 验证创建结果
	assert.NotEqual(suite.T(), pgtype.UUID{}, created.ID)
	assert.Equal(suite.T(), params.Slug, created.Slug)
	assert.Equal(suite.T(), params.Url, created.Url)
	assert.Equal(suite.T(), params.IndexingHookIdentifier, created.IndexingHookIdentifier)
	assert.Equal(suite.T(), params.EnvironmentID, created.EnvironmentID)
	assert.True(suite.T(), created.CreatedAt.Valid)
	assert.True(suite.T(), created.UpdatedAt.Valid)

	// 测试通过ID获取
	endpointID := suite.pgtypeToGoUUID(created.ID)
	retrieved, err := suite.repo.GetEndpointByID(ctx, endpointID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrieved)

	// 验证获取结果
	assert.Equal(suite.T(), created.ID, retrieved.ID)
	assert.Equal(suite.T(), created.Slug, retrieved.Slug)
	assert.Equal(suite.T(), created.Url, retrieved.Url)
	assert.Equal(suite.T(), created.IndexingHookIdentifier, retrieved.IndexingHookIdentifier)
}

func (suite *RepositoryTestSuite) TestGetEndpointBySlug() {
	ctx := context.Background()

	// 创建测试端点
	params := CreateEndpointParams{
		Slug:                   "test-api",
		Url:                    "https://test-api.example.com",
		IndexingHookIdentifier: "test-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	created, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)

	// 测试通过Slug获取
	retrieved, err := suite.repo.GetEndpointBySlug(ctx, suite.testEnvID, "test-api")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrieved)

	// 验证结果
	assert.Equal(suite.T(), created.ID, retrieved.ID)
	assert.Equal(suite.T(), created.Slug, retrieved.Slug)
	assert.Equal(suite.T(), created.EnvironmentID, retrieved.EnvironmentID)
}

func (suite *RepositoryTestSuite) TestUpdateEndpointURL() {
	ctx := context.Background()

	// 创建测试端点
	params := CreateEndpointParams{
		Slug:                   "update-test",
		Url:                    "https://old-url.example.com",
		IndexingHookIdentifier: "upd-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	created, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)

	// 测试更新URL
	newURL := "https://new-url.example.com"
	endpointID := suite.pgtypeToGoUUID(created.ID)
	updated, err := suite.repo.UpdateEndpointURL(ctx, endpointID, newURL)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), updated)

	// 验证更新结果
	assert.Equal(suite.T(), created.ID, updated.ID)
	assert.Equal(suite.T(), newURL, updated.Url)
	assert.NotEqual(suite.T(), created.UpdatedAt, updated.UpdatedAt) // 更新时间应该变化
}

func (suite *RepositoryTestSuite) TestUpsertEndpoint() {
	ctx := context.Background()

	// 测试插入新记录
	params := UpsertEndpointParams{
		Slug:                   "upsert-test",
		Url:                    "https://initial-url.example.com",
		IndexingHookIdentifier: "init-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	inserted, err := suite.repo.UpsertEndpoint(ctx, params)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), inserted)

	// 验证插入结果
	assert.Equal(suite.T(), params.Slug, inserted.Slug)
	assert.Equal(suite.T(), params.Url, inserted.Url)
	initialCreatedAt := inserted.CreatedAt

	// 测试更新现有记录（相同environment_id + slug）
	updateParams := UpsertEndpointParams{
		Slug:                   "upsert-test", // 相同slug
		Url:                    "https://updated-url.example.com",
		IndexingHookIdentifier: "upd-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID), // 相同environment_id
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	updated, err := suite.repo.UpsertEndpoint(ctx, updateParams)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), updated)

	// 验证更新结果
	assert.Equal(suite.T(), inserted.ID, updated.ID) // ID应该相同
	assert.Equal(suite.T(), updateParams.Url, updated.Url)
	assert.Equal(suite.T(), updateParams.IndexingHookIdentifier, updated.IndexingHookIdentifier)
	assert.Equal(suite.T(), initialCreatedAt, updated.CreatedAt)      // 创建时间不变
	assert.NotEqual(suite.T(), inserted.UpdatedAt, updated.UpdatedAt) // 更新时间应该变化
}

func (suite *RepositoryTestSuite) TestDeleteEndpoint() {
	ctx := context.Background()

	// 创建测试端点
	params := CreateEndpointParams{
		Slug:                   "delete-test",
		Url:                    "https://delete-test.example.com",
		IndexingHookIdentifier: "del-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	created, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)

	// 测试删除
	endpointID := suite.pgtypeToGoUUID(created.ID)
	err = suite.repo.DeleteEndpoint(ctx, endpointID)
	require.NoError(suite.T(), err)

	// 验证删除结果 - 应该找不到记录
	_, err = suite.repo.GetEndpointByID(ctx, endpointID)
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointNotFound)
}

// ========== EndpointIndex CRUD 测试 ==========

func (suite *RepositoryTestSuite) TestCreateAndGetEndpointIndex() {
	ctx := context.Background()

	// 先创建父端点
	endpoint := suite.createTestEndpoint(ctx)

	// 准备索引数据
	stats := map[string]interface{}{
		"total_docs": 150,
		"size_mb":    12.5,
	}
	statsBytes, _ := json.Marshal(stats)

	data := map[string]interface{}{
		"schema_version": "1.0",
		"fields":         []string{"title", "content", "tags"},
	}
	dataBytes, _ := json.Marshal(data)

	// 测试创建索引
	indexParams := CreateEndpointIndexParams{
		EndpointID: endpoint.ID,
		Source:     "API",
		Stats:      statsBytes,
		Data:       dataBytes,
	}

	created, err := suite.repo.CreateEndpointIndex(ctx, indexParams)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), created)

	// 验证创建结果
	assert.NotEqual(suite.T(), pgtype.UUID{}, created.ID)
	assert.Equal(suite.T(), endpoint.ID, created.EndpointID)
	assert.Equal(suite.T(), "API", created.Source)
	assert.JSONEq(suite.T(), string(statsBytes), string(created.Stats))
	assert.JSONEq(suite.T(), string(dataBytes), string(created.Data))
	assert.True(suite.T(), created.CreatedAt.Valid)

	// 测试通过ID获取索引
	indexID := suite.pgtypeToGoUUID(created.ID)
	retrieved, err := suite.repo.GetEndpointIndexByID(ctx, indexID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrieved)

	// 验证获取结果
	assert.Equal(suite.T(), created.ID, retrieved.ID)
	assert.Equal(suite.T(), created.EndpointID, retrieved.EndpointID)
	assert.Equal(suite.T(), created.Source, retrieved.Source)
	assert.JSONEq(suite.T(), string(created.Stats), string(retrieved.Stats))
	assert.JSONEq(suite.T(), string(created.Data), string(retrieved.Data))
}

func (suite *RepositoryTestSuite) TestListEndpointIndexes() {
	ctx := context.Background()

	// 创建测试端点
	endpoint := suite.createTestEndpoint(ctx)
	endpointID := suite.pgtypeToGoUUID(endpoint.ID)

	// 创建多个索引
	sources := []string{"API", "HOOK", "MANUAL"}
	var createdIndexes []*CreateEndpointIndexRow

	for _, source := range sources {
		indexParams := CreateEndpointIndexParams{
			EndpointID: endpoint.ID,
			Source:     source,
			Stats:      []byte(`{"docs": 100}`),
			Data:       []byte(`{"type": "test"}`),
		}

		created, err := suite.repo.CreateEndpointIndex(ctx, indexParams)
		require.NoError(suite.T(), err)
		createdIndexes = append(createdIndexes, created)
	}

	// 测试列出索引
	indexes, err := suite.repo.ListEndpointIndexes(ctx, endpointID)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), indexes, 3)

	// 验证结果
	foundSources := make(map[string]bool)
	for _, index := range indexes {
		assert.Equal(suite.T(), endpoint.ID, index.EndpointID)
		foundSources[index.Source] = true
	}

	for _, expectedSource := range sources {
		assert.True(suite.T(), foundSources[expectedSource], "Source %s should be found", expectedSource)
	}
}

func (suite *RepositoryTestSuite) TestDeleteEndpointIndex() {
	ctx := context.Background()

	// 创建测试端点和索引
	endpoint := suite.createTestEndpoint(ctx)
	indexParams := CreateEndpointIndexParams{
		EndpointID: endpoint.ID,
		Source:     "MANUAL",
		Stats:      []byte(`{"docs": 50}`),
		Data:       []byte(`{"test": true}`),
	}

	created, err := suite.repo.CreateEndpointIndex(ctx, indexParams)
	require.NoError(suite.T(), err)

	// 测试删除索引
	indexID := suite.pgtypeToGoUUID(created.ID)
	err = suite.repo.DeleteEndpointIndex(ctx, indexID)
	require.NoError(suite.T(), err)

	// 验证删除结果
	_, err = suite.repo.GetEndpointIndexByID(ctx, indexID)
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointIndexNotFound)
}

// ========== 错误处理测试 ==========

func (suite *RepositoryTestSuite) TestGetEndpointByIDNotFound() {
	ctx := context.Background()
	nonExistentID := uuid.New()

	endpoint, err := suite.repo.GetEndpointByID(ctx, nonExistentID)
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointNotFound)
	assert.Nil(suite.T(), endpoint)
}

func (suite *RepositoryTestSuite) TestGetEndpointBySlugNotFound() {
	ctx := context.Background()
	environmentID := uuid.New()

	endpoint, err := suite.repo.GetEndpointBySlug(ctx, environmentID, "nonexistent-slug")
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointNotFound)
	assert.Nil(suite.T(), endpoint)
}

func (suite *RepositoryTestSuite) TestUpdateEndpointURLNotFound() {
	ctx := context.Background()
	nonExistentID := uuid.New()

	endpoint, err := suite.repo.UpdateEndpointURL(ctx, nonExistentID, "https://new-url.com")
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointNotFound)
	assert.Nil(suite.T(), endpoint)
}

func (suite *RepositoryTestSuite) TestGetEndpointIndexByIDNotFound() {
	ctx := context.Background()
	nonExistentID := uuid.New()

	index, err := suite.repo.GetEndpointIndexByID(ctx, nonExistentID)
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointIndexNotFound)
	assert.Nil(suite.T(), index)
}

// ========== 事务测试 ==========

func (suite *RepositoryTestSuite) TestWithTxCommit() {
	ctx := context.Background()

	var createdEndpoint *CreateEndpointRow
	var createdIndex *CreateEndpointIndexRow

	// 测试事务提交
	err := suite.repo.WithTx(ctx, func(txRepo Repository) error {
		// 在事务中创建端点
		endpointParams := CreateEndpointParams{
			Slug:                   "tx-test",
			Url:                    "https://tx-test.example.com",
			IndexingHookIdentifier: "tx-hook",
			EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
			OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
			ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
		}

		endpoint, err := txRepo.CreateEndpoint(ctx, endpointParams)
		if err != nil {
			return err
		}
		createdEndpoint = endpoint

		// 在事务中创建索引
		indexParams := CreateEndpointIndexParams{
			EndpointID: endpoint.ID,
			Source:     "MANUAL",
			Stats:      []byte(`{"docs": 200}`),
			Data:       []byte(`{"tx": true}`),
		}

		index, err := txRepo.CreateEndpointIndex(ctx, indexParams)
		if err != nil {
			return err
		}
		createdIndex = index

		return nil
	})

	require.NoError(suite.T(), err)

	// 验证事务外能够查询到数据（事务已提交）
	endpointID := suite.pgtypeToGoUUID(createdEndpoint.ID)
	endpoint, err := suite.repo.GetEndpointByID(ctx, endpointID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "tx-test", endpoint.Slug)

	indexID := suite.pgtypeToGoUUID(createdIndex.ID)
	index, err := suite.repo.GetEndpointIndexByID(ctx, indexID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "MANUAL", index.Source)
}

func (suite *RepositoryTestSuite) TestWithTxRollback() {
	ctx := context.Background()

	var endpointID uuid.UUID

	// 测试事务回滚
	err := suite.repo.WithTx(ctx, func(txRepo Repository) error {
		// 在事务中创建端点
		params := CreateEndpointParams{
			Slug:                   "rollback-test",
			Url:                    "https://rollback-test.example.com",
			IndexingHookIdentifier: "rb-hook",
			EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
			OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
			ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
		}

		endpoint, err := txRepo.CreateEndpoint(ctx, params)
		if err != nil {
			return err
		}

		endpointID = suite.pgtypeToGoUUID(endpoint.ID)

		// 故意返回错误触发回滚
		return assert.AnError
	})

	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, assert.AnError)

	// 验证事务外查询不到数据（事务已回滚）
	_, err = suite.repo.GetEndpointByID(ctx, endpointID)
	assert.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, ErrEndpointNotFound)
}

// ========== 约束测试 ==========

func (suite *RepositoryTestSuite) TestUniqueConstraintEnvironmentSlug() {
	ctx := context.Background()

	params := CreateEndpointParams{
		Slug:                   "unique-test",
		Url:                    "https://unique1.example.com",
		IndexingHookIdentifier: "uniq-hk-1",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	// 创建第一个端点
	_, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)

	// 尝试创建相同environment_id + slug的端点（应该失败）
	duplicateParams := CreateEndpointParams{
		Slug:                   "unique-test", // 相同slug
		Url:                    "https://unique2.example.com",
		IndexingHookIdentifier: "uniq-hk-2",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID), // 相同environment_id
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	_, err = suite.repo.CreateEndpoint(ctx, duplicateParams)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "duplicate key value") // PostgreSQL唯一约束错误
}

// ========== 辅助方法 ==========

func (suite *RepositoryTestSuite) newUUID() pgtype.UUID {
	return pgtype.UUID{
		Bytes: uuid.New(),
		Valid: true,
	}
}

func (suite *RepositoryTestSuite) goUUIDToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

func (suite *RepositoryTestSuite) pgtypeToGoUUID(id pgtype.UUID) uuid.UUID {
	return id.Bytes
}

func (suite *RepositoryTestSuite) createTestEndpoint(ctx context.Context) *CreateEndpointRow {
	params := CreateEndpointParams{
		Slug:                   "test-endpoint",
		Url:                    "https://test.example.com",
		IndexingHookIdentifier: "test-hook",
		EnvironmentID:          suite.goUUIDToPgtype(suite.testEnvID),
		OrganizationID:         suite.goUUIDToPgtype(suite.testOrgID),
		ProjectID:              suite.goUUIDToPgtype(suite.testProjectID),
	}

	endpoint, err := suite.repo.CreateEndpoint(ctx, params)
	require.NoError(suite.T(), err)
	return endpoint
}

// TestRepositoryTestSuite 运行所有repository集成测试
func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
