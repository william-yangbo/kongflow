package endpoints

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"kongflow/backend/internal/services/endpointapi"
	"kongflow/backend/internal/services/endpoints/queue"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
)

// Service 端点服务接口
type Service interface {
	CreateEndpoint(ctx context.Context, req EndpointRequest) (*EndpointResponse, error)
	UpsertEndpoint(ctx context.Context, req UpsertEndpointRequest) (*EndpointResponse, error)
	IndexEndpoint(ctx context.Context, req IndexEndpointRequest) (*IndexEndpointResponse, error)
	GetEndpoint(ctx context.Context, id uuid.UUID) (*EndpointResponse, error)
	DeleteEndpoint(ctx context.Context, id uuid.UUID) error
}

// EndpointRequest 创建端点请求
type EndpointRequest struct {
	Slug                   string    `json:"slug"`
	URL                    string    `json:"url"`
	IndexingHookIdentifier string    `json:"indexing_hook_identifier"`
	EnvironmentID          uuid.UUID `json:"environment_id"`
	OrganizationID         uuid.UUID `json:"organization_id"`
	ProjectID              uuid.UUID `json:"project_id"`
}

// UpsertEndpointRequest Upsert端点请求
type UpsertEndpointRequest struct {
	Slug                   string    `json:"slug" validate:"required"`
	URL                    string    `json:"url" validate:"required"`
	EnvironmentID          uuid.UUID `json:"environment_id" validate:"required"`
	OrganizationID         uuid.UUID `json:"organization_id" validate:"required"`
	ProjectID              uuid.UUID `json:"project_id" validate:"required"`
	IndexingHookIdentifier string    `json:"indexing_hook_identifier,omitempty"`
}

// IndexEndpointRequest 端点索引请求
type IndexEndpointRequest struct {
	EndpointID uuid.UUID                 `json:"endpoint_id" validate:"required"`
	Source     queue.EndpointIndexSource `json:"source" validate:"required"`
	Reason     string                    `json:"reason,omitempty"`
	SourceData map[string]interface{}    `json:"source_data,omitempty"`
}

// IndexEndpointResponse 端点索引响应
type IndexEndpointResponse struct {
	IndexID uuid.UUID   `json:"index_id"`
	Stats   IndexStats  `json:"stats"`
	Status  IndexStatus `json:"status"`
}

// IndexStats 索引统计信息
type IndexStats struct {
	Jobs             int `json:"jobs"`
	Sources          int `json:"sources"`
	DynamicTriggers  int `json:"dynamic_triggers"`
	DynamicSchedules int `json:"dynamic_schedules"`
}

// IndexStatus 索引状态
type IndexStatus string

const (
	IndexStatusPending   IndexStatus = "PENDING"
	IndexStatusCompleted IndexStatus = "COMPLETED"
	IndexStatusFailed    IndexStatus = "FAILED"
)

