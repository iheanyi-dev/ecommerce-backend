-- name: CreatePayout :exec
INSERT INTO payouts (
id,
attempt_id,
reference,
business_reference_type,
business_reference_id,
store_id,
order_id,
reason,
beneficiary_snapshot_encrypted,
amount,
currency,
status,
created_at,
updated_at
) VALUES (
$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: GetPayoutByID :one
SELECT *
FROM payouts
WHERE id = $1
LIMIT 1;

-- name: GetPayoutByBusinessReference :one
SELECT *
FROM payouts
WHERE business_reference_type = $1
AND business_reference_id = $2
LIMIT 1;

-- name: ListPayoutsByStoreID :many
SELECT *
FROM payouts
WHERE store_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountPayoutsByStoreID :one
SELECT COUNT(*)
FROM payouts
WHERE store_id = $1;

-- name: UpdatePayout :exec
UPDATE payouts
SET attempt_id = $2,
status = $3,
updated_at = $4
WHERE id = $1;
