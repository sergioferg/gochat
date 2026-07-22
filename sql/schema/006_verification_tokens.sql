-- +goose Up
CREATE TABLE email_verification_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT ((NOW() AT TIME ZONE 'UTC') + INTERVAL '24 hours'),

    CONSTRAINT fk_user_id
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_expires_at ON email_verification_tokens(expires_at);

-- +goose Down
DROP TABLE email_verification_tokens;
