-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (id, transaction_id, wallet_id, entry_type, amount)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, transaction_id, wallet_id, entry_type, amount, created_at;

-- name: FindLedgerEntriesByTransactionID :many
SELECT id, transaction_id, wallet_id, entry_type, amount, created_at
FROM ledger_entries
WHERE transaction_id = $1
ORDER BY created_at ASC;

-- name: CountLedgerEntriesByWalletID :one
SELECT COUNT(*) FROM ledger_entries
WHERE wallet_id = $1;

-- name: CountLedgerEntriesByWalletIDAndType :one
SELECT COUNT(*) FROM ledger_entries
WHERE wallet_id = $1 AND entry_type = $2;

-- name: FindLedgerEntriesByWalletID :many
SELECT id, transaction_id, wallet_id, entry_type, amount, created_at
FROM ledger_entries
WHERE wallet_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: FindLedgerEntriesByWalletIDAndType :many
SELECT id, transaction_id, wallet_id, entry_type, amount, created_at
FROM ledger_entries
WHERE wallet_id = $1 AND entry_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
