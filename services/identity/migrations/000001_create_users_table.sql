-- +goose Up

-- Creates the users table owned exclusively by the Identity service.
--
-- No other microservice should access this table directly.
-- Other services communicate with Identity through APIs and events.

CREATE TABLE users (
    id UUID PRIMARY KEY,

    full_name VARCHAR(150) NOT NULL,

    -- Email is immutable in the User aggregate.
    email VARCHAR(320) NOT NULL UNIQUE,

    password_hash TEXT NOT NULL,

    -- The domain currently supports:
    -- user, vendor, admin.
    role VARCHAR(20) NOT NULL,

    -- User status is mutable and controlled by the domain.
    status VARCHAR(50) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,

    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_users_email ON users (email);

-- +goose Down

DROP TABLE IF EXISTS users;