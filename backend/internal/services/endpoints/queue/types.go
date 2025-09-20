// Package queue provides queue service interface for endpoints service
package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
)

// QueueService 端点队列服务接口
// 基于现有的 workerqueue 包，提供端点特定的队列操作
type QueueService interface {
	// 标准队列操作
	EnqueueIndexEndpoint(ctx context.Context, req *EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterJob(ctx context.Context, req *RegisterJobRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterSource(ctx context.Context, req *RegisterSourceRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterDynamicTrigger(ctx context.Context, req *RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterDynamicSchedule(ctx context.Context, req *RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error)

	// 事务性队列操作
	EnqueueIndexEndpointTx(ctx context.Context, tx pgx.Tx, req *EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterJobTx(ctx context.Context, tx pgx.Tx, req *RegisterJobRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterSourceTx(ctx context.Context, tx pgx.Tx, req *RegisterSourceRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterDynamicTriggerTx(ctx context.Context, tx pgx.Tx, req *RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error)
	EnqueueRegisterDynamicScheduleTx(ctx context.Context, tx pgx.Tx, req *RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error)
}

// EndpointIndexSource 端点索引来源枚举 (对齐trigger.dev)
type EndpointIndexSource string

const (
	EndpointIndexSourceManual   EndpointIndexSource = "MANUAL"
	EndpointIndexSourceAPI      EndpointIndexSource = "API"
	EndpointIndexSourceInternal EndpointIndexSource = "INTERNAL"
	EndpointIndexSourceHook     EndpointIndexSource = "HOOK"
)

// EnqueueIndexEndpointRequest 索引端点队列请求
type EnqueueIndexEndpointRequest struct {
	EndpointID uuid.UUID              `json:"endpoint_id" validate:"required"`
	Source     EndpointIndexSource    `json:"source" validate:"required"`
	Reason     string                 `json:"reason,omitempty"`
	SourceData map[string]interface{} `json:"source_data,omitempty"`
	QueueName  string                 `json:"queue_name,omitempty"`
	RunAt      *time.Time             `json:"run_at,omitempty"`
	Priority   int                    `json:"priority,omitempty"`
}

// RegisterJobRequest 注册作业队列请求
type RegisterJobRequest struct {
	EndpointID  uuid.UUID              `json:"endpoint_id" validate:"required"`
	JobID       string                 `json:"job_id" validate:"required"`
	JobMetadata map[string]interface{} `json:"job_metadata" validate:"required"`
	QueueName   string                 `json:"queue_name,omitempty"`
}

// RegisterSourceRequest 注册源队列请求
type RegisterSourceRequest struct {
	EndpointID     uuid.UUID              `json:"endpoint_id" validate:"required"`
	SourceID       string                 `json:"source_id" validate:"required"`
	SourceMetadata map[string]interface{} `json:"source_metadata" validate:"required"`
	QueueName      string                 `json:"queue_name,omitempty"`
}

// RegisterDynamicTriggerRequest 注册动态触发器队列请求
type RegisterDynamicTriggerRequest struct {
	EndpointID      uuid.UUID              `json:"endpoint_id" validate:"required"`
	TriggerID       string                 `json:"trigger_id" validate:"required"`
	TriggerMetadata map[string]interface{} `json:"trigger_metadata" validate:"required"`
	QueueName       string                 `json:"queue_name,omitempty"`
}

// RegisterDynamicScheduleRequest 注册动态调度队列请求
type RegisterDynamicScheduleRequest struct {
	EndpointID       uuid.UUID              `json:"endpoint_id" validate:"required"`
	ScheduleID       string                 `json:"schedule_id" validate:"required"`
	ScheduleMetadata map[string]interface{} `json:"schedule_metadata" validate:"required"`
	QueueName        string                 `json:"queue_name,omitempty"`
}
