-- +goose Up
CREATE TABLE chat_rooms (
    chat_id UUID NOT NULL,
    user_id UUID NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, user_id),
    CONSTRAINT fk_chat_id
        FOREIGN KEY (chat_id)
        REFERENCES chats(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_chat_rooms_user_id ON chat_rooms(user_id);

-- +goose Down
DROP TABLE IF EXISTS chat_rooms;
