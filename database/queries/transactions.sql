-- name: CreateTransaction :one
INSERT INTO transactions (reference_no, type, status, amount, source_wallet_id, target_wallet_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at;

-- name: FindTransactionByID :one
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE id = $1;

-- name: UpdateTransactionStatus :exec
UPDATE transactions
SET status = $2
WHERE id = $1;

-- name: CountTransactionsByWalletID :one
SELECT COUNT(*) FROM transactions
WHERE (source_wallet_id = $1 OR target_wallet_id = $1);

-- name: CountTransactionsByWalletIDAndType :one
SELECT COUNT(*) FROM transactions
WHERE (source_wallet_id = $1 OR target_wallet_id = $1)
AND type = $2;

-- name: FindTransactionsByWalletID :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE (source_wallet_id = $1 OR target_wallet_id = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindTransactionsByWalletIDAndType :many
SELECT id, reference_no, type, status, amount, source_wallet_id, target_wallet_id, created_at
FROM transactions
WHERE (source_wallet_id = $1 OR target_wallet_id = $1)
AND type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
