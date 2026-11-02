-- name: CreatePaymentAttempt :exec
INSERT INTO payment_attempts (
    id, payment_id, provider, provider_reference,
    customer_email, checkout_url, expires_at,
    metadata, status, failure_reason, created_at, updated_at
)
VALUES (
    $1, $2, $3, $4,
    $5, $6, $7,
    $8, $9, $10, $11, $12
);

-- name: GetPaymentAttemptByID :one
SELECT *
FROM payment_attempts
WHERE id = $1;

-- name: UpdatePaymentAttempt :exec
UPDATE payment_attempts
SET
    provider_reference = $2,
    customer_email = $3,
    checkout_url = $4,
    expires_at = $5,
    metadata = $6,
    status = $7,
    failure_reason = $8,
    updated_at = $9
WHERE id = $1;
