-- +goose Up
CREATE TABLE oauth_accounts(
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,          -- 'google', 'github', 'apple'
    provider_user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    UNIQUE(provider, provider_user_id)
);

-- +goose Down
DROP TABLE IF EXISTS oauth_accounts;
