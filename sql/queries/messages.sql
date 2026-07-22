-- name: GetMessagesBefore :many
SELECT * FROM messages
WHERE chat_id = $1
    AND id < $2
ORDER BY id DESC
LIMIT 50;
--

-- name: UpdateMessage :one
UPDATE messages
SET content = $1,
    updated_at = NOW() AT TIME ZONE 'UTC'
WHERE id = $2
RETURNING *;
--

-- name: CreateMessage :one
INSERT INTO messages(id, content, sender_id, chat_id)
VALUES(
    $1,
    $2,
    $3,
    $4
)
RETURNING *;
--
