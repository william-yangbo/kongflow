// Package queue provides River-based queue service implementation for events
package queue

import (
	"context"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
)

// WorkerQueueManager 定义队列管理器接口，便于测试
type WorkerQueueManager interface {
	EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
	EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
}

// riverQueueService 基于WorkerQueue的Events服务实现
type riverQueueService struct {
	manager WorkerQueueManager
}

// NewRiverQueueService 创建基于WorkerQueue的Events队列服务
func NewRiverQueueService(manager WorkerQueueManager) QueueService {
	return &riverQueueService{
		manager: manager,
	}
}

// EnqueueDeliverEvent 将事件分发任务加入队列
func (r *riverQueueService) EnqueueDeliverEvent(ctx context.Context, req *EnqueueDeliverEventRequest) (*rivertype.JobInsertResult, error) {
	// 使用WorkerQueue的DeliverEventArgs
	args := workerqueue.DeliverEventArgs{
		ID: req.EventID,
		// 可以扩展ProjectID等字段来支持动态路由
	}

	// 构建作业选项
	opts := &workerqueue.JobOptions{
		QueueName: string(workerqueue.QueueEvents),
		Priority:  int(workerqueue.PriorityHigh),
		RunAt:     req.ScheduledFor,
	}

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueInvokeDispatcher 将调度器调用任务加入队列
func (r *riverQueueService) EnqueueInvokeDispatcher(ctx context.Context, req *EnqueueInvokeDispatcherRequest) (*rivertype.JobInsertResult, error) {
	// 使用WorkerQueue的InvokeDispatcherArgs
	args := workerqueue.InvokeDispatcherArgs{
		ID:            req.DispatcherID,
		EventRecordID: req.EventID,
	}

	// 构建作业选项
	opts := &workerqueue.JobOptions{
		QueueName: string(workerqueue.QueueEvents),
		Priority:  int(workerqueue.PriorityHigh),
		RunAt:     req.ScheduledFor,
	}

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueDeliverEventTx 在事务中将事件分发任务加入队列
func (r *riverQueueService) EnqueueDeliverEventTx(ctx context.Context, tx pgx.Tx, req *EnqueueDeliverEventRequest) (*rivertype.JobInsertResult, error) {
	// 使用WorkerQueue的DeliverEventArgs
	args := workerqueue.DeliverEventArgs{
		ID: req.EventID,
	}

	// 构建作业选项
	opts := &workerqueue.JobOptions{
		QueueName: string(workerqueue.QueueEvents),
		Priority:  int(workerqueue.PriorityHigh),
		RunAt:     req.ScheduledFor,
	}

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}

// EnqueueInvokeDispatcherTx 在事务中将调度器调用任务加入队列
func (r *riverQueueService) EnqueueInvokeDispatcherTx(ctx context.Context, tx pgx.Tx, req *EnqueueInvokeDispatcherRequest) (*rivertype.JobInsertResult, error) {
	// 使用WorkerQueue的InvokeDispatcherArgs
	args := workerqueue.InvokeDispatcherArgs{
		ID:            req.DispatcherID,
		EventRecordID: req.EventID,
	}

	// 构建作业选项
	opts := &workerqueue.JobOptions{
		QueueName: string(workerqueue.QueueEvents),
		Priority:  int(workerqueue.PriorityHigh),
		RunAt:     req.ScheduledFor,
	}

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}
