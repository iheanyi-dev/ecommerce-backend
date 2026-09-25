-- +goose Up

-- Stores are owned by Identity users through owner_id.
-- The foreign key is intentionally not used here because Store owns its
-- database independently from Identity and therefore cannot enforce a
-- cross-service relational constraint.
CREATE TABLE stores (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500) NOT NULL DEFAULT '',
    image_reference TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    plan VARCHAR(20) NOT NULL DEFAULT 'basic',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT stores_status_check
        CHECK (status IN ('active', 'inactive')),

    CONSTRAINT stores_plan_check
        CHECK (plan IN ('basic', 'premium'))
);

-- One user can own at most one Store.
CREATE UNIQUE INDEX stores_owner_id_unique
    ON stores (owner_id);

-- Public active-store discovery is performed by status and can optionally
-- filter by name/slug. This index supports the status-first portion of that
-- access pattern.
CREATE INDEX stores_status_idx
    ON stores (status);

-- +goose Down

DROP TABLE IF EXISTS stores;
