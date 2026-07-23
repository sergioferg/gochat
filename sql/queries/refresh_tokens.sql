-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    token_hash,
    created_at,
    updated_at,
    user_id,
    expires_at,
    revoked_at
)
VALUES (
    $1,
    NOW() AT TIME ZONE 'UTC',
    NOW() AT TIME ZONE 'UTC',
    $2,
    (NOW() AT TIME ZONE 'UTC') + INTERVAL '60 days',
    NULL
)
RETURNING *;
--

-- name: GetUserFromRefreshToken :one
SELECT u.*
FROM refresh_tokens rt
JOIN users u ON(rt.user_id = u.id)
WHERE rt.token_hash = $1 AND revoked_at IS NULL AND expires_at > (NOW() AT TIME ZONE 'UTC');
--

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET expires_at = (NOW() AT TIME ZONE 'UTC')
WHERE token_hash = $1;
--

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;
--
