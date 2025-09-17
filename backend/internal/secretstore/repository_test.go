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
	_, err := suite.db.Pool.Exec(context.Background(), `DELETE FROM "SecretStore"`)
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

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
