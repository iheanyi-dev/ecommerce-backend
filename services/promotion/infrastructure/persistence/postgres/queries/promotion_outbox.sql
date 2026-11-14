-- name: CreatePromotionOutbox :exec
INSERT INTO promotion_outbox (
    event_id, promotion_id, event_type, payload
)
VALUES ($1, $2, $3, $4);

-- name: ClaimPendingPromotionOutbox :many
WITH candidates AS (
    SELECT outbox.event_id
    FROM promotion_outbox AS outbox
    WHERE outbox.published_at IS NULL
      AND (
          outbox.claimed_until IS NULL
          OR outbox.claimed_until <= sqlc.arg(now)
      )
      AND NOT EXISTS (
          SELECT 1
          FROM promotion_outbox AS earlier
          WHERE earlier.promotion_id = outbox.promotion_id
            AND earlier.published_at IS NULL
            AND (
                earlier.created_at,
                earlier.event_id
            ) < (
                outbox.created_at,
                outbox.event_id
            )
      )
    ORDER BY outbox.created_at ASC, outbox.event_id ASC
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF outbox SKIP LOCKED
)
UPDATE promotion_outbox AS outbox
SET claim_token = sqlc.arg(claim_token),
    claimed_until = sqlc.arg(lease_until)
FROM candidates
WHERE outbox.event_id = candidates.event_id
RETURNING
    outbox.event_id,
    outbox.promotion_id,
    outbox.event_type,
    outbox.payload,
    outbox.attempts;

-- name: MarkPromotionOutboxPublished :execrows
UPDATE promotion_outbox
SET published_at = NOW(),
    last_error = NULL,
    claim_token = NULL,
    claimed_until = NULL
WHERE event_id = $1
  AND claim_token = $2
  AND published_at IS NULL;

-- name: RecordPromotionOutboxFailure :execrows
UPDATE promotion_outbox
SET attempts = attempts + 1,
    last_error = $3,
    claim_token = NULL,
    claimed_until = NULL
WHERE event_id = $1
  AND claim_token = $2
  AND published_at IS NULL;
