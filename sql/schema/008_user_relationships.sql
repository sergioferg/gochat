-- +goose Up
CREATE TABLE user_relationships (
    id UUID PRIMARY KEY,
    action_user_id UUID NOT NULL,
    target_user_id UUID NOT NULL,
    status TEXT NOT NULL,
    initial_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_action_user
        FOREIGN KEY (action_user_id)
        REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT fk_target_user
        FOREIGN KEY (target_user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT prevent_self_action
        CHECK (action_user_id != target_user_id),

    CONSTRAINT relationship_status_check
        CHECK (status IN ('pending', 'accepted', 'blocked')),

    CONSTRAINT initial_message_length_check
        CHECK (char_length(initial_message) <= 500)
);

CREATE UNIQUE INDEX unique_user_pair
ON user_relationships (
    LEAST(action_user_id, target_user_id),
    GREATEST(action_user_id, target_user_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_relationships;
