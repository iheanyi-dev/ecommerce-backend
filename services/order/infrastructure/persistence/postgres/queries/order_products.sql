-- name: CreateOrderProduct :exec
INSERT INTO order_products (id, order_id, product_id, name, qty, price)
VALUES (
    sqlc.arg(id), sqlc.arg(order_id), sqlc.arg(product_id),
    sqlc.arg(name), sqlc.arg(qty), sqlc.arg(price)
);

-- name: FindOrderProductsByOrderIDs :many
SELECT id, order_id, product_id, name, qty, price
FROM order_products
WHERE order_id = ANY(sqlc.arg(order_ids)::uuid[])
ORDER BY order_id, id;
