-- name: CreatePromotionIfNoConflict :execrows
WITH target_lock AS MATERIALIZED (
    SELECT pg_advisory_xact_lock(
        hashtextextended(
            sqlc.arg(store_id)::uuid::text || ':' ||
            COALESCE(sqlc.narg(product_id)::uuid::text, '<nil>') || ':' ||
            COALESCE(sqlc.narg(customer_id)::uuid::text, '<nil>'),
            0
        )
    )
)
INSERT INTO promotions (
    id, store_id, product_id, customer_id,
    percentage_basis_points, fixed_amount, fixed_currency,
    starts_at, ends_at, status, created_at, updated_at
)
SELECT
    sqlc.arg(id),
    sqlc.arg(store_id),
    sqlc.narg(product_id)::uuid,
    sqlc.narg(customer_id)::uuid,
    sqlc.narg(percentage_basis_points),
    sqlc.narg(fixed_amount),
    sqlc.narg(fixed_currency),
    sqlc.arg(starts_at),
    sqlc.arg(ends_at),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
FROM target_lock
WHERE NOT EXISTS (
    SELECT 1
    FROM promotions AS existing
    WHERE existing.store_id = sqlc.arg(store_id)
      AND existing.product_id IS NOT DISTINCT FROM sqlc.narg(product_id)::uuid
      AND existing.customer_id IS NOT DISTINCT FROM sqlc.narg(customer_id)::uuid
      AND existing.ends_at > sqlc.arg(now)
);

-- name: GetPromotionByID :one
SELECT *
FROM promotions
WHERE id = $1;

-- name: UpdatePromotion :execrows
UPDATE promotions
SET
    percentage_basis_points = sqlc.narg(percentage_basis_points),
    fixed_amount = sqlc.narg(fixed_amount),
    fixed_currency = sqlc.narg(fixed_currency),
    starts_at = sqlc.arg(starts_at),
    ends_at = sqlc.arg(ends_at),
    status = sqlc.arg(status),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeletePromotion :execrows
DELETE FROM promotions
WHERE id = $1;

-- name: ListPromotionsByStore :many
SELECT *
FROM promotions
WHERE store_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: ListActivePromotionsForTarget :many
SELECT *
FROM promotions
WHERE store_id = sqlc.arg(store_id)
  AND status = 'active'
  AND starts_at <= sqlc.arg(now)
  AND ends_at > sqlc.arg(now)
  AND (product_id IS NULL OR product_id = sqlc.arg(product_id))
  AND (customer_id IS NULL OR customer_id = sqlc.arg(customer_id))
ORDER BY created_at DESC, id DESC;

-- name: ListAvailablePromotionsForCustomer :many
SELECT *
FROM promotions
WHERE ends_at > sqlc.arg(now)
  AND status IN ('scheduled', 'active', 'paused')
  AND (customer_id IS NULL OR customer_id = sqlc.arg(customer_id))
  AND (product_id IS NULL OR customer_id = sqlc.arg(customer_id))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);


-- name: ListScheduledPromotionsDue :many
SELECT id
FROM promotions
WHERE status = 'scheduled'
  AND starts_at <= sqlc.arg(now)
  AND ends_at > sqlc.arg(now)
ORDER BY starts_at ASC, id ASC
LIMIT sqlc.arg(batch_size);

-- name: ListExpiredPromotionsDue :many
SELECT id
FROM promotions
WHERE status IN ('scheduled', 'active')
  AND ends_at <= sqlc.arg(now)
ORDER BY ends_at ASC, id ASC
LIMIT sqlc.arg(batch_size);
