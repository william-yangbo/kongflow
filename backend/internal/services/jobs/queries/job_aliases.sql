-- job_aliases.sql
-- JobAlias 作业别名相关查询

-- name: CreateJobAlias :one
INSERT INTO job_aliases (
    job_id, version_id, environment_id, name, value
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, job_id, version_id, environment_id, name, value, created_at, updated_at;

-- name: GetJobAliasByID :one
SELECT id, job_id, version_id, environment_id, name, value, created_at, updated_at
FROM job_aliases 
WHERE id = $1;

-- name: GetJobAliasByName :one
SELECT id, job_id, version_id, environment_id, name, value, created_at, updated_at
FROM job_aliases 
WHERE job_id = $1 AND environment_id = $2 AND name = $3;

-- name: UpsertJobAlias :one
INSERT INTO job_aliases (
    job_id, version_id, environment_id, name, value
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (job_id, environment_id, name) 
DO UPDATE SET 
    version_id = EXCLUDED.version_id,
    value = EXCLUDED.value,
    updated_at = NOW()
RETURNING id, job_id, version_id, environment_id, name, value, created_at, updated_at;

-- name: ListJobAliasesByJob :many
SELECT id, job_id, version_id, environment_id, name, value, created_at, updated_at
FROM job_aliases 
WHERE job_id = $1 AND environment_id = $2
ORDER BY name;

-- name: DeleteJobAlias :exec
DELETE FROM job_aliases WHERE id = $1;

-- name: DeleteJobAliasesByJob :exec
DELETE FROM job_aliases WHERE job_id = $1;