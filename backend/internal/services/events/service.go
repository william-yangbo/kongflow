package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"kongflow/backend/internal/services/apiauth"
	"kongflow/backend/internal/services/events/queue"
	"kongflow/backend/internal/shared"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// determineIfTestEvent 根据环境和选项确定事件是否为测试事件
// 对齐 trigger.dev 的测试事件识别逻辑
func determineIfTestEvent(env *apiauth.AuthenticatedEnvironment, event *SendEventRequest, opts *SendEventOptions) bool {
	// 1. 显式测试标志优先级最高
	if opts != nil && opts.Test != nil {
		return *opts.Test
	}

	// 2. 开发环境默认为测试事件
	if env.Environment.Type == apiauth.EnvironmentTypeDevelopment {
		return true
	}

	// 3. API Key 前缀判断 (tr_dev_ 或 pk_dev_)
	if strings.HasPrefix(env.Environment.APIKey, "tr_dev_") ||
		strings.HasPrefix(env.Environment.APIKey, "pk_dev_") {
		return true
	}

	// 4. 事件名称以 "test." 开头
	if strings.HasPrefix(event.Name, "test.") {
		return true
	}

	// 5. 默认非测试事件
	return false
}

// Service Events 服务接口，严格对齐 trigger.dev 实现
type Service interface {
	// 事件摄取 - 对齐 IngestSendEvent.call
	IngestSendEvent(ctx context.Context, env *apiauth.AuthenticatedEnvironment,
		event *SendEventRequest, opts *SendEventOptions) (*EventRecordResponse, error)

	// 事件分发 - 对齐 DeliverEventService.call
	DeliverEvent(ctx context.Context, eventID string) error

	// 调度器调用 - 对齐 InvokeDispatcherService.call
	InvokeDispatcher(ctx context.Context, dispatcherID string, eventRecordID string) error

	// 事件查询
	GetEventRecord(ctx context.Context, id string) (*EventRecordResponse, error)
	ListEventRecords(ctx context.Context, params ListEventRecordsParams) (*ListEventRecordsResponse, error)

	// 调度器管理
	GetEventDispatcher(ctx context.Context, id string) (*EventDispatcherResponse, error)
	ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) (*ListEventDispatchersResponse, error)
}

// service 实现
type service struct {
	repo          Repository
	sharedQueries *shared.Queries
	queueSvc      queue.QueueService
	logger        *slog.Logger
}

// NewService 创建服务实例
func NewService(repo Repository, sharedQueries *shared.Queries, queueSvc queue.QueueService, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		repo:          repo,
		sharedQueries: sharedQueries,
		queueSvc:      queueSvc,
		logger:        logger,
	}
}

