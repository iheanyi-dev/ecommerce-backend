-- +goose Up
CREATE TABLE promotion_outbox (
    event_id UUID PRIMARY KEY,
    promotion_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

-- Intentionally no FK to promotions: pending events must survive promotion deletion.
CREATE INDEX promotion_outbox_pending_idx
    ON promotion_outbox (created_at, event_id)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS promotion_outbox;
