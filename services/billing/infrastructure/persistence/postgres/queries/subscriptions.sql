-- name: CreateSubscription :exec
INSERT INTO subscriptions (
    id,
    store_id,
    plan_id,
    price_amount_minor,
    currency,
    duration_ns,
    status,
    started_at,
    expires_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateSubscription :exec
UPDATE subscriptions
SET
    status = $2,
    started_at = $3,
    expires_at = $4,
    updated_at = $5
WHERE id = $1;

-- name: GetSubscriptionByID :one
SELECT *
FROM subscriptions
WHERE id = $1;

-- name: GetSubscriptionByIDForUpdate :one
SELECT *
FROM subscriptions
WHERE id = $1
FOR UPDATE;

-- name: GetCurrentSubscriptionByStore :one
SELECT *
FROM subscriptions
WHERE store_id = $1
  AND status IN ('pending', 'active')
LIMIT 1
FOR UPDATE;

-- name: GetSubscriptionsByStoreID :many
SELECT *
FROM subscriptions
WHERE store_id = $1
ORDER BY created_at DESC, id DESC;


-- name: GetDueSubscriptionsForUpdate :many
SELECT *
FROM subscriptions
WHERE status = 'active'
  AND expires_at <= $1
ORDER BY expires_at ASC, id ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;
