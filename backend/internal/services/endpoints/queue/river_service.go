// Package queue provides River-based queue service implementation for endpoints
package queue

import (
	"context"
	"fmt"
	"time"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/rivertype"
)

// WorkerQueueManager 定义队列管理器接口，便于测试
type WorkerQueueManager interface {
	EnqueueJob(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
	EnqueueJobTx(ctx context.Context, tx pgx.Tx, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
}

// riverQueueService 基于 River 队列的服务实现
type riverQueueService struct {
	manager WorkerQueueManager
}

// NewRiverQueueService 创建基于 River 的队列服务
func NewRiverQueueService(manager WorkerQueueManager) QueueService {
	return &riverQueueService{
		manager: manager,
	}
}

// buildJobOptions 构建队列选项的通用函数
func buildJobOptions(queueName string, priority int, runAt interface{}) *workerqueue.JobOptions {
	opts := &workerqueue.JobOptions{
		QueueName: queueName,
		Priority:  priority,
	}

	// 设置默认值
	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}
	if opts.Priority == 0 {
		opts.Priority = int(workerqueue.PriorityNormal)
	}

	// 处理 runAt 参数
	if runAt != nil {
		switch v := runAt.(type) {
		case *time.Time:
			if v != nil {
				opts.RunAt = v
			}
		}
	}

	return opts
}

// EnqueueIndexEndpoint 将端点索引任务加入队列
func (r *riverQueueService) EnqueueIndexEndpoint(ctx context.Context, req *EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.IndexEndpointArgs{
		ID:         req.EndpointID.String(),
		Source:     workerqueue.IndexSource(req.Source),
		SourceData: req.SourceData,
		Reason:     req.Reason,
		JobKey:     fmt.Sprintf("index_endpoint_%s_%s", req.EndpointID.String(), req.Source),
	}

	opts := buildJobOptions(req.QueueName, req.Priority, req.RunAt)
	opts.JobKey = args.JobKey

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueIndexEndpointTx 在事务中将端点索引任务加入队列
func (r *riverQueueService) EnqueueIndexEndpointTx(ctx context.Context, tx pgx.Tx, req *EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.IndexEndpointArgs{
		ID:         req.EndpointID.String(),
		Source:     workerqueue.IndexSource(req.Source),
		SourceData: req.SourceData,
		Reason:     req.Reason,
		JobKey:     fmt.Sprintf("index_endpoint_%s_%s", req.EndpointID.String(), req.Source),
	}

	opts := buildJobOptions(req.QueueName, req.Priority, req.RunAt)
	opts.JobKey = args.JobKey

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}

// EnqueueRegisterJob 将作业注册任务加入队列
func (r *riverQueueService) EnqueueRegisterJob(ctx context.Context, req *RegisterJobRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterJobArgs{
		EndpointID:  req.EndpointID.String(),
		JobID:       req.JobID,
		JobMetadata: req.JobMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_job_%s_%s", req.EndpointID.String(), req.JobID)

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterJobTx 在事务中将作业注册任务加入队列
func (r *riverQueueService) EnqueueRegisterJobTx(ctx context.Context, tx pgx.Tx, req *RegisterJobRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterJobArgs{
		EndpointID:  req.EndpointID.String(),
		JobID:       req.JobID,
		JobMetadata: req.JobMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_job_%s_%s", req.EndpointID.String(), req.JobID)

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}

// EnqueueRegisterSource 将源注册任务加入队列
func (r *riverQueueService) EnqueueRegisterSource(ctx context.Context, req *RegisterSourceRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterSourceArgs{
		EndpointID:     req.EndpointID.String(),
		SourceID:       req.SourceID,
		SourceMetadata: req.SourceMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_source_%s_%s", req.EndpointID.String(), req.SourceID)

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterSourceTx 在事务中将源注册任务加入队列
func (r *riverQueueService) EnqueueRegisterSourceTx(ctx context.Context, tx pgx.Tx, req *RegisterSourceRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterSourceArgs{
		EndpointID:     req.EndpointID.String(),
		SourceID:       req.SourceID,
		SourceMetadata: req.SourceMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_source_%s_%s", req.EndpointID.String(), req.SourceID)

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicTrigger 将动态触发器注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicTrigger(ctx context.Context, req *RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterDynamicTriggerArgs{
		EndpointID:      req.EndpointID.String(),
		TriggerID:       req.TriggerID,
		TriggerMetadata: req.TriggerMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_trigger_%s_%s", req.EndpointID.String(), req.TriggerID)

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicTriggerTx 在事务中将动态触发器注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicTriggerTx(ctx context.Context, tx pgx.Tx, req *RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterDynamicTriggerArgs{
		EndpointID:      req.EndpointID.String(),
		TriggerID:       req.TriggerID,
		TriggerMetadata: req.TriggerMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_trigger_%s_%s", req.EndpointID.String(), req.TriggerID)

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicSchedule 将动态调度注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicSchedule(ctx context.Context, req *RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterDynamicScheduleArgs{
		EndpointID:       req.EndpointID.String(),
		ScheduleID:       req.ScheduleID,
		ScheduleMetadata: req.ScheduleMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_schedule_%s_%s", req.EndpointID.String(), req.ScheduleID)

	return r.manager.EnqueueJob(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicScheduleTx 在事务中将动态调度注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicScheduleTx(ctx context.Context, tx pgx.Tx, req *RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error) {
	args := workerqueue.RegisterDynamicScheduleArgs{
		EndpointID:       req.EndpointID.String(),
		ScheduleID:       req.ScheduleID,
		ScheduleMetadata: req.ScheduleMetadata,
	}

	opts := buildJobOptions(req.QueueName, 0, nil)
	opts.JobKey = fmt.Sprintf("register_schedule_%s_%s", req.EndpointID.String(), req.ScheduleID)

	return r.manager.EnqueueJobTx(ctx, tx, args.Kind(), args, opts)
}
