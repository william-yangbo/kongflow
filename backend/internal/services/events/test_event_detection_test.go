package events

import (
	"testing"

	"kongflow/backend/internal/services/apiauth"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestDetermineIfTestEvent(t *testing.T) {
	// 创建测试环境
	createTestEnv := func(envType apiauth.EnvironmentType, apiKey string) *apiauth.AuthenticatedEnvironment {
		envUUID := pgtype.UUID{}
		envUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

		return &apiauth.AuthenticatedEnvironment{
			Environment: apiauth.RuntimeEnvironment{
				ID:     envUUID,
				Type:   envType,
				APIKey: apiKey,
			},
		}
	}

	// 创建测试事件
	createTestEvent := func(name string) *SendEventRequest {
		return &SendEventRequest{
			ID:   "evt_test_123",
			Name: name,
		}
	}

	t.Run("显式测试标志优先级最高", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		event := createTestEvent("user.created")

		// 显式设置为测试事件
		testTrue := true
		opts := &SendEventOptions{Test: &testTrue}
		assert.True(t, determineIfTestEvent(prodEnv, event, opts))

		// 显式设置为非测试事件
		testFalse := false
		opts = &SendEventOptions{Test: &testFalse}
		assert.False(t, determineIfTestEvent(prodEnv, event, opts))
	})

	t.Run("开发环境默认为测试事件", func(t *testing.T) {
		devEnv := createTestEnv(apiauth.EnvironmentTypeDevelopment, "tr_dev_abcd1234")
		event := createTestEvent("user.created")

		assert.True(t, determineIfTestEvent(devEnv, event, nil))
		assert.True(t, determineIfTestEvent(devEnv, event, &SendEventOptions{}))
	})

	t.Run("生产环境默认为非测试事件", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		event := createTestEvent("user.created")

		assert.False(t, determineIfTestEvent(prodEnv, event, nil))
		assert.False(t, determineIfTestEvent(prodEnv, event, &SendEventOptions{}))
	})

	t.Run("预览环境默认为非测试事件", func(t *testing.T) {
		previewEnv := createTestEnv(apiauth.EnvironmentTypePreview, "tr_preview_abcd1234")
		event := createTestEvent("user.created")

		assert.False(t, determineIfTestEvent(previewEnv, event, nil))
	})

	t.Run("暂存环境默认为非测试事件", func(t *testing.T) {
		stagingEnv := createTestEnv(apiauth.EnvironmentTypeStaging, "tr_stg_abcd1234")
		event := createTestEvent("user.created")

		assert.False(t, determineIfTestEvent(stagingEnv, event, nil))
	})

	t.Run("API Key前缀判断", func(t *testing.T) {
		// tr_dev_ 前缀
		prodEnvWithDevKey := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_dev_abcd1234")
		event := createTestEvent("user.created")
		assert.True(t, determineIfTestEvent(prodEnvWithDevKey, event, nil))

		// pk_dev_ 前缀
		prodEnvWithPkDevKey := createTestEnv(apiauth.EnvironmentTypeProduction, "pk_dev_abcd1234")
		assert.True(t, determineIfTestEvent(prodEnvWithPkDevKey, event, nil))

		// 其他前缀
		prodEnvWithProdKey := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		assert.False(t, determineIfTestEvent(prodEnvWithProdKey, event, nil))
	})

	t.Run("事件名称以test.开头", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")

		// test. 开头的事件
		testEvent := createTestEvent("test.user.created")
		assert.True(t, determineIfTestEvent(prodEnv, testEvent, nil))

		// 普通事件
		normalEvent := createTestEvent("user.created")
		assert.False(t, determineIfTestEvent(prodEnv, normalEvent, nil))

		// 包含test但不以test.开头
		containsTestEvent := createTestEvent("user.test.created")
		assert.False(t, determineIfTestEvent(prodEnv, containsTestEvent, nil))
	})

	t.Run("优先级测试：显式标志 > 环境类型", func(t *testing.T) {
		devEnv := createTestEnv(apiauth.EnvironmentTypeDevelopment, "tr_dev_abcd1234")
		event := createTestEvent("user.created")

		// 开发环境中显式设置为非测试事件
		testFalse := false
		opts := &SendEventOptions{Test: &testFalse}
		assert.False(t, determineIfTestEvent(devEnv, event, opts))
	})

	t.Run("优先级测试：显式标志 > API Key前缀", func(t *testing.T) {
		prodEnvWithDevKey := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_dev_abcd1234")
		event := createTestEvent("user.created")

		// dev key但显式设置为非测试事件
		testFalse := false
		opts := &SendEventOptions{Test: &testFalse}
		assert.False(t, determineIfTestEvent(prodEnvWithDevKey, event, opts))
	})

	t.Run("优先级测试：显式标志 > 事件名称", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		testEvent := createTestEvent("test.user.created")

		// test.开头但显式设置为非测试事件
		testFalse := false
		opts := &SendEventOptions{Test: &testFalse}
		assert.False(t, determineIfTestEvent(prodEnv, testEvent, opts))
	})

	t.Run("复合条件测试", func(t *testing.T) {
		// 开发环境 + dev key + test.事件名称 = 测试事件
		devEnv := createTestEnv(apiauth.EnvironmentTypeDevelopment, "tr_dev_abcd1234")
		testEvent := createTestEvent("test.user.created")
		assert.True(t, determineIfTestEvent(devEnv, testEvent, nil))

		// 生产环境 + prod key + 普通事件名称 = 非测试事件
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		normalEvent := createTestEvent("user.created")
		assert.False(t, determineIfTestEvent(prodEnv, normalEvent, nil))
	})
}

