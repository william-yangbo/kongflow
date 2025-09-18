package apivote

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kongflow/backend/internal/database"
)

func TestApiVoteService_Core(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 使用现有testhelper，复用优化后的基础设施
	// 注意：apivote 服务现在依赖共享实体，需要运行所有迁移
	db := database.SetupTestDBWithMigrations(t, "")
	defer db.Cleanup(t)

	service := NewService(db.Pool)
	ctx := context.Background()

	t.Run("核心投票流程_严格对齐trigger.dev", func(t *testing.T) {
		// 测试参数对齐 trigger.dev 的 call({userId, identifier})
		req := VoteRequest{
			UserID:     "user-123",
			Identifier: "github",
		}

		// 1. 创建新投票
		resp, err := service.Call(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsNewVote, "首次投票应该标记为新投票")
		assert.Equal(t, req.UserID, resp.UserId, "用户ID应该匹配")
		assert.Equal(t, req.Identifier, resp.ApiIdentifier, "API标识符应该匹配")
		assert.NotEmpty(t, resp.ID, "投票ID不应为空")
		assert.True(t, resp.CreatedAt.Valid, "创建时间应该有效")
		assert.True(t, resp.UpdatedAt.Valid, "更新时间应该有效")

		// 2. 重复投票防护（对trigger.dev的改进）
		resp2, err := service.Call(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp2)
		assert.False(t, resp2.IsNewVote, "重复投票应该标记为非新投票")
		assert.Equal(t, resp.ID, resp2.ID, "重复投票应该返回相同的投票记录")

		// 3. 验证数据库唯一性约束生效
		assert.Equal(t, resp.ApiIdentifier, resp2.ApiIdentifier)
		assert.Equal(t, resp.UserId, resp2.UserId)
	})

	t.Run("参数验证_对齐trigger.dev接口要求", func(t *testing.T) {
		// 测试空UserID
		_, err := service.Call(ctx, VoteRequest{UserID: "", Identifier: "github"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "userId is required")

		// 测试空Identifier
		_, err = service.Call(ctx, VoteRequest{UserID: "user-123", Identifier: ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "identifier is required")

		// 测试完全空参数
		_, err = service.Call(ctx, VoteRequest{})
		assert.Error(t, err)
	})

	t.Run("多用户多API投票场景", func(t *testing.T) {
		// 模拟不同用户对不同API的投票
		testCases := []struct {
			userID     string
			identifier string
		}{
			{"user-001", "github"},
			{"user-001", "slack"},   // 同用户不同API
			{"user-002", "github"},  // 不同用户同API
			{"user-002", "discord"}, // 不同用户不同API
		}

		for _, tc := range testCases {
			req := VoteRequest{
				UserID:     tc.userID,
				Identifier: tc.identifier,
			}

			resp, err := service.Call(ctx, req)
			require.NoError(t, err)
			assert.True(t, resp.IsNewVote, "每个用户-API组合的首次投票都应该是新投票")
			assert.Equal(t, tc.userID, resp.UserId)
			assert.Equal(t, tc.identifier, resp.ApiIdentifier)
		}
	})

	t.Run("投票移除功能", func(t *testing.T) {
		// 先创建投票
		req := VoteRequest{
			UserID:     "user-remove-test",
			Identifier: "test-api",
		}

		resp, err := service.Call(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.IsNewVote)

		// 移除投票
		err = service.RemoveVote(ctx, req.UserID, req.Identifier)
		require.NoError(t, err)

		// 验证投票已被移除（再次投票应该创建新记录）
		resp2, err := service.Call(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp2.IsNewVote, "移除后再次投票应该创建新记录")
		assert.NotEqual(t, resp.ID, resp2.ID, "新投票的ID应该不同")
	})

	t.Run("时间戳处理验证", func(t *testing.T) {
		req := VoteRequest{
			UserID:     "user-timestamp-test",
			Identifier: "timestamp-api",
		}

		resp, err := service.Call(ctx, req)
		require.NoError(t, err)

		// 验证时间戳转换功能
		createdAt := ConvertTimestamp(resp.CreatedAt)
		updatedAt := ConvertTimestamp(resp.UpdatedAt)

		assert.False(t, createdAt.IsZero(), "创建时间转换后不应为零值")
		assert.False(t, updatedAt.IsZero(), "更新时间转换后不应为零值")

		// 创建时间和更新时间应该相近（在同一秒内）
		timeDiff := updatedAt.Sub(createdAt)
		assert.True(t, timeDiff >= 0, "更新时间应该不早于创建时间")
		assert.True(t, timeDiff.Seconds() < 1, "创建时间和更新时间应该在1秒内")
	})
}
