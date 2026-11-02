-- name: CreatePayoutAttempt :exec
INSERT INTO payout_attempts (
id,
payout_id,
provider,
provider_reference,
status,
failure_reason,
created_at,
updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetPayoutAttemptByID :one
SELECT *
FROM payout_attempts
WHERE id = $1
LIMIT 1;

-- name: GetLatestPayoutAttemptByPayoutID :one
SELECT *
FROM payout_attempts
WHERE payout_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdatePayoutAttempt :exec
UPDATE payout_attempts
SET provider_reference = $2,
status = $3,
failure_reason = $4,
updated_at = $5
WHERE id = $1;
