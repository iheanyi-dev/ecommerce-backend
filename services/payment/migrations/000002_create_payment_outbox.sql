-- +goose Up
CREATE TABLE payment_outbox (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    CONSTRAINT payment_outbox_payment_event_unique UNIQUE (payment_id, event_type)
);

CREATE INDEX payment_outbox_pending_idx
    ON payment_outbox (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS payment_outbox;
