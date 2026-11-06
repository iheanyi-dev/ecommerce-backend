-- name: CreateSubscriptionOutboxEvent :exec
INSERT INTO subscription_outbox (
    event_id, subscription_id, event_type, payload
) VALUES ($1, $2, $3, $4);

-- name: GetPendingSubscriptionOutboxEvents :many
SELECT event_id, payload, attempts
FROM subscription_outbox
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkSubscriptionOutboxEventPublished :exec
UPDATE subscription_outbox
SET published_at = NOW(), last_error = NULL
WHERE event_id = $1 AND published_at IS NULL;

-- name: RecordSubscriptionOutboxEventFailure :exec
UPDATE subscription_outbox
SET attempts = attempts + 1, last_error = $2
WHERE event_id = $1 AND published_at IS NULL;
