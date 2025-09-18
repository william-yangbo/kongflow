-- Personal Access Tokens queries - trigger.dev PAT alignment
-- name: FindPersonalAccessToken :one
SELECT * FROM personal_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

-- name: FindPersonalAccessTokenWithUser :one
SELECT 
    pat.*,
    u.id as user_id, u.email as user_email, u.name as user_name, u.avatar_url as user_avatar_url
FROM personal_access_tokens pat
INNER JOIN users u ON pat.user_id = u.id
WHERE pat.token = $1 AND pat.expires_at > NOW() LIMIT 1;

-- name: CreatePersonalAccessToken :one
INSERT INTO personal_access_tokens (user_id, token, name, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePersonalTokenLastUsed :exec
UPDATE personal_access_tokens SET last_used_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ListPersonalAccessTokensByUser :many
SELECT * FROM personal_access_tokens 
WHERE user_id = $1 AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: RevokePersonalAccessToken :exec
DELETE FROM personal_access_tokens WHERE id = $1 AND user_id = $2;