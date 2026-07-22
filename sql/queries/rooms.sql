-- name: AddChatRoomMember :exec
INSERT INTO chat_rooms (chat_room_id, user_id)
VALUES ($1, $2);
--
