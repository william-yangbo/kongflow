-- event_examples.sql
-- EventExample 事件示例相关查询

-- name: CreateEventExample :one
INSERT INTO event_examples (
    job_version_id, slug, name, icon, payload
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, job_version_id, slug, name, icon, payload, created_at, updated_at;

-- name: GetEventExampleByID :one
SELECT id, job_version_id, slug, name, icon, payload, created_at, updated_at
FROM event_examples 
WHERE id = $1;

-- name: GetEventExampleBySlug :one
SELECT id, job_version_id, slug, name, icon, payload, created_at, updated_at
FROM event_examples 
WHERE job_version_id = $1 AND slug = $2;

-- name: UpsertEventExample :one
INSERT INTO event_examples (
    job_version_id, slug, name, icon, payload
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (slug, job_version_id) 
DO UPDATE SET 
    name = EXCLUDED.name,
    icon = EXCLUDED.icon,
    payload = EXCLUDED.payload,
    updated_at = NOW()
RETURNING id, job_version_id, slug, name, icon, payload, created_at, updated_at;

-- name: ListEventExamplesByJobVersion :many
SELECT id, job_version_id, slug, name, icon, payload, created_at, updated_at
FROM event_examples 
WHERE job_version_id = $1
ORDER BY name;

-- name: DeleteEventExample :exec
DELETE FROM event_examples WHERE id = $1;

-- name: DeleteEventExamplesByJobVersion :exec
DELETE FROM event_examples WHERE job_version_id = $1;

-- name: DeleteEventExamplesNotInList :exec
DELETE FROM event_examples 
WHERE job_version_id = $1 AND id != ALL($2::uuid[]);