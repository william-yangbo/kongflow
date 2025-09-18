// Package workerqueue provides job argument definitions for KongFlow workers
package workerqueue

import (
	"time"

	"github.com/riverqueue/river"
)

// IndexEndpointArgs represents arguments for the index endpoint job
// This corresponds to trigger.dev's indexEndpoint worker task
type IndexEndpointArgs struct {
	// ID is the endpoint ID to index
	ID string `json:"id"`

	// Source indicates how this indexing was triggered
	Source IndexSource `json:"source"`

	// SourceData contains additional data from the source (optional)
	SourceData map[string]interface{} `json:"source_data,omitempty"`

	// Reason provides a human-readable reason for the indexing (optional)
	Reason string `json:"reason,omitempty"`

	// JobKey is used for uniqueness checks (similar to trigger.dev)
	JobKey string `json:"job_key,omitempty" river:"unique"`
}

// Kind returns the unique identifier for this job type
func (IndexEndpointArgs) Kind() string {
	return "index_endpoint"
}

// InsertOpts provides default insertion options for endpoint indexing jobs
func (IndexEndpointArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueDefault),
		Priority:    int(PriorityNormal),
		MaxAttempts: 7, // 对应trigger.dev的配置
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,             // 使用args进行唯一性检查，包括JobKey字段
			ByPeriod: 15 * time.Minute, // 15分钟内相同端点不重复索引
		},
	}
}

// StartRunArgs represents arguments for starting a job run
// This corresponds to trigger.dev's startRun worker task
type StartRunArgs struct {
	// ID is the run ID to start
	ID string `json:"id"`
}

// Kind returns the unique identifier for this job type
func (StartRunArgs) Kind() string {
	return "start_run"
}

// InsertOpts provides default insertion options for start run jobs
func (StartRunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueExecution),
		Priority:    int(PriorityHigh),
		MaxAttempts: 4,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // 防止相同运行ID重复启动
		},
	}
}

// InvokeDispatcherArgs represents arguments for event dispatcher invocation
// This corresponds to trigger.dev's events.invokeDispatcher worker task
type InvokeDispatcherArgs struct {
	// ID is the event dispatcher ID
	ID string `json:"id"`

	// EventRecordID is the ID of the event record to dispatch
	EventRecordID string `json:"event_record_id"`
}

// Kind returns the unique identifier for this job type
func (InvokeDispatcherArgs) Kind() string {
	return "invoke_dispatcher"
}

// InsertOpts provides default insertion options for dispatcher invocation jobs
func (InvokeDispatcherArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueEvents),
		Priority:    int(PriorityHigh),
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // 防止相同的事件分发重复调用
		},
	}
}

// DeliverEventArgs represents arguments for event delivery
// This corresponds to trigger.dev's deliverEvent worker task
// ✅ Enhanced for Phase 2 project-level routing
type DeliverEventArgs struct {
	// ID is the event record ID to deliver
	ID string `json:"id"`

	// ProjectID enables project-level event routing (Phase 2)
	ProjectID string `json:"projectId,omitempty"`

	// EventType for event categorization
	EventType string `json:"eventType,omitempty"`

	// Priority for dynamic priority handling
	Priority int `json:"priority,omitempty"`
}

// Kind returns the unique identifier for this job type
func (DeliverEventArgs) Kind() string {
	return "deliver_event"
}

// InsertOpts provides default insertion options for event delivery jobs
func (DeliverEventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueEvents),
		Priority:    int(PriorityHigh),
		MaxAttempts: 5,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // 防止相同事件重复投递
		},
	}
}

// PerformRunExecutionV2Args represents arguments for run execution
// This corresponds to trigger.dev's performRunExecutionV2 worker task
// ✅ Enhanced for Phase 2 dynamic queue routing
type PerformRunExecutionV2Args struct {
	// ID is the run ID to execute
	ID string `json:"id"`

	// ProjectID enables project-level queue isolation (Phase 2)
	ProjectID string `json:"projectId"`

	// UserID for user-level routing context (Phase 2)
	UserID string `json:"userId,omitempty"`

	// Reason indicates why this execution is happening
	Reason ExecutionReason `json:"reason"`

	// ResumeTaskID is the task ID to resume from (optional)
	ResumeTaskID string `json:"resume_task_id,omitempty"`

	// IsRetry indicates if this is a retry attempt
	IsRetry bool `json:"is_retry"`
}

