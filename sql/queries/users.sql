-- name: CreateUser :one
INSERT INTO users(id, nickname,  email, hashed_password)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;
--

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;
--

-- name: UpdateUser :one
UPDATE users
SET email = $1,
    hashed_password = $2,
    updated_at = NOW() AT TIME ZONE 'UTC'
WHERE id = $3 AND status = 'active'
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
