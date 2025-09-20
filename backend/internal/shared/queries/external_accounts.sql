-- external_accounts.sql
-- External Accounts 共享查询，对齐 trigger.dev 访问模式

-- name: FindExternalAccountByEnvAndIdentifier :one
SELECT * FROM external_accounts 
WHERE environment_id = $1 AND identifier = $2 
LIMIT 1;

-- name: CreateExternalAccount :one
INSERT INTO external_accounts (identifier, metadata, organization_id, environment_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateExternalAccountMetadata :exec
UPDATE external_accounts 
SET metadata = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListExternalAccountsByEnvironment :many
SELECT * FROM external_accounts 
WHERE environment_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetExternalAccountByID :one
SELECT * FROM external_accounts 
WHERE id = $1 
LIMIT 1;