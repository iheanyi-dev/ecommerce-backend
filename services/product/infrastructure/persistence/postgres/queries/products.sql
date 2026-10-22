-- name: CreateProduct :exec
INSERT INTO products (
    id, store_id, name, description, price_amount, currency,
    quantity, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(store_id), sqlc.arg(name), sqlc.arg(description),
    sqlc.arg(price_amount), sqlc.arg(currency), sqlc.arg(quantity),
    sqlc.arg(created_at), sqlc.arg(updated_at)
);

-- name: GetProductByID :one
SELECT *
FROM products
WHERE id = sqlc.arg(id);

-- Missing IDs are intentionally omitted.
-- name: GetProductsByIDs :many
SELECT *
FROM products
WHERE id = ANY(sqlc.arg(product_ids)::uuid[])
ORDER BY id;

-- name: UpdateProduct :exec
UPDATE products
SET store_id = sqlc.arg(store_id),
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    price_amount = sqlc.arg(price_amount),
    currency = sqlc.arg(currency),
    quantity = sqlc.arg(quantity),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = sqlc.arg(id);

-- name: ListProductsByStore :many
SELECT *
FROM products
WHERE store_id = sqlc.arg(store_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::integer
OFFSET sqlc.arg(page_offset)::integer;

-- name: CountProductsByStore :one
SELECT COUNT(*)::bigint
FROM products
WHERE store_id = sqlc.arg(store_id);

-- name: CountMarketplaceProducts :one
SELECT COUNT(*)::bigint
FROM products
WHERE (
    sqlc.arg(query)::text = ''
    OR name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR description ILIKE '%' || sqlc.arg(query)::text || '%'
)
AND (
    NOT sqlc.arg(has_min)::boolean
    OR (
        currency = sqlc.arg(min_currency)::text
        AND price_amount >= sqlc.arg(min_amount)::bigint
    )
)
AND (
    NOT sqlc.arg(has_max)::boolean
    OR (
        currency = sqlc.arg(max_currency)::text
        AND price_amount <= sqlc.arg(max_amount)::bigint
    )
)
AND (
    NOT sqlc.arg(in_stock_only)::boolean
    OR quantity > 0
);

-- name: ListMarketplaceProducts :many
SELECT *
FROM products
WHERE (
    sqlc.arg(query)::text = ''
    OR name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR description ILIKE '%' || sqlc.arg(query)::text || '%'
)
AND (
    NOT sqlc.arg(has_min)::boolean
    OR (
        currency = sqlc.arg(min_currency)::text
        AND price_amount >= sqlc.arg(min_amount)::bigint
    )
)
AND (
    NOT sqlc.arg(has_max)::boolean
    OR (
        currency = sqlc.arg(max_currency)::text
        AND price_amount <= sqlc.arg(max_amount)::bigint
    )
)
AND (
    NOT sqlc.arg(in_stock_only)::boolean
    OR quantity > 0
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'price_asc'
        THEN price_amount END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'price_desc'
        THEN price_amount END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'name_asc'
        THEN LOWER(name) END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'newest'
        THEN created_at END DESC,
    created_at DESC,
    id DESC
LIMIT sqlc.arg(page_limit)::integer
OFFSET sqlc.arg(page_offset)::integer;

-- name: SearchProductsByStore :many
SELECT *
FROM products
WHERE store_id = sqlc.arg(store_id)
AND (
    name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR description ILIKE '%' || sqlc.arg(query)::text || '%'
)
AND (
    sqlc.arg(currency)::text = ''
    OR currency = sqlc.arg(currency)::text
)
AND (
    NOT sqlc.arg(has_min)::boolean
    OR (
        currency = sqlc.arg(currency)::text
        AND (
            price_amount > sqlc.arg(min_amount)::bigint
            OR (
                sqlc.arg(min_inclusive)::boolean
                AND price_amount = sqlc.arg(min_amount)::bigint
            )
        )
    )
)
AND (
    NOT sqlc.arg(has_max)::boolean
    OR (
        currency = sqlc.arg(currency)::text
        AND (
            price_amount < sqlc.arg(max_amount)::bigint
            OR (
                sqlc.arg(max_inclusive)::boolean
                AND price_amount = sqlc.arg(max_amount)::bigint
            )
        )
    )
)
AND (
    NOT sqlc.arg(available_only)::boolean
    OR quantity > 0
)
ORDER BY LOWER(name), id;

-- name: GetProductImages :many
SELECT product_id, position, storage_reference
FROM product_images
WHERE product_id = sqlc.arg(product_id)
ORDER BY position;

-- name: GetProductImagesByProductIDs :many
SELECT product_id, position, storage_reference
FROM product_images
WHERE product_id = ANY(sqlc.arg(product_ids)::uuid[])
ORDER BY product_id, position;

-- name: DeleteProductImages :exec
DELETE FROM product_images
WHERE product_id = sqlc.arg(product_id);

-- name: CreateProductImage :exec
INSERT INTO product_images (product_id, position, storage_reference)
VALUES (
    sqlc.arg(product_id),
    sqlc.arg(position),
    sqlc.arg(storage_reference)
);
