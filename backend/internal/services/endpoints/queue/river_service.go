// Package queue provides River-based queue service implementation for endpoints
package queue

import (
	"context"
	"fmt"

	"kongflow/backend/internal/services/workerqueue"

	"github.com/riverqueue/river/rivertype"
)

// WorkerQueueClient 定义队列客户端接口，便于测试
type WorkerQueueClient interface {
	Enqueue(ctx context.Context, identifier string, payload interface{}, opts *workerqueue.JobOptions) (*rivertype.JobInsertResult, error)
	EnqueueWithBusinessLogic(ctx context.Context, identifier string, payload interface{}, businessLogic workerqueue.BusinessLogicFunc) (*rivertype.JobInsertResult, error)
}

// riverQueueService 基于 River 队列的服务实现
type riverQueueService struct {
	client WorkerQueueClient
}

// NewRiverQueueService 创建基于 River 的队列服务
func NewRiverQueueService(client WorkerQueueClient) QueueService {
	return &riverQueueService{
		client: client,
	}
}

// EnqueueIndexEndpoint 将端点索引任务加入队列
func (r *riverQueueService) EnqueueIndexEndpoint(ctx context.Context, req EnqueueIndexEndpointRequest) (*rivertype.JobInsertResult, error) {
	// 转换为 workerqueue.IndexEndpointArgs
	args := workerqueue.IndexEndpointArgs{
		ID:         req.EndpointID.String(),
		Source:     workerqueue.IndexSource(req.Source),
		SourceData: req.SourceData,
		Reason:     req.Reason,
		JobKey:     fmt.Sprintf("index_endpoint_%s_%s", req.EndpointID.String(), req.Source),
	}

	// 构建队列选项
	opts := &workerqueue.JobOptions{
		QueueName: req.QueueName,
		RunAt:     req.RunAt,
		Priority:  req.Priority,
		JobKey:    args.JobKey,
	}

	// 如果没有指定队列名称，使用默认值
	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}

	// 如果没有指定优先级，使用正常优先级
	if opts.Priority == 0 {
		opts.Priority = int(workerqueue.PriorityNormal)
	}

	return r.client.Enqueue(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterJob 将作业注册任务加入队列
func (r *riverQueueService) EnqueueRegisterJob(ctx context.Context, req RegisterJobRequest) (*rivertype.JobInsertResult, error) {
	// 转换为 workerqueue.RegisterJobArgs
	args := workerqueue.RegisterJobArgs{
		EndpointID:  req.EndpointID.String(),
		JobID:       req.JobID,
		JobMetadata: req.JobMetadata,
	}

	opts := &workerqueue.JobOptions{
		QueueName: req.QueueName,
		Priority:  int(workerqueue.PriorityNormal),
		JobKey:    fmt.Sprintf("register_job_%s_%s", req.EndpointID.String(), req.JobID),
	}

	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}

	return r.client.Enqueue(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterSource 将源注册任务加入队列
func (r *riverQueueService) EnqueueRegisterSource(ctx context.Context, req RegisterSourceRequest) (*rivertype.JobInsertResult, error) {
	// 转换为 workerqueue.RegisterSourceArgs
	args := workerqueue.RegisterSourceArgs{
		EndpointID:     req.EndpointID.String(),
		SourceID:       req.SourceID,
		SourceMetadata: req.SourceMetadata,
	}

	opts := &workerqueue.JobOptions{
		QueueName: req.QueueName,
		Priority:  int(workerqueue.PriorityNormal),
		JobKey:    fmt.Sprintf("register_source_%s_%s", req.EndpointID.String(), req.SourceID),
	}

	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}

	return r.client.Enqueue(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicTrigger 将动态触发器注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicTrigger(ctx context.Context, req RegisterDynamicTriggerRequest) (*rivertype.JobInsertResult, error) {
	// 转换为 workerqueue.RegisterDynamicTriggerArgs
	args := workerqueue.RegisterDynamicTriggerArgs{
		EndpointID:      req.EndpointID.String(),
		TriggerID:       req.TriggerID,
		TriggerMetadata: req.TriggerMetadata,
	}

	opts := &workerqueue.JobOptions{
		QueueName: req.QueueName,
		Priority:  int(workerqueue.PriorityNormal),
		JobKey:    fmt.Sprintf("register_trigger_%s_%s", req.EndpointID.String(), req.TriggerID),
	}

	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}

	return r.client.Enqueue(ctx, args.Kind(), args, opts)
}

// EnqueueRegisterDynamicSchedule 将动态调度注册任务加入队列
func (r *riverQueueService) EnqueueRegisterDynamicSchedule(ctx context.Context, req RegisterDynamicScheduleRequest) (*rivertype.JobInsertResult, error) {
	// 转换为 workerqueue.RegisterDynamicScheduleArgs
	args := workerqueue.RegisterDynamicScheduleArgs{
		EndpointID:       req.EndpointID.String(),
		ScheduleID:       req.ScheduleID,
		ScheduleMetadata: req.ScheduleMetadata,
	}

	opts := &workerqueue.JobOptions{
		QueueName: req.QueueName,
		Priority:  int(workerqueue.PriorityNormal),
		JobKey:    fmt.Sprintf("register_schedule_%s_%s", req.EndpointID.String(), req.ScheduleID),
	}

	if opts.QueueName == "" {
		opts.QueueName = string(workerqueue.QueueDefault)
	}

	return r.client.Enqueue(ctx, args.Kind(), args, opts)
}
