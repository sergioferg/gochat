-- name: CreateSession :one
INSERT INTO sessions (
    id,
    token_hash,
    created_at,
    updated_at,
    user_id,
    expires_at,
    user_agent,
    ip_address,
    revoked_at
)
VALUES (
    $1,
    $2,
    NOW(),
    NOW(),
    $3,
    NOW() + INTERVAL '60 days',
    $4,
    $5,
    NULL
)
RETURNING *;
--

-- name: GetUserFromSession :one
SELECT u.*
FROM sessions s
JOIN users u ON(s.user_id = u.id)
WHERE s.token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();
--

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = NOW(),
    updated_at = NOW()
WHERE
    id = $1
    AND user_id = $2
    AND revoked_at IS NULL;
--

-- name: DeleteUserSessions :exec
DELETE FROM sessions
WHERE user_id = $1;
--

-- name: GetUserSessions :many
SELECT id, created_at, user_agent, ip_address
FROM sessions
WHERE user_id = $1
    AND expires_at > NOW()
    AND revoked_at IS NULL
ORDER BY created_at DESC;
--

-- name: RevokeSessionByToken :exec
UPDATE sessions
SET
    revoked_at = NOW(),
    updated_at = NOW()
WHERE
    token_hash = $1
    AND revoked_at IS NULL;
--
