package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kongflow/backend/internal/services/apiauth"
	"kongflow/backend/internal/services/events"

	"github.com/google/uuid"
)

// 常量定义，对齐 trigger.dev
const (
	DefaultMaxConcurrentRuns = 100
	DefaultQueueName         = "default"
	LatestAliasName          = "latest"
)

// Service Jobs 服务接口，严格对齐 trigger.dev 的功能
type Service interface {
	// 核心作业管理 - 对齐 RegisterJobService
	RegisterJob(ctx context.Context, endpointID uuid.UUID, req RegisterJobRequest) (*JobResponse, error)
	GetJob(ctx context.Context, id uuid.UUID) (*JobResponse, error)
	GetJobBySlug(ctx context.Context, projectID uuid.UUID, slug string) (*JobResponse, error)
	ListJobs(ctx context.Context, params ListJobsParams) (*ListJobsResponse, error)
	DeleteJob(ctx context.Context, id uuid.UUID) error

	// 作业版本管理
	GetJobVersion(ctx context.Context, id uuid.UUID) (*JobVersionResponse, error)
	ListJobVersions(ctx context.Context, jobID uuid.UUID) (*ListJobVersionsResponse, error)

	// 作业队列管理
	GetJobQueue(ctx context.Context, environmentID uuid.UUID, name string) (*JobQueueResponse, error)
	CreateJobQueue(ctx context.Context, req CreateJobQueueRequest) (*JobQueueResponse, error)

	// 作业测试 - 对齐 TestJobService
	TestJob(ctx context.Context, req TestJobRequest) (*TestJobResponse, error)
}

// RegisterJobRequest 作业注册请求，对齐 trigger.dev 的 JobMetadata
type RegisterJobRequest struct {
	ID             string                     `json:"id" validate:"required"`
	Name           string                     `json:"name" validate:"required"`
	Version        string                     `json:"version" validate:"required"`
	Internal       bool                       `json:"internal"`
	Event          EventSpecification         `json:"event" validate:"required"`
	Trigger        TriggerMetadata            `json:"trigger" validate:"required"`
	Queue          *QueueConfig               `json:"queue,omitempty"`
	Integrations   map[string]IntegrationConf `json:"integrations,omitempty"`
	StartPosition  string                     `json:"startPosition,omitempty"`
	PreprocessRuns bool                       `json:"preprocessRuns"`
}

// EventSpecification 事件规范
type EventSpecification struct {
	Name     string             `json:"name" validate:"required"`
	Source   string             `json:"source,omitempty"`
	Examples []EventExampleData `json:"examples,omitempty"`
}

