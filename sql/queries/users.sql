-- name: CreateUser :one
INSERT INTO users(id, nickname,  email, hashed_password, status)
VALUES (
    $1,
    $2,
    $3,
    $4,
    COALESCE(sqlc.narg('status'), 'unverified')
)
RETURNING *;
--

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;
--

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE(sqlc.narg('email'), email),
    nickname = COALESCE(sqlc.narg('nickname'), nickname),
    hashed_password = COALESCE(sqlc.narg('hashed_password'), hashed_password),
    updated_at = NOW() AT TIME ZONE 'UTC'
WHERE id = sqlc.arg('id') AND status = 'active'
RETURNING *;
--

-- name: VerifyUser :exec
UPDATE users
SET status = 'active',
    updated_at = NOW() AT TIME ZONE 'UTC'
WHERE id = $1 AND status = 'unverified';
--

-- name: AnonymizeUser :exec
UPDATE users
SET
    email = CONCAT('deleted_', id, '@deleted.local'),
    hashed_password = NULL,
    nickname = 'Deleted User',
    status = 'deleted',
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status != 'deleted';
--