// IngestSendEvent 事件摄取，对齐 trigger.dev IngestSendEvent.call
func (s *service) IngestSendEvent(ctx context.Context, env *apiauth.AuthenticatedEnvironment,
	event *SendEventRequest, opts *SendEventOptions) (*EventRecordResponse, error) {

	logger := s.logger.With("operation", "ingest_send_event", "event_name", event.Name)
	logger.Info("Ingesting event", "event_name", event.Name, "environment_id", env.Environment.ID)

	// 计算延迟投递时间，对齐 trigger.dev calculateDeliverAt
	deliverAt := s.calculateDeliverAt(opts)

	// 在事务中创建事件记录
	var eventRecord EventRecords
	err := s.repo.WithTxAndReturn(ctx, func(txRepo Repository, tx pgx.Tx) error {
		// 查找外部账户（如果指定），对齐 trigger.dev
		var externalAccountID pgtype.UUID
		if opts != nil && opts.AccountID != nil {
			// 直接查询共享表，无需独立服务，对齐 trigger.dev 模式
			params := shared.FindExternalAccountByEnvAndIdentifierParams{
				EnvironmentID: env.Environment.ID,
				Identifier:    *opts.AccountID,
			}

			account, err := s.sharedQueries.FindExternalAccountByEnvAndIdentifier(ctx, params)
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("failed to find external account: %w", err)
				}
				// 外部账户不存在，继续处理（符合 trigger.dev 行为）
				logger.Debug("External account not found", "account_id", *opts.AccountID)
			} else {
				externalAccountID = account.ID
				logger.Debug("Using external account", "account_id", account.ID)
			}
		}

		// 生成事件ID（如果未提供）
		eventID := event.ID
		if eventID == "" {
			eventID = uuid.New().String()
		}

		// 设置默认时间戳
		timestamp := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if event.Timestamp != nil {
			timestamp = pgtype.Timestamptz{Time: *event.Timestamp, Valid: true}
		}

		// 序列化 payload 和 context
		payloadBytes, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		contextBytes, err := json.Marshal(event.Context)
		if err != nil {
			return fmt.Errorf("failed to marshal context: %w", err)
		}

		// 设置投递时间
		deliverAtPg := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if deliverAt != nil {
			deliverAtPg = pgtype.Timestamptz{Time: *deliverAt, Valid: true}
		}

		// 创建事件记录
		params := CreateEventRecordParams{
			EventID:           eventID,
			Name:              event.Name,
			Source:            event.Source,
			Payload:           payloadBytes,
			Context:           contextBytes,
			Timestamp:         timestamp,
			EnvironmentID:     env.Environment.ID,
			OrganizationID:    env.Environment.OrganizationID,
			ProjectID:         env.Environment.ProjectID,
			ExternalAccountID: externalAccountID,
			DeliverAt:         deliverAtPg,
			IsTest:            determineIfTestEvent(env, event, opts),
		}

		record, err := txRepo.CreateEventRecord(ctx, params)
		if err != nil {
			// 检查是否是重复键错误，如果是则返回现有记录
			if isDuplicateKeyError(err) {
				existingParams := GetEventRecordByEventIDParams{
					EventID:       eventID,
					EnvironmentID: env.Environment.ID,
				}
				existing, getErr := txRepo.GetEventRecordByEventID(ctx, existingParams)
				if getErr == nil {
					eventRecord = existing
					return nil
				}
			}
			return fmt.Errorf("failed to create event record: %w", err)
		}

		eventRecord = record

		// 触发事件分发作业，对齐 trigger.dev
		// workerQueue.enqueue("deliverEvent", { id: eventLog.id }, { runAt: eventLog.deliverAt, tx })
		payloadStr, err := json.Marshal(map[string]interface{}{
			"id": uuid.UUID(record.ID.Bytes).String(),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal deliver event payload: %w", err)
		}

		queueReq := &queue.EnqueueDeliverEventRequest{
			EventID:      uuid.UUID(record.ID.Bytes).String(),
			EndpointID:   "deliver-event", // 固定端点ID
			Payload:      string(payloadStr),
			ScheduledFor: deliverAt,
		}

		_, queueErr := s.queueSvc.EnqueueDeliverEventTx(ctx, tx, queueReq)
		if queueErr != nil {
			logger.Error("Failed to enqueue deliver event job", "error", queueErr)
			return fmt.Errorf("failed to enqueue deliver event job: %w", queueErr)
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to ingest event", "error", err)
		return nil, err
	}

	logger.Info("Event ingested successfully", "event_id", eventRecord.ID.Bytes)

	return convertEventRecordToResponse(eventRecord), nil
}

// DeliverEvent 事件分发，对齐 trigger.dev DeliverEventService.call
func (s *service) DeliverEvent(ctx context.Context, eventID string) error {
	logger := s.logger.With("operation", "deliver_event", "event_id", eventID)
	logger.Info("Delivering event")

	// 将字符串ID转换为pgtype.UUID
	pgUUID, err := stringToPgUUID(eventID)
	if err != nil {
		logger.Error("Invalid UUID format", "error", err)
		return fmt.Errorf("invalid event ID format: %w", err)
	}

	// 在事务中处理事件分发
	return s.repo.WithTxAndReturn(ctx, func(txRepo Repository, tx pgx.Tx) error {
		// 获取事件记录
		eventRecord, err := txRepo.GetEventRecordByID(ctx, pgUUID)
		if err != nil {
			logger.Error("Failed to get event record", "error", err)
			return fmt.Errorf("failed to get event record: %w", err)
		}

		// 查找可能的事件调度器
		findParams := FindEventDispatchersParams{
			EnvironmentID: eventRecord.EnvironmentID,
			Event:         eventRecord.Name,
			Source:        eventRecord.Source,
			Column4:       false, // enabled filter
			Column5:       false, // manual filter - 非手动调度器
		}

		possibleDispatchers, err := txRepo.FindEventDispatchers(ctx, findParams)
		if err != nil {
			logger.Error("Failed to find event dispatchers", "error", err)
			return fmt.Errorf("failed to find event dispatchers: %w", err)
		}

		logger.Debug("Found possible event dispatchers", "count", len(possibleDispatchers))

		// 过滤匹配的事件调度器
		var matchingDispatchers []EventDispatchers
		for _, dispatcher := range possibleDispatchers {
			if s.evaluateEventRule(dispatcher, eventRecord) {
				matchingDispatchers = append(matchingDispatchers, dispatcher)
			}
		}

		if len(matchingDispatchers) == 0 {
			logger.Debug("No matching event dispatchers")
			return nil
		}

		logger.Debug("Found matching event dispatchers", "count", len(matchingDispatchers))

		// 异步调用匹配的调度器，对齐 trigger.dev
		// workerQueue.enqueue("events.invokeDispatcher", { id, eventRecordId }, { tx })
		for _, dispatcher := range matchingDispatchers {
			payloadStr, err := json.Marshal(map[string]interface{}{
				"dispatcherId":  uuid.UUID(dispatcher.ID.Bytes).String(),
				"eventRecordId": uuid.UUID(eventRecord.ID.Bytes).String(),
			})
			if err != nil {
				logger.Error("Failed to marshal invoke dispatcher payload", "error", err)
				continue
			}

			queueReq := &queue.EnqueueInvokeDispatcherRequest{
				DispatcherID: uuid.UUID(dispatcher.ID.Bytes).String(),
				EventID:      uuid.UUID(eventRecord.ID.Bytes).String(),
				Payload:      string(payloadStr),
			}

			_, queueErr := s.queueSvc.EnqueueInvokeDispatcherTx(ctx, tx, queueReq)
			if queueErr != nil {
				logger.Error("Failed to enqueue invoke dispatcher job",
					"error", queueErr,
					"dispatcher_id", dispatcher.ID.Bytes,
					"event_record_id", eventRecord.ID.Bytes)
				// 继续处理其他调度器，不因为一个失败而终止
				continue
			}

			logger.Debug("Enqueued dispatcher invocation",
				"dispatcher_id", dispatcher.ID.Bytes,
				"event_record_id", eventRecord.ID.Bytes)
		}

		// 更新事件记录为已分发
		updateParams := UpdateEventRecordDeliveredAtParams{
			ID:          eventRecord.ID,
			DeliveredAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}

		err = txRepo.UpdateEventRecordDeliveredAt(ctx, updateParams)
		if err != nil {
			logger.Error("Failed to update event record delivered_at", "error", err)
			return fmt.Errorf("failed to update event record: %w", err)
		}

		logger.Info("Event delivered successfully", "matching_dispatchers", len(matchingDispatchers))
		return nil
	})
}

// InvokeDispatcher 调度器调用，对齐 trigger.dev InvokeDispatcherService.call
func (s *service) InvokeDispatcher(ctx context.Context, dispatcherID string, eventRecordID string) error {
	logger := s.logger.With("operation", "invoke_dispatcher", "dispatcher_id", dispatcherID, "event_record_id", eventRecordID)
	logger.Info("Invoking dispatcher")

	// 将字符串ID转换为pgtype.UUID
	dispatcherPgUUID, err := stringToPgUUID(dispatcherID)
	if err != nil {
		logger.Error("Invalid dispatcher UUID format", "error", err)
		return fmt.Errorf("invalid dispatcher ID format: %w", err)
	}

	eventRecordPgUUID, err := stringToPgUUID(eventRecordID)
	if err != nil {
		logger.Error("Invalid event record UUID format", "error", err)
		return fmt.Errorf("invalid event record ID format: %w", err)
	}

	// 获取事件调度器
	dispatcher, err := s.repo.GetEventDispatcherByID(ctx, dispatcherPgUUID)
	if err != nil {
		logger.Error("Failed to get event dispatcher", "error", err)
		return fmt.Errorf("failed to get event dispatcher: %w", err)
	}

	// 检查调度器是否启用
	if !dispatcher.Enabled {
		logger.Debug("Event dispatcher is disabled")
		return nil
	}

	// 获取事件记录
	eventRecord, err := s.repo.GetEventRecordByID(ctx, eventRecordPgUUID)
	if err != nil {
		logger.Error("Failed to get event record", "error", err)
		return fmt.Errorf("failed to get event record: %w", err)
	}

	logger.Debug("Invoking event dispatcher", "dispatcher_enabled", dispatcher.Enabled)

	// 解析可调度对象
	var dispatchable map[string]interface{}
	if err := json.Unmarshal(dispatcher.Dispatchable, &dispatchable); err != nil {
		logger.Error("Invalid dispatchable", "error", err)
		return fmt.Errorf("invalid dispatchable: %w", err)
	}

	dispatchableType, ok := dispatchable["type"].(string)
	if !ok {
		logger.Error("Missing dispatchable type")
		return fmt.Errorf("missing dispatchable type")
	}

	dispatchableID, ok := dispatchable["id"].(string)
	if !ok {
		logger.Error("Missing dispatchable ID")
		return fmt.Errorf("missing dispatchable ID")
	}

	// 根据可调度对象类型处理
	switch dispatchableType {
	case "JOB_VERSION":
		return s.invokeJobVersion(ctx, dispatchableID, eventRecord, logger)

	case "DYNAMIC_TRIGGER":
		return s.invokeDynamicTrigger(ctx, dispatchableID, eventRecord, logger)

	default:
		logger.Error("Unknown dispatchable type", "type", dispatchableType)
		return fmt.Errorf("unknown dispatchable type: %s", dispatchableType)
	}
}

// invokeJobVersion 调用作业版本，对齐 trigger.dev JOB_VERSION case
func (s *service) invokeJobVersion(ctx context.Context, jobVersionID string, eventRecord EventRecords, logger *slog.Logger) error {
	logger.Info("Invoking job version", "job_version_id", jobVersionID)

	// TODO: 集成Jobs服务
	// 1. 通过jobVersionID查找JobVersion
	// 2. 调用CreateRunService创建作业运行
	//
	// 示例代码结构:
	// jobVersion, err := jobsService.GetJobVersionByID(ctx, jobVersionID)
	// if err != nil {
	//     return fmt.Errorf("failed to get job version: %w", err)
	// }
	//
	// createRunParams := CreateRunParams{
	//     EventID: eventRecord.ID.String(),
	//     Job: jobVersion.Job,
	//     Version: jobVersion,
	//     Environment: environment,
	// }
	//
	// run, err := runsService.CreateRun(ctx, createRunParams)
	// if err != nil {
	//     return fmt.Errorf("failed to create run: %w", err)
	// }

	logger.Info("Job version invocation completed (placeholder)", "job_version_id", jobVersionID)
	return nil
}

// invokeDynamicTrigger 调用动态触发器，对齐 trigger.dev DYNAMIC_TRIGGER case
func (s *service) invokeDynamicTrigger(ctx context.Context, dynamicTriggerID string, eventRecord EventRecords, logger *slog.Logger) error {
	logger.Info("Invoking dynamic trigger", "dynamic_trigger_id", dynamicTriggerID)

	// TODO: 集成DynamicTrigger和Jobs服务
	// 1. 通过dynamicTriggerID查找DynamicTrigger
	// 2. 获取关联的作业列表
	// 3. 为每个作业找到最新版本
	// 4. 为每个作业版本创建运行
	//
	// 示例代码结构:
	// dynamicTrigger, err := triggersService.GetDynamicTriggerByID(ctx, dynamicTriggerID)
	// if err != nil {
	//     return fmt.Errorf("failed to get dynamic trigger: %w", err)
	// }
	//
	// for _, job := range dynamicTrigger.Jobs {
	//     latestVersion, err := jobsService.GetLatestJobVersion(ctx, job.ID, environment.ID)
	//     if err != nil || latestVersion == nil {
	//         continue
	//     }
	//
	//     createRunParams := CreateRunParams{
	//         EventID: eventRecord.ID.String(),
	//         Job: job,
	//         Version: latestVersion,
	//         Environment: environment,
	//     }
	//
	//     _, err = runsService.CreateRun(ctx, createRunParams)
	//     if err != nil {
	//         logger.Error("Failed to create run for job", "job_id", job.ID, "error", err)
	//         continue
	//     }
	// }

	logger.Info("Dynamic trigger invocation completed (placeholder)", "dynamic_trigger_id", dynamicTriggerID)
	return nil
}

// GetEventRecord 获取事件记录
func (s *service) GetEventRecord(ctx context.Context, id string) (*EventRecordResponse, error) {
	logger := s.logger.With("operation", "get_event_record", "event_id", id)

	// 将字符串ID转换为pgtype.UUID
	pgUUID, err := stringToPgUUID(id)
	if err != nil {
		logger.Error("Invalid UUID format", "error", err)
		return nil, err
	}

	eventRecord, err := s.repo.GetEventRecordByID(ctx, pgUUID)
	if err != nil {
		logger.Error("Failed to get event record", "error", err)
		return nil, err
	}

	return convertEventRecordToResponse(eventRecord), nil
}

// ListEventRecords 列出事件记录
func (s *service) ListEventRecords(ctx context.Context, params ListEventRecordsParams) (*ListEventRecordsResponse, error) {
	logger := s.logger.With("operation", "list_event_records")

	events, err := s.repo.ListEventRecords(ctx, params)
	if err != nil {
		logger.Error("Failed to list event records", "error", err)
		return nil, err
	}

	// 暂时跳过总数计算，等待参数类型匹配修复
	// total, err := s.repo.CountEventRecords(ctx, params)
	total := int64(len(events))

	var eventResponses []EventRecordResponse
	for _, event := range events {
		eventResponses = append(eventResponses, *convertEventRecordToResponse(event))
	}

	return &ListEventRecordsResponse{
		Records: eventResponses,
		Total:   total,
	}, nil
}

// GetEventDispatcher 获取事件调度器
func (s *service) GetEventDispatcher(ctx context.Context, id string) (*EventDispatcherResponse, error) {
	logger := s.logger.With("operation", "get_event_dispatcher", "dispatcher_id", id)

	// 将字符串ID转换为pgtype.UUID
	pgUUID, err := stringToPgUUID(id)
	if err != nil {
		logger.Error("Invalid UUID format", "error", err)
		return nil, err
	}

	dispatcher, err := s.repo.GetEventDispatcherByID(ctx, pgUUID)
	if err != nil {
		logger.Error("Failed to get event dispatcher", "error", err)
		return nil, err
	}

	return convertEventDispatcherToResponse(dispatcher), nil
}

// ListEventDispatchers 列出事件调度器
func (s *service) ListEventDispatchers(ctx context.Context, params ListEventDispatchersParams) (*ListEventDispatchersResponse, error) {
	logger := s.logger.With("operation", "list_event_dispatchers")

	dispatchers, err := s.repo.ListEventDispatchers(ctx, params)
	if err != nil {
		logger.Error("Failed to list event dispatchers", "error", err)
		return nil, err
	}

	// 暂时跳过总数计算，等待参数类型匹配修复
	// total, err := s.repo.CountEventDispatchers(ctx, params)
	total := int64(len(dispatchers))

	var dispatcherResponses []EventDispatcherResponse
	for _, dispatcher := range dispatchers {
		dispatcherResponses = append(dispatcherResponses, *convertEventDispatcherToResponse(dispatcher))
	}

	return &ListEventDispatchersResponse{
		Dispatchers: dispatcherResponses,
		Total:       total,
	}, nil
}

// 辅助函数
func stringToPgUUID(s string) (pgtype.UUID, error) {
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}

	return pgtype.UUID{
		Bytes: parsedUUID,
		Valid: true,
	}, nil
}

func uuidToPgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}

// calculateDeliverAt 计算延迟投递时间，对齐 trigger.dev calculateDeliverAt
func (s *service) calculateDeliverAt(opts *SendEventOptions) *time.Time {
	if opts == nil {
		return nil
	}

	// 如果指定了 deliverAt 时间
	if opts.DeliverAt != nil {
		return opts.DeliverAt
	}

	// 如果指定了 deliverAfter 秒数
	if opts.DeliverAfter != nil {
		deliverTime := time.Now().Add(time.Duration(*opts.DeliverAfter) * time.Second)
		return &deliverTime
	}

	return nil
}

// evaluateEventRule 评估事件过滤规则，对齐 trigger.dev evaluateEventRule
func (s *service) evaluateEventRule(dispatcher EventDispatchers, eventRecord EventRecords) bool {
	// 如果没有过滤器，则匹配所有事件
	if len(dispatcher.PayloadFilter) == 0 && len(dispatcher.ContextFilter) == 0 {
		return true
	}

	// 解析过滤器
	var payloadFilter, contextFilter map[string]interface{}
	if len(dispatcher.PayloadFilter) > 0 {
		if err := json.Unmarshal(dispatcher.PayloadFilter, &payloadFilter); err != nil {
			s.logger.Error("Invalid payload filter", "error", err, "dispatcher_id", dispatcher.ID.Bytes)
			return false
		}
	}

	if len(dispatcher.ContextFilter) > 0 {
		if err := json.Unmarshal(dispatcher.ContextFilter, &contextFilter); err != nil {
			s.logger.Error("Invalid context filter", "error", err, "dispatcher_id", dispatcher.ID.Bytes)
			return false
		}
	}

	// 创建事件匹配器
	matcher := NewEventMatcher(eventRecord)

	// 检查是否匹配过滤条件
	filter := EventFilter{
		Payload: payloadFilter,
		Context: contextFilter,
	}

	return matcher.Matches(filter)
}

