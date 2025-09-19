-- event_records.sql
-- EventRecord 事件记录相关查询

-- name: CreateEventRecord :one
INSERT INTO event_records (
    event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at;

-- name: GetEventRecordByID :one
SELECT id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at
FROM event_records 
WHERE id = $1;

-- name: GetEventRecordByEventID :one
SELECT id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at
FROM event_records 
WHERE event_id = $1 AND environment_id = $2;

-- name: ListEventRecordsByEnvironment :many
SELECT id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at
FROM event_records 
WHERE environment_id = $1
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3;

-- name: ListTestEventRecords :many
SELECT id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at
FROM event_records 
WHERE environment_id = $1 AND is_test = true
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3;

-- name: CountEventRecordsByEnvironment :one
SELECT COUNT(*) FROM event_records WHERE environment_id = $1;

-- name: CountTestEventRecords :one
SELECT COUNT(*) FROM event_records WHERE environment_id = $1 AND is_test = true;

-- name: DeleteEventRecord :exec
DELETE FROM event_records WHERE id = $1;

-- name: DeleteEventRecordByEventID :exec
DELETE FROM event_records WHERE event_id = $1 AND environment_id = $2;

-- name: ListEventRecordsByNameAndSource :many
SELECT id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at
FROM event_records 
WHERE environment_id = $1 AND name = $2 AND source = $3
ORDER BY timestamp DESC
LIMIT $4 OFFSET $5;

-- name: UpdateEventRecord :one
UPDATE event_records 
SET name = $2, source = $3, payload = $4, context = $5, timestamp = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, event_id, name, source, payload, context, timestamp,
    environment_id, organization_id, project_id, is_test, created_at, updated_at;