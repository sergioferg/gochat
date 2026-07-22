-- name: GetMessagesBefore :many
SELECT * FROM messages
WHERE chat_room_id = $1
    AND id < $2
ORDER BY id DESC
LIMIT 50;
--

-- name: UpdateMessage :one
UPDATE
-- 