func TestDetermineIfTestEvent_EdgeCases(t *testing.T) {
	// 边界情况测试
	createTestEnv := func(envType apiauth.EnvironmentType, apiKey string) *apiauth.AuthenticatedEnvironment {
		envUUID := pgtype.UUID{}
		envUUID.Scan("123e4567-e89b-12d3-a456-426614174000")

		return &apiauth.AuthenticatedEnvironment{
			Environment: apiauth.RuntimeEnvironment{
				ID:     envUUID,
				Type:   envType,
				APIKey: apiKey,
			},
		}
	}

	t.Run("空事件名称", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")
		event := &SendEventRequest{
			ID:   "evt_test_123",
			Name: "",
		}
		assert.False(t, determineIfTestEvent(prodEnv, event, nil))
	})

	t.Run("nil选项", func(t *testing.T) {
		devEnv := createTestEnv(apiauth.EnvironmentTypeDevelopment, "tr_dev_abcd1234")
		event := &SendEventRequest{
			ID:   "evt_test_123",
			Name: "user.created",
		}
		assert.True(t, determineIfTestEvent(devEnv, event, nil))
	})

	t.Run("空API Key", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "")
		event := &SendEventRequest{
			ID:   "evt_test_123",
			Name: "user.created",
		}
		assert.False(t, determineIfTestEvent(prodEnv, event, nil))
	})

	t.Run("特殊事件名称格式", func(t *testing.T) {
		prodEnv := createTestEnv(apiauth.EnvironmentTypeProduction, "tr_prod_abcd1234")

		// 只有 "test"
		testOnlyEvent := &SendEventRequest{ID: "evt_test_123", Name: "test"}
		assert.False(t, determineIfTestEvent(prodEnv, testOnlyEvent, nil))

		// "test." 后面是空
		testDotEvent := &SendEventRequest{ID: "evt_test_123", Name: "test."}
		assert.True(t, determineIfTestEvent(prodEnv, testDotEvent, nil))

		// 大小写
		TestEvent := &SendEventRequest{ID: "evt_test_123", Name: "Test.user.created"}
		assert.False(t, determineIfTestEvent(prodEnv, TestEvent, nil))
	})
}
