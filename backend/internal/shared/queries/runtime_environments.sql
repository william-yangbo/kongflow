-- Runtime Environments queries - trigger.dev RuntimeEnvironment alignment
-- name: GetRuntimeEnvironment :one
SELECT * FROM runtime_environments WHERE id = $1 LIMIT 1;

-- name: FindRuntimeEnvironmentByAPIKey :one
SELECT * FROM runtime_environments WHERE api_key = $1 LIMIT 1;

-- name: FindRuntimeEnvironmentByPublicAPIKey :one
SELECT * FROM runtime_environments WHERE api_key = $1 AND type != 'PRODUCTION' LIMIT 1;

-- name: GetEnvironmentWithProjectAndOrg :one
SELECT 
    re.id, re.slug, re.api_key, re.type, re.organization_id, re.project_id, 
    re.org_member_id, re.created_at, re.updated_at,
    p.id as project_id, p.slug as project_slug, p.name as project_name,
    o.id as org_id, o.slug as org_slug, o.title as org_title
FROM runtime_environments re
INNER JOIN projects p ON re.project_id = p.id
INNER JOIN organizations o ON re.organization_id = o.id
WHERE re.id = $1 LIMIT 1;

-- name: CreateRuntimeEnvironment :one
INSERT INTO runtime_environments (slug, api_key, type, organization_id, project_id, org_member_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateRuntimeEnvironment :one
UPDATE runtime_environments 
SET slug = $2, type = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListRuntimeEnvironmentsByProject :many
SELECT * FROM runtime_environments 
WHERE project_id = $1
ORDER BY created_at DESC;