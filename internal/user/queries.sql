-- name: UpdateUserCredentials :one
UPDATE users SET email = $2, phone = $3, password_hash = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, email, phone, password_hash, display_name, role, gender, dob, created_at, updated_at, is_active;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserFCMToken :exec
UPDATE users SET fcm_token = $2, device_type = $3, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserAPNSToken :exec
UPDATE users SET apns_token = $2, device_type = $3, updated_at = NOW()
WHERE id = $1;

-- name: GetUserDeviceTokens :one
SELECT fcm_token, apns_token, device_type FROM users WHERE id = $1;

-- name: GetSessionParticipantDeviceTokens :many
SELECT DISTINCT u.fcm_token, u.apns_token, u.device_type
FROM users u
JOIN sessions s ON (s.user_id = u.id OR s.advisor_id = u.id)
WHERE s.id = $1
  AND u.id != $2
  AND (u.fcm_token IS NOT NULL OR u.apns_token IS NOT NULL);