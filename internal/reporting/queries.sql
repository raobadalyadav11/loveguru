-- name: CreateAdminFlag :one
INSERT INTO admin_flags (reported_by, reported_user_id, reported_advisor_id, session_id, reason)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserReports :many
SELECT * FROM admin_flags WHERE reported_user_id = $1 OR reported_advisor_id = $1 ORDER BY created_at DESC;

-- name: GetReportsByStatus :many
SELECT * FROM admin_flags WHERE status = $1 ORDER BY created_at DESC;

-- name: UpdateAdminFlagStatus :exec
UPDATE admin_flags SET status = $2 WHERE id = $1;

-- name: CountTotalReports :one
SELECT COUNT(*) FROM admin_flags;

-- name: CountPendingReports :one
SELECT COUNT(*) FROM admin_flags WHERE status = 'PENDING';

-- name: CountResolvedReports :one
SELECT COUNT(*) FROM admin_flags WHERE status IN ('RESOLVED', 'DISMISSED');

-- name: GetRecentAdminFlags :many
SELECT * FROM admin_flags WHERE created_at >= NOW() - INTERVAL '%s days' ORDER BY created_at DESC;