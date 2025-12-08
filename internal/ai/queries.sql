-- name: InsertAIInteraction :one
INSERT INTO ai_interactions (user_id, prompt, response)
VALUES ($1, $2, $3)
RETURNING *;