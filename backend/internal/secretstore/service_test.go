package secretstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository 实现 Repository 接口的 mock
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetSecret(ctx context.Context, key string) (*SecretStore, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SecretStore), args.Error(1)
}

func (m *MockRepository) UpsertSecret(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockRepository) DeleteSecret(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRepository) ListSecretKeys(ctx context.Context) ([]ListSecretStoreKeysRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ListSecretStoreKeysRow), args.Error(1)
}

func (m *MockRepository) GetSecretCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestService_SetAndGetSecret(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	// 准备测试数据
	testKey := "test-key"
	testValue := map[string]string{"hello": "world"}
	expectedJSON := `{"hello":"world"}`

	// Mock repository 行为
	mockRepo.On("UpsertSecret", ctx, testKey, []byte(expectedJSON)).Return(nil)
	mockRepo.On("GetSecret", ctx, testKey).Return(&SecretStore{
		Key:   testKey,
		Value: []byte(expectedJSON),
	}, nil)

	// 测试 SetSecret
	err := service.SetSecret(ctx, testKey, testValue)
	assert.NoError(t, err)

	// 测试 GetSecret
	var result map[string]string
	err = service.GetSecret(ctx, testKey, &result)
	assert.NoError(t, err)
	assert.Equal(t, testValue, result)

	// 验证 mock 调用
	mockRepo.AssertExpectations(t)
}

func TestService_GetSecretNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	// Mock repository 返回 not found 错误
	mockRepo.On("GetSecret", ctx, "nonexistent").Return(nil, ErrSecretNotFound)

	var result map[string]string
	err := service.GetSecret(ctx, "nonexistent", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret")
}

func TestService_InvalidJSON(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	// 测试无效的 JSON 数据
	mockRepo.On("GetSecret", ctx, "invalid-json").Return(&SecretStore{
		Key:   "invalid-json",
		Value: []byte(`{invalid json`),
	}, nil)

	var result map[string]string
	err := service.GetSecret(ctx, "invalid-json", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestService_ComplexDataTypes(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	// 测试复杂数据结构
	testKey := "complex-data"
	complexData := map[string]interface{}{
		"string":  "test",
		"number":  42,
		"boolean": true,
		"array":   []string{"a", "b", "c"},
		"object": map[string]interface{}{
			"nested": "value",
		},
	}

	expectedJSON := `{"array":["a","b","c"],"boolean":true,"number":42,"object":{"nested":"value"},"string":"test"}`

	// Mock repository 行为
	mockRepo.On("UpsertSecret", ctx, testKey, []byte(expectedJSON)).Return(nil)
	mockRepo.On("GetSecret", ctx, testKey).Return(&SecretStore{
		Key:   testKey,
		Value: []byte(expectedJSON),
	}, nil)

	// 测试 SetSecret
	err := service.SetSecret(ctx, testKey, complexData)
	assert.NoError(t, err)

	// 测试 GetSecret
	var result map[string]interface{}
	err = service.GetSecret(ctx, testKey, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result["string"])
	assert.Equal(t, float64(42), result["number"]) // JSON 数字默认为 float64
	assert.Equal(t, true, result["boolean"])

	// 验证 mock 调用
	mockRepo.AssertExpectations(t)
}

func TestService_GetSecretOrThrow(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	// 测试存在的密钥
	testKey := "existing-key"
	testData := map[string]string{"data": "value"}
	expectedJSON := `{"data":"value"}`

	mockRepo.On("GetSecret", ctx, testKey).Return(&SecretStore{
		Key:   testKey,
		Value: []byte(expectedJSON),
	}, nil)

	var result map[string]string
	err := service.GetSecretOrThrow(ctx, testKey, &result)
	assert.NoError(t, err)
	assert.Equal(t, testData, result)

	// 测试不存在的密钥
	mockRepo.On("GetSecret", ctx, "missing-key").Return(nil, ErrSecretNotFound)

	var missingResult map[string]string
	err = service.GetSecretOrThrow(ctx, "missing-key", &missingResult)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mockRepo.AssertExpectations(t)
}