// Kind returns the unique identifier for this job type
func (PerformRunExecutionV2Args) Kind() string {
	return "perform_run_execution_v2"
}

// InsertOpts provides default insertion options for run execution jobs
func (PerformRunExecutionV2Args) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueExecution),
		Priority:    int(PriorityHigh),
		MaxAttempts: 12,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // 防止相同执行参数的重复调度
			// Note: IsRetry字段的变化会导致不同的唯一性哈希，允许重试
		},
	}
}

// ScheduleEmailArgs represents arguments for email scheduling jobs
// This corresponds to trigger.dev's scheduleEmail worker task
type ScheduleEmailArgs struct {
	// To is the recipient email address
	To string `json:"to"`

	// Subject is the email subject
	Subject string `json:"subject"`

	// Body is the email body content
	Body string `json:"body"`

	// From is the sender email address (optional)
	From string `json:"from,omitempty"`

	// JobKey is used for uniqueness checks (similar to trigger.dev)
	JobKey string `json:"job_key,omitempty" river:"unique"`
}

// Kind returns the unique identifier for this job type
func (ScheduleEmailArgs) Kind() string {
	return "schedule_email"
}

// InsertOpts provides default insertion options for email scheduling jobs
func (ScheduleEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueDefault),
		Priority:    int(PriorityNormal),
		MaxAttempts: 3, // matches trigger.dev
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,            // 使用args进行唯一性检查
			ByPeriod: 5 * time.Minute, // 5分钟内相同邮件不重复发送
		},
	}
}

// StartQueuedRunsArgs represents arguments for starting queued runs
// This corresponds to trigger.dev's startQueuedRuns worker task
// ✅ Phase 2 implementation for project-level queue isolation
type StartQueuedRunsArgs struct {
	// ProjectID enables project-level queue isolation
	ProjectID string `json:"projectId"`

	// UserID for additional routing context
	UserID string `json:"userId,omitempty"`

	// BatchSize for batch processing control
	BatchSize int `json:"batchSize,omitempty"`

	// Priority for job prioritization
	Priority int `json:"priority,omitempty"`

	// Region for geographic routing
	Region string `json:"region,omitempty"`
}

// Kind returns the unique identifier for this job type
func (StartQueuedRunsArgs) Kind() string {
	return "start_queued_runs"
}

// InsertOpts provides default insertion options for starting queued runs
func (StartQueuedRunsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueExecution),
		Priority:    int(PriorityNormal),
		MaxAttempts: 3, // matches trigger.dev
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,             // 防止相同项目的重复调度
			ByPeriod: 10 * time.Minute, // 10分钟内相同项目不重复
		},
	}
}

// UserTaskArgs represents arguments for user-level task routing
// ✅ Phase 2 implementation for user plan and geographic routing
type UserTaskArgs struct {
	// UserID for user identification
	UserID string `json:"userId"`

	// UserPlan for tier-based routing ("enterprise", "pro", "free")
	UserPlan string `json:"userPlan"`

	// Region for geographic routing and compliance
	Region string `json:"region"`

	// TaskType for task categorization
	TaskType string `json:"taskType"`

	// TaskData contains the actual task payload
	TaskData interface{} `json:"taskData"`

	// Priority for job prioritization
	Priority int `json:"priority,omitempty"`

	// TenantID for multi-tenant support
	TenantID string `json:"tenantId,omitempty"`
}

// Kind returns the unique identifier for this job type
func (UserTaskArgs) Kind() string {
	return "user_task"
}

// InsertOpts provides default insertion options for user tasks
func (UserTaskArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       string(QueueDefault), // Will be overridden by dynamic routing
		Priority:    int(PriorityNormal),
		MaxAttempts: 5,
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,             // 防止相同用户任务重复
			ByPeriod: 30 * time.Minute, // 30分钟内相同用户任务不重复
		},
	}
}
