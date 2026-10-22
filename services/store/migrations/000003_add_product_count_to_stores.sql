-- +goose Up

ALTER TABLE stores
    ADD COLUMN product_count INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT stores_product_count_nonnegative
        CHECK (product_count >= 0);

-- +goose Down

ALTER TABLE stores
    DROP CONSTRAINT IF EXISTS stores_product_count_nonnegative,
    DROP COLUMN IF EXISTS product_count;
