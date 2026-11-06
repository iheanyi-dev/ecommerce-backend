-- name: CreatePaymentOutboxEvent :exec
INSERT INTO payment_outbox (
    event_id,
    aggregate_type,
    aggregate_id,
    deduplication_key,
    event_type,
    payload
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (deduplication_key) DO NOTHING;

-- name: GetPendingPaymentOutboxEvents :many
SELECT event_id, event_type, payload, attempts
FROM payment_outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkPaymentOutboxEventPublished :exec
UPDATE payment_outbox
SET published_at = NOW(),
    last_error = NULL
WHERE event_id = $1
  AND published_at IS NULL;

-- name: RecordPaymentOutboxEventFailure :exec
UPDATE payment_outbox
SET attempts = attempts + 1,
    last_error = $2
WHERE event_id = $1
  AND published_at IS NULL;
