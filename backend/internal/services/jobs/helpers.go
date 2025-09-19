package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUID 转换辅助函数

// uuidToPgUUID 将 uuid.UUID 转换为 pgtype.UUID
func uuidToPgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}

// pgUUIDToUUID 将 pgtype.UUID 转换为 uuid.UUID
func pgUUIDToUUID(pu pgtype.UUID) uuid.UUID {
	if !pu.Valid {
		return uuid.Nil
	}
	return pu.Bytes
}

// JSON 转换辅助函数

// jsonbToMap 将 JSONB 转换为 map[string]interface{}
func jsonbToMap(data []byte) map[string]interface{} {
	if len(data) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// mapToJsonb 将 map[string]interface{} 转换为 JSONB
func mapToJsonb(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	return json.Marshal(data)
}

// 验证函数

// validateRegisterJobRequest 验证注册作业请求
func (s *service) validateRegisterJobRequest(req RegisterJobRequest) error {
	if req.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if req.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if req.Version == "" {
		return fmt.Errorf("job version is required")
	}
	if req.Event.Name == "" {
		return fmt.Errorf("event name is required")
	}
	if req.Trigger.Type == "" {
		return fmt.Errorf("trigger type is required")
	}

	// 验证触发器类型
	if req.Trigger.Type != "static" && req.Trigger.Type != "scheduled" {
		return fmt.Errorf("invalid trigger type: %s", req.Trigger.Type)
	}

	// 验证 static 触发器必须有 rule
	if req.Trigger.Type == "static" && req.Trigger.Rule == nil {
		return fmt.Errorf("static trigger requires rule")
	}

	// 验证 scheduled 触发器必须有 schedule
	if req.Trigger.Type == "scheduled" && req.Trigger.Schedule == nil {
		return fmt.Errorf("scheduled trigger requires schedule")
	}

	return nil
}

// 作业管理辅助方法

// upsertJob 创建或更新作业
func (s *service) upsertJob(ctx context.Context, repo Repository, req RegisterJobRequest, endpointID uuid.UUID) (Jobs, error) {
	// 根据项目ID和组织ID来构造参数
	// 这里需要从 endpoint 获取相关信息，简化处理先使用默认值
	orgID := uuid.New()     // TODO: 从 endpoint 获取实际的 organization_id
	projectID := uuid.New() // TODO: 从 endpoint 获取实际的 project_id

	params := UpsertJobParams{
		Slug:           req.ID,
		Title:          req.Name,
		Internal:       req.Internal,
		OrganizationID: uuidToPgUUID(orgID),
		ProjectID:      uuidToPgUUID(projectID),
	}

	return repo.UpsertJob(ctx, params)
}

// upsertJobQueue 创建或更新作业队列
func (s *service) upsertJobQueue(ctx context.Context, repo Repository, queueConfig *QueueConfig, endpointID uuid.UUID) (JobQueues, error) {
	// 确定队列名称和最大并发数
	queueName := DefaultQueueName
	maxConcurrent := DefaultMaxConcurrentRuns

	if queueConfig != nil {
		if queueConfig.Name != "" {
			queueName = queueConfig.Name
		}
		if queueConfig.MaxConcurrent > 0 {
			maxConcurrent = queueConfig.MaxConcurrent
		}
	}

	// TODO: 从 endpoint 获取实际的 environment_id
	environmentID := uuid.New()

	params := UpsertJobQueueParams{
		Name:          queueName,
		EnvironmentID: uuidToPgUUID(environmentID),
		JobCount:      0,
		MaxJobs:       int32(maxConcurrent),
	}

	return repo.UpsertJobQueue(ctx, params)
}

// upsertJobVersion 创建或更新作业版本
func (s *service) upsertJobVersion(ctx context.Context, repo Repository, job Jobs, req RegisterJobRequest, endpointID uuid.UUID, queueID pgtype.UUID) (JobVersions, error) {
	// 构造事件规范
	eventSpec := map[string]interface{}{
		"name":   req.Event.Name,
		"source": req.Event.Source,
	}

	eventSpecJson, err := mapToJsonb(eventSpec)
	if err != nil {
		return JobVersions{}, fmt.Errorf("failed to marshal event specification: %w", err)
	}

	// 构造属性
	var propertiesJson []byte
	if req.Trigger.Properties != nil {
		propertiesJson, err = mapToJsonb(req.Trigger.Properties)
		if err != nil {
			return JobVersions{}, fmt.Errorf("failed to marshal properties: %w", err)
		}
	}

	// 确定开始位置
	startPosition := JobStartPositionINITIAL
	if req.StartPosition == "latest" {
		startPosition = JobStartPositionLATEST
	}

	// TODO: 从 endpoint 获取实际的环境信息
	environmentID := uuid.New()

	params := UpsertJobVersionParams{
		JobID:              job.ID,
		Version:            req.Version,
		EventSpecification: eventSpecJson,
		Properties:         propertiesJson,
		EndpointID:         uuidToPgUUID(endpointID),
		EnvironmentID:      uuidToPgUUID(environmentID),
		OrganizationID:     job.OrganizationID,
		ProjectID:          job.ProjectID,
		QueueID:            queueID,
		StartPosition:      startPosition,
		PreprocessRuns:     req.PreprocessRuns,
	}

	return repo.UpsertJobVersion(ctx, params)
}

// manageEventExamples 管理事件示例
func (s *service) manageEventExamples(ctx context.Context, repo Repository, jobVersionID pgtype.UUID, examples []EventExampleData) error {
	if len(examples) == 0 {
		return nil
	}

	var upsertedIDs []pgtype.UUID

	for _, example := range examples {
		var payloadJson []byte
		var err error

		if example.Payload != nil {
			payloadJson, err = mapToJsonb(example.Payload)
			if err != nil {
				return fmt.Errorf("failed to marshal example payload: %w", err)
			}
		}

		var icon pgtype.Text
		if example.Icon != "" {
			icon = pgtype.Text{String: example.Icon, Valid: true}
		}

		params := UpsertEventExampleParams{
			JobVersionID: jobVersionID,
			Slug:         example.ID,
			Name:         example.Name,
			Icon:         icon,
			Payload:      payloadJson,
		}

		result, err := repo.UpsertEventExample(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to upsert event example: %w", err)
		}

		upsertedIDs = append(upsertedIDs, result.ID)
	}

	// 删除不在列表中的示例
	if len(upsertedIDs) > 0 {
		deleteParams := DeleteEventExamplesNotInListParams{
			JobVersionID: jobVersionID,
			Column2:      upsertedIDs, // sqlc 生成的参数名可能不直观
		}

		if err := repo.DeleteEventExamplesNotInList(ctx, deleteParams); err != nil {
			return fmt.Errorf("failed to delete unused event examples: %w", err)
		}
	}

	return nil
}

// manageJobAlias 管理作业别名
func (s *service) manageJobAlias(ctx context.Context, repo Repository, job Jobs, jobVersion JobVersions, endpointID uuid.UUID) error {
	// 检查是否有更新的版本
	// TODO: 从 jobVersion 获取实际的 environment_id
	environmentID := uuid.New()

	laterCount, err := repo.CountLaterJobVersions(ctx, CountLaterJobVersionsParams{
		JobID:         job.ID,
		EnvironmentID: uuidToPgUUID(environmentID),
		Version:       jobVersion.Version,
	})
	if err != nil {
		return fmt.Errorf("failed to count later job versions: %w", err)
	}

	// 如果没有更新的版本，更新 latest 别名
	if laterCount == 0 {
		params := UpsertJobAliasParams{
			JobID:         job.ID,
			VersionID:     jobVersion.ID,
			EnvironmentID: uuidToPgUUID(environmentID),
			Name:          LatestAliasName,
			Value:         jobVersion.Version,
		}

		_, err := repo.UpsertJobAlias(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to upsert job alias: %w", err)
		}
	}

	return nil
}
