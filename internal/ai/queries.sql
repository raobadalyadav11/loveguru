-- name: InsertAIInteraction :one
INSERT INTO ai_interactions (user_id, prompt, response)
VALUES ($1, $2, $3)
RETURNING *;
-- FAQ Management

-- name: GetAllFAQs :many
SELECT id, question, answer, category, is_active FROM faqs ORDER BY category, question;

-- name: GetFAQsByCategory :many
SELECT id, question, answer, category, is_active FROM faqs WHERE category = $1 AND is_active = true ORDER BY question;

-- name: SearchFAQs :many
SELECT id, question, answer, category, is_active
FROM faqs
WHERE (question ILIKE '%' || $1 || '%' OR answer ILIKE '%' || $1 || '%')
  AND is_active = true
ORDER BY category, question;

-- name: CreateFAQ :one
INSERT INTO faqs (question, answer, category) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateFAQ :exec
UPDATE faqs SET question = $1, answer = $2, category = $3, is_active = $4 WHERE id = $5;

-- name: DeleteFAQ :exec
DELETE FROM faqs WHERE id = $1;