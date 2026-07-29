-- name: AddUserToChat :exec
INSERT INTO chat_participants (chat_id, user_id)
VALUES ($1, $2);
--

-- name: UpdateLastRead :exec
UPDATE chat_participants
SET last_read_at = NOW()
WHERE chat_id = $1 AND user_id = $2;
--

-- name: GetChatParticipantIDs :many
SELECT user_id FROM chat_participants
WHERE chat_id = $1;
--
