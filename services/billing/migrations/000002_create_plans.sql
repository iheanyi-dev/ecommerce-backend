-- +goose Up
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    type VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_amount_minor BIGINT NOT NULL CHECK (price_amount_minor >= 0),
    currency CHAR(3) NOT NULL,
    duration_ns BIGINT NOT NULL CHECK (duration_ns > 0),
    max_products INTEGER NOT NULL CHECK (max_products > 0),
    ai_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    available BOOLEAN NOT NULL DEFAULT FALSE,
    promotion_name VARCHAR(150),
    promotion_discount_amount_minor BIGINT,
    promotion_starts_at TIMESTAMPTZ,
    promotion_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT plans_promotion_check CHECK (
        (promotion_name IS NULL
            AND promotion_discount_amount_minor IS NULL
            AND promotion_starts_at IS NULL
            AND promotion_ends_at IS NULL)
        OR
        (promotion_name IS NOT NULL
            AND promotion_discount_amount_minor > 0
            AND promotion_starts_at IS NOT NULL
            AND promotion_ends_at > promotion_starts_at)
    )
);

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_plan_id_fkey
    FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_fkey;
DROP TABLE IF EXISTS plans;
