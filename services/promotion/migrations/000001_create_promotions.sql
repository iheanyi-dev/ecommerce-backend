-- +goose Up
CREATE TABLE promotions (
    id UUID PRIMARY KEY,
    store_id UUID NOT NULL,
    product_id UUID,
    customer_id UUID,

    percentage_basis_points BIGINT,
    fixed_amount BIGINT,
    fixed_currency VARCHAR(3),

    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT promotions_valid_dates CHECK (ends_at > starts_at),
    CONSTRAINT promotions_valid_status CHECK (
        status IN ('scheduled', 'active', 'expired', 'paused', 'cancelled')
    ),
    CONSTRAINT promotions_valid_percentage CHECK (
        percentage_basis_points IS NULL OR
        percentage_basis_points BETWEEN 1 AND 10000
    ),
    CONSTRAINT promotions_valid_fixed_amount CHECK (
        (fixed_amount IS NULL AND fixed_currency IS NULL) OR
        (fixed_amount > 0 AND fixed_currency IS NOT NULL)
    ),
    CONSTRAINT promotions_valid_discount CHECK (
        percentage_basis_points IS NOT NULL OR fixed_amount IS NOT NULL
    )
);

CREATE INDEX promotions_store_created_idx
    ON promotions (store_id, created_at DESC, id DESC);

CREATE INDEX promotions_target_expiry_idx
    ON promotions (store_id, product_id, customer_id, ends_at);

CREATE INDEX promotions_customer_expiry_idx
    ON promotions (customer_id, ends_at);

-- +goose Down
DROP TABLE IF EXISTS promotions;
