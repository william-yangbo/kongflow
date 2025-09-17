-- name: GetSecretStore :one
SELECT * FROM "SecretStore" WHERE "key" = $1;

-- name: UpsertSecretStore :exec
INSERT INTO "SecretStore" ("key", "value", "createdAt", "updatedAt") 
VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT ("key") DO UPDATE SET
    "value" = EXCLUDED."value",
    "updatedAt" = CURRENT_TIMESTAMP;