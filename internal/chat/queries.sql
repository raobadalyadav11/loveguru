-- name: CreateSession :one
INSERT INTO sessions (user_id, advisor_id, type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1;

-- name: GetMessages :many
SELECT * FROM chat_messages WHERE session_id = $1 ORDER BY created_at LIMIT $2 OFFSET $3;

-- name: InsertMessage :one
INSERT INTO chat_messages (session_id, sender_type, sender_id, content)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSessionStatus :exec
UPDATE sessions SET status = $2, ended_at = NOW() WHERE id = $1;