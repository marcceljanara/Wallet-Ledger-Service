-- name: CreateWallet :one
INSERT INTO wallets (id, user_id, balance, currency)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, balance, currency, created_at, updated_at;

-- name: FindWalletByUserID :one
SELECT id, user_id, balance, currency, created_at, updated_at
FROM wallets
WHERE user_id = $1;

-- name: FindWalletByID :one
SELECT id, user_id, balance, currency, created_at, updated_at
FROM wallets
WHERE id = $1;

-- name: FindWalletByIDForUpdate :one
SELECT id, user_id, balance, currency, created_at, updated_at
FROM wallets
WHERE id = $1
FOR UPDATE;

-- name: UpdateWalletBalance :exec
UPDATE wallets
SET balance = $2, updated_at = NOW()
WHERE id = $1;
