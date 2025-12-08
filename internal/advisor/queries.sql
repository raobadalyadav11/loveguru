-- name: CreateAdvisor :one
INSERT INTO advisors (user_id, bio, experience_years, languages, specializations, hourly_rate)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAdvisorByID :one
SELECT a.*, u.* FROM advisors a JOIN users u ON a.user_id = u.id WHERE a.id = $1;

-- name: GetAdvisorByUserID :one
SELECT * FROM advisors WHERE user_id = $1;

-- name: UpdateAdvisor :one
UPDATE advisors SET bio = $2, experience_years = $3, languages = $4, specializations = $5, hourly_rate = $6, status = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListAdvisors :many
SELECT a.*, u.*, 0 as average_rating
FROM advisors a
JOIN users u ON a.user_id = u.id
WHERE a.status = 'ONLINE'
LIMIT @limit_rows OFFSET @offset_rows;