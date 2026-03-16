-- name: CreateUser :one
INSERT INTO users (email, username, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, username, role, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, username, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, username, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, email, username, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: CreatePasswordReset :one
INSERT INTO password_resets (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, used, created_at;

-- name: GetPasswordResetByTokenHash :one
SELECT id, user_id, token_hash, expires_at, used, created_at
FROM password_resets
WHERE token_hash = $1 AND used = false;

-- name: MarkPasswordResetUsed :exec
UPDATE password_resets
SET used = true
WHERE id = $1;

