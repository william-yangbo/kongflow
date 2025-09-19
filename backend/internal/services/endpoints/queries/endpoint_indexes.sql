-- name: CreateEndpointIndex :one
INSERT INTO endpoint_indexes (
    endpoint_id, source, stats, data
) VALUES ($1, $2, $3, $4)
RETURNING id, endpoint_id, source, stats, data, source_data, reason,
    created_at, updated_at;

-- name: GetEndpointIndexByID :one
SELECT id, endpoint_id, source, stats, data, source_data, reason,
    created_at, updated_at
FROM endpoint_indexes 
WHERE id = $1;

-- name: ListEndpointIndexes :many
SELECT id, endpoint_id, source, stats, data, source_data, reason,
    created_at, updated_at
FROM endpoint_indexes 
WHERE endpoint_id = $1
ORDER BY created_at DESC;

-- name: DeleteEndpointIndex :exec
DELETE FROM endpoint_indexes WHERE id = $1;