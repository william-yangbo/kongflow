-- Organizations queries - trigger.dev Organization entity alignment
-- name: GetOrganization :one
SELECT * FROM organizations WHERE id = $1 LIMIT 1;

-- name: FindOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1 LIMIT 1;

-- name: CreateOrganization :one
INSERT INTO organizations (title, slug)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations 
SET title = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC;