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

func (suite *RepositoryTestSuite) TestListSecretKeys() {
	ctx := context.Background()

	// 插入多个测试数据
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		value, _ := json.Marshal(map[string]string{"data": key})
		err := suite.repo.UpsertSecret(ctx, key, value)
		require.NoError(suite.T(), err)
	}

	// 测试列出密钥
	result, err := suite.repo.ListSecretKeys(ctx)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 3)

	// 验证返回的密钥
	returnedKeys := make([]string, len(result))
	for i, row := range result {
		returnedKeys[i] = row.Key
	}

	for _, expectedKey := range keys {
		assert.Contains(suite.T(), returnedKeys, expectedKey)
	}
}

func (suite *RepositoryTestSuite) TestGetSecretCount() {
	ctx := context.Background()

	// 初始应该为 0
	count, err := suite.repo.GetSecretCount(ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), count)

	// 插入数据
	value, _ := json.Marshal(map[string]string{"test": "data"})
	err = suite.repo.UpsertSecret(ctx, "count-test", value)
	require.NoError(suite.T(), err)

	// 验证计数
	count, err = suite.repo.GetSecretCount(ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), count)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
