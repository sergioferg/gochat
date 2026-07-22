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
SELECT cr.id
FROM chats cr
JOIN chat_rooms cr1 ON cr.id = cr1.chat_id
JOIN chat_rooms cr2 ON cr.id = cr2.chat_id
WHERE cr.is_group = FALSE
  AND cr1.user_id = $1
  AND cr2.user_id = $2
LIMIT 1;
--
