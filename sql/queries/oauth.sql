-- name: GetUserIdFromGithub :one
SELECT user_id
FROM oauth_accounts
WHERE provider = 'github'
    AND provider_user_id = $1;
--

-- name: CreateOAuthGithubAccount :one
INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, created_at)
VALUES(
    $1,
    $2,
    'github',
    $3,
    NOW() AT TIME ZONE 'UTC'
)
RETURNING *;
--
