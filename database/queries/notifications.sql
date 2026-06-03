-- name: CreateNotification :one
INSERT INTO notifications (user_id, title, message)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, message, is_read, created_at;

-- name: FindNotificationsByUserID :many
SELECT id, user_id, title, message, is_read, created_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountNotificationsByUserID :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1;

-- name: MarkNotificationAsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE id = $1 AND user_id = $2;

-- name: ClearNotificationsByUserID :exec
DELETE FROM notifications
WHERE user_id = $1;
