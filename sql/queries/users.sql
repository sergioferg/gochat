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
WHERE id = $3
RETURNING *;
--

-- name: VerifyUser :one
UPDATE users
SET is_verified = TRUE,
    updated_at = NOW() AT TIME ZONE 'UTC'
WHERE id = $1
RETURNING *;
--
