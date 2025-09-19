-- job_versions.sql
-- JobVersion 作业版本相关查询

-- name: CreateJobVersion :one
INSERT INTO job_versions (
    job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at;

-- name: GetJobVersionByID :one
SELECT id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at
FROM job_versions 
WHERE id = $1;

-- name: GetJobVersionByJobAndVersion :one
SELECT id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at
FROM job_versions 
WHERE job_id = $1 AND version = $2 AND environment_id = $3;

-- name: UpsertJobVersion :one
INSERT INTO job_versions (
    job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (job_id, version, environment_id) 
DO UPDATE SET 
    event_specification = EXCLUDED.event_specification,
    properties = EXCLUDED.properties,
    endpoint_id = EXCLUDED.endpoint_id,
    queue_id = EXCLUDED.queue_id,
    start_position = EXCLUDED.start_position,
    preprocess_runs = EXCLUDED.preprocess_runs,
    updated_at = NOW()
RETURNING id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at;

-- name: ListJobVersionsByJob :many
SELECT id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at
FROM job_versions 
WHERE job_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestJobVersion :one
SELECT id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at
FROM job_versions 
WHERE job_id = $1 AND environment_id = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: CountLaterJobVersions :one
SELECT COUNT(*) 
FROM job_versions 
WHERE job_id = $1 AND environment_id = $2 AND version > $3;

-- name: UpdateJobVersionProperties :one
UPDATE job_versions 
SET properties = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, job_id, version, event_specification, properties,
    endpoint_id, environment_id, organization_id, project_id, queue_id,
    start_position, preprocess_runs, created_at, updated_at;

-- name: DeleteJobVersion :exec
DELETE FROM job_versions WHERE id = $1;