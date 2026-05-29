-- name: CountAllTransactions :one
SELECT COUNT(*) FROM transactions;

-- name: CountAllTransactionsByType :one
SELECT COUNT(*) FROM transactions WHERE type = $1;

-- name: CountAllTransactionsByStatus :one
SELECT COUNT(*) FROM transactions WHERE status = $1;

-- name: CountAllTransactionsByTypeAndStatus :one
SELECT COUNT(*) FROM transactions WHERE type = $1 AND status = $2;

-- name: FindAllTransactions :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: FindAllTransactionsByType :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindAllTransactionsByStatus :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindAllTransactionsByTypeAndStatus :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE type = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
