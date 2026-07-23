-- +goose Up
CREATE TABLE users(
    id UUID PRIMARY KEY,
    nickname TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT,
    status TEXT NOT NULL DEFAULT 'unverified',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ DEFAULT NULL,

    CONSTRAINT users_status_check CHECK (status IN ('unverified', 'active', 'deactivated', 'deleted'))
);

-- +goose Down
DROP TABLE IF EXISTS users;
