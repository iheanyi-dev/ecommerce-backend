-- +goose Up

CREATE TABLE product_promotion_index (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL,
    product_id UUID NULL,
    promotion_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL CHECK (LENGTH(TRIM(status)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A store-level promotion has one row per promotion.
CREATE UNIQUE INDEX idx_product_promotion_store_unique
    ON product_promotion_index (promotion_id)
    WHERE product_id IS NULL;

-- A product-level promotion has one row per promotion and product.
CREATE UNIQUE INDEX idx_product_promotion_product_unique
    ON product_promotion_index (promotion_id, product_id)
    WHERE product_id IS NOT NULL;

CREATE INDEX idx_product_promotion_store_id
    ON product_promotion_index (store_id);

CREATE INDEX idx_product_promotion_product_lookup
    ON product_promotion_index (store_id, product_id);

-- +goose Down

DROP TABLE IF EXISTS product_promotion_index;
