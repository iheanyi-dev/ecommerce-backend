-- +goose Up
CREATE INDEX promotions_scheduled_start_idx
    ON promotions (starts_at, id)
    WHERE status = 'scheduled';

CREATE INDEX promotions_expirable_end_idx
    ON promotions (ends_at, id)
    WHERE status IN ('scheduled', 'active');

-- +goose Down
DROP INDEX IF EXISTS promotions_expirable_end_idx;
DROP INDEX IF EXISTS promotions_scheduled_start_idx;
