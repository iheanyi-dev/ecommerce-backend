-- name: CreatePlan :exec
INSERT INTO plans (
    id, type, name, description, price_amount_minor, currency,
    duration_ns, max_products, ai_enabled, available,
    promotion_name, promotion_discount_amount_minor,
    promotion_starts_at, promotion_ends_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16
);

-- name: UpdatePlan :exec
UPDATE plans SET
    type = $2,
    name = $3,
    description = $4,
    price_amount_minor = $5,
    currency = $6,
    duration_ns = $7,
    max_products = $8,
    ai_enabled = $9,
    available = $10,
    promotion_name = $11,
    promotion_discount_amount_minor = $12,
    promotion_starts_at = $13,
    promotion_ends_at = $14,
    updated_at = $15
WHERE id = $1;

-- name: DeletePlanByID :exec
DELETE FROM plans WHERE id = $1;

-- name: GetAllPlans :many
SELECT * FROM plans ORDER BY created_at ASC;

-- name: GetPlanByID :one
SELECT * FROM plans WHERE id = $1;
