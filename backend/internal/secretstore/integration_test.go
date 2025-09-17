package secretstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"kongflow/backend/internal/database"
)

type IntegrationTestSuite struct {
	suite.Suite
	db      *database.TestDB
	service *Service
}

func (suite *IntegrationTestSuite) SetupSuite() {
	suite.db = database.SetupTestDB(suite.T())
	repo := NewRepository(suite.db.Pool)
	suite.service = NewService(repo)
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	suite.db.Cleanup(suite.T())
}

func (suite *IntegrationTestSuite) SetupTest() {
	// 清理测试数据
	_, err := suite.db.Pool.Exec(context.Background(), "DELETE FROM secret_store")
	require.NoError(suite.T(), err)
}

func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
	ctx := context.Background()

	// 测试数据
	secretKey := "oauth.github.client"
	secretData := map[string]interface{}{
		"client_id":     "github_client_123",
		"client_secret": "github_secret_456",
		"scopes":        []string{"read:user", "repo"},
		"metadata": map[string]string{
			"provider": "github",
			"env":      "test",
		},
	}

	// 1. 设置 Secret
	err := suite.service.SetSecret(ctx, secretKey, secretData)
	assert.NoError(suite.T(), err)

	// 2. 获取 Secret
	var retrieved map[string]interface{}
	err = suite.service.GetSecret(ctx, secretKey, &retrieved)
	assert.NoError(suite.T(), err)

	// 验证数据完整性
	assert.Equal(suite.T(), secretData["client_id"], retrieved["client_id"])
	assert.Equal(suite.T(), secretData["client_secret"], retrieved["client_secret"])

	// 验证数组类型
	scopes, ok := retrieved["scopes"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), scopes, 2)

	// 3. 更新 Secret
	updatedData := map[string]interface{}{
		"client_id":     "github_client_789",
		"client_secret": "github_secret_updated",
		"scopes":        []string{"read:user", "repo", "admin:org"},
	}

	err = suite.service.SetSecret(ctx, secretKey, updatedData)
	assert.NoError(suite.T(), err)

	// 4. 验证更新
	var updated map[string]interface{}
	err = suite.service.GetSecret(ctx, secretKey, &updated)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "github_client_789", updated["client_id"])

	updatedScopes, ok := updated["scopes"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), updatedScopes, 3)

	// 5. 测试不存在的 Secret
	var notFound map[string]interface{}
	err = suite.service.GetSecret(ctx, "nonexistent.key", &notFound)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to get secret")
}

func (suite *IntegrationTestSuite) TestMultipleSecretsManagement() {
	ctx := context.Background()

	// 创建多个不同类型的密钥
	secrets := map[string]interface{}{
		"database.postgres": map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"username": "user",
			"password": "pass",
		},
		"api.key": "simple-string-secret",
		"feature.flags": map[string]bool{
			"enable_new_ui": true,
			"debug_mode":    false,
		},
	}

	// 1. 批量设置密钥
	var err error
	for key, value := range secrets {
		err = suite.service.SetSecret(ctx, key, value)
		assert.NoError(suite.T(), err)
	}

	// 2. 验证每个密钥的值
	// 数据库配置
	var dbConfig map[string]interface{}
	err = suite.service.GetSecret(ctx, "database.postgres", &dbConfig)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "localhost", dbConfig["host"])
	assert.Equal(suite.T(), float64(5432), dbConfig["port"]) // JSON 数字为 float64

	// API 密钥
	var apiKey string
	err = suite.service.GetSecret(ctx, "api.key", &apiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "simple-string-secret", apiKey)

	// 功能开关
	var flags map[string]bool
	err = suite.service.GetSecret(ctx, "feature.flags", &flags)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), flags["enable_new_ui"])
	assert.False(suite.T(), flags["debug_mode"])

	// 3. 测试 GetSecretOrThrow
	var dbConfigThrow map[string]interface{}
	err = suite.service.GetSecretOrThrow(ctx, "database.postgres", &dbConfigThrow)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), dbConfig, dbConfigThrow)

	// 4. 测试不存在的密钥
	var nonExistent string
	err = suite.service.GetSecret(ctx, "non.existent", &nonExistent)
	assert.Error(suite.T(), err)
}

func (suite *IntegrationTestSuite) TestGetSecretOrThrowIntegration() {
	ctx := context.Background()

	// 设置测试密钥
	testData := map[string]string{"integration": "test"}
	err := suite.service.SetSecret(ctx, "test.integration", testData)
	require.NoError(suite.T(), err)

	// 测试 GetSecretOrThrow 成功情况
	var result map[string]string
	err = suite.service.GetSecretOrThrow(ctx, "test.integration", &result)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testData, result)

	// 测试 GetSecretOrThrow 失败情况
	var missing map[string]string
	err = suite.service.GetSecretOrThrow(ctx, "missing.key", &missing)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
