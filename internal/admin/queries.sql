-- name: GetPendingAdvisors :many
SELECT a.*, u.* FROM advisors a JOIN users u ON a.user_id = u.id WHERE a.status = 'PENDING' LIMIT $1 OFFSET $2;

-- name: ApproveAdvisor :exec
UPDATE advisors SET is_verified = TRUE, status = 'OFFLINE' WHERE id = $1;

-- name: GetFlags :many
SELECT * FROM admin_flags ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: BlockUser :exec
UPDATE users SET is_active = FALSE WHERE id = $1;
-- Specializations Management

-- name: GetAllSpecializations :many
SELECT id, name, description, category, is_active FROM specializations ORDER BY category, name;

-- name: GetActiveSpecializationsByCategory :many
SELECT id, name, description, category FROM specializations WHERE category = $1 AND is_active = true ORDER BY name;

-- name: CreateSpecialization :one
INSERT INTO specializations (name, description, category) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateSpecialization :exec
UPDATE specializations SET name = $1, description = $2, category = $3, is_active = $4 WHERE id = $5;

-- name: DeleteSpecialization :exec
DELETE FROM specializations WHERE id = $1;

-- name: GetUserSpecializations :many
SELECT s.name, s.category FROM specializations s
JOIN advisors a ON a.specializations && ARRAY[s.name]
WHERE a.user_id = $1;