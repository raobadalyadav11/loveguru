-- name: GetPendingAdvisors :many
SELECT a.*, u.* FROM advisors a JOIN users u ON a.user_id = u.id WHERE a.status = 'PENDING' LIMIT $1 OFFSET $2;

-- name: ApproveAdvisor :exec
UPDATE advisors SET is_verified = TRUE, status = 'OFFLINE' WHERE id = $1;

-- name: GetFlags :many
SELECT * FROM admin_flags ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: BlockUser :exec
UPDATE users SET is_active = FALSE WHERE id = $1;