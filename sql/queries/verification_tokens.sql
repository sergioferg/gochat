-- name: DeleteVerificationTokensByUserID :exec
DELETE FROM email_verification_tokens
WHERE user_id = $1;
--

-- name: CreateVerificationToken :one
INSERT INTO email_verification_tokens (
    token_hash,
    user_id
)
VALUES (
    $1,
    $2
)
RETURNING *;
--

-- name: GetUserFromVerificationToken :one
SELECT u.*
FROM email_verification_tokens evt
JOIN users u ON evt.user_id = u.id
WHERE evt.token_hash = $1
  AND evt.expires_at > (NOW() AT TIME ZONE 'UTC');
--
