-- Device Token Management Queries

-- Update user's FCM token
UPDATE users SET fcm_token = $1, device_type = $2 WHERE id = $3;

-- Update user's APNS token  
UPDATE users SET apns_token = $1, device_type = $2 WHERE id = $3;

-- Get user's device tokens
SELECT fcm_token, apns_token, device_type FROM users WHERE id = $1;

-- Get all active device tokens for a user (excluding their own)
SELECT DISTINCT u.fcm_token, u.apns_token, u.device_type
FROM users u
JOIN sessions s ON (s.user_id = u.id OR s.advisor_id = u.id)
JOIN chat_messages cm ON cm.session_id = s.id
WHERE s.id = $1 
  AND u.id != $2
  AND (u.fcm_token IS NOT NULL OR u.apns_token IS NOT NULL);

-- FAQ Management Queries

-- Get all FAQs
SELECT id, question, answer, category, is_active FROM faqs ORDER BY category, question;

-- Get FAQs by category
SELECT id, question, answer, category, is_active FROM faqs WHERE category = $1 AND is_active = true ORDER BY question;

-- Search FAQs
SELECT id, question, answer, category, is_active 
FROM faqs 
WHERE (question ILIKE '%' || $1 || '%' OR answer ILIKE '%' || $1 || '%')
  AND is_active = true
ORDER BY category, question;

-- Create FAQ
INSERT INTO faqs (question, answer, category) VALUES ($1, $2, $3) RETURNING id;

-- Update FAQ
UPDATE faqs SET question = $1, answer = $2, category = $3, is_active = $4 WHERE id = $5;

-- Delete FAQ
DELETE FROM faqs WHERE id = $1;

-- Call Status and Feedback Queries

-- Update call status
UPDATE call_logs SET status_update = $1, status_timestamp = NOW() WHERE session_id = $2;

-- Get call status
SELECT status_update, status_timestamp FROM call_logs WHERE session_id = $1 ORDER BY status_timestamp DESC LIMIT 1;

-- Create feedback prompt
INSERT INTO call_feedback_prompts (session_id, user_id, advisor_id) VALUES ($1, $2, $3) RETURNING id;

-- Get pending feedback prompts
SELECT cfp.id, cfp.session_id, u.display_name as user_name, a.display_name as advisor_name, cfp.prompt_sent_at
FROM call_feedback_prompts cfp
JOIN users u ON u.id = cfp.user_id
JOIN users a ON a.id = cfp.advisor_id
WHERE cfp.response_received_at IS NULL
ORDER BY cfp.prompt_sent_at DESC;

-- Submit feedback
UPDATE call_feedback_prompts 
SET response_received_at = NOW(), rating = $1, feedback_text = $2 
WHERE id = $3;

-- Specializations Management Queries

-- Get all specializations
SELECT id, name, description, category, is_active FROM specializations ORDER BY category, name;

-- Get active specializations by category
SELECT id, name, description, category FROM specializations WHERE category = $1 AND is_active = true ORDER BY name;

-- Create specialization
INSERT INTO specializations (name, description, category) VALUES ($1, $2, $3) RETURNING id;

-- Update specialization
UPDATE specializations SET name = $1, description = $2, category = $3, is_active = $4 WHERE id = $5;

-- Delete specialization
DELETE FROM specializations WHERE id = $1;

-- Get user specializations for advisors
SELECT s.name, s.category FROM specializations s
JOIN advisors a ON a.specializations && ARRAY[s.name]
WHERE a.user_id = $1;