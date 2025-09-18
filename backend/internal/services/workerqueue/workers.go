// Package workerqueue provides worker implementations for KongFlow
package workerqueue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// EndpointIndexer 端点索引器接口 (避免循环导入)
type EndpointIndexer interface {
	IndexEndpoint(ctx context.Context, req *EndpointIndexRequest) (*EndpointIndexResult, error)
}

// EndpointIndexRequest 端点索引请求
type EndpointIndexRequest struct {
	EndpointID string                 `json:"endpointId"`
	Source     string                 `json:"source"`
	Reason     string                 `json:"reason,omitempty"`
	SourceData map[string]interface{} `json:"sourceData,omitempty"`
}

// EndpointIndexResult 端点索引结果
type EndpointIndexResult struct {
	IndexID string         `json:"indexId"`
	Stats   map[string]int `json:"stats"`
}

// IndexEndpointWorker handles endpoint indexing jobs
type IndexEndpointWorker struct {
	river.WorkerDefaults[IndexEndpointArgs]
	indexer EndpointIndexer
	logger  *slog.Logger
}

// NewIndexEndpointWorker creates a new IndexEndpointWorker
func NewIndexEndpointWorker(indexer EndpointIndexer, logger *slog.Logger) *IndexEndpointWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &IndexEndpointWorker{
		indexer: indexer,
		logger:  logger,
	}
}

// Work processes an endpoint indexing job
func (w *IndexEndpointWorker) Work(ctx context.Context, job *river.Job[IndexEndpointArgs]) error {
	w.logger.Info("Processing index endpoint job",
		"job_id", job.ID,
		"endpoint_id", job.Args.ID,
		"source", job.Args.Source,
		"reason", job.Args.Reason,
		"attempt", job.Attempt,
	)

	// 构建请求
	req := &EndpointIndexRequest{
		EndpointID: job.Args.ID,
		Source:     string(job.Args.Source),
		Reason:     job.Args.Reason,
		SourceData: job.Args.SourceData,
	}

	// 执行索引
	result, err := w.indexer.IndexEndpoint(ctx, req)
	if err != nil {
		w.logger.Error("Endpoint indexing failed",
			"job_id", job.ID,
			"endpoint_id", job.Args.ID,
			"error", err.Error(),
			"attempt", job.Attempt,
		)
		return fmt.Errorf("failed to index endpoint %s: %w", job.Args.ID, err)
	}

	w.logger.Info("Endpoint indexing completed successfully",
		"job_id", job.ID,
		"endpoint_id", job.Args.ID,
		"index_id", result.IndexID,
		"stats", result.Stats,
	)

	return nil
}

// NextRetry 自定义重试策略
func (w *IndexEndpointWorker) NextRetry(job *river.Job[IndexEndpointArgs]) time.Time {
	// 指数退避重试: 2^attempt * 30秒, 最多重试5次
	if job.Attempt >= 5 {
		return time.Time{} // 不再重试
	}

	backoffSeconds := int(30 * (1 << job.Attempt)) // 30, 60, 120, 240, 480 seconds
	return time.Now().Add(time.Duration(backoffSeconds) * time.Second)
}

// Timeout returns the timeout for indexing jobs
func (w *IndexEndpointWorker) Timeout(job *river.Job[IndexEndpointArgs]) time.Duration {
	// Index operations should complete within 2 minutes
	return 2 * time.Minute
}
