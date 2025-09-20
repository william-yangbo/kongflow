-- event_dispatchers.sql
-- Events Service - EventDispatcher 相关查询，对齐 trigger.dev 功能

-- name: CreateEventDispatcher :one
INSERT INTO event_dispatchers (
    event,
    source,
    payload_filter,
    context_filter,
    manual,
    dispatchable_id,
    dispatchable,
    enabled,
    environment_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetEventDispatcherByID :one
SELECT * FROM event_dispatchers 
WHERE id = $1;

-- name: FindEventDispatchers :many
-- 查找匹配的事件调度器，对齐 trigger.dev DeliverEventService 逻辑
SELECT * FROM event_dispatchers
WHERE environment_id = $1
    AND event = $2
    AND source = $3
    AND ($4::BOOLEAN = false OR enabled = true)
    AND ($5::BOOLEAN IS NULL OR manual = $5)
ORDER BY created_at ASC;

-- name: ListEventDispatchers :many
SELECT * FROM event_dispatchers
WHERE environment_id = $1
    AND ($2::TEXT IS NULL OR event = $2)
    AND ($3::TEXT IS NULL OR source = $3)
    AND ($4::BOOLEAN = false OR enabled = true)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountEventDispatchers :one
SELECT COUNT(*) FROM event_dispatchers
WHERE environment_id = $1
    AND ($2::TEXT IS NULL OR event = $2)
    AND ($3::TEXT IS NULL OR source = $3)
    AND ($4::BOOLEAN = false OR enabled = true);

-- name: UpdateEventDispatcherEnabled :exec
UPDATE event_dispatchers 
SET enabled = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteEventDispatcher :exec
DELETE FROM event_dispatchers WHERE id = $1;

-- name: UpsertEventDispatcher :one
-- Upsert 事件调度器，用于端点注册时更新调度器
INSERT INTO event_dispatchers (
    event,
    source,
    payload_filter,
    context_filter,
    manual,
    dispatchable_id,
    dispatchable,
    enabled,
    environment_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (dispatchable_id, environment_id)
DO UPDATE SET
    event = EXCLUDED.event,
    source = EXCLUDED.source,
    payload_filter = EXCLUDED.payload_filter,
    context_filter = EXCLUDED.context_filter,
    manual = EXCLUDED.manual,
    dispatchable = EXCLUDED.dispatchable,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
RETURNING *;