// Package workerqueue provides worker queue functionality using River Queue
package workerqueue

// JobArgs represents the interface that all job arguments must implement
// This mirrors River's JobArgs interface but adds KongFlow-specific functionality
type JobArgs interface {
	// Kind returns the unique string identifier for this job type
	Kind() string
}

// JobPriority represents job priority levels (lower number = higher priority)
// River Queue uses 1-4 range where 1=highest, 4=lowest
type JobPriority int

const (
	PriorityHigh    JobPriority = 1 // Highest priority (events, run execution)
	PriorityNormal  JobPriority = 2 // Normal priority (indexing, basic operations)
	PriorityLow     JobPriority = 3 // Low priority (maintenance, cleanup)
	PriorityVeryLow JobPriority = 4 // Very low priority (background tasks)
)

// JobQueue represents the different queues available in KongFlow
type JobQueue string

const (
	QueueDefault     JobQueue = "default"
	QueueExecution   JobQueue = "execution"
	QueueEvents      JobQueue = "events"
	QueueMaintenance JobQueue = "maintenance"
)

// IndexSource represents the source of an endpoint indexing operation
type IndexSource string

const (
	IndexSourceManual   IndexSource = "MANUAL"
	IndexSourceAPI      IndexSource = "API"
	IndexSourceInternal IndexSource = "INTERNAL"
	IndexSourceHook     IndexSource = "HOOK"
)

// ExecutionReason represents why a run execution is being performed
type ExecutionReason string

const (
	ExecutionReasonPreprocess ExecutionReason = "PREPROCESS"
	ExecutionReasonExecuteJob ExecutionReason = "EXECUTE_JOB"
)
