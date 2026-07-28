-- name: CreateUserRelationship :exec
INSERT INTO user_relationships(id, action_user_id, target_user_id, status, initial_message, created_at, updated_at)
VALUES(
    $1,
    $2,
    $3,
    $4,
    $5,
    NOW() AT TIME ZONE 'UTC',
    NOW() AT TIME ZONE 'UTC'
);
--

-- name: UpdateUserRelationship :exec
UPDATE user_relationships
SET
    status = $1
WHERE action_user_id = $2 AND target_user_id = $3;
--
