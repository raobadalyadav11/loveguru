-- name: CreateCallSession :one
INSERT INTO sessions (user_id, advisor_id, type)
VALUES ($1, $2, 'CALL')
RETURNING *;

-- name: EndCall :exec
UPDATE sessions SET status = 'ENDED', ended_at = NOW() WHERE id = $1;

-- name: InsertCallLog :one
INSERT INTO call_logs (session_id, external_call_id, started_at, ended_at, duration_seconds, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;