-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, created_at, updated_at;

-- name: FindUserByEmail :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: FindUserByID :one
SELECT id, email, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CountAllUsers :one
SELECT COUNT(*) FROM users;

-- name: FindAllUsers :many
SELECT 
    u.id, 
    u.email, 
    u.role, 
    u.created_at, 
    u.updated_at,
    w.id AS wallet_id,
    w.balance AS wallet_balance
FROM users u
LEFT JOIN wallets w ON u.id = w.user_id
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;
