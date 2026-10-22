-- +goose Up

-- Product owns its product records. Store IDs are external identifiers, so
-- this service deliberately does not create a cross-service foreign key.
CREATE TABLE products (
    id UUID PRIMARY KEY,
    store_id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    price_amount BIGINT NOT NULL CHECK (price_amount >= 0),
    currency CHAR(3) NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX products_store_id_created_at_idx
    ON products (store_id, created_at DESC);

CREATE INDEX products_price_currency_idx
    ON products (currency, price_amount);

-- Position is explicit and stable, so image order survives every read.
CREATE TABLE product_images (
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    position SMALLINT NOT NULL CHECK (position >= 0 AND position < 4),
    storage_reference TEXT NOT NULL,
    PRIMARY KEY (product_id, position),
    CONSTRAINT product_images_reference_unique
        UNIQUE (product_id, storage_reference)
);

CREATE INDEX product_images_product_id_idx
    ON product_images (product_id, position);

-- +goose Down

DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS products;
