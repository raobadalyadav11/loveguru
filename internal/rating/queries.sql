-- name: CreateRating :one
INSERT INTO ratings (session_id, user_id, advisor_id, rating, review_text)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAdvisorRatings :many
SELECT * FROM ratings WHERE advisor_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;