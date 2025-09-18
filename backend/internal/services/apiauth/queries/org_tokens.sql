-- Organization Access Tokens queries - trigger.dev OAT alignment
-- name: FindOrganizationAccessToken :one
SELECT * FROM organization_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

-- name: FindOrganizationAccessTokenWithOrg :one
SELECT 
    oat.*,
    o.id as org_id, o.title as org_title, o.slug as org_slug
FROM organization_access_tokens oat
INNER JOIN organizations o ON oat.organization_id = o.id
WHERE oat.token = $1 AND oat.expires_at > NOW() LIMIT 1;

-- name: CreateOrganizationAccessToken :one
INSERT INTO organization_access_tokens (organization_id, token, name, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateOrgTokenLastUsed :exec
UPDATE organization_access_tokens SET last_used_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ListOrganizationAccessTokensByOrg :many
SELECT * FROM organization_access_tokens 
WHERE organization_id = $1 AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: RevokeOrganizationAccessToken :exec
DELETE FROM organization_access_tokens WHERE id = $1 AND organization_id = $2;