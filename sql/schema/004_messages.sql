-- +goose Up
CREATE TABLE messages(
    id UUID PRIMARY KEY,
    content TEXT NOT NULL,
    sender_id UUID NOT NULL,
    chat_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_id
        FOREIGN KEY (sender_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_chat_id
        FOREIGN KEY (chat_id)
        REFERENCES chats(id) ON DELETE CASCADE
);
CREATE INDEX idx_messages_chat_id_id ON messages(chat_id, id DESC);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);

-- +goose Down
DROP TABLE IF EXISTS messages;
