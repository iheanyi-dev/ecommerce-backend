-- +goose Up
-- Orders and their product snapshots are stored separately.
-- Order creation and product-line insertion must share one transaction.

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    cart_id UUID NOT NULL,
    store_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    delivery_location TEXT NOT NULL,
    price_product BIGINT NOT NULL CHECK (price_product >= 0),
    price_shipping BIGINT NOT NULL DEFAULT 0 CHECK (price_shipping >= 0),
    currency VARCHAR(3) NOT NULL,
    state VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX orders_customer_created_at_idx
    ON orders (customer_id, created_at DESC, id);

CREATE INDEX orders_store_created_at_idx
    ON orders (store_id, created_at DESC, id);

CREATE INDEX orders_cart_id_idx
    ON orders (cart_id);

CREATE TABLE order_products (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    name TEXT NOT NULL,
    qty BIGINT NOT NULL CHECK (qty > 0),
    price BIGINT NOT NULL CHECK (price >= 0)
);

CREATE INDEX order_products_order_id_idx
    ON order_products (order_id);

-- +goose Down
DROP TABLE IF EXISTS order_products;
DROP TABLE IF EXISTS orders;
