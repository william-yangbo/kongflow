-- job_queues.sql
-- JobQueue 作业队列相关查询

-- name: CreateJobQueue :one
INSERT INTO job_queues (
    name, environment_id, job_count, max_jobs
) VALUES ($1, $2, $3, $4)
RETURNING id, name, environment_id, job_count, max_jobs, created_at, updated_at;

-- name: GetJobQueueByID :one
SELECT id, name, environment_id, job_count, max_jobs, created_at, updated_at
FROM job_queues 
WHERE id = $1;

-- name: GetJobQueueByName :one
SELECT id, name, environment_id, job_count, max_jobs, created_at, updated_at
FROM job_queues 
WHERE environment_id = $1 AND name = $2;

-- name: UpsertJobQueue :one
INSERT INTO job_queues (
    name, environment_id, job_count, max_jobs
) VALUES ($1, $2, $3, $4)
ON CONFLICT (environment_id, name) 
DO UPDATE SET 
    max_jobs = EXCLUDED.max_jobs,
    updated_at = NOW()
RETURNING id, name, environment_id, job_count, max_jobs, created_at, updated_at;

-- name: ListJobQueuesByEnvironment :many
SELECT id, name, environment_id, job_count, max_jobs, created_at, updated_at
FROM job_queues 
WHERE environment_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: UpdateJobQueueCounts :one
UPDATE job_queues 
SET job_count = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, environment_id, job_count, max_jobs, created_at, updated_at;

-- name: IncrementJobCount :one
UPDATE job_queues 
SET job_count = job_count + 1, updated_at = NOW()
WHERE id = $1
RETURNING id, name, environment_id, job_count, max_jobs, created_at, updated_at;

-- name: DecrementJobCount :one
UPDATE job_queues 
SET job_count = GREATEST(job_count - 1, 0), updated_at = NOW()
WHERE id = $1
RETURNING id, name, environment_id, job_count, max_jobs, created_at, updated_at;

-- name: DeleteJobQueue :exec
DELETE FROM job_queues WHERE id = $1;