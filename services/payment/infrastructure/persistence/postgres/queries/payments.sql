-- name: CreatePayment :exec
INSERT INTO payments (
    id, attempt_id, reference, payer_id, purpose,
    business_reference_type, business_reference_id,
    amount, currency, status, created_at, updated_at
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7,
    $8, $9, $10, $11, $12
);

-- name: GetPaymentByID :one
SELECT *
FROM payments
WHERE id = $1;

-- name: GetPaymentByBusinessReference :one
SELECT *
FROM payments
WHERE business_reference_type = $1
  AND business_reference_id = $2;

-- name: ListPaymentsByPayerID :many
SELECT *
FROM payments
WHERE payer_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2
OFFSET $3;

-- name: CountPaymentsByPayerID :one
SELECT COUNT(*)
FROM payments
WHERE payer_id = $1;

-- name: UpdatePayment :exec
UPDATE payments
SET
    attempt_id = $2,
    status = $3,
    updated_at = $4
WHERE id = $1;