// TriggerMetadata 触发器元数据
type TriggerMetadata struct {
	Type       string                 `json:"type" validate:"required"` // "static" | "scheduled"
	Rule       *TriggerRule           `json:"rule,omitempty"`
	Schedule   *ScheduleMetadata      `json:"schedule,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// TriggerRule 触发规则
type TriggerRule struct {
	Event   string                 `json:"event"`
	Source  string                 `json:"source"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ScheduleMetadata 调度元数据
type ScheduleMetadata struct {
	Cron     string `json:"cron,omitempty"`
	Interval string `json:"interval,omitempty"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	Name          string `json:"name,omitempty"`
	MaxConcurrent int    `json:"maxConcurrent,omitempty"`
}

// IntegrationConf 集成配置
type IntegrationConf struct {
	ID       string              `json:"id"`
	Metadata IntegrationMetadata `json:"metadata"`
}

// IntegrationMetadata 集成元数据
type IntegrationMetadata struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Instructions string `json:"instructions,omitempty"`
}

// EventExampleData 事件示例数据
type EventExampleData struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Icon    string                 `json:"icon,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// ListJobsParams 列表查询参数
type ListJobsParams struct {
	ProjectID uuid.UUID `json:"project_id"`
	Limit     int32     `json:"limit"`
	Offset    int32     `json:"offset"`
}

// TestJobRequest 作业测试请求
type TestJobRequest struct {
	EnvironmentID uuid.UUID              `json:"environment_id" validate:"required"`
	VersionID     uuid.UUID              `json:"version_id" validate:"required"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

// CreateJobQueueRequest 创建队列请求
type CreateJobQueueRequest struct {
	Name          string    `json:"name" validate:"required"`
	EnvironmentID uuid.UUID `json:"environment_id" validate:"required"`
	MaxJobs       int32     `json:"max_jobs"`
}

// Response DTOs
type JobResponse struct {
	ID             uuid.UUID            `json:"id"`
	Slug           string               `json:"slug"`
	Title          string               `json:"title"`
	Internal       bool                 `json:"internal"`
	OrganizationID uuid.UUID            `json:"organization_id"`
	ProjectID      uuid.UUID            `json:"project_id"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	CurrentVersion *JobVersionResponse  `json:"current_version,omitempty"`
	Versions       []JobVersionResponse `json:"versions,omitempty"`
}

type JobVersionResponse struct {
	ID                 uuid.UUID              `json:"id"`
	Version            string                 `json:"version"`
	EventSpecification map[string]interface{} `json:"event_specification"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
	StartPosition      JobStartPosition       `json:"start_position"`
	PreprocessRuns     bool                   `json:"preprocess_runs"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type JobQueueResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	JobCount      int32     `json:"job_count"`
	MaxJobs       int32     `json:"max_jobs"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ListJobsResponse struct {
	Jobs    []JobResponse `json:"jobs"`
	Total   int64         `json:"total"`
	Limit   int32         `json:"limit"`
	Offset  int32         `json:"offset"`
	HasMore bool          `json:"has_more"`
}

type ListJobVersionsResponse struct {
	Versions []JobVersionResponse `json:"versions"`
	Total    int                  `json:"total"`
}

type TestJobResponse struct {
	RunID   uuid.UUID `json:"run_id"`
	EventID uuid.UUID `json:"event_id"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

// service 实现
type service struct {
	repo      Repository
	eventsSvc events.Service
	logger    *slog.Logger
}

// NewService 创建服务实例
func NewService(repo Repository, eventsSvc events.Service, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		repo:      repo,
		eventsSvc: eventsSvc,
		logger:    logger,
	}
}

// RegisterJob 注册作业，严格对齐 trigger.dev RegisterJobService.call
func (s *service) RegisterJob(ctx context.Context, endpointID uuid.UUID, req RegisterJobRequest) (*JobResponse, error) {
	logger := s.logger.With(
		"operation", "register_job",
		"job_id", req.ID,
		"job_name", req.Name,
		"version", req.Version,
		"endpoint_id", endpointID.String(),
	)
	logger.Info("Starting job registration")

	// 输入验证
	if err := s.validateRegisterJobRequest(req); err != nil {
		logger.Error("Invalid register job request", "error", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var result *JobResponse
	err := s.repo.WithTx(ctx, func(txRepo Repository) error {
		// 1. Upsert Job - 对齐 trigger.dev 的 #upsertJob 逻辑
		job, err := s.upsertJob(ctx, txRepo, req, endpointID)
		if err != nil {
			return fmt.Errorf("failed to upsert job: %w", err)
		}

		// 2. Upsert JobQueue - 对齐 queue 管理逻辑
		jobQueue, err := s.upsertJobQueue(ctx, txRepo, req.Queue, endpointID)
		if err != nil {
			return fmt.Errorf("failed to upsert job queue: %w", err)
		}

		// 3. Upsert JobVersion - 核心版本管理逻辑
		jobVersion, err := s.upsertJobVersion(ctx, txRepo, job, req, endpointID, jobQueue.ID)
		if err != nil {
			return fmt.Errorf("failed to upsert job version: %w", err)
		}

		// 4. 管理 EventExamples - 对齐示例管理逻辑
		if err := s.manageEventExamples(ctx, txRepo, jobVersion.ID, req.Event.Examples); err != nil {
			return fmt.Errorf("failed to manage event examples: %w", err)
		}

		// 5. 管理 JobAlias - 对齐别名管理逻辑（如果是最新版本）
		if err := s.manageJobAlias(ctx, txRepo, job, jobVersion, endpointID); err != nil {
			return fmt.Errorf("failed to manage job alias: %w", err)
		}

		// 构造响应
		result = &JobResponse{
			ID:             pgUUIDToUUID(job.ID),
			Slug:           job.Slug,
			Title:          job.Title,
			Internal:       job.Internal,
			OrganizationID: pgUUIDToUUID(job.OrganizationID),
			ProjectID:      pgUUIDToUUID(job.ProjectID),
			CreatedAt:      job.CreatedAt.Time,
			UpdatedAt:      job.UpdatedAt.Time,
			CurrentVersion: &JobVersionResponse{
				ID:                 pgUUIDToUUID(jobVersion.ID),
				Version:            jobVersion.Version,
				EventSpecification: jsonbToMap(jobVersion.EventSpecification),
				Properties:         jsonbToMap(jobVersion.Properties),
				StartPosition:      jobVersion.StartPosition,
				PreprocessRuns:     jobVersion.PreprocessRuns,
				CreatedAt:          jobVersion.CreatedAt.Time,
				UpdatedAt:          jobVersion.UpdatedAt.Time,
			},
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to register job", "error", err)
		return nil, err
	}

	logger.Info("Job registered successfully", "job_id", result.ID)
	return result, nil
}

// GetJob 获取作业详情
func (s *service) GetJob(ctx context.Context, id uuid.UUID) (*JobResponse, error) {
	logger := s.logger.With("operation", "get_job", "job_id", id.String())

	job, err := s.repo.GetJobByID(ctx, uuidToPgUUID(id))
	if err != nil {
		logger.Error("Failed to get job", "error", err)
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &JobResponse{
		ID:             pgUUIDToUUID(job.ID),
		Slug:           job.Slug,
		Title:          job.Title,
		Internal:       job.Internal,
		OrganizationID: pgUUIDToUUID(job.OrganizationID),
		ProjectID:      pgUUIDToUUID(job.ProjectID),
		CreatedAt:      job.CreatedAt.Time,
		UpdatedAt:      job.UpdatedAt.Time,
	}, nil
}

// GetJobBySlug 根据 slug 获取作业
func (s *service) GetJobBySlug(ctx context.Context, projectID uuid.UUID, slug string) (*JobResponse, error) {
	logger := s.logger.With("operation", "get_job_by_slug", "project_id", projectID.String(), "slug", slug)

	job, err := s.repo.GetJobBySlug(ctx, uuidToPgUUID(projectID), slug)
	if err != nil {
		logger.Error("Failed to get job by slug", "error", err)
		return nil, fmt.Errorf("failed to get job by slug: %w", err)
	}

	return &JobResponse{
		ID:             pgUUIDToUUID(job.ID),
		Slug:           job.Slug,
		Title:          job.Title,
		Internal:       job.Internal,
		OrganizationID: pgUUIDToUUID(job.OrganizationID),
		ProjectID:      pgUUIDToUUID(job.ProjectID),
		CreatedAt:      job.CreatedAt.Time,
		UpdatedAt:      job.UpdatedAt.Time,
	}, nil
}

// ListJobs 列出作业
func (s *service) ListJobs(ctx context.Context, params ListJobsParams) (*ListJobsResponse, error) {
	jobs, err := s.repo.ListJobsByProject(ctx, ListJobsByProjectParams{
		ProjectID: uuidToPgUUID(params.ProjectID),
		Limit:     params.Limit,
		Offset:    params.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	total, err := s.repo.CountJobsByProject(ctx, uuidToPgUUID(params.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	var jobResponses []JobResponse
	for _, job := range jobs {
		jobResponses = append(jobResponses, JobResponse{
			ID:             pgUUIDToUUID(job.ID),
			Slug:           job.Slug,
			Title:          job.Title,
			Internal:       job.Internal,
			OrganizationID: pgUUIDToUUID(job.OrganizationID),
			ProjectID:      pgUUIDToUUID(job.ProjectID),
			CreatedAt:      job.CreatedAt.Time,
			UpdatedAt:      job.UpdatedAt.Time,
		})
	}

	return &ListJobsResponse{
		Jobs:    jobResponses,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: params.Offset+params.Limit < int32(total),
	}, nil
}

// DeleteJob 删除作业
func (s *service) DeleteJob(ctx context.Context, id uuid.UUID) error {
	logger := s.logger.With("operation", "delete_job", "job_id", id.String())

	err := s.repo.DeleteJob(ctx, uuidToPgUUID(id))
	if err != nil {
		logger.Error("Failed to delete job", "error", err)
		return fmt.Errorf("failed to delete job: %w", err)
	}

	logger.Info("Job deleted successfully")
	return nil
}

// GetJobVersion 获取作业版本
func (s *service) GetJobVersion(ctx context.Context, id uuid.UUID) (*JobVersionResponse, error) {
	version, err := s.repo.GetJobVersionByID(ctx, uuidToPgUUID(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get job version: %w", err)
	}

	return &JobVersionResponse{
		ID:                 pgUUIDToUUID(version.ID),
		Version:            version.Version,
		EventSpecification: jsonbToMap(version.EventSpecification),
		Properties:         jsonbToMap(version.Properties),
		StartPosition:      version.StartPosition,
		PreprocessRuns:     version.PreprocessRuns,
		CreatedAt:          version.CreatedAt.Time,
		UpdatedAt:          version.UpdatedAt.Time,
	}, nil
}

// ListJobVersions 列出作业版本
func (s *service) ListJobVersions(ctx context.Context, jobID uuid.UUID) (*ListJobVersionsResponse, error) {
	versions, err := s.repo.ListJobVersionsByJob(ctx, ListJobVersionsByJobParams{
		JobID:  uuidToPgUUID(jobID),
		Limit:  100, // 默认限制
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list job versions: %w", err)
	}

	var versionResponses []JobVersionResponse
	for _, version := range versions {
		versionResponses = append(versionResponses, JobVersionResponse{
			ID:                 pgUUIDToUUID(version.ID),
			Version:            version.Version,
			EventSpecification: jsonbToMap(version.EventSpecification),
			Properties:         jsonbToMap(version.Properties),
			StartPosition:      version.StartPosition,
			PreprocessRuns:     version.PreprocessRuns,
			CreatedAt:          version.CreatedAt.Time,
			UpdatedAt:          version.UpdatedAt.Time,
		})
	}

	return &ListJobVersionsResponse{
		Versions: versionResponses,
		Total:    len(versionResponses),
	}, nil
}

// GetJobQueue 获取作业队列
func (s *service) GetJobQueue(ctx context.Context, environmentID uuid.UUID, name string) (*JobQueueResponse, error) {
	queue, err := s.repo.GetJobQueueByName(ctx, GetJobQueueByNameParams{
		EnvironmentID: uuidToPgUUID(environmentID),
		Name:          name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get job queue: %w", err)
	}

	return &JobQueueResponse{
		ID:            pgUUIDToUUID(queue.ID),
		Name:          queue.Name,
		EnvironmentID: pgUUIDToUUID(queue.EnvironmentID),
		JobCount:      queue.JobCount,
		MaxJobs:       queue.MaxJobs,
		CreatedAt:     queue.CreatedAt.Time,
		UpdatedAt:     queue.UpdatedAt.Time,
	}, nil
}

// CreateJobQueue 创建作业队列
func (s *service) CreateJobQueue(ctx context.Context, req CreateJobQueueRequest) (*JobQueueResponse, error) {
	maxJobs := req.MaxJobs
	if maxJobs <= 0 {
		maxJobs = DefaultMaxConcurrentRuns
	}

	queue, err := s.repo.CreateJobQueue(ctx, CreateJobQueueParams{
		Name:          req.Name,
		EnvironmentID: uuidToPgUUID(req.EnvironmentID),
		JobCount:      0,
		MaxJobs:       maxJobs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create job queue: %w", err)
	}

	return &JobQueueResponse{
		ID:            pgUUIDToUUID(queue.ID),
		Name:          queue.Name,
		EnvironmentID: pgUUIDToUUID(queue.EnvironmentID),
		JobCount:      queue.JobCount,
		MaxJobs:       queue.MaxJobs,
		CreatedAt:     queue.CreatedAt.Time,
		UpdatedAt:     queue.UpdatedAt.Time,
	}, nil
}

// TestJob 测试作业，对齐 trigger.dev TestJobService
func (s *service) TestJob(ctx context.Context, req TestJobRequest) (*TestJobResponse, error) {
	logger := s.logger.With(
		"operation", "test_job",
		"version_id", req.VersionID.String(),
		"environment_id", req.EnvironmentID.String(),
	)
	logger.Info("Starting job test")

	// 获取作业版本信息
	version, err := s.repo.GetJobVersionByID(ctx, uuidToPgUUID(req.VersionID))
	if err != nil {
		logger.Error("Failed to get job version", "error", err)
		return nil, fmt.Errorf("failed to get job version: %w", err)
	}

	// 解析事件规范
	eventSpec := jsonbToMap(version.EventSpecification)
	eventName, ok := eventSpec["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid event specification: missing name")
	}

	// 获取事件源（可选）
	eventSource := "trigger.dev"
	if source, exists := eventSpec["source"].(string); exists && source != "" {
		eventSource = source
	}

	// 生成测试事件ID
	eventID := uuid.New().String()

	// 构造事件上下文，标记为测试事件
	contextData := map[string]interface{}{
		"test":        true,
		"job_version": version.Version,
		"source":      "job_test",
	}

	// 使用 payload，如果没有则使用空 map
	payload := req.Payload
	if payload == nil {
		payload = make(map[string]interface{})
	}

	// 创建发送事件请求
	sendEventReq := &events.SendEventRequest{
		ID:      eventID,
		Name:    eventName,
		Source:  eventSource,
		Payload: payload,
		Context: contextData,
	}

	// 创建发送事件选项
	sendEventOpts := &events.SendEventOptions{
		// 测试事件立即投递
		DeliverAt: nil,
	}

	// 构造认证环境（需要从 EnvironmentID 获取）
	// TODO: 这里需要从实际的环境服务获取环境信息
	// 暂时使用模拟的环境信息
	authenticatedEnv := &apiauth.AuthenticatedEnvironment{
		Environment: apiauth.RuntimeEnvironment{
			ID:             uuidToPgUUID(req.EnvironmentID),
			OrganizationID: uuidToPgUUID(uuid.New()), // TODO: 从环境服务获取真实的组织ID
			ProjectID:      uuidToPgUUID(uuid.New()), // TODO: 从环境服务获取真实的项目ID
		},
	}

	// 通过 Events 服务创建事件记录
	eventRecord, err := s.eventsSvc.IngestSendEvent(ctx, authenticatedEnv, sendEventReq, sendEventOpts)
	if err != nil {
		logger.Error("Failed to create event record", "error", err)
		return nil, fmt.Errorf("failed to create event record: %w", err)
	}

	logger.Info("Job test completed successfully",
		"event_id", eventRecord.EventID,
		"event_record_id", eventRecord.ID,
		"version_id", version.ID.String(),
		"event_name", eventName)

	// 解析事件记录ID为UUID
	eventUUID, parseErr := uuid.Parse(eventRecord.EventID)
	if parseErr != nil {
		logger.Error("Failed to parse event ID", "error", parseErr)
		return nil, fmt.Errorf("failed to parse event ID: %w", parseErr)
	}

	return &TestJobResponse{
		RunID:   uuid.New(), // TODO: 等待 Runs 服务集成后返回真实的 run ID
		EventID: eventUUID,
		Status:  "pending",
		Message: "Test job submitted successfully",
	}, nil
}

// 内部辅助方法
