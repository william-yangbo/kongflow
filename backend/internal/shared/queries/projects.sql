-- Projects queries - trigger.dev Project entity alignment  
-- name: GetProject :one
SELECT * FROM projects WHERE id = $1 LIMIT 1;

-- name: FindProjectBySlug :one
SELECT * FROM projects WHERE organization_id = $1 AND slug = $2 LIMIT 1;

-- name: CreateProject :one
INSERT INTO projects (name, slug, organization_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects 
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListProjectsByOrganization :many
SELECT * FROM projects 
WHERE organization_id = $1
ORDER BY created_at DESC;