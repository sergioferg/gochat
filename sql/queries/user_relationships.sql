-- name: CreateUserRelationship :one
INSERT INTO user_relationships (
    id, action_user_id, target_user_id, status, initial_message, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, NOW(), NOW()
)
RETURNING *;

-- name: GetRelationshipBetweenUsers :one
SELECT *
FROM user_relationships
WHERE (action_user_id = $1 AND target_user_id = $2)
   OR (action_user_id = $2 AND target_user_id = $1);

-- name: GetPendingRequestsForUser :many
SELECT
    r.id,
    r.action_user_id AS sender_id,
    u.nickname AS sender_nickname,
    u.real_name AS sender_real_name,
    r.initial_message,
    r.created_at
FROM user_relationships r
JOIN users u ON u.id = r.action_user_id
WHERE r.target_user_id = $1 AND r.status = 'pending';

-- name: UpdateRelationshipStatus :exec
UPDATE user_relationships
SET
    status = $1,
    action_user_id = $2,
    target_user_id = $3,
    updated_at = NOW()
WHERE id = $4;

-- name: DeleteRelationship :exec
DELETE FROM user_relationships
WHERE id = $1;

-- name: GetRelationshipByID :one
SELECT id, action_user_id, target_user_id, status, initial_message, created_at, updated_at
FROM user_relationships
WHERE id = $1;
