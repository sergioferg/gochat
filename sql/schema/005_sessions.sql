-- +goose Up
CREATE TABLE sessions(
    id UUID PRIMARY KEY,
    token_hash TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    user_agent TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '60 days',
    revoked_at TIMESTAMPTZ,
    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);
CREATE INDEX idx_refresh_tokens_user_id ON sessions(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON sessions(expires_at);

-- +goose Down
DROP TABLE sessions;
