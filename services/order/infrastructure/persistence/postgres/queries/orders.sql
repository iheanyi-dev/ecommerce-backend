-- name: CreateOrder :exec
INSERT INTO orders (
    id, cart_id, store_id, customer_id, delivery_location,
    price_product, price_shipping, currency, state, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(cart_id), sqlc.arg(store_id), sqlc.arg(customer_id),
    sqlc.arg(delivery_location), sqlc.arg(price_product), sqlc.arg(price_shipping),
    sqlc.arg(currency), sqlc.arg(state), sqlc.arg(created_at), sqlc.arg(updated_at)
);

-- name: GetOrderByID :one
SELECT id, cart_id, store_id, customer_id, delivery_location,
       price_product, price_shipping, currency, state, created_at, updated_at
FROM orders
WHERE id = sqlc.arg(id);

-- name: UpdateOrder :execrows
UPDATE orders
SET price_shipping = sqlc.arg(price_shipping),
    state = sqlc.arg(state),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ListOrdersByCustomerID :many
SELECT id, cart_id, store_id, customer_id, delivery_location,
       price_product, price_shipping, currency, state, created_at, updated_at
FROM orders
WHERE customer_id = sqlc.arg(customer_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::bigint
OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountOrdersByCustomerID :one
SELECT COUNT(*) FROM orders
WHERE customer_id = sqlc.arg(customer_id);

-- name: ListOrdersByStoreID :many
SELECT id, cart_id, store_id, customer_id, delivery_location,
       price_product, price_shipping, currency, state, created_at, updated_at
FROM orders
WHERE store_id = sqlc.arg(store_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::bigint
OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountOrdersByStoreID :one
SELECT COUNT(*) FROM orders
WHERE store_id = sqlc.arg(store_id);
