-- +goose Up
CREATE TABLE refresh_tokens(
    token_hash TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    user_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC') + INTERVAL '60 days',
    revoked_at TIMESTAMPTZ,
    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- +goose Down
DROP TABLE refresh_tokens;
