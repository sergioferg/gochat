-- name: CreateUser :one
INSERT INTO users(
    id,
    nickname,
    real_name,
    birth_date,
    email,
    hashed_password,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    COALESCE(sqlc.narg('status'), 'unverified')
)
RETURNING *;
--

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;
--

-- name: GetUserByID :many
SELECT
    u.id,
    u.email,
    u.nickname,
    u.real_name,
    u.birth_date,
    u.created_at,
    u.updated_at,
    oa.provider,
    oa.provider_user_id
FROM users u
JOIN oauth_accounts oa ON(oa.user_id = u.id)
WHERE u.id = $1;
--

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE(sqlc.narg('email'), email),
    nickname = COALESCE(sqlc.narg('nickname'), nickname),
    real_name = COALESCE(sqlc.narg('real_name'), real_name),
    birth_date = COALESCE(sqlc.narg('birth_date'), birth_date),
    hashed_password = COALESCE(sqlc.narg('hashed_password'), hashed_password),
    updated_at = NOW()
WHERE id = sqlc.arg('id') AND status = 'active'
RETURNING *;
--

-- name: VerifyUser :exec
UPDATE users
SET status = 'active',
    updated_at = NOW()
WHERE id = $1 AND status = 'unverified';
--

-- name: AnonymizeUser :exec
UPDATE users
SET
    email = CONCAT('deleted_', id, '@deleted.local'),
    hashed_password = NULL,
    nickname = CONCAT('deleted_', id),
    real_name = 'Deleted User',
    status = 'deleted',
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND status != 'deleted';
--

-- name: GetBirthDateById :one
SELECT birth_date FROM users
WHERE id = $1;
--

-- name: GetUsersByNickname :many
SELECT id, nickname, real_name
FROM users
WHERE nickname ILIKE $1 AND id != $2 AND status NOT IN ('deactivated', 'deleted')
LIMIT 20;

-- name: GetUserSingleByID :one
SELECT id, nickname, real_name, status, created_at
FROM users
WHERE id = $1;

