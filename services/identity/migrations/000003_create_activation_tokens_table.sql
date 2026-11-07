-- +goose Up

CREATE TABLE activation_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_activation_tokens_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_activation_tokens_user_id
    ON activation_tokens (user_id);

CREATE INDEX idx_activation_tokens_expires_at
    ON activation_tokens (expires_at);

-- +goose Down

DROP TABLE IF EXISTS activation_tokens;
