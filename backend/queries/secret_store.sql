-- name: GetSecretStore :one
SELECT * FROM secret_store WHERE key = $1;

-- name: UpsertSecretStore :exec
INSERT INTO secret_store (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = NOW();

-- name: DeleteSecretStore :exec
DELETE FROM secret_store WHERE key = $1;

-- name: ListSecretStoreKeys :many
SELECT key, created_at, updated_at FROM secret_store
ORDER BY created_at DESC;

-- name: GetSecretStoreCount :one
SELECT COUNT(*) FROM secret_store;