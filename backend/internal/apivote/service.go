package apivote

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service 提供API投票业务逻辑
// 严格对齐 trigger.dev 的 ApiVoteService 接口
type Service struct {
	queries *Queries
}

// NewService 创建新的API投票服务实例
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		queries: New(db),
	}
}

// VoteRequest 投票请求参数
// 严格对齐 trigger.dev 的 call() 方法参数: {userId, identifier}
type VoteRequest struct {
	UserID     string `json:"userId" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
}

// VoteResponse 投票响应
// 扩展 trigger.dev 的返回值，添加是否为新投票的标识
type VoteResponse struct {
	*ApiIntegrationVote
	IsNewVote bool `json:"isNewVote"`
}

// Call 执行投票操作
// 严格对齐 trigger.dev 的 ApiVoteService.call({userId, identifier}) 方法
//
// trigger.dev 原始实现:
//
//	return this.#prismaClient.apiIntegrationVote.create({
//	  data: {
//	    user: { connect: { id: userId } },
//	    apiIdentifier: identifier,
//	  },
//	});
//
// 注意：trigger.dev 的原实现没有重复投票检查，但根据数据库唯一约束，
// 重复投票会抛出错误。我们在Go实现中主动检查，提供更好的用户体验。
func (s *Service) Call(ctx context.Context, req VoteRequest) (*VoteResponse, error) {
	// 1. 参数验证
	if req.UserID == "" {
		return nil, fmt.Errorf("userId is required")
	}
	if req.Identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}

	// 2. 检查是否已存在投票
	// 这是对 trigger.dev 行为的改进，避免数据库约束错误
	params := GetApiVoteByUserAndIdentifierParams{
		ApiIdentifier: req.Identifier,
		UserId:        req.UserID,
	}

	existingVote, err := s.queries.GetApiVoteByUserAndIdentifier(ctx, params)
	if err == nil {
		// 找到现有投票，返回现有记录
		return &VoteResponse{
			ApiIntegrationVote: &existingVote,
			IsNewVote:          false,
		}, nil
	}

	if err != pgx.ErrNoRows {
		// 数据库查询错误
		return nil, fmt.Errorf("failed to check existing vote: %w", err)
	}

	// 3. 创建新投票
	// 严格对齐 trigger.dev 的创建逻辑
	id := uuid.New().String() // 对齐 trigger.dev 的 cuid() 生成

	createParams := CreateApiVoteParams{
		ID:            id,
		ApiIdentifier: req.Identifier,
		UserId:        req.UserID,
	}

	newVote, err := s.queries.CreateApiVote(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create vote: %w", err)
	}

	return &VoteResponse{
		ApiIntegrationVote: &newVote,
		IsNewVote:          true,
	}, nil
}

// RemoveVote 移除投票
// 扩展功能，trigger.dev 中未实现但在实际应用中可能需要
func (s *Service) RemoveVote(ctx context.Context, userID, apiIdentifier string) error {
	if userID == "" || apiIdentifier == "" {
		return fmt.Errorf("userID and apiIdentifier are required")
	}

	params := DeleteApiVoteByUserAndIdentifierParams{
		ApiIdentifier: apiIdentifier,
		UserId:        userID,
	}

	err := s.queries.DeleteApiVoteByUserAndIdentifier(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to remove vote: %w", err)
	}

	return nil
}

// ConvertTimestamp 辅助方法：将 pgtype.Timestamp 转换为标准 time.Time
// 用于更好的 JSON 序列化和业务逻辑处理
func ConvertTimestamp(ts pgtype.Timestamp) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}
