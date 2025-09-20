// Package queue provides queue service interface for events service
package queue

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
)

// QueueService Events队列服务接口，对齐trigger.dev和WorkerQueue最佳实践
type QueueService interface {
	// 标准队列操作
	EnqueueDeliverEvent(ctx context.Context, req *EnqueueDeliverEventRequest) (*rivertype.JobInsertResult, error)
	EnqueueInvokeDispatcher(ctx context.Context, req *EnqueueInvokeDispatcherRequest) (*rivertype.JobInsertResult, error)

	// 事务性队列操作
	EnqueueDeliverEventTx(ctx context.Context, tx pgx.Tx, req *EnqueueDeliverEventRequest) (*rivertype.JobInsertResult, error)
	EnqueueInvokeDispatcherTx(ctx context.Context, tx pgx.Tx, req *EnqueueInvokeDispatcherRequest) (*rivertype.JobInsertResult, error)
}

// EnqueueDeliverEventRequest 事件分发队列请求
// 简化设计，只包含业务必要的字段
type EnqueueDeliverEventRequest struct {
	// EventID 事件ID
	EventID string `json:"eventId" validate:"required"`

	// EndpointID 端点ID
	EndpointID string `json:"endpointId" validate:"required"`

	// Payload 事件负载数据
	Payload string `json:"payload" validate:"required"`

	// ScheduledFor 计划执行时间（可选）
	ScheduledFor *time.Time `json:"scheduledFor,omitempty"`
}

// EnqueueInvokeDispatcherRequest 调度器调用队列请求
type EnqueueInvokeDispatcherRequest struct {
	// DispatcherID 调度器ID
	DispatcherID string `json:"dispatcherId" validate:"required"`

	// EventID 事件ID
	EventID string `json:"eventId" validate:"required"`

	// Payload 调度器负载数据
	Payload string `json:"payload" validate:"required"`

	// ScheduledFor 计划执行时间（可选）
	ScheduledFor *time.Time `json:"scheduledFor,omitempty"`
}
