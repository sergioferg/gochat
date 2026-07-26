-- name: CreateChat :one
INSERT INTO chats(id, name, is_group)
VALUES(
    $1,
    $2,
    $3
)
RETURNING *;
--

-- name: GetDirectChatBetweenUsers :one
SELECT c.id
FROM chats c
JOIN chat_participants cp1 ON c.id = cp1.chat_id
JOIN chat_participants cp2 ON c.id = cp2.chat_id
WHERE c.is_group = FALSE
  AND cp1.user_id = $1
  AND cp2.user_id = $2
LIMIT 1;
--

-- name: GetUserChats :many
SELECT
    c.id AS chat_id,
    c.name AS chat_name,
    c.is_group,
    cp.last_read_at,
    COALESCE((SELECT content FROM messages m WHERE m.chat_id = c.id ORDER BY m.id DESC LIMIT 1), '') AS last_message_content,
    COALESCE((SELECT id FROM messages m WHERE m.chat_id = c.id ORDER BY m.id DESC LIMIT 1), '00000000-0000-0000-0000-000000000000'::uuid) AS last_message_id
FROM chats c
JOIN chat_participants cp ON c.id = cp.chat_id
WHERE cp.user_id = $1
ORDER BY last_message_id DESC;
--
