-- +goose Up
ALTER TABLE promotion_outbox
    ADD COLUMN claim_token UUID,
    ADD COLUMN claimed_until TIMESTAMPTZ;

CREATE INDEX promotion_outbox_claimable_idx
    ON promotion_outbox (claimed_until, created_at, event_id)
    WHERE published_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS promotion_outbox_claimable_idx;

ALTER TABLE promotion_outbox
    DROP COLUMN IF EXISTS claimed_until,
    DROP COLUMN IF EXISTS claim_token;
