package endpoints

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"kongflow/backend/internal/database"
)

// TestCreateEndpointValidation 测试创建端点的输入验证
func TestCreateEndpointValidation(t *testing.T) {
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewRepository(testDB.Pool)
	service := NewService(repo, nil, nil, nil) // 测试验证逻辑，不需要外部依赖

	ctx := context.Background()

	// 测试空 slug
	req := EndpointRequest{
		URL:            "https://example.com/api",
		EnvironmentID:  uuid.New(),
		OrganizationID: uuid.New(),
		ProjectID:      uuid.New(),
	}

	result, err := service.CreateEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "slug is required")

	// 测试空 URL
	req = EndpointRequest{
		Slug:           "test-endpoint",
		EnvironmentID:  uuid.New(),
		OrganizationID: uuid.New(),
		ProjectID:      uuid.New(),
	}

	result, err = service.CreateEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "url is required")

	// 测试缺少 environment ID
	req = EndpointRequest{
		Slug:           "test-endpoint",
		URL:            "https://example.com/api",
		OrganizationID: uuid.New(),
		ProjectID:      uuid.New(),
	}

	result, err = service.CreateEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "environment_id is required")
}

// TestUpsertEndpointValidation 测试 Upsert 端点的输入验证
func TestUpsertEndpointValidation(t *testing.T) {
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewRepository(testDB.Pool)
	service := NewService(repo, nil, nil, nil) // 测试验证逻辑，不需要外部依赖

	ctx := context.Background()

	// 测试空 slug
	req := UpsertEndpointRequest{
		URL:            "https://example.com/api",
		EnvironmentID:  uuid.New(),
		OrganizationID: uuid.New(),
		ProjectID:      uuid.New(),
	}

	result, err := service.UpsertEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "slug is required")

	// 测试空 URL
	req = UpsertEndpointRequest{
		Slug:           "test-endpoint",
		EnvironmentID:  uuid.New(),
		OrganizationID: uuid.New(),
		ProjectID:      uuid.New(),
	}

	result, err = service.UpsertEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "url is required")
}

// TestIndexEndpointValidation 测试索引端点的输入验证
func TestIndexEndpointValidation(t *testing.T) {
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewRepository(testDB.Pool)
	service := NewService(repo, nil, nil, nil) // 测试验证逻辑，不需要外部依赖

	ctx := context.Background()

	// 测试空 endpoint ID
	req := IndexEndpointRequest{}

	result, err := service.IndexEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "endpoint_id is required")

	// 测试无效的 endpoint ID (不存在的端点)
	req = IndexEndpointRequest{
		EndpointID: uuid.New(), // 随机 UUID，数据库中不存在
	}

	result, err = service.IndexEndpoint(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "endpoint not found")
}

// TestServiceMethods 测试服务方法的基本结构和签名
func TestServiceMethods(t *testing.T) {
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewRepository(testDB.Pool)
	service := NewService(repo, nil, nil, nil)

	// 确保服务实现了 Service 接口
	var _ Service = service

	t.Run("service creation", func(t *testing.T) {
		assert.NotNil(t, service)
	})
}

// TestGenerateHookIdentifier 测试 Hook 标识符生成
func TestGenerateHookIdentifier(t *testing.T) {
	testDB := database.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := NewRepository(testDB.Pool)
	svc := NewService(repo, nil, nil, nil).(*service)

	// 测试生成的标识符
	hookID1 := svc.generateHookIdentifier()
	hookID2 := svc.generateHookIdentifier()

	// 验证标识符格式
	assert.NotEmpty(t, hookID1)
	assert.NotEmpty(t, hookID2)
	assert.NotEqual(t, hookID1, hookID2) // 应该是唯一的
	assert.Len(t, hookID1, 10)           // 应该是10个字符
	assert.Len(t, hookID2, 10)           // 应该是10个字符

	// 验证只包含预期的字符集
	const charset = "0123456789abcdefghijklmnopqrstuvxyz"
	for _, r := range hookID1 {
		assert.Contains(t, charset, string(r))
	}
	for _, r := range hookID2 {
		assert.Contains(t, charset, string(r))
	}
}
