-- name: CreateEndpoint :one
INSERT INTO endpoints (
    slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id,
    created_at, updated_at;

-- name: GetEndpointByID :one
SELECT id, slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id,
    created_at, updated_at
FROM endpoints 
WHERE id = $1;

-- name: GetEndpointBySlug :one
SELECT id, slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id,
    created_at, updated_at
FROM endpoints 
WHERE environment_id = $1 AND slug = $2;

-- name: UpdateEndpointURL :one
UPDATE endpoints 
SET url = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id,
    created_at, updated_at;

-- name: UpsertEndpoint :one
INSERT INTO endpoints (
    slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (environment_id, slug) 
DO UPDATE SET 
    url = EXCLUDED.url,
    indexing_hook_identifier = EXCLUDED.indexing_hook_identifier,
    updated_at = NOW()
RETURNING id, slug, url, indexing_hook_identifier,
    environment_id, organization_id, project_id,
    created_at, updated_at;

-- name: DeleteEndpoint :exec
DELETE FROM endpoints WHERE id = $1;