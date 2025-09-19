-- jobs.sql
-- Job 作业相关查询，对齐 trigger.dev 的 Job 操作

-- name: CreateJob :one
INSERT INTO jobs (
    slug, title, internal, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, slug, title, internal, organization_id, project_id, created_at, updated_at;

-- name: GetJobByID :one
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs 
WHERE id = $1;

-- name: GetJobBySlug :one
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs 
WHERE project_id = $1 AND slug = $2;

-- name: UpsertJob :one
INSERT INTO jobs (
    slug, title, internal, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, slug) 
DO UPDATE SET 
    title = EXCLUDED.title,
    internal = EXCLUDED.internal,
    updated_at = NOW()
RETURNING id, slug, title, internal, organization_id, project_id, created_at, updated_at;

-- name: ListJobsByProject :many
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs 
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountJobsByProject :one
SELECT COUNT(*) FROM jobs WHERE project_id = $1;

-- name: UpdateJob :one
UPDATE jobs 
SET title = $2, internal = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, slug, title, internal, organization_id, project_id, created_at, updated_at;

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = $1;

-- name: ListJobsByOrganization :many
SELECT id, slug, title, internal, organization_id, project_id, created_at, updated_at
FROM jobs 
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;