-- name: GetRecommendedAdvisors :many
SELECT a.*, u.*, 
       COALESCE(AVG(r.rating), 0) as average_rating
FROM advisors a
JOIN users u ON a.user_id = u.id
LEFT JOIN ratings r ON a.user_id = r.advisor_id
WHERE a.status = 'ONLINE' 
AND a.is_verified = true
AND ($1::text[] IS NULL OR a.specializations && $1::text[])
AND ($2::text[] IS NULL OR a.languages && $2::text[])
GROUP BY a.id, u.id
ORDER BY average_rating DESC, a.experience_years DESC
LIMIT $3 OFFSET $4;

-- name: GetUserSessionHistory :many
SELECT s.*, a.user_id as advisor_user_id
FROM sessions s
LEFT JOIN advisors a ON s.advisor_id = a.id
WHERE s.user_id = $1
ORDER BY s.started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAdvisorRatingsWithReviewer :many
SELECT r.*, u.display_name as reviewer_name
FROM ratings r
JOIN users u ON r.user_id = u.id
WHERE r.advisor_id = $1
ORDER BY r.created_at DESC;