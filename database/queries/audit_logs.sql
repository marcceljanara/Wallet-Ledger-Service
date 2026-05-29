-- name: CreateAuditLog :one
INSERT INTO audit_logs (id, user_id, action, ip_address, endpoint)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, action, ip_address, endpoint, created_at;

-- name: CountAuditLogsByUserID :one
SELECT COUNT(*) FROM audit_logs
WHERE user_id = $1;

-- name: FindAuditLogsByUserID :many
SELECT id, user_id, action, ip_address, endpoint, created_at
FROM audit_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAllAuditLogs :one
SELECT COUNT(*) FROM audit_logs;

-- name: CountAllAuditLogsByUserID :one
SELECT COUNT(*) FROM audit_logs
WHERE user_id = $1;

-- name: CountAllAuditLogsByAction :one
SELECT COUNT(*) FROM audit_logs
WHERE action = $1;

-- name: CountAllAuditLogsByUserIDAndAction :one
SELECT COUNT(*) FROM audit_logs
WHERE user_id = $1 AND action = $2;

-- name: FindAllAuditLogs :many
SELECT id, user_id, action, ip_address, endpoint, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: FindAllAuditLogsByUserID :many
SELECT id, user_id, action, ip_address, endpoint, created_at
FROM audit_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindAllAuditLogsByAction :many
SELECT id, user_id, action, ip_address, endpoint, created_at
FROM audit_logs
WHERE action = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindAllAuditLogsByUserIDAndAction :many
SELECT id, user_id, action, ip_address, endpoint, created_at
FROM audit_logs
WHERE user_id = $1 AND action = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
