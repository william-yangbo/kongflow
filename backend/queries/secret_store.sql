-- name: GetSecretStore :one
SELECT * FROM secret_store WHERE key = $1;

-- name: UpsertSecretStore :exec
INSERT INTO secret_store (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = NOW();