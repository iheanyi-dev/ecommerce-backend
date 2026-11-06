-- +goose Up
ALTER TABLE payment_outbox
    ADD COLUMN aggregate_type VARCHAR(50),
    ADD COLUMN aggregate_id UUID,
    ADD COLUMN deduplication_key VARCHAR(255);

UPDATE payment_outbox
SET aggregate_type = 'payment',
    aggregate_id = payment_id,
    deduplication_key = payment_id::TEXT || ':' || event_type;

ALTER TABLE payment_outbox
    ALTER COLUMN aggregate_type SET NOT NULL,
    ALTER COLUMN aggregate_id SET NOT NULL,
    ALTER COLUMN deduplication_key SET NOT NULL;

ALTER TABLE payment_outbox
    DROP CONSTRAINT IF EXISTS payment_outbox_payment_event_unique,
    DROP CONSTRAINT IF EXISTS payment_outbox_payment_id_fkey,
    DROP COLUMN payment_id,
    ADD CONSTRAINT payment_outbox_deduplication_key_unique
        UNIQUE (deduplication_key);

-- +goose Down
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM payment_outbox
        WHERE aggregate_type <> 'payment'
    ) THEN
        RAISE EXCEPTION
            'Cannot reverse generic outbox migration while non-payment events exist';
    END IF;
END;
$$;

ALTER TABLE payment_outbox
    ADD COLUMN payment_id UUID;

UPDATE payment_outbox
SET payment_id = aggregate_id;

ALTER TABLE payment_outbox
    ALTER COLUMN payment_id SET NOT NULL;

ALTER TABLE payment_outbox
    ADD CONSTRAINT payment_outbox_payment_id_fkey
        FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE CASCADE,
    ADD CONSTRAINT payment_outbox_payment_event_unique
        UNIQUE (payment_id, event_type);

ALTER TABLE payment_outbox
    DROP CONSTRAINT payment_outbox_deduplication_key_unique,
    DROP COLUMN aggregate_type,
    DROP COLUMN aggregate_id,
    DROP COLUMN deduplication_key;
