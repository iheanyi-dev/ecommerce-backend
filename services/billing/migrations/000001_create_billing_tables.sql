-- +goose Up
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    store_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    price_amount_minor BIGINT NOT NULL CHECK (price_amount_minor >= 0),
    currency CHAR(3) NOT NULL,
    duration_ns BIGINT NOT NULL CHECK (duration_ns > 0),
    status VARCHAR(20) NOT NULL CHECK (
        status IN ('pending', 'active', 'failed', 'expired', 'cancelled')
    ),
    started_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT subscriptions_period_check CHECK (
        (status IN ('pending', 'failed') AND started_at IS NULL AND expires_at IS NULL)
        OR
        (status IN ('active', 'expired') AND started_at IS NOT NULL AND expires_at IS NOT NULL)
        OR
        (status = 'cancelled' AND (
            (started_at IS NOT NULL AND expires_at IS NOT NULL)
            OR (started_at IS NULL AND expires_at IS NULL)
        ))
    )
);

CREATE INDEX subscriptions_store_created_idx
    ON subscriptions (store_id, created_at DESC);

CREATE UNIQUE INDEX subscriptions_one_current_per_store_idx
    ON subscriptions (store_id)
    WHERE status IN ('pending', 'active');

CREATE TABLE subscription_outbox (
    event_id UUID PRIMARY KEY,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX subscription_outbox_pending_idx
    ON subscription_outbox (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS subscription_outbox;
DROP TABLE IF EXISTS subscriptions;
