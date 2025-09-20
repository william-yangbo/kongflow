-- event_records.sql
-- Events Service - EventRecord 相关查询，对齐 trigger.dev 功能

-- name: CreateEventRecord :one
INSERT INTO event_records (
    event_id,
    name,
    timestamp,
    payload,
    context,
    source,
    organization_id,
    environment_id,
    project_id,
    external_account_id,
    deliver_at,
    is_test
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetEventRecordByID :one
SELECT * FROM event_records 
WHERE id = $1;

-- name: GetEventRecordByEventID :one
SELECT * FROM event_records 
WHERE event_id = $1 AND environment_id = $2;

-- name: UpdateEventRecordDeliveredAt :exec
UPDATE event_records 
SET delivered_at = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListEventRecords :many
SELECT * FROM event_records
WHERE 
    ($1::UUID IS NULL OR environment_id = $1) AND
    ($2::UUID IS NULL OR project_id = $2) AND
    ($3::TEXT IS NULL OR source = $3) AND
    ($4::BOOLEAN IS NULL OR ($4 = true AND delivered_at IS NOT NULL) OR ($4 = false AND delivered_at IS NULL)) AND
    ($5::TIMESTAMPTZ IS NULL OR created_at >= $5) AND
    ($6::TIMESTAMPTZ IS NULL OR created_at <= $6)
ORDER BY created_at DESC
LIMIT $7 OFFSET $8;

-- name: CountEventRecords :one
SELECT COUNT(*) FROM event_records
WHERE 
    ($1::UUID IS NULL OR environment_id = $1) AND
    ($2::UUID IS NULL OR project_id = $2) AND
    ($3::TEXT IS NULL OR source = $3) AND
    ($4::BOOLEAN IS NULL OR ($4 = true AND delivered_at IS NOT NULL) OR ($4 = false AND delivered_at IS NULL)) AND
    ($5::TIMESTAMPTZ IS NULL OR created_at >= $5) AND
    ($6::TIMESTAMPTZ IS NULL OR created_at <= $6);

-- name: ListPendingEventRecords :many
-- 获取待投递的事件记录，用于调度
SELECT * FROM event_records
WHERE delivered_at IS NULL 
    AND deliver_at <= NOW()
    AND ($1::UUID IS NULL OR environment_id = $1)
ORDER BY deliver_at ASC
LIMIT $2;

-- name: DeleteEventRecord :exec
DELETE FROM event_records WHERE id = $1;