// EndpointResponse 端点响应
type EndpointResponse struct {
	ID                     uuid.UUID `json:"id"`
	Slug                   string    `json:"slug"`
	URL                    string    `json:"url"`
	IndexingHookIdentifier string    `json:"indexing_hook_identifier"`
	EnvironmentID          uuid.UUID `json:"environment_id"`
	OrganizationID         uuid.UUID `json:"organization_id"`
	ProjectID              uuid.UUID `json:"project_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// service 实现
type service struct {
	repo         Repository
	apiClient    endpointapi.EndpointAPIClient
	queueService queue.QueueService
	logger       *slog.Logger
}

// NewService 创建服务实例
func NewService(repo Repository, apiClient endpointapi.EndpointAPIClient, queueService queue.QueueService, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		repo:         repo,
		apiClient:    apiClient,
		queueService: queueService,
		logger:       logger,
	}
}

// CreateEndpoint 创建端点 (增强版，对齐trigger.dev)
func (s *service) CreateEndpoint(ctx context.Context, req EndpointRequest) (*EndpointResponse, error) {
	logger := s.logger.With("operation", "create_endpoint", "slug", req.Slug, "url", req.URL)
	logger.Info("Starting endpoint creation")

	// 1. 输入验证
	if err := s.validateCreateRequest(req); err != nil {
		logger.Error("Invalid request", "error", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 2. 端点可达性验证 (对齐trigger.dev ping逻辑)
	pingResp, err := s.apiClient.Ping(ctx)
	if err != nil {
		logger.Error("Endpoint ping failed", "error", err)
		return nil, fmt.Errorf("endpoint ping failed: %w", err)
	}
	if !pingResp.OK {
		logger.Error("Endpoint ping returned error", "error", pingResp.Error)
		return nil, fmt.Errorf("endpoint ping failed: %s", pingResp.Error)
	}

	// 3. 生成indexingHookIdentifier (对齐trigger.dev逻辑)
	hookIdentifier := req.IndexingHookIdentifier
	if hookIdentifier == "" {
		hookIdentifier = s.generateHookIdentifier()
	}

	// 4. 事务性创建端点
	endpoint, err := s.createEndpointInTransaction(ctx, req, hookIdentifier)
	if err != nil {
		logger.Error("Failed to create endpoint", "error", err)
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}

	// 5. 异步触发索引 (对齐trigger.dev自动索引)
	if _, err := s.enqueueIndexEndpoint(ctx, endpoint.ID, queue.EndpointIndexSourceInternal, "Auto-triggered after endpoint creation"); err != nil {
		// 记录警告但不失败创建
		logger.Warn("Failed to enqueue index endpoint", "endpoint_id", endpoint.ID, "error", err)
	}

	logger.Info("Endpoint created successfully", "endpoint_id", endpoint.ID)
	return endpoint, nil
}

// validateCreateRequest 验证创建请求
func (s *service) validateCreateRequest(req EndpointRequest) error {
	if req.Slug == "" {
		return errors.New("slug is required")
	}
	if req.URL == "" {
		return errors.New("url is required")
	}
	if req.EnvironmentID == uuid.Nil {
		return errors.New("environment_id is required")
	}
	if req.OrganizationID == uuid.Nil {
		return errors.New("organization_id is required")
	}
	if req.ProjectID == uuid.Nil {
		return errors.New("project_id is required")
	}
	return nil
}

// generateHookIdentifier 生成Hook标识符 (对齐trigger.dev customAlphabet)
func (s *service) generateHookIdentifier() string {
	const charset = "0123456789abcdefghijklmnopqrstuvxyz"
	const length = 10

	b := make([]byte, length)
	for i := range b {
		randByte := make([]byte, 1)
		rand.Read(randByte)
		b[i] = charset[randByte[0]%byte(len(charset))]
	}
	return string(b)
}

// createEndpointInTransaction 在事务中创建端点
func (s *service) createEndpointInTransaction(ctx context.Context, req EndpointRequest, hookIdentifier string) (*EndpointResponse, error) {
	params := CreateEndpointParams{
		Slug:                   req.Slug,
		Url:                    req.URL,
		IndexingHookIdentifier: hookIdentifier,
		EnvironmentID:          uuidToPgtype(req.EnvironmentID),
		OrganizationID:         uuidToPgtype(req.OrganizationID),
		ProjectID:              uuidToPgtype(req.ProjectID),
	}

	endpoint, err := s.repo.CreateEndpoint(ctx, params)
	if err != nil {
		return nil, err
	}

	return &EndpointResponse{
		ID:                     pgtypeToUUID(endpoint.ID),
		Slug:                   endpoint.Slug,
		URL:                    endpoint.Url,
		IndexingHookIdentifier: endpoint.IndexingHookIdentifier,
		EnvironmentID:          pgtypeToUUID(endpoint.EnvironmentID),
		OrganizationID:         pgtypeToUUID(endpoint.OrganizationID),
		ProjectID:              pgtypeToUUID(endpoint.ProjectID),
		CreatedAt:              endpoint.CreatedAt.Time,
		UpdatedAt:              endpoint.UpdatedAt.Time,
	}, nil
}

// enqueueIndexEndpoint 将端点索引任务加入队列
func (s *service) enqueueIndexEndpoint(ctx context.Context, endpointID uuid.UUID, source queue.EndpointIndexSource, reason string) (*rivertype.JobInsertResult, error) {
	req := queue.EnqueueIndexEndpointRequest{
		EndpointID: endpointID,
		Source:     source,
		Reason:     reason,
	}

	return s.queueService.EnqueueIndexEndpoint(ctx, req)
} // UpsertEndpoint 创建或更新端点 (对齐trigger.dev endpoint.upsert)
func (s *service) UpsertEndpoint(ctx context.Context, req UpsertEndpointRequest) (*EndpointResponse, error) {
	logger := s.logger.With("operation", "upsert_endpoint", "slug", req.Slug, "url", req.URL)
	logger.Info("Starting endpoint upsert")

	// 1. 输入验证
	if err := s.validateUpsertRequest(req); err != nil {
		logger.Error("Invalid request", "error", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// 2. 端点可达性验证
	pingResp, err := s.apiClient.Ping(ctx)
	if err != nil {
		logger.Error("Endpoint ping failed", "error", err)
		return nil, fmt.Errorf("endpoint ping failed: %w", err)
	}
	if !pingResp.OK {
		logger.Error("Endpoint ping returned error", "error", pingResp.Error)
		return nil, fmt.Errorf("endpoint ping failed: %s", pingResp.Error)
	}

	// 3. 在事务中处理upsert逻辑
	endpoint, err := s.upsertEndpointInTransaction(ctx, req)
	if err != nil {
		logger.Error("Failed to upsert endpoint", "error", err)
		return nil, fmt.Errorf("failed to upsert endpoint: %w", err)
	}

	// 4. 异步触发索引
	if _, err := s.enqueueIndexEndpoint(ctx, endpoint.ID, queue.EndpointIndexSourceInternal, "Auto-triggered after endpoint upsert"); err != nil {
		logger.Warn("Failed to enqueue index endpoint", "endpoint_id", endpoint.ID, "error", err)
	}

	logger.Info("Endpoint upserted successfully", "endpoint_id", endpoint.ID)
	return endpoint, nil
}

// validateUpsertRequest 验证upsert请求
func (s *service) validateUpsertRequest(req UpsertEndpointRequest) error {
	if req.Slug == "" {
		return errors.New("slug is required")
	}
	if req.URL == "" {
		return errors.New("url is required")
	}
	if req.EnvironmentID == uuid.Nil {
		return errors.New("environment_id is required")
	}
	if req.OrganizationID == uuid.Nil {
		return errors.New("organization_id is required")
	}
	if req.ProjectID == uuid.Nil {
		return errors.New("project_id is required")
	}
	return nil
}

// upsertEndpointInTransaction 在事务中执行upsert逻辑
func (s *service) upsertEndpointInTransaction(ctx context.Context, req UpsertEndpointRequest) (*EndpointResponse, error) {
	// TODO: 需要实现事务包装器，目前先用简单逻辑

	// 1. 尝试查找现有端点
	existingEndpoint, err := s.repo.GetEndpointBySlug(ctx, req.EnvironmentID, req.Slug)
	if err != nil {
		// 如果没找到，创建新端点
		hookIdentifier := req.IndexingHookIdentifier
		if hookIdentifier == "" {
			hookIdentifier = s.generateHookIdentifier()
		}

		createParams := CreateEndpointParams{
			Slug:                   req.Slug,
			Url:                    req.URL,
			IndexingHookIdentifier: hookIdentifier,
			EnvironmentID:          uuidToPgtype(req.EnvironmentID),
			OrganizationID:         uuidToPgtype(req.OrganizationID),
			ProjectID:              uuidToPgtype(req.ProjectID),
		}

		endpoint, err := s.repo.CreateEndpoint(ctx, createParams)
		if err != nil {
			return nil, err
		}

		return &EndpointResponse{
			ID:                     pgtypeToUUID(endpoint.ID),
			Slug:                   endpoint.Slug,
			URL:                    endpoint.Url,
			IndexingHookIdentifier: endpoint.IndexingHookIdentifier,
			EnvironmentID:          pgtypeToUUID(endpoint.EnvironmentID),
			OrganizationID:         pgtypeToUUID(endpoint.OrganizationID),
			ProjectID:              pgtypeToUUID(endpoint.ProjectID),
			CreatedAt:              endpoint.CreatedAt.Time,
			UpdatedAt:              endpoint.UpdatedAt.Time,
		}, nil
	} else {
		// 如果找到了，更新URL
		endpoint, err := s.repo.UpdateEndpointURL(ctx, pgtypeToUUID(existingEndpoint.ID), req.URL)
		if err != nil {
			return nil, err
		}

		return &EndpointResponse{
			ID:                     pgtypeToUUID(endpoint.ID),
			Slug:                   endpoint.Slug,
			URL:                    endpoint.Url,
			IndexingHookIdentifier: endpoint.IndexingHookIdentifier,
			EnvironmentID:          pgtypeToUUID(endpoint.EnvironmentID),
			OrganizationID:         pgtypeToUUID(endpoint.OrganizationID),
			ProjectID:              pgtypeToUUID(endpoint.ProjectID),
			CreatedAt:              endpoint.CreatedAt.Time,
			UpdatedAt:              endpoint.UpdatedAt.Time,
		}, nil
	}
}

// IndexEndpoint 触发端点索引
func (s *service) IndexEndpoint(ctx context.Context, req IndexEndpointRequest) (*IndexEndpointResponse, error) {
	logger := s.logger.With("operation", "index_endpoint", "endpoint_id", req.EndpointID)
	logger.Info("Starting endpoint indexing")

	// 1. 输入验证
	if req.EndpointID == uuid.Nil {
		return nil, errors.New("endpoint_id is required")
	}

	// 2. 验证端点是否存在
	_, err := s.repo.GetEndpointByID(ctx, req.EndpointID)
	if err != nil {
		logger.Error("Endpoint not found", "error", err)
		return nil, fmt.Errorf("endpoint not found: %w", err)
	}

	// 3. 异步触发索引
	source := req.Source
	if source == "" {
		source = queue.EndpointIndexSourceManual
	}

	reason := req.Reason
	if reason == "" {
		reason = "Manual indexing request"
	}

	result, err := s.enqueueIndexEndpoint(ctx, req.EndpointID, source, reason)
	if err != nil {
		logger.Error("Failed to enqueue index endpoint", "error", err)
		return nil, fmt.Errorf("failed to enqueue index endpoint: %w", err)
	}

	logger.Info("Endpoint indexing enqueued successfully", "job_id", result.Job.ID)
	return &IndexEndpointResponse{
		IndexID: req.EndpointID,
		Stats: IndexStats{
			Jobs:             0, // 这些需要从实际索引结果获取
			Sources:          0,
			DynamicTriggers:  0,
			DynamicSchedules: 0,
		},
		Status: IndexStatusPending,
	}, nil
}

// GetEndpoint 获取端点
func (s *service) GetEndpoint(ctx context.Context, id uuid.UUID) (*EndpointResponse, error) {
	endpoint, err := s.repo.GetEndpointByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &EndpointResponse{
		ID:                     pgtypeToUUID(endpoint.ID),
		Slug:                   endpoint.Slug,
		URL:                    endpoint.Url,
		IndexingHookIdentifier: endpoint.IndexingHookIdentifier,
		EnvironmentID:          pgtypeToUUID(endpoint.EnvironmentID),
		OrganizationID:         pgtypeToUUID(endpoint.OrganizationID),
		ProjectID:              pgtypeToUUID(endpoint.ProjectID),
		CreatedAt:              endpoint.CreatedAt.Time,
		UpdatedAt:              endpoint.UpdatedAt.Time,
	}, nil
}

// DeleteEndpoint 删除端点
func (s *service) DeleteEndpoint(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEndpoint(ctx, id)
}