// isDuplicateKeyError 检查是否是重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// 检查 PostgreSQL 唯一约束错误
	return strings.Contains(err.Error(), "SQLSTATE 23505") ||
		strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint")
}

// EventMatcher 事件匹配器，对齐 trigger.dev EventMatcher
type EventMatcher struct {
	event EventRecords
}

// NewEventMatcher 创建事件匹配器
func NewEventMatcher(event EventRecords) *EventMatcher {
	return &EventMatcher{event: event}
}

// Matches 检查事件是否匹配过滤条件
func (m *EventMatcher) Matches(filter EventFilter) bool {
	// 解析事件的payload和context
	var eventPayload, eventContext map[string]interface{}
	if len(m.event.Payload) > 0 {
		json.Unmarshal(m.event.Payload, &eventPayload)
	}
	if len(m.event.Context) > 0 {
		json.Unmarshal(m.event.Context, &eventContext)
	}

	// 创建完整的事件对象用于匹配
	eventData := map[string]interface{}{
		"payload": eventPayload,
		"context": eventContext,
	}

	return patternMatches(eventData, map[string]interface{}{
		"payload": filter.Payload,
		"context": filter.Context,
	})
}

// patternMatches 模式匹配函数，对齐 trigger.dev patternMatches
func patternMatches(payload interface{}, pattern interface{}) bool {
	if pattern == nil {
		return true
	}

	patternMap, ok := pattern.(map[string]interface{})
	if !ok {
		return payload == pattern
	}

	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		return false
	}

	for patternKey, patternValue := range patternMap {
		payloadValue, exists := payloadMap[patternKey]
		if !exists && patternValue != nil {
			return false
		}

		// 处理数组模式匹配
		if patternArray, isArray := patternValue.([]interface{}); isArray {
			if len(patternArray) > 0 {
				found := false
				for _, item := range patternArray {
					if payloadValue == item {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		} else if patternObj, isObj := patternValue.(map[string]interface{}); isObj {
			// 处理嵌套对象匹配
			if payloadArray, isPayloadArray := payloadValue.([]interface{}); isPayloadArray {
				// 检查数组中的任何元素是否匹配
				found := false
				for _, item := range payloadArray {
					if patternMatches(item, patternObj) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			} else {
				if !patternMatches(payloadValue, patternObj) {
					return false
				}
			}
		} else if patternValue != nil && payloadValue != patternValue {
			return false
		}
	}

	return true
}

// 辅助转换函数
func convertEventRecordToResponse(record EventRecords) *EventRecordResponse {
	// 解析 payload 和 context
	var payload, context map[string]interface{}
	json.Unmarshal(record.Payload, &payload)
	json.Unmarshal(record.Context, &context)

	// 处理投递时间
	var deliverAt *time.Time
	if record.DeliverAt.Valid {
		deliverAt = &record.DeliverAt.Time
	}

	return &EventRecordResponse{
		ID:        uuid.UUID(record.ID.Bytes).String(),
		EventID:   record.EventID,
		Name:      record.Name,
		Source:    record.Source,
		Payload:   payload,
		Context:   context,
		Timestamp: record.Timestamp.Time,
		DeliverAt: deliverAt,
		IsTest:    record.IsTest,
		CreatedAt: record.CreatedAt.Time,
	}
}

func convertEventDispatcherToResponse(dispatcher EventDispatchers) *EventDispatcherResponse {
	return &EventDispatcherResponse{
		ID:      uuid.UUID(dispatcher.ID.Bytes).String(),
		Event:   dispatcher.Event,
		Source:  dispatcher.Source,
		Enabled: dispatcher.Enabled,
	}
}
