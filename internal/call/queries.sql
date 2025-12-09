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
-- Call Status and Feedback

-- name: UpdateCallStatus :exec
UPDATE call_logs SET status_update = $1, status_timestamp = NOW() WHERE session_id = $2;

-- name: GetCallStatus :one
SELECT status_update, status_timestamp FROM call_logs WHERE session_id = $1 ORDER BY status_timestamp DESC LIMIT 1;

-- name: CreateFeedbackPrompt :one
INSERT INTO call_feedback_prompts (session_id, user_id, advisor_id) VALUES ($1, $2, $3) RETURNING id;

-- name: GetPendingFeedbackPrompts :many
SELECT cfp.id, cfp.session_id, u.display_name as user_name, a.display_name as advisor_name, cfp.prompt_sent_at
FROM call_feedback_prompts cfp
JOIN users u ON u.id = cfp.user_id
JOIN users a ON a.id = cfp.advisor_id
WHERE cfp.response_received_at IS NULL
ORDER BY cfp.prompt_sent_at DESC;

-- name: SubmitFeedback :exec
UPDATE call_feedback_prompts
SET response_received_at = NOW(), rating = $1, feedback_text = $2
WHERE id = $3;

-- name: GetRecentEndedSessions :many
SELECT id, user_id, advisor_id, type, started_at, ended_at, status
FROM sessions
WHERE status = 'ENDED'
AND ended_at >= NOW() - INTERVAL '24 hours'
ORDER BY ended_at DESC;

-- name: GetFeedbackPromptBySession :one
SELECT id, session_id, user_id, advisor_id, prompt_sent_at, response_received_at, rating, feedback_text
FROM call_feedback_prompts
WHERE session_id = $1
ORDER BY prompt_sent_at DESC
LIMIT 1;

-- name: GetCallSessionByID :one
SELECT id, user_id, advisor_id, type, started_at, ended_at, status FROM sessions WHERE id = $1;