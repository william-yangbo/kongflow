-- ApiVote Service SQL Queries
-- 严格对齐 trigger.dev 的 ApiVoteService 功能

-- name: CreateApiVote :one
-- 对齐 trigger.dev: apiIntegrationVote.create()
INSERT INTO "ApiIntegrationVote" (
    "id",
    "apiIdentifier", 
    "userId",
    "createdAt",
    "updatedAt"
) VALUES (
    $1, $2, $3,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) RETURNING *;

-- name: GetApiVoteByUserAndIdentifier :one
-- 查询用户对特定API的投票记录，用于防重复投票逻辑
SELECT * FROM "ApiIntegrationVote"
WHERE "apiIdentifier" = $1 AND "userId" = $2;

-- name: DeleteApiVoteByUserAndIdentifier :exec
-- 删除用户对特定API的投票（可选功能）
DELETE FROM "ApiIntegrationVote"
WHERE "apiIdentifier" = $1 AND "userId" = $2;