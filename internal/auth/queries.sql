-- name: CreateUser :one
INSERT INTO users (email, phone, password_hash, display_name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUser :one
UPDATE users SET display_name = $2, gender = $3, dob = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserSessions :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3;